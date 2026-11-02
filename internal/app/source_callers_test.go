package app_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func TestSourceCallersRemainOutsideRankingAndEnterBoundedPlanning(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "source-callers")
	request := app.Request{RepositoryPath: root, OldBPMNPath: filepath.Join(root, "old.bpmn"), NewBPMNPath: filepath.Join(root, "new.bpmn"), Method: domain.MethodBM25, Retrieval: mapping.Options{TopK: 3}}
	base := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{})
	base.IncludeSourceCallers = false
	base.Indexers = nil
	next := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{})
	before, err := base.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	after, err := next.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i, c := range after.Changes {
		if !reflect.DeepEqual(before.Changes[i].Impacts, c.Impacts) {
			t.Fatal("source context changed retrieval/actions")
		}
		if len(c.RelatedFiles) > 0 {
			found = true
			if c.RelatedFiles[0].File != "ui/panel.tsx" {
				t.Fatalf("unexpected caller: %+v", c.RelatedFiles)
			}
		}
	}
	if !found {
		t.Fatal("failed to recover known client")
	}
	bundle := planning.BuildEvidenceBundle(after, planning.DefaultLimits())
	candidateFound := false
	for _, c := range bundle.Changes {
		for _, v := range c.Candidates {
			if v.Origin == "source-caller" {
				candidateFound = true
				if v.RetrievalScore != nil || v.File != "ui/panel.tsx" {
					t.Fatal("invented structural confidence/path")
				}
			}
		}
	}
	if !candidateFound {
		t.Fatal("planner did not receive connected client")
	}
	again, err := next.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := report.JSON(after)
	b, _ := report.JSON(again)
	if string(a) != string(b) {
		t.Fatal("source context output not deterministic")
	}
}
