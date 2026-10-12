package evaluation

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

func Markdown(value Report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Evaluation\n\n")
	fmt.Fprintf(&output, "- Evaluation ID: `%s`\n", value.EvaluationID)
	fmt.Fprintf(&output, "- Analysis ID: `%s`\n", value.AnalysisID)
	fmt.Fprintf(&output, "- Method: `%s`\n", value.Method)
	fmt.Fprintf(&output, "- Prediction SHA-256: `%s`\n", value.PredictionSHA256)
	fmt.Fprintf(&output, "- Ground truth SHA-256: `%s`\n\n", value.GroundTruthSHA256)
	metrics := value.Metrics
	output.WriteString("| Metric | Value |\n|---|---:|\n")
	fmt.Fprintf(&output, "| Recall@1 / @3 / @5 | %.4f / %.4f / %.4f |\n", metrics.Recall.At1, metrics.Recall.At3, metrics.Recall.At5)
	fmt.Fprintf(&output, "| Precision@1 / @3 / @5 | %.4f / %.4f / %.4f |\n", metrics.Precision.At1, metrics.Precision.At3, metrics.Precision.At5)
	fmt.Fprintf(&output, "| MRR | %.4f |\n| MAP | %.4f |\n", metrics.MRR, metrics.MAP)
	fmt.Fprintf(&output, "| File precision / recall / F1 | %.4f / %.4f / %.4f |\n", metrics.FilePrecision, metrics.FileRecall, metrics.FileF1)
	fmt.Fprintf(&output, "| Module / layer / action accuracy | %.4f / %.4f / %.4f |\n", metrics.ModuleAccuracy, metrics.LayerAccuracy, metrics.ActionAccuracy)
	fmt.Fprintf(&output, "| Unresolved rate | %.4f |\n| False discovery proportion (FP/(TP+FP)) | %.4f |\n\n", metrics.UnresolvedRate, metrics.FalsePositiveRate)
	output.WriteString("## Per change\n\n| Change | Element | Expected | Predicted | RR | AP | Unresolved |\n|---|---|---|---|---:|---:|---|\n")
	for _, change := range value.Changes {
		fmt.Fprintf(&output, "| %s | `%s` | %d | %d | %.4f | %.4f | %t |\n", change.ChangeType, change.ElementID, len(change.ExpectedFiles), len(change.PredictedFiles), change.ReciprocalRank, change.AveragePrecision, change.Unresolved)
	}
	return output.Bytes()
}
