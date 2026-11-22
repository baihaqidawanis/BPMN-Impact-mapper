package agenttools

import (
	"context"
	"strings"
	"testing"
)

func TestTestCallSitesKeepsEveryMatchingFixtureAndRespectsScope(t *testing.T) {
	const file = "test/lifecycle_test.go"
	const code = `package lifecycle
import inventory "example/inventory"
import purchasing "example/purchasing"
func TestSnapshot() {
	s := inventory.NewService()
	s.Transition(ctx, id, "submit", actor, "", version, "first")
	s.Transition(ctx, id, "approve", actor, "", version, "approval")
	{
		s := purchasing.NewService()
		s.Transition(ctx, id, "submit", actor, "", version, "other")
	}
}
func TestRecovery() {
	s := inventory.NewService()
	s.Transition(ctx, id, "submit", actor, "", version, "before restart")
	s.Transition(ctx, id, "submit", actor, "", version, "after restart")
	s.Transition(ctx, id, "submit", actor, "already set", version, "control")
}`
	snapshot := Snapshot{files: map[string]string{file: code}}
	sites, err := snapshot.TestCallSites(context.Background(), TestCallQuery{
		File: file, Constructor: "inventory.NewService", Method: "Transition", Literals: map[int]string{3: "submit", 5: ""},
	})
	if err != nil || len(sites) != 3 {
		t.Fatalf("expected three exact call sites, got %+v, %v", sites, err)
	}
	for i, want := range []string{"first", "before restart", "after restart"} {
		if !strings.Contains(sites[i].Evidence.Excerpt, want) || sites[i].Evidence.ID == "" || sites[i].Reference == "" {
			t.Fatalf("wrong observation: %+v", sites[i])
		}
	}
	if sites[0].Test != "TestSnapshot" || sites[1].Test != "TestRecovery" || sites[2].Test != "TestRecovery" {
		t.Fatalf("wrong test owners: %+v", sites)
	}
}
