// Offline calibration and transfer evaluation. Runtime selection never sees labels.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/selection"
	"os"
	"path/filepath"
)

type studyRow struct {
	Scenario     string `json:"scenario"`
	Variant      string `json:"variant"`
	Hash         string `json:"prediction_sha256"`
	Input        string `json:"input_fingerprint"`
	Reproducible bool   `json:"reproducible"`
}
type sample struct {
	scenario   string
	prediction domain.Report
	truth      evaluation.GroundTruth
}
type fileHash struct {
	File string `json:"file"`
	Hash string `json:"sha256"`
}
type stats struct {
	Nodes     int     `json:"nodes"`
	Expected  int     `json:"expected_pairs"`
	Selected  int     `json:"selected_pairs"`
	Hits      int     `json:"true_positive_pairs"`
	Deferred  int     `json:"deferred_pairs"`
	Abstained int     `json:"abstained_nodes"`
	Precision float64 `json:"actual_selected_precision"`
	Recall    float64 `json:"recall"`
}
type trial struct {
	Policy selection.Policy `json:"policy"`
	Stats  stats            `json:"stats"`
}
type chosen struct {
	Variant string           `json:"variant"`
	Policy  selection.Policy `json:"policy"`
	Reduced bool             `json:"reduces_development_candidates"`
	Trials  []trial          `json:"development_trials"`
}
type frozen struct {
	Version  string     `json:"version"`
	Rule     string     `json:"selection_rule"`
	Variants []chosen   `json:"variants"`
	Files    []fileHash `json:"development_bindings"`
}
type nodeResult struct {
	Scenario        string            `json:"scenario"`
	ID              string            `json:"element_id"`
	Selected        []string          `json:"selected"`
	Deferred        []string          `json:"deferred"`
	DeferredReasons map[string]string `json:"deferred_reasons"`
}

func read(path string) ([]byte, error) {
	s, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() || s.Size() > 16<<20 {
		return nil, fmt.Errorf("expected regular input below 16MiB")
	}
	return os.ReadFile(path)
}
func hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func write(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func load(root string) (map[string][]sample, []fileHash, error) {
	bindings := []fileHash{}
	data, err := read(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, nil, err
	}
	bindings = append(bindings, fileHash{filepath.ToSlash(filepath.Join(root, "manifest.json")), hash(data)})
	var manifest benchmark.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, err
	}
	cases := map[string]benchmark.Scenario{}
	for _, c := range manifest.Scenarios {
		cases[c.ID] = c
	}
	data, err = read(filepath.Join(root, "study-summary.json"))
	if err != nil {
		return nil, nil, err
	}
	bindings = append(bindings, fileHash{filepath.ToSlash(filepath.Join(root, "study-summary.json")), hash(data)})
	var rows []studyRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, nil, err
	}
	if len(rows) != len(cases)*2 {
		return nil, nil, fmt.Errorf("study must contain two variants for all cases")
	}
	result := map[string][]sample{}
	seen := map[string]bool{}
	for _, r := range rows {
		if !r.Reproducible || (r.Variant != "full" && r.Variant != "multi-hop4") || seen[r.Scenario+":"+r.Variant] {
			return nil, nil, fmt.Errorf("invalid study cells")
		}
		seen[r.Scenario+":"+r.Variant] = true
		c, ok := cases[r.Scenario]
		if !ok {
			return nil, nil, fmt.Errorf("unknown case")
		}
		for path, want := range map[string]string{c.OldBPMNPath: c.OldBPMNSHA256, c.NewBPMNPath: c.NewBPMNSHA256} {
			b, err := read(path)
			if err != nil {
				return nil, nil, err
			}
			if hash(b) != want {
				return nil, nil, fmt.Errorf("BPMN drift")
			}
		}
		truthBytes, err := read(c.GroundTruthPath)
		if err != nil {
			return nil, nil, err
		}
		if hash(truthBytes) != c.GroundTruthSHA256 {
			return nil, nil, fmt.Errorf("reference drift")
		}
		var truth evaluation.GroundTruth
		if err := json.Unmarshal(truthBytes, &truth); err != nil {
			return nil, nil, err
		}
		var prediction domain.Report
		for repeat := 1; repeat <= 2; repeat++ {
			path := filepath.Join(root, r.Scenario, r.Variant, fmt.Sprintf("run%d", repeat), "impact-report.json")
			b, err := read(path)
			if err != nil {
				return nil, nil, err
			}
			if hash(b) != r.Hash {
				return nil, nil, fmt.Errorf("prediction/repetition drift")
			}
			bindings = append(bindings, fileHash{filepath.ToSlash(path), hash(b)})
			if err := json.Unmarshal(b, &prediction); err != nil {
				return nil, nil, err
			}
		}
		if prediction.InputFingerprint != r.Input {
			return nil, nil, fmt.Errorf("input binding mismatch")
		}
		nodes := evaluation.PrimaryNodeProjection(prediction)
		if len(nodes.Changes) == 0 {
			if len(truth.Cases) != 0 {
				return nil, nil, fmt.Errorf("labels on unchanged input")
			}
		} else if err := evaluation.ValidateCoverage(nodes, truth); err != nil {
			return nil, nil, err
		}
		bindings = append(bindings, fileHash{c.GroundTruthPath, hash(truthBytes)})
		result[r.Variant] = append(result[r.Variant], sample{r.Scenario, nodes, truth})
	}
	return result, bindings, nil
}

