package llm

import (
	"encoding/json"
	"sort"
)

const evidencePoolInstruction = " source_excerpts is a shared source-text table. For evidence containing excerpt_ref, read source_excerpts[excerpt_ref] as its exact excerpt. Each item retains its own location, type and evidence_index; shared text does not establish identical responsibilities across files or functions. No source text was summarized or removed."

// Compress repeated excerpts in transport only; never mutate stored evidence.
func shareEvidenceExcerpts(input []byte) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(input, &root); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	var walk func(any, func(map[string]any, string))
	walk = func(v any, visit func(map[string]any, string)) {
		switch node := v.(type) {
		case map[string]any:
			if text, ok := node["excerpt"].(string); ok && len(text) > 160 && node["type"] != nil && node["location"] != nil {
				visit(node, text)
			}
			for _, child := range node {
				walk(child, visit)
			}
		case []any:
			for _, child := range node {
				walk(child, visit)
			}
		}
	}
	walk(root, func(_ map[string]any, text string) { counts[text]++ })
	pool := []string{}
	for text, count := range counts {
		if count > 1 {
			pool = append(pool, text)
		}
	}
	if len(pool) == 0 {
		return input, nil
	}
	sort.Strings(pool)
	indices := map[string]int{}
	for i, text := range pool {
		indices[text] = i
	}
	walk(root, func(item map[string]any, text string) {
		if i, ok := indices[text]; ok {
			delete(item, "excerpt")
			item["excerpt_ref"] = i
		}
	})
	root["source_excerpts"] = pool
	encoded, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if len(encoded) >= len(input) {
		return input, nil
	}
	return encoded, nil
}
