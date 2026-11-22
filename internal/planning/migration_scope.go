package planning

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

var numberedSQLMigration = regexp.MustCompile(`^[0-9]+[^/]*\.(up|down)\.sql$`)

// A numbered migration in the analyzed snapshot may already be applied.
// Describe new DDL in the storage plan while preserving migration history.
func validateMigrationEdit(p FilePlan) error {
	if p.Action != domain.ActionModify && p.Action != domain.ActionRemove {
		return nil
	}
	file := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
	if !strings.Contains("/"+file, "/migrations/") || !numberedSQLMigration.MatchString(path.Base(file)) {
		return nil
	}
	return fmt.Errorf("%w: existing migration %q is schema evidence; do not edit or remove it. Use NO_CHANGE for this target and describe a new forward migration in the affected storage proposal if the source constraint requires one", ErrAgentPlanIncomplete, p.File)
}
