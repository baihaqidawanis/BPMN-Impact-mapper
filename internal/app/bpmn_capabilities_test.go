package app_test

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUncomparedMultiInstanceEditCannotLookLikeCompleteCleanAnalysis(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	old := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><userTask id="A" name="Review claim"/></process></definitions>`
	next := strings.Replace(old, `name="Review claim"/>`, `name="Review claim"><multiInstanceLoopCharacteristics isSequential="false"/></userTask>`, 1)
	before, after := filepath.Join(root, "old.bpmn"), filepath.Join(root, "new.bpmn")
	for path, data := range map[string]string{before: old, after: next} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	analyzer := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{})
	req := app.Request{RepositoryPath: repo, OldBPMNPath: before, NewBPMNPath: after, Method: domain.MethodBM25}
	first, err := analyzer.Analyze(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Changes) != 0 {
		t.Fatal("test must exercise a currently unmodelled semantic-only edit")
	}
	found := false
	for _, warning := range first.Warnings {
		if warning.Code == "BPMN_PARTIAL_SUPPORT" && strings.Contains(warning.Message, "changed:") && strings.Contains(warning.Message, "multi-instance") {
			found = true
		}
	}
	if !found || first.MethodMetadata["bpmn_capability_policy"] == "" {
		t.Fatal("report implies complete clean diff", first)
	}
	json1, _ := report.JSON(first)
	second, err := analyzer.Analyze(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	json2, _ := report.JSON(second)
	if string(json1) != string(json2) {
		t.Fatal("diagnostics made report nondeterministic")
	}
	if !strings.Contains(string(report.Markdown(first)), "BPMN_PARTIAL_SUPPORT") {
		t.Fatal("Markdown hid capability warning")
	}
}
