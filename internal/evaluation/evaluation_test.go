package evaluation_test

import (
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

func TestEvaluateComputesHandCheckableMetrics(t *testing.T) {
	prediction := domain.Report{AnalysisID: "analysis", Method: domain.MethodHybrid, Changes: []domain.Change{
		{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "A", Type: "task"}, Impacts: []domain.Impact{
			{File: "wrong.go", Module: "other", Layer: "test", RequiredAction: domain.ActionVerify},
			{File: "a.go", Module: "core", Layer: "application", RequiredAction: domain.ActionVerify},
		}},
		{ChangeType: domain.ModifyTaskLabel, NewElement: &domain.Element{ID: "B", Type: "task"}, Impacts: []domain.Impact{
			{File: "b.go", Module: "core", Layer: "domain", RequiredAction: domain.ActionModify},
		}},
	}}
	truth := evaluation.GroundTruth{Version: "1", Cases: []evaluation.Case{
		{ChangeType: domain.AddTask, ElementID: "A", Expected: []evaluation.ExpectedImpact{{File: "a.go", Module: "core", Layer: "application", Action: domain.ActionAdd}}},
		{ChangeType: domain.ModifyTaskLabel, ElementID: "B", Expected: []evaluation.ExpectedImpact{{File: "b.go", Module: "core", Layer: "domain", Action: domain.ActionModify}}},
	}}
	result, err := evaluation.Evaluate(prediction, []byte("prediction"), truth, []byte("truth"))
	if err != nil {
		t.Fatal(err)
	}
	metrics := result.Metrics
	if metrics.Recall.At1 != 0.5 || metrics.Recall.At3 != 1 || metrics.Precision.At3 != 0.3333 || metrics.MRR != 0.75 || metrics.MAP != 0.75 {
		t.Fatalf("ranking metrics=%#v", metrics)
	}
	if metrics.FilePrecision != 0.6667 || metrics.FileRecall != 1 || metrics.FileF1 != 0.8 || metrics.ActionAccuracy != 0.5 {
		t.Fatalf("classification metrics=%#v", metrics)
	}
	if metrics.FalsePositiveRate != 0.3333 || metrics.UnresolvedRate != 0 {
		t.Fatalf("rates=%#v", metrics)
	}
}

func TestEvaluateRejectsMismatchedBindingAndDuplicates(t *testing.T) {
	prediction := domain.Report{AnalysisID: "one", Method: domain.MethodHybrid}
	truth := evaluation.GroundTruth{Version: "1", AnalysisID: "two"}
	if _, err := evaluation.Evaluate(prediction, nil, truth, nil); err == nil {
		t.Fatal("expected analysis binding error")
	}
	truth = evaluation.GroundTruth{Version: "1", Cases: []evaluation.Case{
		{ChangeType: domain.AddTask, ElementID: "A"}, {ChangeType: domain.AddTask, ElementID: "A"},
	}}
	if err := truth.Validate(); err == nil {
		t.Fatal("expected duplicate case rejection")
	}
}
