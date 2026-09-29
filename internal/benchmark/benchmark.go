// Package benchmark orchestrates fair repeated experiments through injected
// ports. It never exposes hidden ground truth to the analyzer.
package benchmark

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type Manifest struct {
	Version     string                  `json:"version"`
	Repetitions int                     `json:"repetitions"`
	TopK        int                     `json:"top_k"`
	Methods     []domain.AnalysisMethod `json:"methods"`
	Scenarios   []Scenario              `json:"scenarios"`
}

type Scenario struct {
	ID                       string `json:"id"`
	RepositoryPath           string `json:"repository_path"`
	RepositoryName           string `json:"repository_name"`
	Revision                 string `json:"revision"`
	OldBPMNPath              string `json:"old_bpmn_path"`
	OldBPMNSHA256            string `json:"old_bpmn_sha256"`
	NewBPMNPath              string `json:"new_bpmn_path"`
	NewBPMNSHA256            string `json:"new_bpmn_sha256"`
	MappingConfigPath        string `json:"mapping_config_path"`
	MappingConfigSHA256      string `json:"mapping_config_sha256"`
	GroundTruthPath          string `json:"ground_truth_path"`
	GroundTruthSHA256        string `json:"ground_truth_sha256"`
	ExpectedInputFingerprint string `json:"expected_input_fingerprint,omitempty"`
}

type Analyzer interface {
	Analyze(context.Context, app.Request) (domain.Report, error)
}

type Files interface {
	Read(path string, maximumBytes int64) ([]byte, error)
	Write(relativePath string, data []byte) error
}

type PredictionRenderer interface {
	JSON(domain.Report) ([]byte, error)
	Markdown(domain.Report) []byte
}

type Runner struct {
	Analyzer    Analyzer
	Files       Files
	Predictions PredictionRenderer
	Now         func() time.Time
}

type ScenarioMethodResult struct {
	ScenarioID       string                `json:"scenario_id"`
	Method           domain.AnalysisMethod `json:"method"`
	AnalysisID       string                `json:"analysis_id"`
	InputFingerprint string                `json:"input_fingerprint"`
	PredictionSHA256 string                `json:"prediction_sha256"`
	EvaluationID     string                `json:"evaluation_id"`
	Repetitions      int                   `json:"repetitions"`
	Reproducible     bool                  `json:"reproducible"`
	Metrics          evaluation.Metrics    `json:"metrics"`
}

type MethodAggregate struct {
	Method    domain.AnalysisMethod `json:"method"`
	Scenarios int                   `json:"scenarios"`
	Metrics   evaluation.Metrics    `json:"metrics"`
}

type Report struct {
	BenchmarkID string                 `json:"benchmark_id"`
	Results     []ScenarioMethodResult `json:"results"`
	Aggregates  []MethodAggregate      `json:"aggregates"`
}

type Timing struct {
	BenchmarkID string        `json:"benchmark_id"`
	Runs        []TimingEntry `json:"runs"`
}

type TimingEntry struct {
	ScenarioID   string                `json:"scenario_id"`
	Method       domain.AnalysisMethod `json:"method"`
	Repetition   int                   `json:"repetition"`
	Milliseconds int64                 `json:"milliseconds"`
}

func (manifest Manifest) Validate() error {
	if manifest.Version != "1" {
		return fmt.Errorf("unsupported benchmark manifest version %q", manifest.Version)
	}
	if manifest.Repetitions < 2 || manifest.Repetitions > 10 {
		return errors.New("benchmark repetitions must be between 2 and 10")
	}
	if manifest.TopK <= 0 || manifest.TopK > 100 {
		return errors.New("benchmark top_k must be between 1 and 100")
	}
	methodSet := map[domain.AnalysisMethod]bool{}
	for _, method := range manifest.Methods {
		if !method.Valid() || methodSet[method] {
			return fmt.Errorf("invalid or duplicate benchmark method %q", method)
		}
		methodSet[method] = true
	}
	for _, required := range []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodConfiguration} {
		if !methodSet[required] {
			return fmt.Errorf("benchmark manifest requires method %q", required)
		}
	}
	seen := map[string]bool{}
	for index, scenario := range manifest.Scenarios {
		if strings.TrimSpace(scenario.ID) == "" || seen[scenario.ID] {
			return fmt.Errorf("scenario %d has empty or duplicate id %q", index, scenario.ID)
		}
		seen[scenario.ID] = true
		if scenario.RepositoryPath == "" || scenario.OldBPMNPath == "" || scenario.NewBPMNPath == "" || scenario.MappingConfigPath == "" || scenario.GroundTruthPath == "" {
			return fmt.Errorf("scenario %s is missing required paths", scenario.ID)
		}
		for label, hash := range map[string]string{"old BPMN": scenario.OldBPMNSHA256, "new BPMN": scenario.NewBPMNSHA256, "mapping config": scenario.MappingConfigSHA256, "ground truth": scenario.GroundTruthSHA256} {
			if !validHash(hash) {
				return fmt.Errorf("scenario %s has invalid %s SHA-256", scenario.ID, label)
			}
		}
		if scenario.ExpectedInputFingerprint != "" && len(scenario.ExpectedInputFingerprint) != 32 {
			return fmt.Errorf("scenario %s has invalid expected input fingerprint", scenario.ID)
		}
	}
	if len(manifest.Scenarios) == 0 {
		return errors.New("benchmark manifest requires at least one scenario")
	}
	return nil
}

