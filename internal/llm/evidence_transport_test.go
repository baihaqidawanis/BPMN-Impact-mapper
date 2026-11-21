package llm

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSharedEvidencePreservesTextLocationsAndIndices(t *testing.T) {
	text := strings.Repeat("source statement; ", 40)
	in, _ := json.Marshal(map[string]any{"bundle": map[string]any{"evidence": []any{
		map[string]any{"type": "SOURCE_TOOL", "location": "first.go:12", "evidence_index": 0, "excerpt": text},
		map[string]any{"type": "SOURCE_TOOL", "location": "second.go:20", "evidence_index": 1, "excerpt": text},
		map[string]any{"type": "SOURCE_TOOL", "location": "first.go:13", "evidence_index": 2, "excerpt": "unique proof"},
	}}})
	original := append([]byte(nil), in...)
	out, err := shareEvidenceExcerpts(in)
	if err != nil || len(out) >= len(in) || !bytes.Equal(in, original) {
		t.Fatal("compression failed or mutated input", err)
	}
	again, _ := shareEvidenceExcerpts(in)
	if !bytes.Equal(out, again) {
		t.Fatal("unstable transport")
	}
	var root map[string]any
	json.Unmarshal(out, &root)
	pool := root["source_excerpts"].([]any)
	items := root["bundle"].(map[string]any)["evidence"].([]any)
	for i, item := range items {
		e := item.(map[string]any)
		if e["evidence_index"] != float64(i) {
			t.Fatal("changed evidence order/index")
		}
		expected := "unique proof"
		if i < 2 {
			expected = text
		}
		actual, _ := e["excerpt"].(string)
		if ref, ok := e["excerpt_ref"].(float64); ok {
			actual = pool[int(ref)].(string)
		}
		if actual != expected {
			t.Fatal("lost source text")
		}
	}
	if items[0].(map[string]any)["location"] != "first.go:12" || items[1].(map[string]any)["location"] != "second.go:20" {
		t.Fatal("merged distinct source identities")
	}
	if _, err := shareEvidenceExcerpts([]byte("invalid")); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}
