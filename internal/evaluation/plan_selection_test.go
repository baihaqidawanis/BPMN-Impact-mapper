package evaluation_test

import (
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

func TestEvaluatePlanSelectionDeduplicatesPathsExcludesNonEditsAndMatchesScope(t *testing.T) {
	truth := evaluation.GroundTruth{
		Version: "1",
		Cases: []evaluation.Case{
			{
				ChangeType: domain.AddGateway,
				ElementID:  "ZeroDeltaDecision",
				Expected: []evaluation.ExpectedImpact{
					{File: "backend/internal/inventory/adapter/postgres/store.go", Action: domain.ActionModify},
					{File: "backend/test/integration/operations_test.go", Action: domain.ActionModify},
				},
			},
			{
				ChangeType: domain.ModifyTaskLabel,
				ElementID:  "NoChangeControl",
				Expected:   []evaluation.ExpectedImpact{},
			},
		},
	}
	entries := []evaluation.PlanEntry{
		// Duplicate path with Windows separator and ./ prefix must deduplicate cleanly.
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "backend\\internal\\inventory\\adapter\\postgres\\store.go", Action: domain.ActionModify},
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "./backend/internal/inventory/adapter/postgres/store.go", Action: domain.ActionModify},
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "backend/test/integration/operations_test.go", Action: domain.ActionModify},
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "frontend/features/inventory/inventory-adjustment-panel.tsx", Action: domain.ActionModify},
		// Non-edit verdicts must be excluded from proposed edits.
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "backend/migrations/000005_operations.up.sql", Action: domain.ActionNoAction},
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "frontend/features/inventory/inventory-workspace.tsx", Action: domain.ActionVerify},
		// Different BPMN element outside scope must not leak into ZeroDeltaDecision.
		{ChangeType: domain.AddTask, ElementID: "InvalidAdjustment", File: "frontend/app/inventory/[section]/page.tsx", Action: domain.ActionModify},
		// No-change control with only NO_ACTION must pass without false edit.
		{ChangeType: domain.ModifyTaskLabel, ElementID: "NoChangeControl", File: "README.md", Action: domain.ActionNoAction},
	}

	report, err := evaluation.EvaluatePlanSelection(entries, truth)
	if err != nil {
		t.Fatal(err)
	}
	if report.PositiveCases != 1 || !report.AllMandatoryFound {
		t.Fatalf("unexpected positive summary: %+v", report)
	}
	if report.MacroPrecision != 0.6667 || report.MacroRecall != 1 || report.MacroF1 != 0.8 {
		t.Fatalf("expected P=0.6667 R=1 F1=0.8, got P=%.4f R=%.4f F1=%.4f", report.MacroPrecision, report.MacroRecall, report.MacroF1)
	}
	if report.NoChangeControls != 1 || report.FalseEditControls != 0 {
		t.Fatalf("expected 1 passing no-change control, got %+v", report)
	}
	if report.ExcludedNonEditRows != 3 {
		t.Fatalf("expected 3 excluded non-edit rows, got %d", report.ExcludedNonEditRows)
	}

	// Verify positive reference with zero proposed edits records 0 precision and 0 recall.
	emptyReport, err := evaluation.EvaluatePlanSelection([]evaluation.PlanEntry{
		{ChangeType: domain.AddGateway, ElementID: "ZeroDeltaDecision", File: "backend/internal/inventory/adapter/postgres/store.go", Action: domain.ActionVerify},
		{ChangeType: domain.ModifyTaskLabel, ElementID: "NoChangeControl", File: "README.md", Action: domain.ActionModify},
	}, truth)
	if err != nil {
		t.Fatal(err)
	}
	if emptyReport.MacroPrecision != 0 || emptyReport.MacroRecall != 0 || emptyReport.AllMandatoryFound || emptyReport.FalseEditControls != 1 {
		t.Fatalf("unexpected zero-edit/false-edit handling: %+v", emptyReport)
	}
}

func TestEvaluateNormalizesDuplicateWindowsAndDotSlashPaths(t *testing.T) {
	prediction := domain.Report{
		AnalysisID: "a1",
		Method:     domain.MethodHybrid,
		Changes: []domain.Change{{
			ChangeType: domain.AddTask,
			NewElement: &domain.Element{ID: "T1", Type: "serviceTask"},
			Impacts: []domain.Impact{
				{File: "backend\\store.go", RequiredAction: domain.ActionVerify},
				{File: "./backend/store.go", RequiredAction: domain.ActionVerify},
			},
		}},
	}
	truth := evaluation.GroundTruth{
		Version: "1",
		Cases: []evaluation.Case{{
			ChangeType: domain.AddTask,
			ElementID:  "T1",
			Expected:   []evaluation.ExpectedImpact{{File: "backend/store.go", Action: domain.ActionModify}},
		}},
	}
	res, err := evaluation.Evaluate(prediction, []byte("p"), truth, []byte("t"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Metrics.PredictedUniqueFiles != 1 || res.Metrics.FilePrecision != 1 || res.Metrics.FileRecall != 1 {
		t.Fatalf("expected normalized deduplication to match 1/1, got %+v", res.Metrics)
	}
}
