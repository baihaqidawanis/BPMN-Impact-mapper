package planning

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
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
	return validatePlanScopeWithContext(nil, Candidate{}, p)
}

func validatePlanScopeWithContext(pc *domain.ProcessContext, candidate Candidate, p FilePlan) error {
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
	if err := validateFrontendBackendMismatch(p); err != nil {
		return err
	}
	if err := validateServerLaneFrontendCalculation(pc, p); err != nil {
		return err
	}
	if err := validateCrossSubmoduleEntityMismatch(pc, candidate, p); err != nil {
		return err
	}
	return nil
}

var nonFrontendStepPrefix = regexp.MustCompile(`(?i)^(?:backend(?:\s+service)?|server(?:\s+handler)?|database|sql|migration|test|unit\s+test|integration\s+test)\s*:`)

func validateFrontendBackendMismatch(p FilePlan) error {
	if p.Action == "NO_ACTION" {
		return nil
	}
	ext := strings.ToLower(path.Ext(p.File))
	isFrontend := p.Layer == "frontend" || ext == ".tsx" || ext == ".jsx" || ext == ".vue" || ext == ".svelte"
	if !isFrontend || strings.Contains(strings.ToLower(p.File), ".test.") || strings.Contains(strings.ToLower(p.File), ".spec.") {
		return nil
	}
	items := cleanStrings(p.ProposedChanges)
	if len(items) == 0 {
		items = cleanStrings(p.Steps)
	}
	if len(items) == 0 {
		return nil
	}
	backendPrefixed := 0
	for _, item := range items {
		if nonFrontendStepPrefix.MatchString(strings.TrimSpace(item)) {
			backendPrefixed++
		}
	}
	if backendPrefixed == len(items) {
		return fmt.Errorf("%w: frontend target %q contains only backend/test steps; return NO_CHANGE with empty edits if this UI file needs no code change", ErrAgentPlanIncomplete, p.File)
	}
	return nil
}

var clientSnapshotDeltaCue = regexp.MustCompile(`(?i)\b(?:client-side\s+(?:delta|snapshot|calculation)|compar(?:e|es|ing)\s+\w+\s+against\s+(?:a\s+|the\s+)?(?:stored\s+|current\s+|database\s+)?snapshot|calculate\s+(?:the\s+)?(?:stock\s+|line\s+|item\s+)?deltas?\b)`)

func validateServerLaneFrontendCalculation(pc *domain.ProcessContext, p FilePlan) error {
	if p.Action == domain.ActionNoAction || pc == nil || pc.Before == nil || pc.After == nil {
		return nil
	}
	ext := strings.ToLower(path.Ext(p.File))
	isFrontend := p.Layer == "frontend" || ext == ".tsx" || ext == ".jsx" || ext == ".vue" || ext == ".svelte"
	if !isFrontend || strings.Contains(strings.ToLower(p.File), ".test.") || strings.Contains(strings.ToLower(p.File), ".spec.") {
		return nil
	}
	if hasChangedUserTask(pc) {
		return nil
	}
	for _, item := range append(append([]string{}, p.ProposedChanges...), p.Steps...) {
		if matchUnnegated(clientSnapshotDeltaCue, item) {
			return fmt.Errorf("%w: frontend target %q moves server-lane snapshot/delta computation into the client (offending text: %q); no userTask changed in the BPMN process — use NO_CHANGE unless a concrete client request payload field must change", ErrAgentPlanIncomplete, p.File, compactSnippet(item, 140))
		}
	}
	return nil
}

func hasChangedUserTask(pc *domain.ProcessContext) bool {
	for id, afterEl := range pc.After.Elements {
		if afterEl == nil || afterEl.Type != "userTask" {
			continue
		}
		beforeEl := pc.Before.Elements[id]
		if beforeEl == nil || beforeEl.Name != afterEl.Name || beforeEl.Documentation != afterEl.Documentation {
			return true
		}
	}
	for id, beforeEl := range pc.Before.Elements {
		if beforeEl != nil && beforeEl.Type == "userTask" && pc.After.Elements[id] == nil {
			return true
		}
	}
	return false
}

var internalSubmodulePath = regexp.MustCompile(`(?:^|/)internal/([a-z][a-z0-9_-]+)/`)
var editedGoFuncMention = regexp.MustCompile(`\b(?:Transition|Create|Update|Submit|Approve|Reject|Post|Cancel|Reverse)([A-Z][a-z]{3,})\b`)
var sharedInfraSubmodules = map[string]bool{
	"platform": true, "common": true, "shared": true, "pkg": true,
	"auth": true, "middleware": true, "config": true, "audit": true,
	"database": true, "db": true, "http": true, "transport": true, "events": true,
}

