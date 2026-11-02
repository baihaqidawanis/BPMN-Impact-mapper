// Offline research evaluation of saved predictions; no inference or ERP execution.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"os"
	"path/filepath"
	"strings"
)

type row struct {
	Method           domain.AnalysisMethod `json:"method"`
	AnalysisID       string                `json:"analysis_id"`
	PredictionSHA256 string                `json:"prediction_sha256"`
	NodeChanges      int                   `json:"node_changes"`
	ExcludedChanges  int                   `json:"excluded_changes"`
	Metrics          evaluation.Metrics    `json:"metrics"`
	Union5           evaluation.FileUnion  `json:"union_top_5_per_node"`
	Union10          evaluation.FileUnion  `json:"union_top_10_per_node"`
}

func read(path string) ([]byte, error) {
	s, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() || s.Size() > 16<<20 {
		return nil, fmt.Errorf("expected regular JSON below 16 MiB")
	}
	return os.ReadFile(path)
}
func hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	reports := flag.String("reports", "outputs/current-staff-benchmark-final/normalized-staff-combined", "saved four-method prediction directory")
	reference := flag.String("reference", "experiments/simple-erp/current-staff/combined/node-reference.json", "expanded frozen node reference")
	output := flag.String("output", "outputs/compound-node-evaluation", "new output directory")
	reviewPath := flag.String("review", "experiments/simple-erp/current-staff/combined/node-reference-review.json", "reference/source/BPMN hash binding")
	benchmarkPath := flag.String("benchmark", "outputs/current-staff-benchmark-final/benchmark-report.json", "original saved benchmark binding")
	manifestPath := flag.String("manifest", "experiments/simple-erp/current-staff/manifest.json", "original benchmark inputs")
	caseID := flag.String("case-id", "normalized-staff-combined", "saved benchmark case id")
	repo := flag.String("repo", "../Simple-ERP", "local source snapshot to verify, read only")
	flag.Parse()
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must be new")
	}
	truthBytes, err := read(*reference)
	if err != nil {
		return err
	}
	var truth evaluation.GroundTruth
	if err := json.Unmarshal(truthBytes, &truth); err != nil {
		return err
	}
	if err := truth.Validate(); err != nil {
		return err
	}
	for _, item := range truth.Cases {
		if len(item.Expected) == 0 {
			return fmt.Errorf("node diagnostic requires positive references; evaluate negative controls separately")
		}
	}
	reviewBytes, err := read(*reviewPath)
	if err != nil {
		return err
	}
	var binding struct {
		BaselinePath  string            `json:"baseline_path"`
		BaselineHash  string            `json:"baseline_sha256"`
		ChangedPath   string            `json:"changed_path"`
		ChangedHash   string            `json:"changed_sha256"`
		ReferenceHash string            `json:"reference_sha256"`
		Sources       map[string]string `json:"source_hashes"`
	}
	if err := json.Unmarshal(reviewBytes, &binding); err != nil {
		return err
	}
	if hash(truthBytes) != binding.ReferenceHash || len(binding.Sources) == 0 {
		return fmt.Errorf("reference binding mismatch")
	}
	for path, want := range map[string]string{binding.BaselinePath: binding.BaselineHash, binding.ChangedPath: binding.ChangedHash} {
		data, err := read(path)
		if err != nil {
			return err
		}
		if hash(data) != want {
			return fmt.Errorf("BPMN binding drift")
		}
	}
	for path, want := range binding.Sources {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if filepath.IsAbs(path) || filepath.VolumeName(path) != "" || clean != path || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("invalid relative source")
		}
		data, err := read(filepath.Join(*repo, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		if hash(data) != want {
			return fmt.Errorf("source snapshot drift: %s", path)
		}
	}
	manifestBytes, err := read(*manifestPath)
	if err != nil {
		return err
	}
	var manifest benchmark.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return err
	}
	foundCase := false
	for _, c := range manifest.Scenarios {
		if c.ID == *caseID {
			foundCase = true
			if c.OldBPMNSHA256 != binding.BaselineHash || c.NewBPMNSHA256 != binding.ChangedHash {
				return fmt.Errorf("saved input binding mismatch")
			}
		}
	}
	if !foundCase {
		return fmt.Errorf("case missing in original manifest")
	}
	benchmarkBytes, err := read(*benchmarkPath)
	if err != nil {
		return err
	}
	var saved benchmark.Report
	if err := json.Unmarshal(benchmarkBytes, &saved); err != nil {
		return err
	}
	// All labels must match all changed nodes; incomplete labels are never silently
	// reported as complete node coverage. Original full prediction bytes bind hashes.
	rows := []row{}
	artifacts := map[string][]byte{}
	fingerprint := ""
	for _, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion, domain.MethodHybrid} {
		data, err := read(filepath.Join(*reports, string(method), "impact-report.json"))
		if err != nil {
			return err
		}
		var prediction domain.Report
		if err := json.Unmarshal(data, &prediction); err != nil {
			return err
		}
		if prediction.Method != method {
			return fmt.Errorf("prediction method mismatch")
		}
		bound := false
		for _, r := range saved.Results {
			if r.ScenarioID == *caseID && r.Method == method {
				bound = true
				if r.PredictionSHA256 != hash(data) || r.AnalysisID != prediction.AnalysisID || r.InputFingerprint != prediction.InputFingerprint || !r.Reproducible {
					return fmt.Errorf("saved prediction binding mismatch")
				}
			}
		}
		if !bound {
			return fmt.Errorf("missing saved method result")
		}
		if fingerprint == "" {
			fingerprint = prediction.InputFingerprint
		}
		if prediction.InputFingerprint != fingerprint {
			return fmt.Errorf("cross-method input drift")
		}
		nodes := evaluation.PrimaryNodeProjection(prediction)
		if err := evaluation.ValidateCoverage(nodes, truth); err != nil {
			return fmt.Errorf("%s primary-node coverage: %w", method, err)
		}
		result, err := evaluation.Evaluate(nodes, data, truth, truthBytes)
		if err != nil {
			return err
		}
		u5, err := evaluation.NodeFileUnion(prediction, truth, 5)
		if err != nil {
			return err
		}
		u10, err := evaluation.NodeFileUnion(prediction, truth, 10)
		if err != nil {
			return err
		}
		rows = append(rows, row{method, prediction.AnalysisID, hash(data), len(nodes.Changes), len(prediction.Changes) - len(nodes.Changes), result.Metrics, u5, u10})
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		artifacts[string(method)+"-evaluation.json"] = append(encoded, '\n')
	}
	summary := struct {
		Scope         string `json:"scope"`
		Status        string `json:"reference_status"`
		ReferenceHash string `json:"reference_sha256"`
		Fingerprint   string `json:"input_fingerprint"`
		Rows          []row  `json:"rows"`
	}{"all changed primary nodes; sequence flows excluded", "retrospective exposed development case; provisional source review", hash(truthBytes), fingerprint, rows}
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	artifacts["node-evaluation.json"] = append(encoded, '\n')
	artifacts["reference.json"] = truthBytes
	artifacts["reference-review.json"] = reviewBytes
	artifacts["node-evaluation.md"] = render(rows, hash(truthBytes))
	if err := os.MkdirAll(*output, 0700); err != nil {
		return err
	}
	for name, data := range artifacts {
		if err := os.WriteFile(filepath.Join(*output, name), data, 0600); err != nil {
			return err
		}
	}
	for _, r := range rows {
		p := r.Metrics.PositiveRanking
		fmt.Printf("%s: nodes=%d P5=%.4f R5=%.4f MAP=%.4f union5=%d files precision=%.4f recall=%.4f\n", r.Method, r.NodeChanges, p.Precision5, p.Recall5, p.MAP, len(r.Union5.Predicted), r.Union5.Precision, r.Union5.Recall)
	}
	return nil
}

