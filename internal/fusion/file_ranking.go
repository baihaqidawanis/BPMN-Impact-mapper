package fusion

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

// A file gets the position of its first occurrence, with ranks compacted after
// deduplication. Votes across different functions support the file, not a claim
// that both retrievers identified the same function. Preserve original evidence.
func fileRanking(impacts []domain.Impact) []domain.Impact {
	seen := map[string]bool{}
	result := make([]domain.Impact, 0, len(impacts))
	for _, candidate := range impacts {
		if seen[candidate.File] {
			continue
		}
		seen[candidate.File] = true
		candidate.Symbol = ""
		candidate.Evidence = append([]domain.Evidence(nil), candidate.Evidence...)
		result = append(result, candidate)
	}
	return result
}
