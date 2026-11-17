package planning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func JSON(plan ChangePlan) ([]byte, error) {
	if plan.Plans == nil {
		plan.Plans = []FilePlan{}
	}
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
	if plan.Status != StatusComplete {
		output.WriteString("> Unapproved draft. Resolve review issues before implementation. Any recommendations below remain provisional; their presence does not mean the combined plan passed review.\n\n")
	}
	if p := plan.FullReview; p != nil {
		fmt.Fprintf(&output, "## Full repository review coverage\n\n- Eligible files reviewed: %d/%d\n- Source chunks reviewed: %d/%d\n- Planning batches reviewed: %d/%d\n- Coverage records successful chunk classifications, not guaranteed correctness. Excluded files are outside this count.\n\n", p.FilesReviewed, p.FilesTotal, p.ChunksReviewed, p.ChunksTotal, p.BatchesReviewed, p.BatchesTotal)
		fmt.Fprintf(&output, "- Source snapshot: `%s`\n- Final consistency review: %s\n\n", p.SourceFingerprint, p.ConsistencyReview)
	}
	if len(plan.ReviewIssues) > 0 {
		output.WriteString("## Review issues\n\n")
		for _, issue := range plan.ReviewIssues {
			fmt.Fprintf(&output, "- %s\n", singleLine(issue))
		}
		output.WriteByte('\n')

	}
	if len(plan.ReviewFindings) > 0 {
		output.WriteString("## Reviewer checks\n\nThese are model conclusions, not independent proof of correctness. Proposal-only checks do not revalidate source.\n\n")
		for _, f := range plan.ReviewFindings {
			fmt.Fprintf(&output, "### `%s` / `%s`\n\n- Basis: `%s`\n- Evidence: `%s`\n\n| Check | Result | Finding |\n|---|---|---|\n", singleLine(f.File), singleLine(f.ElementID), singleLine(f.Basis), strings.Join(f.EvidenceIDs, "`, `"))
			for _, c := range f.Checks {
				fmt.Fprintf(&output, "| %s | %s | %s |\n", tableText(c.Criterion), tableText(c.Status), tableText(c.Finding))
			}
			output.WriteByte('\n')
			if f.SourceExcerpt != "" {
				fmt.Fprintf(&output, "Compared source: %s\n\n", reviewCodeSpan(f.SourceExcerpt))
			}
			if f.ImplementationExcerpt != "" {
				fmt.Fprintf(&output, "Compared plan item: %s\n\n", reviewCodeSpan(f.ImplementationExcerpt))
			}
		}
	}
	if len(plan.ChangeCoverage) > 0 {
		output.WriteString("## BPMN change coverage\n\nA plan is a proposal. UNRESOLVED means no evidence-backed file plan was produced; it does not mean no impact.\n\n| Change | Element | File plans | Status |\n|---|---|---:|---|\n")
		for _, c := range plan.ChangeCoverage {
			fmt.Fprintf(&output, "| %s | %s | %d | %s |\n", tableText(string(c.ChangeType)), tableText(c.ElementID), c.Plans, c.Status)
		}
		output.WriteByte('\n')
	}
	if len(plan.Plans) == 0 {
		output.WriteString("No evidence-grounded file plan was produced.\n")
		return output.Bytes()
	}
	groups := map[string][]FilePlan{}
	for _, item := range plan.Plans {
		groups[item.File] = append(groups[item.File], item)
	}
	files := make([]string, 0, len(groups))
	for file := range groups {
		files = append(files, file)
	}
	sort.Strings(files)
	fmt.Fprintf(&output, "## File overview\n\n%d files reviewed / %d files with proposed edits / %d scoped assessments. NO_ACTION assessments are excluded from the proposed edit count. Sections preserve each BPMN change and its evidence; do not apply duplicate steps blindly.\n\n", len(files), len(ProposedFiles(plan.Plans)), len(plan.Plans))
	for index, file := range files {
		fmt.Fprintf(&output, "## %d. `%s`\n\n", index+1, singleLine(file))
		for n, item := range groups[file] {
			fmt.Fprintf(&output, "### Requirement %d: %s / %s\n\n", n+1, singleLine(string(item.ChangeType)), singleLine(item.ElementID))
			fmt.Fprintf(&output, "- Change: `%s` / `%s`\n- Layer: `%s`\n- Module: `%s`\n- Action: **%s**\n", item.ChangeType, item.ElementID, item.Layer, item.Module, item.Action)
			if item.Symbol != "" {
				fmt.Fprintf(&output, "- Symbol: `%s`\n", item.Symbol)
			}
			fmt.Fprintf(&output, "- Evidence: `%s`\n\n%s\n\n", strings.Join(item.EvidenceIDs, "`, `"), singleLine(item.Summary))
			if item.Action == "NO_ACTION" {
				output.WriteString("No source edits are proposed for this target. The existing-contract conclusion remains subject to the cited source and model review.\n\n")
			}
			for _, section := range []struct{ title, value string }{{"Current behavior", item.CurrentBehavior}, {"Required behavior", item.TargetBehavior}, {"Design rationale", item.DesignRationale}} {
				if section.value != "" {
					fmt.Fprintf(&output, "#### %s\n\n%s\n\n", section.title, singleLine(section.value))
				}
			}
			if len(item.Assumptions) > 0 {
				output.WriteString("#### Assumptions and open decisions\n\n")
				for _, v := range item.Assumptions {
					fmt.Fprintf(&output, "- %s\n", singleLine(v))
				}
				output.WriteByte('\n')
			}
			if len(item.Steps) > 0 {
				output.WriteString("#### Implementation steps\n\n")
			}
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
				fmt.Fprintf(&output, "#### %s\n\n", group.title)
				for _, line := range group.items {
					fmt.Fprintf(&output, "- %s\n", singleLine(line))
				}
				output.WriteByte('\n')
			}
			output.WriteByte('\n')
		}
	}

	return output.Bytes()
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func tableText(s string) string { return strings.ReplaceAll(singleLine(s), "|", "\\|") }

func reviewCodeSpan(s string) string {
	s = singleLine(s)
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	return fence + " " + s + " " + fence
}