func render(rows []row, referenceHash string) []byte {
	var out strings.Builder
	fmt.Fprintf(&out, "# Compound primary-node evaluation\n\nReference SHA-256: `%s`. Retrospective, exposed development case with provisional source-derived labels.\n\n", referenceHash)
	out.WriteString("All seven changed task/gateway nodes are scored; sequence-flow changes are excluded, not declared negative. File-level metrics do not prove function correctness or implementation changes.\n\n| Method | Nodes | Excluded flow changes | P@5 | R@5 | MAP | MRR |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		p := r.Metrics.PositiveRanking
		fmt.Fprintf(&out, "| %s | %d | %d | %.4f | %.4f | %.4f | %.4f |\n", r.Method, r.NodeChanges, r.ExcludedChanges, p.Precision5, p.Recall5, p.MAP, p.MRR)
	}
	out.WriteString("\n## Union of each node's top 5 distinct files\n\nThis is an unranked set, not Precision@5 for one ranked list. Shared expected files count once; returned file count is the actual union denominator.\n\n| Method | Expected files | Returned files | Precision | Recall |\n|---|---:|---:|---:|---:|\n")
	for _, r := range rows {
		u := r.Union5
		fmt.Fprintf(&out, "| %s | %d | %d | %.4f | %.4f |\n", r.Method, len(u.Expected), len(u.Predicted), u.Precision, u.Recall)
	}
	out.WriteString("\nThese seven nodes describe related requirements in one scenario, not seven independent business cases. References were expanded after the original benchmark existed. Prediction hashes are bound to that original benchmark; no new inference, ranking tuning, repeated-run claim or held-out-validation claim is made. Required-action classification remains separate from file relevance.\n")
	return []byte(out.String())
}
