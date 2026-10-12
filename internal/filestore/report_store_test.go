package filestore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
)

func TestReportStoreRoundTrip(t *testing.T) {
	store := ReportStore{Root: t.TempDir()}
	value := domain.Report{
		AnalysisID:       "0123456789abcdef0123456789abcdef",
		InputFingerprint: "fingerprint",
		Method:           domain.MethodHybrid,
		Repository:       domain.Repository{Name: "synthetic", Revision: "v1"},
		Changes:          []domain.Change{},
		Warnings:         []domain.Warning{},
	}
	if err := store.Save(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), value); err != nil {
		t.Fatalf("idempotent save: %v", err)
	}
	loaded, err := store.Get(context.Background(), value.AnalysisID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AnalysisID != value.AnalysisID || loaded.Method != value.Method {
		t.Fatalf("loaded=%#v", loaded)
	}
	listed, err := store.List(context.Background())
	if err != nil || len(listed) != 1 || listed[0].AnalysisID != value.AnalysisID {
		t.Fatalf("listed=%#v err=%v", listed, err)
	}
	for _, name := range []string{"impact-report.json", "impact-report.md"} {
		info, err := os.Stat(filepath.Join(store.Root, value.AnalysisID, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
	}
}

func TestReportStoreListIsDeterministicAndSkipsUnsafeEntries(t *testing.T) {
	store := ReportStore{Root: t.TempDir()}
	for _, id := range []string{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		value := domain.Report{AnalysisID: id, InputFingerprint: "fingerprint", Method: domain.MethodHybrid, Repository: domain.Repository{Name: "fixture", Revision: "one"}, Changes: []domain.Change{{Impacts: []domain.Impact{{File: "file.go"}}}}, Warnings: []domain.Warning{}}
		if err := store.Save(context.Background(), value); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(store.Root, "not-an-analysis"), 0o750); err != nil {
		t.Fatal(err)
	}
	values, err := store.List(context.Background())
	if err != nil || len(values) != 2 || values[0].AnalysisID[0] != 'a' || values[0].Candidates != 1 {
		t.Fatalf("values=%#v err=%v", values, err)
	}
}

func TestReportStoreRejectsTraversalAndMissing(t *testing.T) {
	store := ReportStore{Root: t.TempDir()}
	if err := store.Save(context.Background(), domain.Report{AnalysisID: "../escape"}); err == nil {
		t.Fatal("expected invalid ID error")
	}
	_, err := store.Get(context.Background(), "0123456789abcdef0123456789abcdef")
	if !errors.Is(err, reportstore.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}
