package planning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func JSON(plan ChangePlan) ([]byte, error) {
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func BundleJSON(bundle EvidenceBundle) ([]byte, error) {
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func Markdown(plan ChangePlan) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# BPMN change plan\n\n- Plan ID: `%s`\n- Context ID: `%s`\n- Analysis ID: `%s`\n- Provider: `%s`\n- Model: `%s`\n- Prompt: `%s`\n- Status: **%s**\n\n", plan.PlanID, plan.ContextID, plan.AnalysisID, plan.Provider.Provider, plan.Provider.Model, plan.Provider.PromptVersion, plan.Status)
	if p := plan.FullReview; p != nil {
		fmt.Fprintf(&output, "## Full repository review coverage\n\n- Eligible files reviewed: %d/%d\n- Source chunks reviewed: %d/%d\n- Planning batches reviewed: %d/%d\n- Coverage records successful chunk classifications, not guaranteed correctness. Excluded files are outside this count.\n\n", p.FilesReviewed, p.FilesTotal, p.ChunksReviewed, p.ChunksTotal, p.BatchesReviewed, p.BatchesTotal)
		fmt.Fprintf(&output, "- Source snapshot: `%s`\n\n", p.SourceFingerprint)
	}
	if len(plan.ReviewIssues) > 0 {
		output.WriteString("## Review issues\n\n")
		for _, issue := range plan.ReviewIssues {
			fmt.Fprintf(&output, "- %s\n", singleLine(issue))
		}
		output.WriteByte('\n')

	}
	if len(plan.Plans) == 0 {
		output.WriteString("No evidence-grounded file plan was produced.\n")
		return output.Bytes()
	}
	for index, item := range plan.Plans {
		fmt.Fprintf(&output, "## %d. `%s`\n\n", index+1, item.File)
		fmt.Fprintf(&output, "- Change: `%s` / `%s`\n- Layer: `%s`\n- Module: `%s`\n- Action: **%s**\n", item.ChangeType, item.ElementID, item.Layer, item.Module, item.Action)
		if item.Symbol != "" {
			fmt.Fprintf(&output, "- Symbol: `%s`\n", item.Symbol)
		}
		fmt.Fprintf(&output, "- Evidence: `%s`\n\n%s\n\n", strings.Join(item.EvidenceIDs, "`, `"), singleLine(item.Summary))
		for stepIndex, step := range item.Steps {
			fmt.Fprintf(&output, "%d. %s\n", stepIndex+1, singleLine(step))
		}
		for _, group := range []struct {
			title string
			items []string
		}{{"Proposed changes", item.ProposedChanges}, {"Acceptance checks", item.AcceptanceChecks}} {
			if len(group.items) == 0 {
				continue
			}
			fmt.Fprintf(&output, "### %s\n\n", group.title)
			for _, line := range group.items {
				fmt.Fprintf(&output, "- %s\n", singleLine(line))
			}
			output.WriteByte('\n')
		}
		output.WriteByte('\n')
	}
	return output.Bytes()
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
