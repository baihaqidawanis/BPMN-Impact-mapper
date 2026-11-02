package fusion_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

func TestFileFusionIgnoresDuplicateFunctionCountAndDoesNotInventFunctionAgreement(t *testing.T) {
	a := impact("a.go", domain.ActionVerify, "BM25")
	a.Symbol = "Alpha"
	b := impact("b.go", domain.ActionVerify, "BM25")
	b.Symbol = "Beta"
	rightB := b
	rightB.Symbol = "DifferentFunction"
	base := []domain.Change{{ChangeType: domain.AddTask}}
	var previous []domain.Change
	for _, copies := range []int{1, 20} {
		left := []domain.Impact{}
		for i := 0; i < copies; i++ {
			duplicate := a
			duplicate.Symbol += string(rune('A' + i))
			left = append(left, duplicate)
		}
		left = append(left, b)
		strategy := fusion.Strategy{MethodName: domain.MethodHybrid, FileLevelFusion: true, DiversifyFiles: true,
			Left:  fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{Impacts: left}}},
			Right: fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{rightB, a}}}}}
		got, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{TopK: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(got[0].Impacts) != 2 || got[0].Impacts[0].Symbol != "" || got[0].Impacts[1].Symbol != "" {
			t.Fatalf("invalid file result: %#v", got)
		}
		if previous != nil && !reflect.DeepEqual(previous, got) {
			t.Fatal("duplicate functions changed file ranks")
		}
		previous = got
		if left[0].Symbol == "" || rightB.Symbol != "DifferentFunction" {
			t.Fatal("input candidates mutated")
		}
	}
}
