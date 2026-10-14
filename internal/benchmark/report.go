package benchmark

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

type TimingAggregate struct {
	Method       string
	Runs         int
	MinMillis    int64
	MedianMillis float64
	MeanMillis   float64
	MaxMillis    int64
}

func AggregateTiming(value Timing) []TimingAggregate {
	grouped := map[string][]int64{}
	for _, run := range value.Runs {
		grouped[string(run.Method)] = append(grouped[string(run.Method)], run.Milliseconds)
	}
	methods := make([]string, 0, len(grouped))
	for method := range grouped {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	result := make([]TimingAggregate, 0, len(methods))
	for _, method := range methods {
		values := append([]int64(nil), grouped[method]...)
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		var sum int64
		for _, value := range values {
			sum += value
		}
		median := float64(values[len(values)/2])
		if len(values)%2 == 0 {
			median = float64(values[len(values)/2-1]+values[len(values)/2]) / 2
		}
		result = append(result, TimingAggregate{Method: method, Runs: len(values), MinMillis: values[0], MedianMillis: median, MeanMillis: float64(sum) / float64(len(values)), MaxMillis: values[len(values)-1]})
	}
	return result
}

// TimingMarkdown intentionally combines deterministic quality metrics with
// separately labeled volatile latency so experiment reports remain easy to
// compare without including timing in the benchmark identity.
func TimingMarkdown(report Report, timing Timing) []byte {
	metrics := map[string]evaluation.Metrics{}
	for _, aggregate := range report.Aggregates {
		metrics[string(aggregate.Method)] = aggregate.Metrics
	}
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Benchmark - Metrics and Latency\n\n- Benchmark ID: `%s`\n- Latency: volatile wall-clock milliseconds; excluded from reproducibility hashes\n- Runtime protocol: shared-process sequential execution; the first embedding method includes cold model startup, so this table is not a fair standalone method-speed comparison\n\n", report.BenchmarkID)
	output.WriteString("| Method | Runs | Recall@5 | MRR | File F1 | Min ms | Median ms | Mean ms | Max ms |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range AggregateTiming(timing) {
		metric := metrics[aggregate.Method]
		fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %d | %.1f | %.1f | %d |\n", aggregate.Method, aggregate.Runs, metric.Recall.At5, metric.MRR, metric.FileF1, aggregate.MinMillis, aggregate.MedianMillis, aggregate.MeanMillis, aggregate.MaxMillis)
	}
	return output.Bytes()
}

func TimingCSV(report Report, timing Timing) ([]byte, error) {
	metrics := map[string]evaluation.Metrics{}
	for _, aggregate := range report.Aggregates {
		metrics[string(aggregate.Method)] = aggregate.Metrics
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{"benchmark_id", "method", "runs", "recall_at_5", "mrr", "file_f1", "min_ms", "median_ms", "mean_ms", "max_ms", "latency_semantics"}); err != nil {
		return nil, err
	}
	for _, aggregate := range AggregateTiming(timing) {
		metric := metrics[aggregate.Method]
		row := []string{report.BenchmarkID, aggregate.Method, strconv.Itoa(aggregate.Runs), format4(metric.Recall.At5), format4(metric.MRR), format4(metric.FileF1), strconv.FormatInt(aggregate.MinMillis, 10), format1(aggregate.MedianMillis), format1(aggregate.MeanMillis), strconv.FormatInt(aggregate.MaxMillis, 10), "volatile shared-process sequential wall-clock; first embedding method includes cold startup; excluded from reproducibility hash"}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func format1(value float64) string { return strconv.FormatFloat(value, 'f', 1, 64) }
func format4(value float64) string { return strconv.FormatFloat(value, 'f', 4, 64) }

// AblationMarkdown presents paired aggregate deltas against the complete Hybrid
// configuration. The deltas are descriptive and do not imply causality.
func AblationMarkdown(value Report) []byte {
	hybrid, found := aggregateFor(value, "hybrid")
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Benchmark - Method Ablation\n\n- Benchmark ID: `%s`\n- Reference: Graph-Aware Hybrid\n- Interpretation: paired aggregate differences on the same frozen scenarios; descriptive, not causal\n\n", value.BenchmarkID)
	if !found {
		output.WriteString("Hybrid aggregate is unavailable; deltas cannot be computed.\n")
		return output.Bytes()
	}
	output.WriteString("| Configuration | Recall@5 | Delta | MRR | Delta | File F1 | Delta | Action macro-F1 | Delta |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		fmt.Fprintf(&output, "| %s | %.4f | %+.4f | %.4f | %+.4f | %.4f | %+.4f | %.4f | %+.4f |\n", aggregate.Method,
			m.Recall.At5, m.Recall.At5-hybrid.Recall.At5, m.MRR, m.MRR-hybrid.MRR,
			m.FileF1, m.FileF1-hybrid.FileF1, m.ActionMacroF1, m.ActionMacroF1-hybrid.ActionMacroF1)
	}
	return output.Bytes()
}

func AblationCSV(value Report) ([]byte, error) {
	hybrid, found := aggregateFor(value, "hybrid")
	if !found {
		return nil, errors.New("hybrid aggregate is required for ablation output")
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{"benchmark_id", "method", "reference", "recall_at_5", "delta_recall_at_5", "mrr", "delta_mrr", "file_f1", "delta_file_f1", "action_macro_f1", "delta_action_macro_f1", "interpretation"}); err != nil {
		return nil, err
	}
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		row := []string{value.BenchmarkID, string(aggregate.Method), "hybrid", format4(m.Recall.At5), signed4(m.Recall.At5 - hybrid.Recall.At5), format4(m.MRR), signed4(m.MRR - hybrid.MRR), format4(m.FileF1), signed4(m.FileF1 - hybrid.FileF1), format4(m.ActionMacroF1), signed4(m.ActionMacroF1 - hybrid.ActionMacroF1), "paired aggregate difference on frozen scenarios; descriptive, not causal"}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), writer.Error()
}

func aggregateFor(value Report, method string) (evaluation.Metrics, bool) {
	for _, aggregate := range value.Aggregates {
		if string(aggregate.Method) == method {
			return aggregate.Metrics, true
		}
	}
	return evaluation.Metrics{}, false
}

func signed4(value float64) string { return fmt.Sprintf("%+.4f", value) }

func Markdown(value Report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Benchmark\n\n- Benchmark ID: `%s`\n- Scenario-method runs: %d\n\n", value.BenchmarkID, len(value.Results))
	output.WriteString("| Method | Scenarios | Recall@1 | Recall@3 | File F1 | Action accuracy | Action macro-F1 | Unresolved | False discovery proportion |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f |\n", aggregate.Method, aggregate.Scenarios, m.Recall.At1, m.Recall.At3, m.FileF1, m.ActionAccuracy, m.ActionMacroF1, m.UnresolvedRate, m.FalsePositiveRate)
	}
	output.WriteString("\nAll predictions were reproduced byte-for-byte for the configured repetition count. Timing is stored separately because it is intentionally non-deterministic.\n")
	return output.Bytes()
}

