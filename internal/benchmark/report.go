package benchmark

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

func JSON(value Report) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func TimingJSON(value Timing) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Markdown(value Report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Benchmark\n\n- Benchmark ID: `%s`\n- Scenario-method runs: %d\n\n", value.BenchmarkID, len(value.Results))
	output.WriteString("| Method | Scenarios | Recall@1 | Recall@3 | File F1 | Action accuracy | Unresolved | False discovery proportion |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f |\n", aggregate.Method, aggregate.Scenarios, m.Recall.At1, m.Recall.At3, m.FileF1, m.ActionAccuracy, m.UnresolvedRate, m.FalsePositiveRate)
	}
	output.WriteString("\nAll predictions were reproduced byte-for-byte for the configured repetition count. Timing is stored separately because it is intentionally non-deterministic.\n")
	return output.Bytes()
}

// CSV renders deterministic, spreadsheet-ready scenario and aggregate rows.
// Timing is excluded because it is intentionally stored as volatile data.
func CSV(value Report) ([]byte, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := []string{"benchmark_id", "row_type", "scenario_id", "method", "scenarios", "repetitions", "reproducible", "recall_at_1", "recall_at_3", "recall_at_5", "precision_at_1", "precision_at_3", "precision_at_5", "mrr", "map", "file_precision", "file_recall", "file_f1", "module_accuracy", "layer_accuracy", "action_accuracy", "unresolved_rate", "false_discovery_proportion", "evaluated_changes", "expected_impacts", "predicted_unique_files"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, result := range value.Results {
		if err := writer.Write(csvRow(value.BenchmarkID, "scenario", result.ScenarioID, string(result.Method), "", strconv.Itoa(result.Repetitions), strconv.FormatBool(result.Reproducible), result.Metrics)); err != nil {
			return nil, err
		}
	}
	for _, aggregate := range value.Aggregates {
		if err := writer.Write(csvRow(value.BenchmarkID, "aggregate", "", string(aggregate.Method), strconv.Itoa(aggregate.Scenarios), "", "", aggregate.Metrics)); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func csvRow(benchmarkID, rowType, scenarioID, method, scenarios, repetitions, reproducible string, metrics evaluation.Metrics) []string {
	f := func(value float64) string { return strconv.FormatFloat(value, 'f', 4, 64) }
	return []string{benchmarkID, rowType, scenarioID, method, scenarios, repetitions, reproducible,
		f(metrics.Recall.At1), f(metrics.Recall.At3), f(metrics.Recall.At5), f(metrics.Precision.At1), f(metrics.Precision.At3), f(metrics.Precision.At5),
		f(metrics.MRR), f(metrics.MAP), f(metrics.FilePrecision), f(metrics.FileRecall), f(metrics.FileF1), f(metrics.ModuleAccuracy), f(metrics.LayerAccuracy), f(metrics.ActionAccuracy), f(metrics.UnresolvedRate), f(metrics.FalsePositiveRate),
		strconv.Itoa(metrics.EvaluatedChanges), strconv.Itoa(metrics.ExpectedImpacts), strconv.Itoa(metrics.PredictedUniqueFiles)}
}