func (runner Runner) Run(ctx context.Context, manifest Manifest) (Report, Timing, error) {
	if err := manifest.Validate(); err != nil {
		return Report{}, Timing{}, err
	}
	if runner.Analyzer == nil || runner.Files == nil || runner.Predictions == nil {
		return Report{}, Timing{}, errors.New("benchmark runner ports are not configured")
	}
	if runner.Now == nil {
		runner.Now = time.Now
	}
	scenarios := append([]Scenario(nil), manifest.Scenarios...)
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].ID < scenarios[j].ID })
	methods := append([]domain.AnalysisMethod(nil), manifest.Methods...)
	sort.Slice(methods, func(i, j int) bool { return methods[i] < methods[j] })
	var results []ScenarioMethodResult
	var timings []TimingEntry
	for _, scenario := range scenarios {
		if err := runner.verifyPublicInputs(scenario); err != nil {
			return Report{}, Timing{}, err
		}
		commonFingerprint := ""
		for _, method := range methods {
			var predictionHash, evaluationID, analysisID string
			var firstMetrics evaluation.Metrics
			reproducible := true
			for repetition := 1; repetition <= manifest.Repetitions; repetition++ {
				request := app.Request{RepositoryPath: scenario.RepositoryPath, RepositoryName: scenario.RepositoryName, Revision: scenario.Revision, OldBPMNPath: scenario.OldBPMNPath, NewBPMNPath: scenario.NewBPMNPath, Method: method, Retrieval: mapping.Options{TopK: manifest.TopK}}
				if method == domain.MethodConfiguration {
					request.MappingConfigPath = scenario.MappingConfigPath
				}
				started := runner.Now()
				prediction, err := runner.Analyzer.Analyze(ctx, request)
				elapsed := runner.Now().Sub(started)
				if err != nil {
					return Report{}, Timing{}, fmt.Errorf("scenario %s method %s repetition %d: %w", scenario.ID, method, repetition, err)
				}
				timings = append(timings, TimingEntry{ScenarioID: scenario.ID, Method: method, Repetition: repetition, Milliseconds: elapsed.Milliseconds()})
				if commonFingerprint == "" {
					commonFingerprint = prediction.InputFingerprint
				}
				if prediction.InputFingerprint != commonFingerprint || (scenario.ExpectedInputFingerprint != "" && prediction.InputFingerprint != scenario.ExpectedInputFingerprint) {
					return Report{}, Timing{}, fmt.Errorf("scenario %s input fingerprint drift", scenario.ID)
				}
				predictionBytes, err := runner.Predictions.JSON(prediction)
				if err != nil {
					return Report{}, Timing{}, err
				}
				currentHash := sha256Hex(predictionBytes)
				truthBytes, err := runner.Files.Read(scenario.GroundTruthPath, 8<<20)
				if err != nil {
					return Report{}, Timing{}, fmt.Errorf("scenario %s ground truth after inference: %w", scenario.ID, err)
				}
				if sha256Hex(truthBytes) != strings.ToLower(scenario.GroundTruthSHA256) {
					return Report{}, Timing{}, fmt.Errorf("scenario %s ground truth hash mismatch", scenario.ID)
				}
				var truth evaluation.GroundTruth
				if err := decodeStrict(truthBytes, &truth); err != nil {
					return Report{}, Timing{}, err
				}
				evaluated, err := evaluation.Evaluate(prediction, predictionBytes, truth, truthBytes)
				if err != nil {
					return Report{}, Timing{}, err
				}
				if repetition == 1 {
					predictionHash, evaluationID, analysisID, firstMetrics = currentHash, evaluated.EvaluationID, prediction.AnalysisID, evaluated.Metrics
					if err := runner.writeRunArtifacts(scenario.ID, method, prediction, predictionBytes, evaluated); err != nil {
						return Report{}, Timing{}, err
					}
				} else if currentHash != predictionHash || evaluated.EvaluationID != evaluationID {
					reproducible = false
				}
			}
			if !reproducible {
				return Report{}, Timing{}, fmt.Errorf("scenario %s method %s is not reproducible", scenario.ID, method)
			}
			results = append(results, ScenarioMethodResult{ScenarioID: scenario.ID, Method: method, AnalysisID: analysisID, InputFingerprint: commonFingerprint, PredictionSHA256: predictionHash, EvaluationID: evaluationID, Repetitions: manifest.Repetitions, Reproducible: true, Metrics: firstMetrics})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].ScenarioID != results[j].ScenarioID {
			return results[i].ScenarioID < results[j].ScenarioID
		}
		return results[i].Method < results[j].Method
	})
	report := Report{Results: results, Aggregates: aggregate(results)}
	seed, _ := json.Marshal(report)
	report.BenchmarkID = sha256Hex(seed)[:32]
	return report, Timing{BenchmarkID: report.BenchmarkID, Runs: timings}, nil
}

