package textutil

import (
	"path/filepath"
	"strings"
	"unicode"
)

func Normalize(s string) string {
	return strings.Join(Tokenize(s), " ")
}

func Tokenize(s string) []string {
	var words []string
	var b strings.Builder
	var previous rune
	flush := func() {
		if b.Len() > 0 {
			words = append(words, strings.ToLower(b.String()))
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if b.Len() > 0 && unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous)) {
				flush()
			}
			b.WriteRune(r)
		} else {
			flush()
		}
		previous = r
	}
	flush()
	return words
}

func RelativePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func UniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	slicesSort(result)
	return result
}

// ContextAnchor selects a short process phrase shared by a changed element's
// documentation and an adjacent BPMN label. If documentation has no shared
// phrase, a single token common to at least two neighbors is used. It never
// returns an entire neighboring task description or a generated BPMN ID.
func ContextAnchor(documentation string, neighborNames []string) string {
	documentTokens := Tokenize(documentation)
	neighbors := make([][]string, 0, len(neighborNames))
	for _, name := range neighborNames {
		if tokens := Tokenize(name); len(tokens) > 0 {
			neighbors = append(neighbors, tokens)
		}
	}
	for size := 3; size >= 2; size-- {
		for start := 0; start+size <= len(documentTokens); start++ {
			phrase := documentTokens[start : start+size]
			if !meaningfulPhrase(phrase) {
				continue
			}
			for _, neighbor := range neighbors {
				if containsPhrase(neighbor, phrase) {
					return strings.Join(phrase, " ")
				}
			}
		}
	}
	if len(neighbors) < 2 {
		return ""
	}
	for _, token := range neighbors[0] {
		if len([]rune(token)) < 4 {
			continue
		}
		shared := true
		for _, neighbor := range neighbors[1:] {
			if !containsPhrase(neighbor, []string{token}) {
				shared = false
				break
			}
		}
		if shared {
			return token
		}
	}
	return ""
}

func meaningfulPhrase(tokens []string) bool {
	for _, token := range tokens {
		if len([]rune(token)) < 4 {
			return false
		}
	}
	return true
}

func containsPhrase(haystack, needle []string) bool {
	for start := 0; start+len(needle) <= len(haystack); start++ {
		match := true
		for index, token := range needle {
			if haystack[start+index] != token {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func slicesSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
