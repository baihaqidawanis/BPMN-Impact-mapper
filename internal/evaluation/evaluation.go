// Package evaluation compares an already-produced canonical report with hidden
// ground truth. It has no repository, BPMN parser, or retrieval dependency.
package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"path"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type GroundTruth struct {
	Version    string                `json:"version"`
	AnalysisID string                `json:"analysis_id,omitempty"`
	Method     domain.AnalysisMethod `json:"method,omitempty"`
	Cases      []Case                `json:"cases"`
}

type Case struct {
	ChangeType domain.ChangeType `json:"change_type"`
	ElementID  string            `json:"element_id"`
	Expected   []ExpectedImpact  `json:"expected"`
}

type ExpectedImpact struct {
	File   string                `json:"file"`
	Symbol string                `json:"symbol,omitempty"`
	Module string                `json:"module,omitempty"`
	Layer  string                `json:"layer,omitempty"`
	Action domain.RequiredAction `json:"required_action"`
}

type AtK struct {
	At1 float64 `json:"at_1"`
	At3 float64 `json:"at_3"`
	At5 float64 `json:"at_5"`
}

type Metrics struct {
	Recall               AtK     `json:"recall"`
	Precision            AtK     `json:"precision"`
	MRR                  float64 `json:"mrr"`
	MAP                  float64 `json:"map"`
	FilePrecision        float64 `json:"file_precision"`
	FileRecall           float64 `json:"file_recall"`
	FileF1               float64 `json:"file_f1"`
	ModuleAccuracy       float64 `json:"module_accuracy"`
	LayerAccuracy        float64 `json:"layer_accuracy"`
	ActionAccuracy       float64 `json:"required_action_accuracy"`
	UnresolvedRate       float64 `json:"unresolved_rate"`
	FalsePositiveRate    float64 `json:"false_positive_rate"`
	LatencyMilliseconds  int64   `json:"latency_ms"`
	EvaluatedChanges     int     `json:"evaluated_changes"`
	ExpectedImpacts      int     `json:"expected_impacts"`
	PredictedUniqueFiles int     `json:"predicted_unique_files"`
}

type ChangeResult struct {
	ChangeType       domain.ChangeType `json:"change_type"`
	ElementID        string            `json:"element_id"`
	ExpectedFiles    []string          `json:"expected_files"`
	PredictedFiles   []string          `json:"predicted_files"`
	ReciprocalRank   float64           `json:"reciprocal_rank"`
	AveragePrecision float64           `json:"average_precision"`
	Unresolved       bool              `json:"unresolved"`
}

type Report struct {
	EvaluationID      string                `json:"evaluation_id"`
	AnalysisID        string                `json:"analysis_id"`
	Method            domain.AnalysisMethod `json:"method"`
	PredictionSHA256  string                `json:"prediction_sha256"`
	GroundTruthSHA256 string                `json:"ground_truth_sha256"`
	Metrics           Metrics               `json:"metrics"`
	Changes           []ChangeResult        `json:"changes"`
}

func (truth GroundTruth) Validate() error {
	if truth.Version != "1" {
		return fmt.Errorf("unsupported ground-truth version %q", truth.Version)
	}
	seenCases := map[string]bool{}
	for index, item := range truth.Cases {
		if item.ChangeType == "" || item.ElementID == "" {
			return fmt.Errorf("ground-truth case %d requires change_type and element_id", index)
		}
		key := caseKey(item.ChangeType, item.ElementID)
		if seenCases[key] {
			return fmt.Errorf("duplicate ground-truth case %s", key)
		}
		seenCases[key] = true
		seenFiles := map[string]bool{}
		for expectedIndex, expected := range item.Expected {
			switch expected.Action {
			case domain.ActionAdd, domain.ActionModify, domain.ActionRemove, domain.ActionVerify, domain.ActionNoAction, domain.ActionUnresolved:
			default:
				return errors.New("unsupported expected action")
			}
			if expected.File == "" || expected.Action == "" {
				return fmt.Errorf("ground-truth case %d expected impact %d requires file and action", index, expectedIndex)
			}
			clean := path.Clean(expected.File)
			if clean == "." || strings.ContainsAny(expected.File, ":\x00\r\n") || clean != expected.File || path.IsAbs(expected.File) || strings.Contains(expected.File, "\\") || clean == ".." || strings.HasPrefix(clean, "../") {
				return fmt.Errorf("ground-truth case %d expected impact %d requires a clean relative slash path", index, expectedIndex)
			}
			if seenFiles[expected.File] {
				return fmt.Errorf("ground-truth case %d duplicates file %q", index, expected.File)
			}
			seenFiles[expected.File] = true
		}
	}
	return nil
}

