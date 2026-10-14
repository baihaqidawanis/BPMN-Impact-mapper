package filestore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planstore"
)

func TestPlanStoreRoundTripIsImmutable(t *testing.T) {
	store := PlanStore{Root: t.TempDir()}
	value := fixturePlan(t)
	if err := store.Save(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), value); err != nil {
		t.Fatalf("idempotent save: %v", err)
	}
	loaded, err := store.Get(context.Background(), value.PlanID)
	if err != nil || !reflect.DeepEqual(loaded, value) {
		t.Fatalf("loaded=%#v err=%v", loaded, err)
	}
	listed, err := store.ListByAnalysis(context.Background(), value.AnalysisID)
	if err != nil || len(listed) != 1 || listed[0].PlanID != value.PlanID || listed[0].Status != planning.StatusDisabled {
		t.Fatalf("listed=%#v err=%v", listed, err)
	}
	for _, name := range []string{"change-plan.json", "change-plan.md"} {
		info, err := os.Stat(filepath.Join(store.Root, value.PlanID, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
	}

	changed := value
	changed.Status = planning.StatusComplete
	if err := store.Save(context.Background(), changed); err == nil {
		t.Fatal("expected changed plan with stale identity to be rejected")
	}
}

func TestPlanStoreRejectsTraversalAndTampering(t *testing.T) {
	store := PlanStore{Root: t.TempDir()}
	if _, err := store.Get(context.Background(), "../escape"); !errors.Is(err, planstore.ErrNotFound) {
		t.Fatalf("traversal error=%v", err)
	}
	value := fixturePlan(t)
	if err := store.Save(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, value.PlanID, "change-plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-3] = 'x'
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), value.PlanID); err == nil {
		t.Fatal("expected tampered plan to be rejected")
	}
}

func fixturePlan(t *testing.T) planning.ChangePlan {
	t.Helper()
	value := planning.ChangePlan{
		Version: "1", AnalysisID: "0123456789abcdef0123456789abcdef",
		ContextID: "abcdef0123456789abcdef0123456789",
		Provider:  planning.ProviderIdentity{Provider: "none", Model: "none", PromptVersion: "none"},
		Status:    planning.StatusDisabled, Plans: []planning.FilePlan{}, ReviewIssues: []string{},
	}
	id, err := planning.ComputePlanID(value)
	if err != nil {
		t.Fatal(err)
	}
	value.PlanID = id
	return value
}