func (runner Runner) verifyPublicInputs(scenario Scenario) error {
	inputs := []struct {
		path, expected, label string
		limit                 int64
	}{
		{scenario.OldBPMNPath, scenario.OldBPMNSHA256, "old BPMN", 8 << 20},
		{scenario.NewBPMNPath, scenario.NewBPMNSHA256, "new BPMN", 8 << 20},
		{scenario.MappingConfigPath, scenario.MappingConfigSHA256, "mapping config", 1 << 20},
	}
	for _, input := range inputs {
		data, err := runner.Files.Read(input.path, input.limit)
		if err != nil {
			return err
		}
		if sha256Hex(data) != strings.ToLower(input.expected) {
			return fmt.Errorf("scenario %s %s hash mismatch", scenario.ID, input.label)
		}
	}
	return nil
}

func (runner Runner) writeRunArtifacts(scenarioID string, method domain.AnalysisMethod, prediction domain.Report, predictionBytes []byte, evaluated evaluation.Report) error {
	base := scenarioID + "/" + string(method)
	artifacts := map[string][]byte{base + "/impact-report.json": predictionBytes, base + "/impact-report.md": runner.Predictions.Markdown(prediction), base + "/evaluation-report.md": evaluation.Markdown(evaluated)}
	evaluationJSON, err := evaluation.JSON(evaluated)
	if err != nil {
		return err
	}
	artifacts[base+"/evaluation-report.json"] = evaluationJSON
	keys := make([]string, 0, len(artifacts))
	for key := range artifacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := runner.Files.Write(key, artifacts[key]); err != nil {
			return err
		}
	}
	return nil
}

func aggregate(results []ScenarioMethodResult) []MethodAggregate {
	groups := map[domain.AnalysisMethod][]evaluation.Metrics{}
	for _, result := range results {
		groups[result.Method] = append(groups[result.Method], result.Metrics)
	}
	methods := make([]domain.AnalysisMethod, 0, len(groups))
	for method := range groups {
		methods = append(methods, method)
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i] < methods[j] })
	aggregates := make([]MethodAggregate, 0, len(methods))
	for _, method := range methods {
		aggregates = append(aggregates, MethodAggregate{Method: method, Scenarios: len(groups[method]), Metrics: meanMetrics(groups[method])})
	}
	return aggregates
}

func meanMetrics(values []evaluation.Metrics) evaluation.Metrics {
	var result evaluation.Metrics
	for _, value := range values {
		result.Recall.At1 += value.Recall.At1
		result.Recall.At3 += value.Recall.At3
		result.Recall.At5 += value.Recall.At5
		result.Precision.At1 += value.Precision.At1
		result.Precision.At3 += value.Precision.At3
		result.Precision.At5 += value.Precision.At5
		result.MRR += value.MRR
		result.MAP += value.MAP
		result.FilePrecision += value.FilePrecision
		result.FileRecall += value.FileRecall
		result.FileF1 += value.FileF1
		result.ModuleAccuracy += value.ModuleAccuracy
		result.LayerAccuracy += value.LayerAccuracy
		result.ActionAccuracy += value.ActionAccuracy
		result.UnresolvedRate += value.UnresolvedRate
		result.FalsePositiveRate += value.FalsePositiveRate
		result.EvaluatedChanges += value.EvaluatedChanges
		result.ExpectedImpacts += value.ExpectedImpacts
		result.PredictedUniqueFiles += value.PredictedUniqueFiles
	}
	count := float64(len(values))
	if count == 0 {
		return result
	}
	result.Recall.At1 = round(result.Recall.At1 / count)
	result.Recall.At3 = round(result.Recall.At3 / count)
	result.Recall.At5 = round(result.Recall.At5 / count)
	result.Precision.At1 = round(result.Precision.At1 / count)
	result.Precision.At3 = round(result.Precision.At3 / count)
	result.Precision.At5 = round(result.Precision.At5 / count)
	result.MRR = round(result.MRR / count)
	result.MAP = round(result.MAP / count)
	result.FilePrecision = round(result.FilePrecision / count)
	result.FileRecall = round(result.FileRecall / count)
	result.FileF1 = round(result.FileF1 / count)
	result.ModuleAccuracy = round(result.ModuleAccuracy / count)
	result.LayerAccuracy = round(result.LayerAccuracy / count)
	result.ActionAccuracy = round(result.ActionAccuracy / count)
	result.UnresolvedRate = round(result.UnresolvedRate / count)
	result.FalsePositiveRate = round(result.FalsePositiveRate / count)
	return result
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func validHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func sha256Hex(value []byte) string { hash := sha256.Sum256(value); return hex.EncodeToString(hash[:]) }
func round(value float64) float64   { return float64(int(value*10000+0.5)) / 10000 }
