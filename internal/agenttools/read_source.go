package agenttools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// readSource reads an eight-line window from the immutable, ingestion-approved
// snapshot. Query locates the window; start_line pages it. No live path is read.
func (s *Snapshot) readSource(ctx context.Context, q planning.ToolRequest) (planning.ToolResult, error) {
	out := planning.ToolResult{Candidates: []planning.Candidate{}}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	content, ok := s.files[q.File]
	if !safeFile(q.File) || !ok || q.StartLine < 0 {
		return out, errors.New("source file/window unavailable in approved snapshot")
	}
	lines := strings.Split(content, "\n")
	start := max(0, q.StartLine-1)
	matched := -1
	if q.Query != "" {
		found := false
		for i := start; i < len(lines); i++ {
			if strings.Contains(strings.ToLower(lines[i]), strings.ToLower(q.Query)) {
				start = max(0, i-2)
				matched = i
				found = true
				break
			}
		}
		if !found {
			return out, errors.New("query absent from this source window onward")
		}
	}
	if start >= len(lines) {
		return out, errors.New("source line beyond file")
	}
	c := planning.Candidate{File: q.File, Origin: "agent-source-tool", SuggestedAction: domain.ActionVerify}
	for _, a := range s.artifacts {
		if a.File == q.File {
			c.Module = a.Module
			c.Layer = a.Layer
			break
		}
	}
	end := min(start+8, len(lines))
	for i := start; i < end; i++ {
		text := []rune(strings.TrimSpace(lines[i]))
		if len(text) == 0 {
			continue
		}
		if len(text) > 240 {
			position := 0
			if i == matched {
				if offset := strings.Index(strings.ToLower(string(text)), strings.ToLower(q.Query)); offset >= 0 {
					position = max(0, len([]rune(string(text)[:offset]))-80)
				}
			}
			text = text[position:min(position+240, len(text))]
			out.Truncated = true
		}
		e := planning.EvidenceItem{Type: "SOURCE_TOOL", Location: fmt.Sprintf("%s:%d", q.File, i+1), Excerpt: string(text)}
		e.ID = digest(e)[:24]
		c.Evidence = append(c.Evidence, e)
	}
	if len(c.Evidence) > 0 {
		out.Candidates = append(out.Candidates, c)
	}
	out.Truncated = out.Truncated || end < len(lines)
	if end < len(lines) {
		out.NextLine = end + 1
	}
	return out, nil
}
