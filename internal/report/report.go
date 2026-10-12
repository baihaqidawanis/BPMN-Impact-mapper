package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func JSON(value domain.Report) ([]byte, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func Markdown(value domain.Report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN Impact Report\n\n")
	fmt.Fprintf(&output, "- Analysis ID: `%s`\n", escape(value.AnalysisID))
	fmt.Fprintf(&output, "- Input fingerprint: `%s`\n", escape(value.InputFingerprint))
	fmt.Fprintf(&output, "- Method: `%s`\n", escape(string(value.Method)))
	if len(value.MethodMetadata) > 0 {
		keys := make([]string, 0, len(value.MethodMetadata))
		for key := range value.MethodMetadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&output, "- Method %s: `%s`\n", escape(key), escape(value.MethodMetadata[key]))
		}
	}
	fmt.Fprintf(&output, "- Repository: `%s`\n", escape(value.Repository.Name))
	fmt.Fprintf(&output, "- Revision: `%s`\n", escape(value.Repository.Revision))
	fmt.Fprintf(&output, "- Changes: %d\n\n", len(value.Changes))
	if len(value.SourceAnalysis) > 0 {
		output.WriteString("## Source analysis coverage\n\n| File | Language | Adapter | Capabilities | Fallback |\n|---|---|---|---|---|\n")
		for _, file := range value.SourceAnalysis {
			caps := []string{}
			for _, c := range file.Capabilities {
				caps = append(caps, string(c))
			}
			fmt.Fprintf(&output, "| %s | %s | %s | %s | %t |\n", table(file.File), table(file.Language), table(file.Adapter), table(strings.Join(caps, ", ")), file.Fallback)
		}
		output.WriteByte('\n')
	}
	for index, change := range value.Changes {
		id, name := elementIdentity(change)
		fmt.Fprintf(&output, "## %d. %s — %s\n\n", index+1, escape(string(change.ChangeType)), escape(id))
		if name != "" {
			fmt.Fprintf(&output, "Element: %s\n\n", escape(name))
		}
		if change.Reason != "" {
			fmt.Fprintf(&output, "Reason: %s\n\n", escape(change.Reason))
		}
		if change.Context != nil {
			fmt.Fprintf(&output, "Composite pattern: `%s`\n\n", escape(string(change.Context.Pattern)))
			if values := contextElements(change.Context.Predecessors); values != "" {
				fmt.Fprintf(&output, "Predecessors: %s\n\n", values)
			}
			if values := contextElements(change.Context.Successors); values != "" {
				fmt.Fprintf(&output, "Successors: %s\n\n", values)
			}
		}
		if len(change.Impacts) > 0 {
			output.WriteString("| Action | Confidence | Module | Layer | File | Symbol |\n")
			output.WriteString("|---|---:|---|---|---|---|\n")
			for _, impact := range change.Impacts {
				fmt.Fprintf(&output, "| %s | %.4f | %s | %s | `%s` | `%s` |\n", impact.RequiredAction, impact.Confidence, table(impact.Module), table(impact.Layer), table(impact.File), table(impact.Symbol))
			}
			output.WriteByte('\n')
			for _, impact := range change.Impacts {
				fmt.Fprintf(&output, "### Evidence: `%s`", escape(impact.File))
				if impact.Symbol != "" {
					fmt.Fprintf(&output, " / `%s`", escape(impact.Symbol))
				}
				output.WriteString("\n\n")
				if impact.ScoreBreakdown.RRF > 0 {
					fmt.Fprintf(&output, "Score: trace=%.4f, symbol=%.4f, path=%.4f, lexical=%.4f, structured=%.4f, BM25=%.4f, semantic=%.4f, RRF=%.4f, total=%.4f.\n\n", impact.ScoreBreakdown.TraceID, impact.ScoreBreakdown.Symbol, impact.ScoreBreakdown.Path, impact.ScoreBreakdown.Lexical, impact.ScoreBreakdown.Structured, impact.ScoreBreakdown.BM25, impact.ScoreBreakdown.Semantic, impact.ScoreBreakdown.RRF, impact.ScoreBreakdown.Total)
				} else {
					fmt.Fprintf(&output, "Score: trace=%.4f, symbol=%.4f, path=%.4f, lexical=%.4f, structured=%.4f, BM25=%.4f, semantic=%.4f, total=%.4f.\n\n", impact.ScoreBreakdown.TraceID, impact.ScoreBreakdown.Symbol, impact.ScoreBreakdown.Path, impact.ScoreBreakdown.Lexical, impact.ScoreBreakdown.Structured, impact.ScoreBreakdown.BM25, impact.ScoreBreakdown.Semantic, impact.ScoreBreakdown.Total)
				}
				for _, evidence := range impact.Evidence {
					if evidence.Type == "PROPAGATION_SEED" {
						fmt.Fprintf(&output, "Dependency score: %.4f (one-hop, VERIFY).\n\n", impact.ScoreBreakdown.Dependency)
					}
					fmt.Fprintf(&output, "- **%s** `%s`: %s\n", escape(evidence.Type), escape(evidence.Location), escape(evidence.Excerpt))
				}
				output.WriteByte('\n')
			}
		} else {
			output.WriteString("No candidate met the evidence threshold.\n\n")
		}
		if len(change.UnresolvedImpact) > 0 {
			output.WriteString("Unresolved:\n\n")
			for _, unresolved := range change.UnresolvedImpact {
				fmt.Fprintf(&output, "- %s\n", escape(unresolved))
			}
			output.WriteByte('\n')
		}
	}
	if len(value.Warnings) > 0 {
		output.WriteString("## Warnings\n\n")
		for _, warning := range value.Warnings {
			fmt.Fprintf(&output, "- **%s**: %s\n", escape(warning.Code), escape(warning.Message))
		}
	}
	return output.Bytes()
}

func elementIdentity(change domain.Change) (string, string) {
	if change.NewElement != nil {
		return change.NewElement.ID, change.NewElement.Name
	}
	if change.OldElement != nil {
		return change.OldElement.ID, change.OldElement.Name
	}
	return "unknown", ""
}

func escape(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func table(value string) string { return strings.ReplaceAll(escape(value), "|", "\\|") }

func contextElements(elements []domain.ElementContext) string {
	values := make([]string, 0, len(elements))
	for _, element := range elements {
		label := element.ID
		if element.Name != "" {
			label += " (" + element.Name + ")"
		}
		values = append(values, "`"+escape(label)+"`")
	}
	return strings.Join(values, ", ")
}