func Evaluate(prediction domain.Report, predictionBytes []byte, truth GroundTruth, truthBytes []byte) (Report, error) {
	if err := truth.Validate(); err != nil {
		return Report{}, err
	}
	if truth.AnalysisID != "" && truth.AnalysisID != prediction.AnalysisID {
		return Report{}, errors.New("ground truth analysis_id does not match prediction")
	}
	if truth.Method != "" && truth.Method != prediction.Method {
		return Report{}, errors.New("ground truth method does not match prediction")
	}
	predictionByCase := map[string]domain.Change{}
	for _, change := range prediction.Changes {
		predictionByCase[caseKey(change.ChangeType, elementID(change))] = change
	}
	sortedCases := append([]Case(nil), truth.Cases...)
	sort.Slice(sortedCases, func(i, j int) bool {
		return caseKey(sortedCases[i].ChangeType, sortedCases[i].ElementID) < caseKey(sortedCases[j].ChangeType, sortedCases[j].ElementID)
	})
	metrics := Metrics{EvaluatedChanges: len(sortedCases)}
	results := make([]ChangeResult, 0, len(sortedCases))
	var recall1, recall3, recall5, precision1, precision3, precision5 float64
	truePositive, falsePositive, falseNegative := 0, 0, 0
	matchedPairs, moduleCorrect, layerCorrect, actionCorrect := 0, 0, 0, 0
	for _, truthCase := range sortedCases {
		change, exists := predictionByCase[caseKey(truthCase.ChangeType, truthCase.ElementID)]
		predictions := deduplicate(change.Impacts)
		expected := expectedMap(truthCase.Expected)
		metrics.ExpectedImpacts += len(expected)
		metrics.PredictedUniqueFiles += len(predictions)
		result := ChangeResult{ChangeType: truthCase.ChangeType, ElementID: truthCase.ElementID, ExpectedFiles: sortedKeys(expected), PredictedFiles: impactFiles(predictions), Unresolved: !exists || len(predictions) == 0 || len(change.UnresolvedImpact) > 0}
		if result.Unresolved {
			metrics.UnresolvedRate++
		}
		recall1 += recallAt(predictions, expected, 1)
		recall3 += recallAt(predictions, expected, 3)
		recall5 += recallAt(predictions, expected, 5)
		precision1 += precisionAt(predictions, expected, 1)
		precision3 += precisionAt(predictions, expected, 3)
		precision5 += precisionAt(predictions, expected, 5)
		result.ReciprocalRank = reciprocalRank(predictions, expected)
		result.AveragePrecision = averagePrecision(predictions, expected)
		metrics.MRR += result.ReciprocalRank
		metrics.MAP += result.AveragePrecision
		predictedSet := map[string]domain.Impact{}
		for _, impact := range predictions {
			predictedSet[impact.File] = impact
		}
		for file, impact := range predictedSet {
			expectedImpact, relevant := expected[file]
			if relevant {
				truePositive++
				matchedPairs++
				if expectedImpact.Module == "" || expectedImpact.Module == impact.Module {
					moduleCorrect++
				}
				if expectedImpact.Layer == "" || expectedImpact.Layer == impact.Layer {
					layerCorrect++
				}
				if expectedImpact.Action == impact.RequiredAction {
					actionCorrect++
				}
			} else {
				falsePositive++
			}
		}
		for file := range expected {
			if _, found := predictedSet[file]; !found {
				falseNegative++
			}
		}
		results = append(results, result)
	}
	count := float64(len(sortedCases))
	if count > 0 {
		metrics.Recall = AtK{At1: round(recall1 / count), At3: round(recall3 / count), At5: round(recall5 / count)}
		metrics.Precision = AtK{At1: round(precision1 / count), At3: round(precision3 / count), At5: round(precision5 / count)}
		metrics.MRR = round(metrics.MRR / count)
		metrics.MAP = round(metrics.MAP / count)
		metrics.UnresolvedRate = round(metrics.UnresolvedRate / count)
	}
	metrics.FilePrecision = ratio(truePositive, truePositive+falsePositive)
	metrics.FileRecall = ratio(truePositive, truePositive+falseNegative)
	if metrics.FilePrecision+metrics.FileRecall > 0 {
		metrics.FileF1 = round(2 * metrics.FilePrecision * metrics.FileRecall / (metrics.FilePrecision + metrics.FileRecall))
	}
	metrics.FalsePositiveRate = ratio(falsePositive, truePositive+falsePositive)
	metrics.ModuleAccuracy = ratio(moduleCorrect, matchedPairs)
	metrics.LayerAccuracy = ratio(layerCorrect, matchedPairs)
	metrics.ActionAccuracy = ratio(actionCorrect, matchedPairs)
	predictionHash, truthHash := sha256Hex(predictionBytes), sha256Hex(truthBytes)
	evaluationSeed := []byte(predictionHash + ":" + truthHash)
	return Report{EvaluationID: sha256Hex(evaluationSeed)[:32], AnalysisID: prediction.AnalysisID, Method: prediction.Method, PredictionSHA256: predictionHash, GroundTruthSHA256: truthHash, Metrics: metrics, Changes: results}, nil
}

