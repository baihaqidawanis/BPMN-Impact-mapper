package evaluation

import (
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// ValidateCoverage forbids reporting a convenient subset as a complete web evaluation.
func ValidateCoverage(prediction domain.Report, truth GroundTruth) error {
	if err := truth.Validate(); err != nil {
		return err
	}
	if len(truth.Cases) == 0 || len(truth.Cases) != len(prediction.Changes) {
		return errors.New("incomplete cases")
	}
	keys := map[string]bool{}
	for _, change := range prediction.Changes {
		key := caseKey(change.ChangeType, elementID(change))
		if keys[key] {
			return errors.New("ambiguous change key")
		}
		keys[key] = true
	}
	for _, item := range truth.Cases {
		if !keys[caseKey(item.ChangeType, item.ElementID)] {
			return errors.New("unknown case")
		}
	}
	return nil
}
