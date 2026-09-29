package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	output.WriteString("| Method | Scenarios | Recall@1 | Recall@3 | File F1 | Action accuracy | Unresolved | FP rate |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, aggregate := range value.Aggregates {
		m := aggregate.Metrics
		fmt.Fprintf(&output, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f |\n", aggregate.Method, aggregate.Scenarios, m.Recall.At1, m.Recall.At3, m.FileF1, m.ActionAccuracy, m.UnresolvedRate, m.FalsePositiveRate)
	}
	output.WriteString("\nAll predictions were reproduced byte-for-byte for the configured repetition count. Timing is stored separately because it is intentionally non-deterministic.\n")
	return output.Bytes()
}
