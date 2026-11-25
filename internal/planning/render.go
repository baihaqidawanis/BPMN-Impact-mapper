package planning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
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

// Markdown is the developer handoff. The JSON export retains the full review record.
func Markdown(plan ChangePlan) []byte {
	var out bytes.Buffer
	out.WriteString("# BPMN change plan\n\n")
	switch plan.Status {
	case StatusComplete:
		out.WriteString("The agent review is complete. The following changes are proposed for implementation.\n\n")
	case StatusDisabled:
		out.WriteString("Agent review was disabled for this analysis.\n\n")
	default:
		out.WriteString("This draft is awaiting review. Resolve the open questions before implementing it.\n\n")
	}
	if p := plan.FullReview; p != nil {
		fmt.Fprintf(&out, "Full repository review coverage. Source chunks reviewed: %d/%d - Files reviewed: %d/%d - Planning batches: %d/%d\n\n", p.ChunksReviewed, p.ChunksTotal, p.FilesReviewed, p.FilesTotal, p.BatchesReviewed, p.BatchesTotal)
	}
	type changeKey struct{ kind, id string }
	changeTitles := map[changeKey]string{}
	if len(plan.ChangeCoverage) > 0 {
		out.WriteString("## Process changes\n\n")
		for i, c := range plan.ChangeCoverage {
			title := readableName(string(c.ChangeType))
			label := singleLine(c.ElementName)
			if label == "" || label == c.ElementID {
				label = readableName(c.ElementID)
			}
			title += " (" + label + ")"
			changeTitles[changeKey{string(c.ChangeType), c.ElementID}] = title
			outcome := map[string]string{"UNRESOLVED": "The affected code still needs to be identified.", "PLANNED": "A file change is proposed.", "NO_CHANGE_PROPOSED": "No edit was proposed."}[c.Status]
			if outcome == "" {
				outcome = readableName(c.Status)
			}
			fmt.Fprintf(&out, "%d. **%s.** %s\n", i+1, title, outcome)
		}
		out.WriteByte('\n')
	}

	type target struct{ file, symbol, change, element string }
	contested := map[target]string{}
	for _, finding := range plan.ReviewFindings {
		for _, check := range finding.Checks {
			if check.Criterion == "scope" && check.Status == "REVISE" {
				contested[target{finding.File, finding.Symbol, finding.ChangeType, finding.ElementID}] = check.Finding
			}
		}
	}
	groups := map[string][]FilePlan{}
	var held []string
	for _, item := range plan.Plans {
		if item.Action == "NO_ACTION" {
			continue
		}
		reason := contested[target{item.File, item.Symbol, string(item.ChangeType), item.ElementID}]
		if validatePlanScope(item) != nil {
			reason = "The proposal edits the diagram. A change to this file still needs supporting evidence."
		}
		if reason != "" {
			label := changeTitles[changeKey{string(item.ChangeType), item.ElementID}]
			if label == "" {
				label = readableName(item.ElementID)
			}
			name := fmt.Sprintf("`%s`", item.File)
			if item.Symbol != "" {
				name += " (`" + singleLine(item.Symbol) + "`)"
			}
			held = append(held, fmt.Sprintf("%s. %s. %s", name, label, reason))
			continue
		}
		groups[item.File] = append(groups[item.File], item)
	}
	if len(groups) > 0 {
		out.WriteString("## Proposed implementation\n\n")
	} else if len(held) > 0 {
		out.WriteString("All proposed edits still have open questions about scope.\n\n")
	} else {
		out.WriteString("No source edits are proposed in this export.\n\n")
	}
	var files []string
	for file := range groups {
		files = append(files, file)
	}
	sort.Strings(files)
	for i, file := range files {
		fmt.Fprintf(&out, "### %d. `%s`\n\n", i+1, singleLine(file))
		for _, warning := range fileConflictWarnings(plan.ReviewIssues, file) {
			fmt.Fprintf(&out, "> **Review warning:** %s\n\n", singleLine(warning))
		}
		for _, item := range groups[file] {
			title := changeTitles[changeKey{string(item.ChangeType), item.ElementID}]
			if title == "" {
				title = readableName(item.ElementID)
			}
			if title != "" {
				fmt.Fprintf(&out, "**%s.** ", title)
			}
			if item.Symbol != "" {
				fmt.Fprintf(&out, "`%s`. ", singleLine(item.Symbol))
			}
			fmt.Fprintf(&out, "%s\n\n", singleLine(item.Summary))
			if item.CurrentBehavior != "" {
				fmt.Fprintf(&out, "Today: %s\n\n", singleLine(item.CurrentBehavior))
			}
			if item.TargetBehavior != "" && singleLine(item.TargetBehavior) != singleLine(item.Summary) {
				fmt.Fprintf(&out, "Required: %s\n\n", singleLine(item.TargetBehavior))
			}
			if item.DesignRationale != "" {
				fmt.Fprintf(&out, "**Why this edit**\n\n%s\n\n", singleLine(item.DesignRationale))
			}
			edits := item.ProposedChanges
			if len(edits) == 0 {
				edits = item.Steps
			}
			for index, edit := range edits {
				fmt.Fprintf(&out, "%d. %s\n", index+1, singleLine(edit))
			}
			if len(item.AcceptanceChecks) > 0 {
				out.WriteString("\n**Acceptance checks**\n\n")
				for _, check := range item.AcceptanceChecks {
					fmt.Fprintf(&out, "- %s\n", singleLine(check))
				}
			}
			for _, a := range item.Assumptions {
				fmt.Fprintf(&out, "- Confirm: %s\n", singleLine(a))
			}
			if len(item.EvidenceIDs) > 0 {
				fmt.Fprintf(&out, "\nEvidence: `%s`\n", strings.Join(cleanStrings(item.EvidenceIDs), "`, `"))
			}
			out.WriteByte('\n')
		}
	}
	if len(held) > 0 || len(plan.ReviewIssues) > 0 || len(plan.ReviewFindings) > 0 {
		out.WriteString("## Open review items\n\n")
		seen := map[string]bool{}
		writeIssue := func(issue string) {
			line := strings.NewReplacer("NO_ACTION", "leave the file unchanged", "NEED_EVIDENCE", "more evidence required", "NEEDS_REVIEW", "awaiting review").Replace(singleLine(issue))
			if line != "" && !seen[line] {
				fmt.Fprintf(&out, "- %s\n", line)
				seen[line] = true
			}
		}
		for _, concern := range held {
			writeIssue(concern)
		}
		// Ollama can also serialize the structured findings into ReviewIssues.
		// Drop only exact copies; keep standalone feedback from older providers.
		summaries := map[string]bool{}
		for _, f := range plan.ReviewFindings {
			var joined []string
			for _, c := range f.Checks {
				if c.Status == "PASS" {
					continue
				}
				writeIssue(fmt.Sprintf("`%s` (%s). %s", f.File, c.Criterion, c.Finding))
				joined = append(joined, c.Criterion+": "+c.Finding)
			}
			if len(joined) > 0 {
				summaries[singleLine(strings.Join(joined, "; "))] = true
			}
		}
		for _, issue := range plan.ReviewIssues {
			prefix, body, ok := strings.Cut(issue, ": ")
			if ok && strings.HasPrefix(prefix, "Plan ") {
				if _, err := strconv.Atoi(strings.TrimPrefix(prefix, "Plan ")); err == nil && summaries[singleLine(body)] {
					continue
				}
			}
			issue = strings.Replace(issue, "PARTIAL CHECKPOINT EXPORT:", "Partial export:", 1)
			for _, c := range plan.ChangeCoverage {
				issue = strings.ReplaceAll(issue, string(c.ChangeType)+"/"+c.ElementID, changeTitles[changeKey{string(c.ChangeType), c.ElementID}])
			}
			writeIssue(strings.ReplaceAll(issue, " ()", ""))
		}
		if len(seen) == 0 {
			out.WriteString("The model reviewer reported no open findings.\n")
		}
		out.WriteByte('\n')
	}
	if plan.PlanID != "" {
		fmt.Fprintf(&out, "Plan `%s`", plan.PlanID)
		if plan.AnalysisID != "" {
			fmt.Fprintf(&out, " from analysis `%s`", plan.AnalysisID)
		}
		out.WriteString(". ")
	}
	out.WriteString("The JSON export contains all assessments, source references and reviewer checks.\n")
	return out.Bytes()
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func fileConflictWarnings(issues []string, file string) []string {
	if file == "" || len(issues) == 0 {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, issue := range issues {
		clean := singleLine(issue)
		if !strings.Contains(clean, file) {
			continue
		}
		lower := strings.ToLower(clean)
		if strings.Contains(lower, "conflict") || strings.Contains(lower, "schema prerequisite missing") || strings.Contains(lower, "overlapping focused batches") {
			if !seen[clean] {
				seen[clean] = true
				out = append(out, clean)
			}
		}
	}
	return out
}

// Display system enums and legacy element IDs as words. Preserve real code paths
// and function names wherever a developer needs them to locate the edit.
func readableName(s string) string {
	var out strings.Builder
	var prev rune
	for _, r := range s {
		if r == '_' || r == '-' {
			out.WriteByte(' ')
		} else {
			if unicode.IsLower(prev) && unicode.IsUpper(r) {
				out.WriteByte(' ')
			}
			out.WriteRune(unicode.ToLower(r))
		}
		prev = r
	}
	words := []rune(singleLine(out.String()))
	if len(words) == 0 {
		return ""
	}
	words[0] = unicode.ToUpper(words[0])
	return string(words)
}
