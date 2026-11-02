package semantic

import "strings"

// Model profiles describe the input convention used during model training.
// They do not calibrate cosine scores or change the configured threshold.
func modelInputProfile(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "intfloat/multilingual-e5-small", "intfloat/multilingual-e5-base", "intfloat/multilingual-e5-large":
		return "e5-query-passage-v1"
	default:
		return "plain-v1"
	}
}

func embeddingInput(profile, text string, query bool) string {
	if profile != "e5-query-passage-v1" {
		return text
	}
	if query {
		return "query: " + text
	}
	return "passage: " + text
}
