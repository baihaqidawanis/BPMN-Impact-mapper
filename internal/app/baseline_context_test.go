package app_test

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineContextExperimentKeepsInputsAndProductionIsolated(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "claims.go"), []byte("package claims\nfunc ValidateClaimDraftVersion() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><task id="Before" name="Submit claim"/><task id="After" name="Validate claim DRAFT version"/></process></definitions>`
	after := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><task id="Before" name="Submit claim"/><task id="After" name="Validate claim DRAFT version"/><task id="New" name="Gather validations"/><parallelGateway id="Join" name="Combine validations"/><sequenceFlow id="A" sourceRef="Before" targetRef="New"/><sequenceFlow id="B" sourceRef="New" targetRef="Join"/><sequenceFlow id="C" sourceRef="Join" targetRef="After"/></process></definitions>`
	oldPath, newPath := filepath.Join(dir, "old.bpmn"), filepath.Join(dir, "new.bpmn")
	for p, s := range map[string]string{oldPath: before, newPath: after} {
		if err := os.WriteFile(p, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := app.Request{RepositoryPath: repo, OldBPMNPath: oldPath, NewBPMNPath: newPath, Method: domain.MethodHybrid}
	var base []byte
	fingerprint, id := "", ""
	for _, mode := range []string{"full", "multi-hop4", "full"} {
		analyzer, err := wiring.NewContextStudyAnalyzer(&constantTestEmbedder{}, mode)
		if err != nil {
			t.Fatal(err)
		}
		value, err := analyzer.Analyze(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := report.JSON(value)
		anchors := 0
		for _, c := range value.Changes {
			if c.Context != nil {
				anchors += len(c.Context.BaselineAnchors)
			}
		}
		if mode == "multi-hop4" {
			if anchors == 0 || value.InputFingerprint != fingerprint || value.AnalysisID == id || value.MethodMetadata["baseline_context_policy"] == "" {
				t.Fatal("invalid experiment isolation")
			}
		} else if base == nil {
			base = data
			fingerprint = value.InputFingerprint
			id = value.AnalysisID
			if anchors != 0 {
				t.Fatal("production expanded anchors")
			}
		} else if string(base) != string(data) {
			t.Fatal("experiment changed subsequent production output")
		}
	}
}