// CSV renders deterministic, spreadsheet-ready scenario and aggregate rows.
// Timing is excluded because it is intentionally stored as volatile data.
func CSV(value Report) ([]byte, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := []string{"benchmark_id", "row_type", "scenario_id", "method", "scenarios", "repetitions", "reproducible", "recall_at_1", "recall_at_3", "recall_at_5", "precision_at_1", "precision_at_3", "precision_at_5", "mrr", "map", "file_precision", "file_recall", "file_f1", "module_accuracy", "layer_accuracy", "action_accuracy", "action_macro_f1", "unresolved_rate", "false_discovery_proportion", "evaluated_changes", "expected_impacts", "predicted_unique_files"}
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
	return []string{benchmarkID, rowType, scenarioID, method, scenarios, repetitions, reproducible,
		format4(metrics.Recall.At1), format4(metrics.Recall.At3), format4(metrics.Recall.At5), format4(metrics.Precision.At1), format4(metrics.Precision.At3), format4(metrics.Precision.At5),
		format4(metrics.MRR), format4(metrics.MAP), format4(metrics.FilePrecision), format4(metrics.FileRecall), format4(metrics.FileF1), format4(metrics.ModuleAccuracy), format4(metrics.LayerAccuracy), format4(metrics.ActionAccuracy), format4(metrics.ActionMacroF1), format4(metrics.UnresolvedRate), format4(metrics.FalsePositiveRate),
		strconv.Itoa(metrics.EvaluatedChanges), strconv.Itoa(metrics.ExpectedImpacts), strconv.Itoa(metrics.PredictedUniqueFiles)}
}
