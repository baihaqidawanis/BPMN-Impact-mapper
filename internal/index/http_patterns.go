package index

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

var routeParameter = regexp.MustCompile(`^\{[A-Za-z_][A-Za-z0-9_]*\}$`)
var templateParameter = regexp.MustCompile(`^\$\{[A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*\}$`)
var templateHTTPCall = regexp.MustCompile("\\b(?:api(?:<[^>\\n]+>)?|fetch)\\s*\\(\\s*`([^`\\n]+)`\\s*,\\s*\\{[^}\\n]*\\bmethod\\s*:\\s*[\"'](GET|POST|PUT|PATCH|DELETE)[\"']")

func routeParts(value string, client bool) ([]string, bool) {
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#*:\\\t\r\n ") {
		return nil, false
	}
	parts := strings.Split(value, "/")
	literal := 0
	for _, part := range parts[1:] {
		if (client && templateParameter.MatchString(part)) || (!client && routeParameter.MatchString(part)) {
			continue
		}
		if strings.ContainsAny(part, "{}$`") {
			return nil, false
		}
		if part != "" {
			literal++
		}
	}
	return parts, literal > 0
}
func supportedRoutePattern(value string) bool { _, ok := routeParts(value, false); return ok }

// Mask comments and quoted strings for the call start. The URL itself is parsed
// separately. This is deliberately limited text recognition, not a JS parser.
func codeStarts(content string) []bool {
	result := make([]bool, len(content))
	var quote byte
	block, line, escape := false, false, false
	for i := 0; i < len(content); i++ {
		c := content[i]
		if line {
			if c == '\n' {
				line = false
			}
			continue
		}
		if block {
			if c == '*' && i+1 < len(content) && content[i+1] == '/' {
				block = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(content) {
			if content[i+1] == '/' {
				line = true
				i++
				continue
			}
			if content[i+1] == '*' {
				block = true
				i++
				continue
			}
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			continue
		}
		result[i] = true
	}
	return result
}

func patternClientLinks(file ingest.File, routes map[string][]httpRoute) []domain.SourceRelation {
	content := string(file.Content)
	mask := codeStarts(content)
	result := []domain.SourceRelation{}
	keys := make([]string, 0, len(routes))
	for key := range routes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, bounds := range templateHTTPCall.FindAllStringSubmatchIndex(content, -1) {
		if !mask[bounds[0]] {
			continue
		}
		value, method := content[bounds[2]:bounds[3]], content[bounds[4]:bounds[5]]
		parts, ok := routeParts(value, true)
		if !ok {
			continue
		}
		matches := []httpRoute{}
		for _, key := range keys {
			m, path, ok := strings.Cut(key, " ")
			if !ok || m != method {
				continue
			}
			server, ok := routeParts(path, false)
			if !ok || len(parts) != len(server) {
				continue
			}
			compatible := true
			for i := range parts {
				if templateParameter.MatchString(parts[i]) {
					if !routeParameter.MatchString(server[i]) {
						compatible = false
						break
					}
				} else if parts[i] != server[i] && !routeParameter.MatchString(server[i]) {
					compatible = false
					break
				}
			}
			if compatible {
				matches = append(matches, routes[key]...)
			}
		}
		if len(matches) != 1 || matches[0].targetID == "" {
			continue
		}
		line := strings.Count(content[:bounds[0]], "\n") + 1
		lineStart := strings.LastIndex(content[:bounds[0]], "\n") + 1
		lineEnd := strings.IndexByte(content[bounds[0]:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += bounds[0]
		}
		r := matches[0]
		result = append(result, domain.SourceRelation{Kind: domain.RelationHTTPPattern, TargetID: r.targetID, Location: fmt.Sprintf("%s:%d", file.RelativePath, line), Excerpt: routeExcerpt(content[lineStart:lineEnd]), TargetLocation: r.location, TargetExcerpt: r.excerpt})
	}
	return result
}
