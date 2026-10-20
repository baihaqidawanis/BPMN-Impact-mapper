package semantic

import (
	"crypto/sha256"
	"strings"
	"testing"
)

func TestEmbeddingCacheCopiesValuesAndRejectsDifferentKey(t *testing.T) {
	key := sha256.Sum256([]byte("cache-test-one"))
	other := sha256.Sum256([]byte("cache-test-two"))
	runtime := Identity{PythonVersion: "3.12.0", SentenceTransformersVersion: "5.2.3", ModelArtifactSHA256: strings.Repeat("a", 64), EmbeddingDimension: 2}
	storeCachedVectors(key, [][]float64{{1, 2}}, runtime)
	first := cachedVectors(key)
	if len(first) != 1 || len(first[0]) != 2 || first[0][0] != 1 {
		t.Fatalf("unexpected cached vectors: %#v", first)
	}
	first[0][0] = 99
	second := cachedVectors(key)
	if second[0][0] != 1 {
		t.Fatal("caller mutated cached vectors")
	}
	if cachedVectors(other) != nil {
		t.Fatal("cache returned vectors for a different input")
	}
	if cachedRuntimeIdentity(key).PythonVersion != "3.12.0" || cachedRuntimeIdentity(other).PythonVersion != "" {
		t.Fatal("runtime identity did not follow the cached embedding key")
	}
}

func TestEmbeddingCacheKeepsCorpusAndQueryTogether(t *testing.T) {
	corpus := sha256.Sum256([]byte("two-entry-corpus"))
	query := sha256.Sum256([]byte("two-entry-query"))
	identity := Identity{PythonVersion: "3.12.0"}
	storeCachedVectors(corpus, [][]float64{{1, 0}}, identity)
	storeCachedVectors(query, [][]float64{{0, 1}}, identity)
	if got := cachedVectors(corpus); len(got) != 1 || got[0][0] != 1 {
		t.Fatalf("corpus was evicted by query: %v", got)
	}
	if got := cachedVectors(query); len(got) != 1 || got[0][1] != 1 {
		t.Fatalf("query was not cached: %v", got)
	}
}
