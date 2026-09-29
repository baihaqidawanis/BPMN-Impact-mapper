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
	for _, name := range []string{"impact-report.json", "impact-report.md"} {
		info, err := os.Stat(filepath.Join(store.Root, value.AnalysisID, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
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
