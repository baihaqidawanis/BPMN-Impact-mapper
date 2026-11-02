package semantic

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strings"
	"testing"
)

func TestBaselineAnchorQueriesAreOptInAndUseLabelsNotIDs(t *testing.T) {
	c := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "New", Type: "serviceTask", Name: "Aggregate checks"}, Context: &domain.ChangeContext{BaselineAnchors: []domain.BaselineAnchor{{ElementContext: domain.ElementContext{ID: "BASELINE-SECRET-ID", Name: "Validate invoice version"}, Hops: 2}}}}
	plain := (Strategy{}).queryText(c)
	expanded := (Strategy{UseBaselineAnchors: true}).queryText(c)
	if strings.Contains(plain, "Validate invoice version") || !strings.Contains(expanded, "Validate invoice version") || strings.Contains(expanded, "BASELINE-SECRET-ID") {
		t.Fatal(plain, expanded)
	}
}
