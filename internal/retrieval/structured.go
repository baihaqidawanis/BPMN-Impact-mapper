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

// A SQL line may mention task words in a variable or column while querying an
// unrelated table. Credit DATABASE_TABLE only when the captured table name
// itself overlaps the BPMN query. This is a conservative syntax hint, not SQL
// parsing: joins, aliases, and dynamically constructed names may abstain.
var databaseTableName = regexp.MustCompile("(?i)\\b(?:insert\\s+into|create\\s+table|delete\\s+from|from|update)\\s+[\\\"`\\[]?([a-z_][a-z0-9_.]*)")
var auditActionLiteral = regexp.MustCompile("[\\\"']([A-Z][A-Z0-9_.:-]*)[\\\"']")
var routePathLiteral = regexp.MustCompile("[\\\"'`](/[^\\\"'`]+)[\\\"'`]")
var routeYAMLPath = regexp.MustCompile(`^\s*(/[\w/{}/:.-]+)\s*:`)

func structuredSignal(artifact domain.Artifact, query []string) (float64, domain.Evidence) {
	for _, hint := range structuredPatterns {
		for index, line := range strings.Split(artifact.Content, "\n") {
			// Never claim evidence that was clipped out of the displayed excerpt.
			shown := bounded(strings.TrimSpace(line), 240)
			if strings.HasPrefix(shown, "//") || strings.HasPrefix(shown, "#") || strings.HasPrefix(shown, "--") {
				continue
			}
			if !hint.pattern.MatchString(shown) {
				continue
			}
			support := shown
			strength := hint.weight
			if hint.kind == "DATABASE_TABLE" {
				match := databaseTableName.FindStringSubmatch(shown)
				if len(match) < 2 {
					continue
				}
				support = match[1]
				strength = tableTermCoverage(support, query)
			} else if hint.kind == "AUDIT_ACTION" {
				support = firstMatchingLiteral(auditActionLiteral, shown, query)
			} else if hint.kind == "API_ROUTE" {
				support = firstMatchingLiteral(routePathLiteral, shown, query)
				if support == "" {
					support = firstMatchingLiteral(routeYAMLPath, shown, query)
				}
			} else if hint.kind == "STATE_STATUS" || hint.kind == "TEST_NAME" {
				support = hint.pattern.FindString(shown)
			}
			if strength == 0 || (hint.kind != "DATABASE_TABLE" && overlap(query, textutil.Tokenize(support)) == 0) {
				continue
			}
			lineNumber := index + 1
			if artifact.Symbol.StartLine > 0 {
				lineNumber += artifact.Symbol.StartLine - 1
			}
			return strength, domain.Evidence{Type: hint.kind, Location: fmt.Sprintf("%s:%d", artifact.File, lineNumber), Excerpt: shown}
		}
	}
	return 0, domain.Evidence{}
}

func firstMatchingLiteral(pattern *regexp.Regexp, line string, query []string) string {
	for _, match := range pattern.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 && overlap(query, textutil.Tokenize(match[1])) > 0 {
			return match[1]
		}
	}
	return ""
}

func tableTermCoverage(table string, query []string) float64 {
	terms := textutil.UniqueSorted(textutil.Tokenize(table))
	if len(terms) == 0 {
		return 0
	}
	querySet := make(map[string]bool, len(query))
	for _, term := range query {
		querySet[term] = true
	}
	matched := 0
	for _, term := range terms {
		if querySet[term] || containsInflection(querySet, term) {
			matched++
		}
	}
	return float64(matched) / float64(len(terms))
}
