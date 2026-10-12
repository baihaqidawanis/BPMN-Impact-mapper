package reviewpack_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reviewpack"
)

func TestBuildIsBlindAndOrderIndependent(t *testing.T) {
	a := input(domain.MethodHybrid, "hybrid-id", "hybrid bytes")
	b := input(domain.MethodBM25, "bm25-id", "bm25 bytes")
	first, err := reviewpack.Build([]reviewpack.Input{a, b}, "reviewer-a", "reviewer-b")
	if err != nil {
		t.Fatal(err)
	}
	second, err := reviewpack.Build([]reviewpack.Input{b, a}, "reviewer-a", "reviewer-b")
	if err != nil {
		t.Fatal(err)
	}
	left, _ := reviewpack.JSON(first)
	right, _ := reviewpack.JSON(second)
	if !bytes.Equal(left, right) {
		t.Fatal("pack changed when report argument order changed")
	}
	guide := reviewpack.Guide(first.Manifest)
	for _, forbidden := range []string{"hybrid-id", "bm25-id", "service.go", "confidence", "0.9", "BM25"} {
		if bytes.Contains(guide, []byte(forbidden)) {
			t.Fatalf("blind guide leaked %q", forbidden)
		}
	}
	template, _ := reviewpack.JSON(first.ReviewerA)
	if bytes.Contains(template, []byte("service.go")) || !bytes.Contains(template, []byte(`"file": "REVIEW_REQUIRED"`)) {
		t.Fatalf("review template leaked predictions or lacks fail-closed sentinel: %s", template)
	}
	if _, err := review.Reconcile(review.Annotation(first.ReviewerA), review.Annotation(first.ReviewerB)); err == nil {
		t.Fatal("untouched templates must not be accepted as ground truth")
	}
	if len(first.Manifest.Predictions) != 2 || first.Manifest.PackID == "" || first.Manifest.CaseDigest == "" {
		t.Fatal("coordinator provenance is incomplete")
	}
}

func TestBuildRejectsMismatchedAndInvalidInputs(t *testing.T) {
	base := input(domain.MethodHybrid, "one", "one")
	tests := []struct {
		name   string
		inputs []reviewpack.Input
	}{
		{"duplicate method", []reviewpack.Input{base, input(domain.MethodHybrid, "two", "two")}},
		{"duplicate analysis", []reviewpack.Input{base, input(domain.MethodBM25, "one", "two")}},
		{"fingerprint mismatch", []reviewpack.Input{base, mutate(input(domain.MethodBM25, "two", "two"), func(report *domain.Report) { report.InputFingerprint = "other" })}},
		{"repository mismatch", []reviewpack.Input{base, mutate(input(domain.MethodBM25, "two", "two"), func(report *domain.Report) { report.Repository.Revision = "other" })}},
		{"changes mismatch", []reviewpack.Input{base, mutate(input(domain.MethodBM25, "two", "two"), func(report *domain.Report) { report.Changes[0].NewElement.Name = "different" })}},
		{"duplicate case", []reviewpack.Input{mutate(base, func(report *domain.Report) { report.Changes = append(report.Changes, report.Changes[0]) })}},
		{"empty changes", []reviewpack.Input{mutate(base, func(report *domain.Report) { report.Changes = nil })}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := reviewpack.Build(test.inputs, "a", "b"); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if _, err := reviewpack.Build([]reviewpack.Input{base}, "same", "same"); err == nil {
		t.Fatal("expected independent reviewer error")
	}
}

func TestManifestValidationAndCompletedCoverage(t *testing.T) {
	pack, err := reviewpack.Build([]reviewpack.Input{input(domain.MethodHybrid, "one", "one")}, "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if err := reviewpack.ValidateManifest(pack.Manifest); err != nil {
		t.Fatal(err)
	}
	completed := pack.ReviewerA.Cases
	completed[0].Expected = []evaluation.ExpectedImpact{}
	if err := reviewpack.ValidateCompletedCases(pack.Manifest, completed); err != nil {
		t.Fatal(err)
	}
	if err := reviewpack.ValidateCompletedCases(pack.Manifest, nil); err == nil {
		t.Fatal("expected missing coverage error")
	}
	tampered := pack.Manifest
	tampered.CaseDigest = strings.Repeat("0", 64)
	if err := reviewpack.ValidateManifest(tampered); err == nil {
		t.Fatal("expected manifest integrity error")
	}
}

func input(method domain.AnalysisMethod, analysisID, content string) reviewpack.Input {
	digest := sha256.Sum256([]byte(content))
	return reviewpack.Input{SHA256: hex.EncodeToString(digest[:]), Report: domain.Report{
		AnalysisID: analysisID, InputFingerprint: strings.Repeat("f", 32), Method: method,
		Repository: domain.Repository{Name: "fixture", Revision: "rev-1"},
		Changes:    []domain.Change{{ChangeType: domain.ModifyTaskLabel, OldElement: &domain.Element{ID: "TASK-1", Name: "Old", Type: "task"}, NewElement: &domain.Element{ID: "TASK-1", Name: "New", Type: "task"}, Reason: "element label changed", Impacts: []domain.Impact{{File: "service.go", Confidence: 0.9}}}},
	}}
}

func mutate(value reviewpack.Input, change func(*domain.Report)) reviewpack.Input {
	change(&value.Report)
	return value
}
