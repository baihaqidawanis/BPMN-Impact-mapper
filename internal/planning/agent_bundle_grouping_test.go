package planning_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func parsePatternGraphs(t *testing.T, dir string) (*domain.Graph, *domain.Graph) {
	t.Helper()
	read := func(name string) *domain.Graph {
		data, err := os.ReadFile(filepath.Join("..", "..", "experiments", "simple-erp", "poc-patterns", dir, name))
		if err != nil {
			t.Fatal(err)
		}
		g, err := bpmn.Parse(data, bpmn.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	return read("before.bpmn"), read("after.bpmn")
}

func TestSkipBundlePreservesBaselineJoinAndDoesNotPolluteGatewayLeads(t *testing.T) {
	before, after := parsePatternGraphs(t, "skip")
	changes := diff.Compare(before, after)
	for i := range changes {
		el := changes[i].NewElement
		if el == nil {
			continue
		}
		switch el.ID {
		case "ZeroDeltaDecision":
			changes[i].Impacts = []domain.Impact{{
				File:     "backend/internal/inventory/adapter/postgres/store.go",
				Evidence: []domain.Evidence{{Type: "BM25", Location: "store.go:120", Excerpt: "TransitionAdjustment"}},
			}}
		case "BypassApproval":
			// Contextless flow-only lexical noise must not pollute a gateway that already has ranked candidates.
			changes[i].Impacts = []domain.Impact{
				{File: "frontend/tests/dashboard-workspace.test.tsx", Evidence: []domain.Evidence{{Type: "BM25", Location: "dashboard-workspace.test.tsx:26", Excerpt: "1 item stok nol"}}},
				{File: "backend/internal/inventory/adapter/postgres/store.go", Evidence: []domain.Evidence{{Type: "BM25", Location: "store.go:180", Excerpt: "SUBMITTED"}}},
			}
		}
	}
	report := domain.Report{ProcessContext: &domain.ProcessContext{Before: before, After: after}, Changes: changes}
	bundle := planning.BuildAgentBundle(report, planning.Limits{GroupGatewayChanges: true, GroupGatewayBranches: true})

	byID := map[string]planning.ChangeInput{}
	for _, c := range bundle.Changes {
		byID[c.ElementID] = c
	}
	for _, suffix := range []string{"Flow_5", "Flow_6", "Flow_6_prepare_result", "Flow_6_prepared", "result_merge", "result_joined"} {
		id := "role_staff_l3_adj_submit_" + suffix
		if _, leaked := byID[id]; leaked {
			t.Fatalf("unchanged baseline node or flow %s became a review unit", id)
		}
	}
	zero, ok := byID["ZeroDeltaDecision"]
	if !ok {
		t.Fatal("missing ZeroDeltaDecision")
	}
	if len(zero.Candidates) != 1 || zero.Candidates[0].File != "backend/internal/inventory/adapter/postgres/store.go" {
		t.Fatalf("expected only store.go (with merged evidence) on ZeroDeltaDecision, got %+v", zero.Candidates)
	}
	if len(zero.Candidates[0].Evidence) != 2 {
		t.Fatalf("expected matching-file evidence from BypassApproval merged onto store.go, got %+v", zero.Candidates[0].Evidence)
	}
}

func TestAgentBundleGroupingFoldsNewTaskToTaskWiringAcrossPatterns(t *testing.T) {
	for _, tc := range []struct {
		dir       string
		mustFold  []string
		mustKeep  []string
	}{
		{
			dir:      "redo",
			mustFold: []string{"ReturnedToStaff", "ApplyRevision", "role_supervisor_l3_pr_review_Flow_3"},
			mustKeep: []string{"ReturnPRForRevision", "EditReturnedPR", "ResubmitRevisedPR"},
		},
		{
			dir:      "non-free-choice",
			mustFold: []string{"RouteLocked", "ReadLockedRoute"},
			mustKeep: []string{"LockApprovalRoute", "LoadLockedApprovalRoute", "LockedRouteDecision", "ContinueToReview"},
		},
		{
			dir:      "switch",
			mustFold: []string{"role_supervisor_l3_pr_review_Flow_3"},
			mustKeep: []string{"ApprovalProductDecision", "CheckProductsAtApproval"},
		},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			before, after := parsePatternGraphs(t, tc.dir)
			report := domain.Report{ProcessContext: &domain.ProcessContext{Before: before, After: after}, Changes: diff.Compare(before, after)}
			bundle := planning.BuildAgentBundle(report, planning.Limits{GroupGatewayChanges: true, GroupGatewayBranches: true})
			present := map[string]bool{}
			for _, c := range bundle.Changes {
				present[c.ElementID] = true
			}
			for _, id := range tc.mustFold {
				if present[id] {
					t.Errorf("expected supporting wiring flow %q to fold in %s", id, tc.dir)
				}
			}
			for _, id := range tc.mustKeep {
				if !present[id] {
					t.Errorf("expected business unit %q to remain in %s", id, tc.dir)
				}
			}
		})
	}
}
