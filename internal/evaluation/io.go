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
	output.WriteString("Ranking metrics depend on the reference, which may be provisional. R-precision checks the top R files, where R is the reference file count. AP rewards all relevant files occurring early; MAP averages AP. Empty-reference cases have undefined positive ranking metrics and are evaluated as abstentions. Fixed-k precision divides by k even when fewer candidates exist.\n\n")
	if p := metrics.PositiveRanking; p != nil {
		fmt.Fprintf(&output, "## Positive-reference changes (%d)\n\n| Metric | Value |\n|---|---:|\n| MAP | %.4f |\n| MRR | %.4f |\n| R-precision | %.4f |\n| Precision@5 / @10 | %.4f / %.4f |\n| Recall@5 / @10 | %.4f / %.4f |\n\n", p.Count, p.MAP, p.MRR, p.RPrecision, p.Precision5, p.Precision10, p.Recall5, p.Recall10)
	} else {
		output.WriteString("Positive-reference ranking: n/a (no positive reference cases).\n\n")
	}
	output.WriteString("## All labeled atomic changes (legacy aggregate)\n\n")
	output.WriteString("| Metric | Value |\n|---|---:|\n")
	fmt.Fprintf(&output, "| Recall@1 / @3 / @5 | %.4f / %.4f / %.4f |\n", metrics.Recall.At1, metrics.Recall.At3, metrics.Recall.At5)
	fmt.Fprintf(&output, "| Precision@1 / @3 / @5 | %.4f / %.4f / %.4f |\n", metrics.Precision.At1, metrics.Precision.At3, metrics.Precision.At5)
	fmt.Fprintf(&output, "| MRR | %.4f |\n| MAP | %.4f |\n", metrics.MRR, metrics.MAP)
	fmt.Fprintf(&output, "| File precision / recall / F1 | %.4f / %.4f / %.4f |\n", metrics.FilePrecision, metrics.FileRecall, metrics.FileF1)
	fmt.Fprintf(&output, "| Module / layer / action accuracy | %.4f / %.4f / %.4f |\n", metrics.ModuleAccuracy, metrics.LayerAccuracy, metrics.ActionAccuracy)
	fmt.Fprintf(&output, "| Required-action macro-F1 | %.4f |\n", metrics.ActionMacroF1)
	fmt.Fprintf(&output, "| Unresolved/abstention rate | %.4f |\n| False discovery proportion (FP/(TP+FP)) | %.4f |\n", metrics.UnresolvedRate, metrics.FalsePositiveRate)
	if metrics.NegativeControls > 0 {
		fmt.Fprintf(&output, "| Correct negative controls | %d / %d (%.4f) |\n", metrics.CorrectAbstentions, metrics.NegativeControls, metrics.NegativeControlAccuracy)
	} else {
		output.WriteString("| Correct negative controls | n/a (0 labeled negative cases) |\n")
	}
	output.WriteByte('\n')
	output.WriteString("## Required-action confusion matrix\n\nRows are expected actions; columns are predicted actions. A relevant file that was not retrieved is predicted as `UNRESOLVED`. Macro-F1 includes only classes with ground-truth support.\n\n| Expected \\ Predicted |")
	for _, label := range metrics.ActionConfusion.Labels {
		fmt.Fprintf(&output, " %s |", label)
	}
	output.WriteString("\n|---|")
	for range metrics.ActionConfusion.Labels {
		output.WriteString("---:|")
	}
	output.WriteByte('\n')
	for row, label := range metrics.ActionConfusion.Labels {
		fmt.Fprintf(&output, "| %s |", label)
		for _, count := range metrics.ActionConfusion.Rows[row] {
			fmt.Fprintf(&output, " %d |", count)
		}
		output.WriteByte('\n')
	}
	output.WriteByte('\n')
	output.WriteString("## Per change\n\n| Change | Element | Expected | Predicted | RR | AP | Unresolved |\n|---|---|---|---|---:|---:|---|\n")
	for _, change := range value.Changes {
		fmt.Fprintf(&output, "| %s | `%s` | %d | %d | %.4f | %.4f | %t |\n", change.ChangeType, change.ElementID, len(change.ExpectedFiles), len(change.PredictedFiles), change.ReciprocalRank, change.AveragePrecision, change.Unresolved)
	}
	return output.Bytes()
}
