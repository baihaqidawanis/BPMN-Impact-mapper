package retrieval

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// Conservative single-line hints, not a SQL parser or verified dependencies.
// Both the syntax and a query token must occur in the reported excerpt.
var structuredPatterns = []struct {
	kind    string
	weight  float64
	pattern *regexp.Regexp
}{
	{"TEST_NAME", 1, regexp.MustCompile(`(?i)(?:\bfunc\s+(?:Test|Example|Benchmark)\w*\s*\(|\bdef\s+test_\w*\s*\(|\b(?:test|it|describe)\s*\(|\bfunction\s+test\w*\s*\()`)},
	{"API_ROUTE", 1, regexp.MustCompile(`(?i)(?:\b(?:get|post|put|patch|delete|handlefunc|handle)\s*\(\s*["'\x60]/[^"'\x60]+|^\s*/[\w/{}/:.-]+\s*:)`)},
	{"DATABASE_TABLE", 1, regexp.MustCompile("(?i)\\b(?:(?:insert\\s+into|create\\s+table|delete\\s+from|select\\s+.+?\\s+from)\\s+[\\\"`\\[]?[a-z_][a-z0-9_.]*|update\\s+[\\\"`\\[]?[a-z_][a-z0-9_.]*[\\\"`\\]]?\\s+set\\s+)")},
	{"AUDIT_ACTION", 1, regexp.MustCompile(`(?i)\b(?:audit|logaudit|recordaudit|audit\.\w+)\s*\(`)},
	{"STATE_STATUS", 0.7, regexp.MustCompile(`(?i)\b\w*(?:state|status)\w*\s*(?::=|=|:|==|!=)\s*["'][A-Z][A-Z_]+["']`)},
}

func structuredSignal(artifact domain.Artifact, query []string) (float64, domain.Evidence) {
	for _, hint := range structuredPatterns {
		for index, line := range strings.Split(artifact.Content, "\n") {
			// Never claim evidence that was clipped out of the displayed excerpt.
			shown := bounded(strings.TrimSpace(line), 240)
			if strings.HasPrefix(shown, "//") || strings.HasPrefix(shown, "#") || strings.HasPrefix(shown, "--") {
				continue
			}
			if !hint.pattern.MatchString(shown) || overlap(query, textutil.Tokenize(shown)) == 0 {
				continue
			}
			lineNumber := index + 1
			if artifact.Symbol.StartLine > 0 {
				lineNumber += artifact.Symbol.StartLine - 1
			}
			return hint.weight, domain.Evidence{Type: hint.kind, Location: fmt.Sprintf("%s:%d", artifact.File, lineNumber), Excerpt: shown}
		}
	}
	return 0, domain.Evidence{}
}
