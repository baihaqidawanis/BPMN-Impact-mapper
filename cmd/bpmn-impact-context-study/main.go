// Offline development study. Ground truth is read only after prediction.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/experiment"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type row struct {
	Scope            string                `json:"scope,omitempty"`
	Union5           *evaluation.FileUnion `json:"union_top5_per_node,omitempty"`
	Union10          *evaluation.FileUnion `json:"union_top10_per_node,omitempty"`
	Scenario         string                `json:"scenario"`
	Variant          string                `json:"variant"`
	AnalysisID       string                `json:"analysis_id"`
	Input            string                `json:"input_fingerprint"`
	PredictionSHA256 string                `json:"prediction_sha256"`
	Reproducible     bool                  `json:"reproducible"`
	Metrics          evaluation.Metrics    `json:"metrics"`
}

func run() error {
	manifestPath := flag.String("manifest", "experiments/simple-erp/staff-consolidated-2026-10-29-manifest.json", "frozen benchmark manifest; paths relative to cwd")
	output := flag.String("output", "outputs/context-study", "new output directory")
	timeout := flag.Duration("timeout", time.Hour, "whole study deadline")
	variantList := flag.String("variants", "full,no-context,no-start-end,both-top10", "comma-separated experimental modes")
	nodeScope := flag.Bool("node-scope", false, "require labels for all changed primary nodes; exclude flow changes")
	flag.Parse()
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must be a new directory")
	}
	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		return err
	}
	var manifest benchmark.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	store, err := experiment.NewLocalFiles(*output)
	if err != nil {
		return err
	}
	if err := store.Write("manifest.json", data); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	embedder := semantic.NewPythonEmbedderFromEnvironment()
	variants := strings.Split(*variantList, ",")
	seenVariants := map[string]bool{}
	for _, variant := range variants {
		if seenVariants[variant] {
			return fmt.Errorf("duplicate variant")
		}
		seenVariants[variant] = true
		if _, err := wiring.NewContextStudyAnalyzer(embedder, variant); err != nil {
			return err
		}
	}
	rows := []row{}
	for _, scenario := range manifest.Scenarios {
		if err := verify(scenario.OldBPMNPath, scenario.OldBPMNSHA256); err != nil {
			return err
		}
		if err := verify(scenario.NewBPMNPath, scenario.NewBPMNSHA256); err != nil {
			return err
		}
		fingerprint := ""
		for _, variant := range variants {
			analyzer, err := wiring.NewContextStudyAnalyzer(embedder, variant)
			if err != nil {
				return err
			}
			var firstHash string
			var first evaluation.Report
			var union5, union10 *evaluation.FileUnion
			for repeat := 1; repeat <= manifest.Repetitions; repeat++ {
				fmt.Printf("%s %s run %d\n", scenario.ID, variant, repeat)
				prediction, err := analyzer.Analyze(ctx, app.Request{RepositoryPath: scenario.RepositoryPath, RepositoryName: scenario.RepositoryName, Revision: scenario.Revision, OldBPMNPath: scenario.OldBPMNPath, NewBPMNPath: scenario.NewBPMNPath, Method: domain.MethodHybrid, Retrieval: mapping.Options{TopK: manifest.TopK}})
				if err != nil {
					return err
				}
				if fingerprint == "" {
					fingerprint = prediction.InputFingerprint
				}
				if prediction.InputFingerprint != fingerprint {
					return fmt.Errorf("inputs changed during study: %s", scenario.ID)
				}
				bytes, err := report.JSON(prediction)
				if err != nil {
					return err
				}
				prefix := filepath.ToSlash(filepath.Join(scenario.ID, variant, fmt.Sprintf("run%d", repeat)))
				if err := store.Write(prefix+"/impact-report.json", bytes); err != nil {
					return err
				}
				// Closed labels never enter analyzer requests or constructors.
				if err := verify(scenario.GroundTruthPath, scenario.GroundTruthSHA256); err != nil {
					return err
				}
				truthBytes, err := os.ReadFile(scenario.GroundTruthPath)
				if err != nil {
					return err
				}
				var truth evaluation.GroundTruth
				if err := json.Unmarshal(truthBytes, &truth); err != nil {
					return err
				}
				view := prediction
				if *nodeScope {
					view = evaluation.PrimaryNodeProjection(prediction)
					if len(view.Changes) == 0 {
						if len(truth.Cases) > 0 {
							return fmt.Errorf("labels on unchanged control")
						}
					} else {
						if err := evaluation.ValidateCoverage(view, truth); err != nil {
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
						union5 = &u5
						union10 = &u10
					}
				}
				evaluated, err := evaluation.Evaluate(view, bytes, truth, truthBytes)
				if err != nil {
					return err
				}
				evalBytes, err := evaluation.JSON(evaluated)
				if err != nil {
					return err
				}
				if err := store.Write(prefix+"/evaluation.json", evalBytes); err != nil {
					return err
				}
				current := hash(bytes)
				if repeat == 1 {
					firstHash, first = current, evaluated
				} else if current != firstHash {
					return fmt.Errorf("non-reproducible prediction: %s %s", scenario.ID, variant)
				}
			}
			scope := ""
			if *nodeScope {
				scope = "primary-nodes; flow changes excluded"
			}
			rows = append(rows, row{Scope: scope, Union5: union5, Union10: union10, Scenario: scenario.ID, Variant: variant, AnalysisID: first.AnalysisID, Input: fingerprint, PredictionSHA256: firstHash, Reproducible: true, Metrics: first.Metrics})
			summary, _ := json.MarshalIndent(rows, "", "  ")
			if err := store.Write("study-summary.json", append(summary, '\n')); err != nil {
				return err
			}
		}
	}
	return nil
}
func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func verify(path, expected string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if hash(data) != expected {
		return fmt.Errorf("frozen input changed: %s", path)
	}
	return nil
}
