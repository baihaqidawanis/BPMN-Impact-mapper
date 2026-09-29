package retrieval_test

import (
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
)

func TestBM25RankingIsDeterministicWithStableTies(t *testing.T) {
	documents := []domain.Artifact{
		{ID: "b", Tokens: []string{"request", "validate"}},
		{ID: "a", Tokens: []string{"request", "validate"}},
		{ID: "c", Tokens: []string{"unrelated"}},
	}
	model := retrieval.NewBM25(documents)
	first := model.Search([]string{"validate", "request"})
	second := model.Search([]string{"request", "validate"})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("rankings differ: %#v %#v", first, second)
	}
	if len(first) != 2 || first[0].Artifact.ID != "a" || first[1].Artifact.ID != "b" {
		t.Fatalf("unexpected ranking: %#v", first)
	}
}
