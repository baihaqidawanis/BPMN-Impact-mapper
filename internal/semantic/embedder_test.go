package semantic

import (
	"crypto/sha256"
	"testing"
)

func TestEmbeddingCacheCopiesValuesAndRejectsDifferentKey(t *testing.T) {
	key := sha256.Sum256([]byte("cache-test-one"))
	other := sha256.Sum256([]byte("cache-test-two"))
	storeCachedVectors(key, [][]float64{{1, 2}})
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
}
