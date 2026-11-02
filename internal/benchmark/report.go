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
	writeTruthScopeNote(&output, report)
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
	if err := writer.Write([]string{"benchmark_id", "method", "runs", "recall_at_5", "mrr", "file_f1", "min_ms", "median_ms", "mean_ms", "max_ms", "latency_semantics", "label_provenance"}); err != nil {
		return nil, err
	}
	for _, aggregate := range AggregateTiming(timing) {
		metric := metrics[aggregate.Method]
		row := []string{report.BenchmarkID, aggregate.Method, strconv.Itoa(aggregate.Runs), format4(metric.Recall.At5), format4(metric.MRR), format4(metric.FileF1), strconv.FormatInt(aggregate.MinMillis, 10), format1(aggregate.MedianMillis), format1(aggregate.MeanMillis), strconv.FormatInt(aggregate.MaxMillis, 10), "volatile shared-process sequential wall-clock; first embedding method includes cold startup; excluded from reproducibility hash; quality aggregates use complete truth only", report.LabelProvenance}
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
	fmt.Fprintf(&output, "# BPMN Impact Benchmark - Method Ablation\n\n- Benchmark ID: `%s`\n- Reference: BPMN-Context-Aware Hybrid Retrieval\n- Interpretation: paired aggregate differences on the same frozen scenarios; descriptive, not causal\n\n", value.BenchmarkID)
	writeTruthScopeNote(&output, value)
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
	if err := writer.Write([]string{"benchmark_id", "method", "reference", "recall_at_5", "delta_recall_at_5", "mrr", "delta_mrr", "file_f1", "delta_file_f1", "action_macro_f1", "delta_action_macro_f1", "interpretation", "label_provenance"}); err != nil {
		return nil, err
	}
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		row := []string{value.BenchmarkID, string(aggregate.Method), "hybrid", format4(m.Recall.At5), signed4(m.Recall.At5 - hybrid.Recall.At5), format4(m.MRR), signed4(m.MRR - hybrid.MRR), format4(m.FileF1), signed4(m.FileF1 - hybrid.FileF1), format4(m.ActionMacroF1), signed4(m.ActionMacroF1 - hybrid.ActionMacroF1), "paired aggregate difference on complete-truth frozen scenarios; subsets excluded; descriptive, not causal", value.LabelProvenance}
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
	writeTruthScopeNote(&output, value)
	output.WriteString("| Method | Scenarios | Recall@1 | Recall@3 | File F1 | Action accuracy | Action macro-F1 | Unresolved | False discovery proportion | Correct negative controls |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		negativeControls := "n/a"
		if m.NegativeControls > 0 {
			negativeControls = fmt.Sprintf("%d/%d (%.4f)", m.CorrectAbstentions, m.NegativeControls, m.NegativeControlAccuracy)
		}
		fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %s |\n", aggregate.Method, aggregate.Scenarios, m.Recall.At1, m.Recall.At3, m.FileF1, m.ActionAccuracy, m.ActionMacroF1, m.UnresolvedRate, m.FalsePositiveRate, negativeControls)
	}
	output.WriteString("\n## Positive-reference ranking\n\nEmpty-reference cases are excluded here and retain their separate abstention metrics. Values are weighted by positive-change count across complete-reference scenarios. Older reports show n/a.\n\n| Method | Positive changes | MAP | MRR | R-precision | P@5 | P@10 | R@5 | R@10 |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, a := range value.Aggregates {
		if p := a.Metrics.PositiveRanking; p != nil {
			fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f |\n", a.Method, p.Count, p.MAP, p.MRR, p.RPrecision, p.Precision5, p.Precision10, p.Recall5, p.Recall10)
		} else {
			fmt.Fprintf(&output, "| %s | 0 | n/a | n/a | n/a | n/a | n/a | n/a | n/a |\n", a.Method)
		}
	}
	output.WriteString("\nTiming is stored separately because it is intentionally non-deterministic. Inspect per-result reproducibility flags before claiming identical repetitions.\n")
	return output.Bytes()
}

