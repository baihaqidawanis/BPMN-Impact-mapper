package index

import (
	"fmt"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// TextWindowAdapter is a generic, query-independent text adapter, not an AST.
// Keep the complete file plus at most eight contiguous windows. Large files
// use larger windows to cover all text; semantic passage limits still apply.
type TextWindowAdapter struct{ CodeOnly bool }

func (a TextWindowAdapter) Name() string {
	if a.CodeOnly {
		return "generic-code-windows-v1-min1200-max8"
	}
	return "generic-text-windows-v1-min1200-max8"
}
func (TextWindowAdapter) Capabilities() []domain.Capability {
	return []domain.Capability{domain.TextOnly}
}
func (a TextWindowAdapter) Supports(file ingest.File) bool {
	if (GoAdapter{}).Supports(file) {
		return false
	}
	if !a.CodeOnly {
		return true
	}
	switch language(file.RelativePath) {
	case "JavaScript", "TypeScript", "Java", "Python", "C#":
		return true
	}
	return false
}
func (TextWindowAdapter) Index(file ingest.File) ([]domain.Artifact, error) {
	parent := baseArtifact(file, string(file.Content))
	parent.ID = file.RelativePath
	result := []domain.Artifact{parent}
	runes := []rune(parent.Content)
	if len(runes) <= 1200 {
		return result, nil
	}
	size := max(1200, (len(runes)+7)/8)
	line := 1
	for start := 0; start < len(runes); start += size {
		end := min(len(runes), start+size)
		content := string(runes[start:end])
		window := parent
		window.ID = fmt.Sprintf("%s#text-window:%d:%d", file.RelativePath, start, end)
		window.Content = content
		window.Symbol = domain.Symbol{Kind: "text_window", StartLine: line, EndLine: line + strings.Count(content, "\n")}
		window.Tokens = textutil.Tokenize(file.RelativePath + " " + content)
		result = append(result, window)
		line += strings.Count(content, "\n")
	}
	return result, nil
}