func validateCrossSubmoduleEntityMismatch(pc *domain.ProcessContext, candidate Candidate, p FilePlan) error {
	if p.Action == domain.ActionNoAction || pc == nil || pc.After == nil || p.Layer == "test" || p.Layer == "documentation" {
		return nil
	}
	norm := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
	m := internalSubmodulePath.FindStringSubmatch(norm)
	if len(m) < 2 || sharedInfraSubmodules[m[1]] {
		return nil
	}
	submod := m[1]
	for _, ev := range candidate.Evidence {
		if strings.HasPrefix(ev.Type, "CALLS") || strings.HasPrefix(ev.Type, "CALLED_BY") || ev.Type == "INTERFACE_DISPATCH" || ev.Type == "IMPLEMENTS_INTERFACE" || ev.Type == "TRACE_ID" {
			return nil
		}
	}
	processWords := collectProcessWordPrefixes(pc)
	if matchesProcessPrefix(submod, processWords) {
		return nil
	}
	joined := p.Symbol + " " + strings.Join(append(append([]string{}, p.ProposedChanges...), p.Steps...), " ")
	funcMatches := editedGoFuncMention.FindAllStringSubmatch(joined, -1)
	if len(funcMatches) == 0 {
		return nil
	}
	for _, fm := range funcMatches {
		entity := strings.ToLower(fm[1])
		if matchesProcessPrefix(entity, processWords) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q modifies %s in unrelated submodule %q with no process or call-graph connection to this BPMN workflow; return NO_CHANGE unless cited call evidence proves cross-module invocation", ErrAgentPlanIncomplete, p.File, funcMatches[0][0], submod)
}

// validateUncalledDomainMethodEdit blocks modifying a /domain/ method when
// find_callers has already been executed on that target and proved zero
// non-test production callers exist.
func validateUncalledDomainMethodEdit(observations []ToolObservation, p FilePlan) error {
	if p.Action != domain.ActionModify && p.Action != domain.ActionAdd {
		return nil
	}
	lowerFile := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
	if p.Layer != "domain" && !strings.Contains("/"+lowerFile, "/domain/") {
		return nil
	}
	for _, obs := range observations {
		if obs.Request.Name != "find_callers" || obs.Result.Error != "" || obs.Request.File != p.File {
			continue
		}
		if p.Symbol != "" && obs.Request.Symbol != "" && obs.Request.Symbol != p.Symbol {
			continue
		}
		hasProdCaller := false
		for _, c := range obs.Result.Candidates {
			lf := strings.ToLower(c.File)
			if !strings.HasSuffix(lf, "_test.go") && !strings.Contains("/"+lf, "/test/") && !strings.Contains("/"+lf, "/tests/") {
				hasProdCaller = true
				break
			}
		}
		if !hasProdCaller {
			sym := p.Symbol
			if sym == "" {
				sym = obs.Request.Symbol
			}
			return fmt.Errorf("%w: %q (%s) has no production callers according to find_callers; return NO_CHANGE rather than placing runtime workflow logic in an uncalled or test-only domain method", ErrAgentPlanIncomplete, p.File, sym)
		}
	}
	return nil
}

func collectProcessWordPrefixes(pc *domain.ProcessContext) map[string]bool {
	prefixes := map[string]bool{}
	if pc == nil {
		return prefixes
	}
	add := func(s string) {
		for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if len(w) >= 4 {
				prefixes[w[:4]] = true
				prefixes[w] = true
			}
		}
	}
	for _, g := range []*domain.Graph{pc.Before, pc.After} {
		if g == nil {
			continue
		}
		for id, name := range g.ProcessNames {
			add(id)
			add(name)
		}
		for _, el := range g.Elements {
			if el != nil {
				add(el.ID)
				add(el.Name)
				add(el.Documentation)
			}
		}
		for _, f := range g.Flows {
			if f != nil {
				add(f.Name)
				add(f.Condition)
			}
		}
	}
	return prefixes
}

func matchesProcessPrefix(word string, prefixes map[string]bool) bool {
	word = strings.ToLower(word)
	if prefixes[word] {
		return true
	}
	if len(word) >= 4 && prefixes[word[:4]] {
		return true
	}
	return false
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