// CSV renders deterministic, spreadsheet-ready scenario and aggregate rows.
// Timing is excluded because it is intentionally stored as volatile data.
func CSV(value Report) ([]byte, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := []string{"benchmark_id", "row_type", "scenario_id", "method", "scenarios", "repetitions", "reproducible", "recall_at_1", "recall_at_3", "recall_at_5", "precision_at_1", "precision_at_3", "precision_at_5", "mrr", "map", "file_precision", "file_recall", "file_f1", "module_accuracy", "layer_accuracy", "action_accuracy", "action_macro_f1", "unresolved_rate", "false_discovery_proportion", "negative_controls", "correct_abstentions", "negative_control_accuracy", "evaluated_changes", "expected_impacts", "predicted_unique_files", "truth_scope", "label_provenance"}
	header = append(header, "positive_changes", "positive_map", "positive_mrr", "r_precision", "positive_precision_at_5", "positive_precision_at_10", "positive_recall_at_5", "positive_recall_at_10")
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, result := range value.Results {
		scope := "complete"
		if result.TruthScope == "subset" {
			scope = "subset"
		}
		if err := writer.Write(csvRow(value.BenchmarkID, "scenario", result.ScenarioID, string(result.Method), "", strconv.Itoa(result.Repetitions), strconv.FormatBool(result.Reproducible), result.Metrics, scope, value.LabelProvenance)); err != nil {
			return nil, err
		}
	}
	for _, aggregate := range value.Aggregates {
		if err := writer.Write(csvRow(value.BenchmarkID, "aggregate", "", string(aggregate.Method), strconv.Itoa(aggregate.Scenarios), "", "", aggregate.Metrics, "complete-only", value.LabelProvenance)); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func csvRow(benchmarkID, rowType, scenarioID, method, scenarios, repetitions, reproducible string, metrics evaluation.Metrics, truthScope, labelProvenance string) []string {
	negativeControlAccuracy := ""
	if metrics.NegativeControls > 0 {
		negativeControlAccuracy = format4(metrics.NegativeControlAccuracy)
	}
	row := []string{benchmarkID, rowType, scenarioID, method, scenarios, repetitions, reproducible,
		format4(metrics.Recall.At1), format4(metrics.Recall.At3), format4(metrics.Recall.At5), format4(metrics.Precision.At1), format4(metrics.Precision.At3), format4(metrics.Precision.At5),
		format4(metrics.MRR), format4(metrics.MAP), format4(metrics.FilePrecision), format4(metrics.FileRecall), format4(metrics.FileF1), format4(metrics.ModuleAccuracy), format4(metrics.LayerAccuracy), format4(metrics.ActionAccuracy), format4(metrics.ActionMacroF1), format4(metrics.UnresolvedRate), format4(metrics.FalsePositiveRate), strconv.Itoa(metrics.NegativeControls), strconv.Itoa(metrics.CorrectAbstentions), negativeControlAccuracy,
		strconv.Itoa(metrics.EvaluatedChanges), strconv.Itoa(metrics.ExpectedImpacts), strconv.Itoa(metrics.PredictedUniqueFiles), truthScope, labelProvenance}
	if p := metrics.PositiveRanking; p != nil {
		return append(row, strconv.Itoa(p.Count), format4(p.MAP), format4(p.MRR), format4(p.RPrecision), format4(p.Precision5), format4(p.Precision10), format4(p.Recall5), format4(p.Recall10))
	}
	return append(row, "0", "", "", "", "", "", "", "")
}

func writeTruthScopeNote(output *bytes.Buffer, value Report) {
	if value.LabelProvenance != "" {
		fmt.Fprintf(output, "- Label provenance: `%s` (manifest declaration; the runner cannot verify reviewer independence).\n\n", value.LabelProvenance)
	}
	if len(value.PartialScenarios) > 0 {
		fmt.Fprintf(output, "- Partial ground truth: %v. Their metrics are diagnostic only and excluded from all method aggregates below. Latency still includes their runs.\n\n", value.PartialScenarios)
	}
}
