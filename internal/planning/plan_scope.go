package planning

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var diagramTaskEdit = regexp.MustCompile(`(?i)\b(add|insert|create)\s+(a\s+)?(new\s+)?(user|service|manual|send|receive|script|business rule)\s+task\b`)
var diagramModelEdit = regexp.MustCompile(`(?i)\b(edit|modify|update)\s+(the\s+)?(bpmn|workflow xml)\b`)

// Match an instruction to author diagram nodes, not a description of what a
// developer should document or implement in service code.
var diagramNodeEdit = regexp.MustCompile(`(?i)^(add|insert|create)\s+(a\s+|an\s+|the\s+)?(new\s+)?(BPMN\s+(element|gateway|sequence flow)|gateway element)\b`)

// A diagram-edit recipe assigned to an implementation/doc file cannot implement
// a blueprint change. This guard only catches explicit, high-confidence cases;
// semantic source correctness still requires review.
func validatePlanScope(p FilePlan) error {
	ext := strings.ToLower(path.Ext(p.File))
	if ext == ".bpmn" || ext == ".xml" {
		return nil
	}
	steps := append(append([]string{}, p.Steps...), p.ProposedChanges...)
	authoring := false
	for _, step := range steps {
		if diagramNodeEdit.MatchString(strings.TrimSpace(step)) {
			authoring = true
			break
		}
	}
	text := strings.Join(steps, " ")
	lower := strings.ToLower(text)
	if authoring || diagramModelEdit.MatchString(text) || (diagramTaskEdit.MatchString(text) && (strings.Contains(lower, "bpmn") || strings.Contains(lower, "flow connections"))) {
		return fmt.Errorf("%w: diagram-edit steps are assigned to non-diagram file %q; plan its software or documentation changes instead of modifying the supplied BPMN", ErrAgentPlanIncomplete, p.File)
	}
	return nil
}

// Documentation may be useful, but it does not resolve uncertain implementation
// impacts of an added task/gateway. Flag the gap rather than invent code edits.
func implementationCoverageIssues(bundle EvidenceBundle, plans []FilePlan) []string {
	var issues []string
	for _, change := range bundle.Changes {
		if change.ChangeType != "ADD_TASK" && change.ChangeType != "ADD_GATEWAY" {
			continue
		}
		hasCode, hasDocs, hasImplementation := false, false, false
		for _, c := range change.Candidates {
			if c.Layer != "documentation" && !strings.HasSuffix(strings.ToLower(c.File), ".md") {
				hasCode = true
			}
		}
		for _, p := range plans {
			if p.ChangeType != change.ChangeType || p.ElementID != change.ElementID {
				continue
			}
			if p.Action == "NO_ACTION" {
				continue
			}
			if p.Layer == "documentation" || strings.HasSuffix(strings.ToLower(p.File), ".md") {
				hasDocs = true
			} else {
				hasImplementation = true
			}
		}
		if hasCode && hasDocs && !hasImplementation {
			issues = append(issues, fmt.Sprintf("%s/%s has only documentation proposals; implementation candidates remain unresolved. Documentation alone does not establish software coverage.", change.ChangeType, change.ElementID))
		}
	}
	return issues
}