func score(samples []sample, policy selection.Policy) (stats, []nodeResult, error) {
	s := stats{}
	records := []nodeResult{}
	for _, sample := range samples {
		labels := map[string]map[string]bool{}
		for _, c := range sample.truth.Cases {
			e := map[string]bool{}
			for _, f := range c.Expected {
				e[f.File] = true
			}
			labels[string(c.ChangeType)+":"+c.ElementID] = e
		}
		for _, c := range sample.prediction.Changes {
			id := ""
			if c.NewElement != nil {
				id = c.NewElement.ID
			} else if c.OldElement != nil {
				id = c.OldElement.ID
			}
			expected := labels[string(c.ChangeType)+":"+id]
			result, err := selection.Apply(c.Impacts, policy)
			if err != nil {
				return s, nil, err
			}
			s.Nodes++
			s.Expected += len(expected)
			s.Selected += len(result.Selected)
			s.Deferred += len(result.Deferred)
			if len(result.Selected) == 0 {
				s.Abstained++
			}
			record := nodeResult{Scenario: sample.scenario, ID: id, Selected: []string{}, Deferred: []string{}, DeferredReasons: map[string]string{}}
			for _, i := range result.Selected {
				record.Selected = append(record.Selected, i.File)
				if expected[i.File] {
					s.Hits++
				}
			}
			for _, i := range result.Deferred {
				record.Deferred = append(record.Deferred, i.Candidate.File)
				record.DeferredReasons[i.Candidate.File] = i.Reason
			}
			records = append(records, record)
		}
	}
	if s.Selected > 0 {
		s.Precision = float64(s.Hits) / float64(s.Selected)
	}
	if s.Expected > 0 {
		s.Recall = float64(s.Hits) / float64(s.Expected)
	}
	return s, records, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	dev := flag.String("development", "outputs/baseline-context-study", "prior development study")
	validation := flag.String("validation", "outputs/p2p-transfer-study", "new transfer study")
	output := flag.String("output", "outputs/selection-calibration", "new output directory")
	calibrateOnly := flag.Bool("calibrate-only", false, "write locked policies without reading transfer predictions")
	policyPath := flag.String("policy", "", "frozen policy JSON; skips calibration")
	flag.Parse()
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must be new")
	}
	config := frozen{}
	var err error
	if *policyPath == "" {
		data, bindings, err := load(*dev)
		if err != nil {
			return err
		}
		config = frozen{Version: "rank-agreement-selection-v1", Rule: "Choose highest actual selected precision among all/both-within-5/10/20 with no additional development true-positive pair loss; tie keeps earlier policy. No confidence probability.", Files: bindings}
		for _, variant := range []string{"full", "multi-hop4"} {
			samples := data[variant]
			base, _, err := score(samples, selection.Policy{Name: "all"})
			if err != nil {
				return err
			}
			item := chosen{Variant: variant, Policy: selection.Policy{Name: "all"}, Trials: []trial{{selection.Policy{Name: "all"}, base}}}
			best := base
			for _, bound := range []int{5, 10, 20} {
				p := selection.Policy{Name: "both-within", Bound: bound}
				s, _, err := score(samples, p)
				if err != nil {
					return err
				}
				item.Trials = append(item.Trials, trial{p, s})
				if s.Hits == base.Hits && s.Precision > best.Precision {
					item.Policy = p
					best = s
				}
			}
			item.Reduced = best.Selected < base.Selected
			config.Variants = append(config.Variants, item)
		}
	} else {
		b, err := read(*policyPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &config); err != nil {
			return err
		}
		if config.Version != "rank-agreement-selection-v1" || len(config.Variants) != 2 {
			return fmt.Errorf("invalid frozen policy")
		}
		seen := map[string]bool{}
		for _, v := range config.Variants {
			if (v.Variant != "full" && v.Variant != "multi-hop4") || seen[v.Variant] {
				return fmt.Errorf("invalid frozen variant")
			}
			seen[v.Variant] = true
			if _, err := selection.Apply(nil, v.Policy); err != nil {
				return err
			}
		}
		for _, f := range config.Files {
			b, err := read(filepath.FromSlash(f.File))
			if err != nil {
				return err
			}
			if hash(b) != f.Hash {
				return fmt.Errorf("development binding drift")
			}
		}
	}
	if err = os.MkdirAll(*output, 0700); err != nil {
		return err
	}
	if err = write(filepath.Join(*output, "policy.json"), config); err != nil {
		return err
	}
	for _, c := range config.Variants {
		fmt.Printf("%s locked policy: %s bound=%d; development reduction=%t\n", c.Variant, c.Policy.Name, c.Policy.Bound, c.Reduced)
	}
	if *calibrateOnly {
		return nil
	}
	data, bindings, err := load(*validation)
	if err != nil {
		return err
	}
	results := []struct {
		Variant   string           `json:"variant"`
		Policy    selection.Policy `json:"policy"`
		Baseline  stats            `json:"baseline"`
		Selection stats            `json:"selection"`
		Records   []nodeResult     `json:"nodes"`
	}{}
	for _, c := range config.Variants {
		base, _, err := score(data[c.Variant], selection.Policy{Name: "all"})
		if err != nil {
			return err
		}
		selected, records, err := score(data[c.Variant], c.Policy)
		if err != nil {
			return err
		}
		results = append(results, struct {
			Variant   string           `json:"variant"`
			Policy    selection.Policy `json:"policy"`
			Baseline  stats            `json:"baseline"`
			Selection stats            `json:"selection"`
			Records   []nodeResult     `json:"nodes"`
		}{c.Variant, c.Policy, base, selected, records})
		fmt.Printf("%s transfer: candidates %d -> %d; precision %.4f -> %.4f; recall %.4f -> %.4f; abstained=%d\n", c.Variant, base.Selected, selected.Selected, base.Precision, selected.Precision, base.Recall, selected.Recall, selected.Abstained)
	}
	if err = write(filepath.Join(*output, "validation-bindings.json"), bindings); err != nil {
		return err
	}
	return write(filepath.Join(*output, "validation.json"), results)
}
