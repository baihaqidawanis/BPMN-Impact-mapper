package evaluation

import (
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"sort"
)

// PrimaryNodeProjection excludes sequence-flow details from an explicitly
// node-scoped experiment. It must never be called complete BPMN coverage.
func PrimaryNodeProjection(report domain.Report) domain.Report {
	projected := report
	projected.Changes = []domain.Change{}
	for _, change := range report.Changes {
		if change.PartOf != "" || change.ChangeType == domain.ModifySequenceFlow {
			continue
		}
		projected.Changes = append(projected.Changes, change)
	}
	return projected
}

// UnionAtK evaluates a set of distinct files from each labelled node's first K
// distinct-file candidates. It is not precision@K for a single ranked list.
type FileUnion struct {
	PerNodeK  int      `json:"per_node_k"`
	Expected  []string `json:"expected"`
	Predicted []string `json:"predicted"`
	Missed    []string `json:"missed"`
	Extra     []string `json:"extra"`
	Precision float64  `json:"precision"`
	Recall    float64  `json:"recall"`
}

func NodeFileUnion(prediction domain.Report, truth GroundTruth, k int) (FileUnion, error) {
	projected := PrimaryNodeProjection(prediction)
	if err := ValidateCoverage(projected, truth); err != nil {
		return FileUnion{}, err
	}
	if k <= 0 {
		return FileUnion{}, errors.New("per-node k must be positive")
	}
	expected, predicted := map[string]bool{}, map[string]bool{}
	for _, c := range truth.Cases {
		for _, e := range c.Expected {
			expected[e.File] = true
		}
	}
	for _, c := range projected.Changes {
		candidates := deduplicate(c.Impacts)
		for i, candidate := range candidates {
			if i >= k {
				break
			}
			predicted[candidate.File] = true
		}
	}
	result := FileUnion{PerNodeK: k, Expected: fileSetKeys(expected), Predicted: fileSetKeys(predicted), Missed: []string{}, Extra: []string{}}
	hits := 0
	for _, file := range result.Expected {
		if predicted[file] {
			hits++
		} else {
			result.Missed = append(result.Missed, file)
		}
	}
	for _, file := range result.Predicted {
		if !expected[file] {
			result.Extra = append(result.Extra, file)
		}
	}
	if len(predicted) > 0 {
		result.Precision = float64(hits) / float64(len(predicted))
	}
	if len(expected) > 0 {
		result.Recall = float64(hits) / float64(len(expected))
	}
	return result, nil
}

func fileSetKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
