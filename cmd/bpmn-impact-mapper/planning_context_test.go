package main

import (
	"bytes"
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanningContextExportUsesSnapshotAndRefusesOverwrite(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "golden", "report.json")
	var stdout bytes.Buffer
	if err := run([]string{"planning-context", "--report", file}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"context_id"`)) || bytes.Contains(stdout.Bytes(), []byte(`"plans"`)) {
		t.Fatal("export did not produce a context-only package")
	}
	output := filepath.Join(t.TempDir(), "context.json")
	args := []string{"planning-context", "--report", file, "--output", output}
	if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(output)
	if !bytes.Equal(got, stdout.Bytes()) {
		t.Fatal("non-deterministic exported context")
	}
	if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("overwrote an existing context")
	}
}

func TestPlanningContextSmallerModelBudget(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "golden", "report.json")
	var out bytes.Buffer
	if err := run([]string{"planning-context", "--report", file, "--candidates", "1", "--related", "0", "--evidence", "1", "--excerpt", "32"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var bundle planning.EvidenceBundle
	if err := json.Unmarshal(out.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	for _, change := range bundle.Changes {
		if len(change.Candidates) > 1 {
			t.Fatal("candidate bound exceeded")
		}
		for _, candidate := range change.Candidates {
			if len(candidate.Evidence) > 1 {
				t.Fatal("evidence bound exceeded")
			}
			for _, e := range candidate.Evidence {
				if len([]rune(e.Excerpt)) > 32 {
					t.Fatal("excerpt bound exceeded")
				}
			}
		}
	}
	if err := run([]string{"planning-context", "--report", file, "--candidates", "0"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("invalid bound accepted")
	}
}
