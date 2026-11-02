package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func TestConditionFocusPreservesProductionBytesAndInputIdentity(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(repo, "claims.go"): "package claims\nfunc CopyClaimRejected() {}\n",
		filepath.Join(dir, "old.bpmn"):   `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><exclusiveGateway id="G" name="Copy claim allowed?"/><task id="T"/><sequenceFlow id="F" sourceRef="G" targetRef="T" name="REJECTED"/></process></definitions>`,
		filepath.Join(dir, "new.bpmn"):   `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><exclusiveGateway id="G" name="Copy claim allowed?"/><task id="T"/><sequenceFlow id="F" sourceRef="G" targetRef="T" name="REJECTED CANCELLED"/></process></definitions>`,
	}
	for p, data := range files {
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := app.Request{RepositoryPath: repo, OldBPMNPath: filepath.Join(dir, "old.bpmn"), NewBPMNPath: filepath.Join(dir, "new.bpmn"), Method: domain.MethodHybrid}
	var baseline []byte
	var fingerprint, id string
	for _, mode := range []string{"full", "condition-focus", "full"} {
		a, err := wiring.NewContextStudyAnalyzer(&constantTestEmbedder{}, mode)
		if err != nil {
			t.Fatal(err)
		}
		r, err := a.Analyze(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := report.JSON(r)
		found := false
		for _, c := range r.Changes {
			if c.ChangeType != domain.ModifyGatewayCondition {
				continue
			}
			found = true
			if mode == "condition-focus" {
				if len(c.Context.ConditionDelta) != 1 || c.Context.ConditionDelta[0] != "cancelled" {
					t.Fatal("missing auditable lexical delta")
				}
			} else if len(c.Context.ConditionDelta) != 0 {
				t.Fatal("production annotation leaked")
			}
		}
		if !found {
			t.Fatal("missing gateway condition change")
		}
		if baseline == nil {
			baseline = data
			fingerprint = r.InputFingerprint
			id = r.AnalysisID
		} else if mode == "condition-focus" {
			if r.InputFingerprint != fingerprint || r.AnalysisID == id {
				t.Fatal("input/config identities mixed")
			}
		} else if string(baseline) != string(data) {
			t.Fatal("experiment mutated subsequent production output")
		}
	}
}
