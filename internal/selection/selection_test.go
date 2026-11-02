package selection

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"testing"
)

func TestSelectionUsesAgreementNotConfidenceAndPreservesInput(t *testing.T) {
	makeCandidate := func(file, rank string, score float64) domain.Impact {
		return domain.Impact{File: file, RequiredAction: domain.ActionVerify, Confidence: score, Evidence: []domain.Evidence{{Type: "RRF_FUSION", Excerpt: rank}}}
	}
	input := []domain.Impact{makeCandidate("one.go", "left_rank=absent;right_rank=1", 1), makeCandidate("two.go", "left_rank=2;right_rank=3", 0.1), makeCandidate("three.go", "left_rank=2;right_rank=20", 0.9), makeCandidate("two.go", "left_rank=1;right_rank=1", 1)}
	before, _ := json.Marshal(input)
	got, err := Apply(input, Policy{Name: "both-within", Bound: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Selected) != 1 || got.Selected[0].File != "two.go" || got.Selected[0].RequiredAction != domain.ActionVerify || len(got.Deferred) != 2 || got.DuplicateFiles != 1 {
		t.Fatal(got)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("mutated original candidates")
	}
}

func TestSelectionDefersAmbiguousRanksAndCanAbstain(t *testing.T) {
	for _, evidence := range [][]domain.Evidence{
		nil, {{Type: "RRF_FUSION", Excerpt: "left_rank=2;left_rank=1;right_rank=1"}},
		{{Type: "RRF_FUSION", Excerpt: "left_rank=1;right_rank=1"}, {Type: "RRF_FUSION", Excerpt: "left_rank=1;right_rank=1"}},
		{{Type: "RRF_FUSION", Excerpt: "left_rank=0;right_rank=1"}},
	} {
		got, err := Apply([]domain.Impact{{File: "a.go", Evidence: evidence}}, Policy{Name: "both-within", Bound: 5})
		if err != nil || len(got.Selected) != 0 || len(got.Deferred) != 1 {
			t.Fatal(got, err)
		}
	}
	if _, err := Apply(nil, Policy{Name: "both-within", Bound: 0}); err == nil {
		t.Fatal("invalid bound accepted")
	}
	if _, err := Apply(nil, Policy{Name: "unknown"}); err == nil {
		t.Fatal("unknown policy accepted")
	}
}