func caseKey(changeType domain.ChangeType, elementID string) string {
	return string(changeType) + "|" + elementID
}

func elementID(change domain.Change) string {
	if change.NewElement != nil {
		return change.NewElement.ID
	}
	if change.OldElement != nil {
		return change.OldElement.ID
	}
	return ""
}

func deduplicate(impacts []domain.Impact) []domain.Impact {
	seen := map[string]bool{}
	result := make([]domain.Impact, 0, len(impacts))
	for _, impact := range impacts {
		if impact.File != "" && !seen[impact.File] {
			seen[impact.File] = true
			result = append(result, impact)
		}
	}
	return result
}

func expectedMap(values []ExpectedImpact) map[string]ExpectedImpact {
	result := map[string]ExpectedImpact{}
	for _, value := range values {
		result[value.File] = value
	}
	return result
}

func recallAt(predicted []domain.Impact, expected map[string]ExpectedImpact, k int) float64 {
	if len(expected) == 0 {
		return 0
	}
	return float64(hitsAt(predicted, expected, k)) / float64(len(expected))
}

func precisionAt(predicted []domain.Impact, expected map[string]ExpectedImpact, k int) float64 {
	return float64(hitsAt(predicted, expected, k)) / float64(k)
}

func hitsAt(predicted []domain.Impact, expected map[string]ExpectedImpact, k int) int {
	if k > len(predicted) {
		k = len(predicted)
	}
	hits := 0
	for _, impact := range predicted[:k] {
		if _, ok := expected[impact.File]; ok {
			hits++
		}
	}
	return hits
}

func reciprocalRank(predicted []domain.Impact, expected map[string]ExpectedImpact) float64 {
	for index, impact := range predicted {
		if _, ok := expected[impact.File]; ok {
			return round(1 / float64(index+1))
		}
	}
	return 0
}

func averagePrecision(predicted []domain.Impact, expected map[string]ExpectedImpact) float64 {
	if len(expected) == 0 {
		return 0
	}
	hits, total := 0, 0.0
	for index, impact := range predicted {
		if _, ok := expected[impact.File]; ok {
			hits++
			total += float64(hits) / float64(index+1)
		}
	}
	return round(total / float64(len(expected)))
}

func impactFiles(impacts []domain.Impact) []string {
	result := make([]string, 0, len(impacts))
	for _, impact := range impacts {
		result = append(result, impact.File)
	}
	return result
}

func sortedKeys(values map[string]ExpectedImpact) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return round(float64(numerator) / float64(denominator))
}

func round(value float64) float64 { return math.Round(value*10000) / 10000 }

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
