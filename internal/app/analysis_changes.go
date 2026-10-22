package app

import (
	"context"
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

// attachPrimaryChanges maps each independent process change once. A plain
// sequence-flow split remains in canonical JSON as a supporting graph diff,
// but cannot create three duplicate candidate lists for one inserted task.
func attachPrimaryChanges(ctx context.Context, strategy mapping.Strategy, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	indices := make([]int, 0, len(changes))
	primary := make([]domain.Change, 0, len(changes))
	for index, change := range changes {
		if change.PartOf == "" {
			indices = append(indices, index)
			primary = append(primary, change)
		}
	}
	mapped, warnings, err := strategy.Attach(ctx, primary, artifacts, options)
	if err != nil {
		return nil, nil, err
	}
	if len(mapped) != len(indices) {
		return nil, nil, fmt.Errorf("mapping strategy returned %d changes for %d primary changes", len(mapped), len(indices))
	}
	for index, originalIndex := range indices {
		changes[originalIndex] = mapped[index]
	}
	return changes, warnings, nil
}
