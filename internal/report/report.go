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
	if strings.HasPrefix(value.MethodMetadata["left_ranking_version"], "context-bm25-") {
		output.WriteString("Hybrid combines BPMN-context BM25 and Semantic positions with RRF. IDs, function names, paths, and code patterns are optional evidence, not separate ranking weights. A new BPMN ID absent from old source causes no score penalty. Scores order candidates; they are not probabilities of a required change.\n\n")
	} else {
		output.WriteString("Retrieval scores order candidates; they are not probabilities of a required change. BM25, semantic similarity and rank agreement have different scales. TRACE_ID links the changed element; CONTEXT_TRACE_ID links a neighbouring element only.\n\n")
	}
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
	primaryIDs := map[string]bool{}
	for _, change := range value.Changes {
		if change.PartOf == "" {
			if change.NewElement != nil {
				primaryIDs[change.NewElement.ID] = true
			} else if change.OldElement != nil {
				primaryIDs[change.OldElement.ID] = true
			}
		}
	}
	for index, change := range value.Changes {
		if change.PartOf != "" && primaryIDs[change.PartOf] {
			continue
		}
		id, name := elementIdentity(change)
		fmt.Fprintf(&output, "## %d. %s — %s\n\n", index+1, escape(string(change.ChangeType)), escape(id))
		if name != "" {
			fmt.Fprintf(&output, "Element: %s\n\n", escape(name))
		}
		if change.Reason != "" {
			fmt.Fprintf(&output, "Reason: %s\n\n", escape(change.Reason))
		}
		if change.Context != nil {
			if change.Context.Pattern != "" || len(change.Context.BaselineAnchors) == 0 {
				fmt.Fprintf(&output, "Composite pattern: `%s`\n\n", escape(string(change.Context.Pattern)))
			}
			if len(change.Context.BaselineAnchors) > 0 {
				output.WriteString("Experimental baseline task anchors (query context, not proven code dependencies):\n\n")
				for _, a := range change.Context.BaselineAnchors {
					fmt.Fprintf(&output, "- %s: `%s` %s; %d hops via `%s`\n", escape(a.Direction), escape(a.ID), escape(a.Name), a.Hops, escape(strings.Join(a.FlowPath, " -> ")))
				}
				output.WriteByte('\n')
			}
			if values := contextElements(change.Context.Predecessors); values != "" {
				fmt.Fprintf(&output, "Predecessors: %s\n\n", values)
			}
			if values := contextElements(change.Context.Successors); values != "" {
				fmt.Fprintf(&output, "Successors: %s\n\n", values)
			}
			for _, branch := range change.Context.BranchChanges {
				old, next := branch.OldCondition, branch.NewCondition
				if old == "" {
					old = branch.OldLabel
				}
				if next == "" {
					next = branch.NewLabel
				}
				fmt.Fprintf(&output, "Branch `%s`: before `%s`; after `%s`.\n\n", escape(branch.FlowID), escape(old), escape(next))
			}
		}
		var supportingFlows []string
		for _, detail := range value.Changes {
			if detail.PartOf == id {
				flowID, _ := elementIdentity(detail)
				supportingFlows = append(supportingFlows, flowID)
			}
		}
		if len(supportingFlows) > 0 {
			fmt.Fprintf(&output, "Supporting sequence-flow edits (mapped with this change): %s.\n\n", escape(strings.Join(supportingFlows, ", ")))
		}
		if len(change.RelatedFiles) > 0 {
			output.WriteString("### Related source files (unranked)\n\nObserved source connections, not proof these files must change. They are separate from ranked impacts and retrieval metrics.\n\n")
			for _, f := range change.RelatedFiles {
				fmt.Fprintf(&output, "- VERIFY `%s`: possible caller of `%s` in `%s` (%d source edges).\n", escape(f.File), escape(f.SeedFunction), escape(f.SeedFile), f.Hops)
				for _, e := range f.Evidence {
					fmt.Fprintf(&output, "  - %s `%s`: %s\n", escape(e.Type), escape(e.Location), escape(e.Excerpt))
				}
			}
			output.WriteByte('\n')
		}
		if len(change.Impacts) > 0 {
			output.WriteString("| Action | Retrieval score | Module | Layer | File | Function / code name |\n")
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
				if impact.ScoreBreakdown.EvidenceScore != nil {
					fmt.Fprintf(&output, "Code-evidence score: %.4f. ", *impact.ScoreBreakdown.EvidenceScore)
				} else if strings.HasPrefix(value.MethodMetadata["left_ranking_version"], "context-bm25-") {
					fmt.Fprintf(&output, "BM25 word match: %.4f. ", impact.ScoreBreakdown.BM25)
				} else {
					output.WriteString("Code-evidence score: not scored. ")
				}
				if impact.ScoreBreakdown.Semantic > 0 {
					fmt.Fprintf(&output, "Meaning match: %.4f. ", impact.ScoreBreakdown.Semantic)
				} else {
					output.WriteString("Meaning match: not scored. ")
				}
				if strings.HasPrefix(value.MethodMetadata["left_ranking_version"], "context-bm25-") {
					fmt.Fprintf(&output, "Score: BM25=%.4f, semantic=%.4f, RRF=%.4f (ranking only).\n\n", impact.ScoreBreakdown.BM25, impact.ScoreBreakdown.Semantic, impact.ScoreBreakdown.RRF)
				} else if impact.ScoreBreakdown.RRF > 0 {
					fmt.Fprintf(&output, "Rank agreement: %.4f.\n\n", impact.ScoreBreakdown.RRF)
				} else {
					output.WriteString("Rank agreement: not combined.\n\n")
				}
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
