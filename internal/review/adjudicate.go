package review

import (
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"sort"
)

// Adjudicate preserves agreements and requires exactly one explicit decision per conflict.
func Adjudicate(a, b Annotation, resolutions []evaluation.Case) (evaluation.GroundTruth, error) {
	result, err := Reconcile(a, b)
	if err != nil {
		return evaluation.GroundTruth{}, err
	}
	decisions := evaluation.GroundTruth{Version: "1", Cases: resolutions}
	if err := decisions.Validate(); err != nil {
		return evaluation.GroundTruth{}, err
	}
	pending := map[string]bool{}
	for _, conflict := range result.Disagreements {
		pending[conflict.ChangeType+"|"+conflict.ElementID] = true
	}
	for _, decision := range resolutions {
		key := string(decision.ChangeType) + "|" + decision.ElementID
		if !pending[key] {
			return evaluation.GroundTruth{}, errors.New("unexpected resolution")
		}
		delete(pending, key)
		decision.Expected = sortedExpected(decision.Expected)
		result.Agreed.Cases = append(result.Agreed.Cases, decision)
	}
	if len(pending) > 0 {
		return evaluation.GroundTruth{}, errors.New("unresolved disagreements")
	}
	sort.Slice(result.Agreed.Cases, func(i, j int) bool {
		a, b := result.Agreed.Cases[i], result.Agreed.Cases[j]
		return string(a.ChangeType)+"|"+a.ElementID < string(b.ChangeType)+"|"+b.ElementID
	})
	return result.Agreed, nil
}
