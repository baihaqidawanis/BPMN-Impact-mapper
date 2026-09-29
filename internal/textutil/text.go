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

func slicesSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
