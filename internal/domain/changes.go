package domain

// IndependentChanges returns impact-analysis units. Supporting BPMN flow edits
// remain in the canonical report, but their parent insertion owns retrieval
// and review. Orphaned part_of references remain independent rather than
// silently disappearing from older or malformed reports.
func IndependentChanges(changes []Change) []Change {
	primaryIDs := make(map[string]bool, len(changes))
	for _, change := range changes {
		if change.PartOf != "" {
			continue
		}
		if change.NewElement != nil {
			primaryIDs[change.NewElement.ID] = true
		} else if change.OldElement != nil {
			primaryIDs[change.OldElement.ID] = true
		}
	}
	result := make([]Change, 0, len(changes))
	for _, change := range changes {
		if change.PartOf == "" || !primaryIDs[change.PartOf] {
			result = append(result, change)
		}
	}
	return result
}
