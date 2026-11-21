package planning

import (
	"path"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// Inspect the highest ranked seed for each role before spending spare reads on
// uncertain targets. An early verdict based on excerpts can miss a caller.
func weakSourceRequests(bundle EvidenceBundle, decisions []CandidateDecision, limit int) []ToolRequest {
	if len(bundle.Changes) != 1 || limit <= 0 {
		return nil
	}
	change := bundle.Changes[0]
	processName := ""
	if context := bundle.ProcessContext; context != nil {
		for _, graph := range []*domain.Graph{context.After, context.Before} {
			if graph != nil {
				if element := graph.Elements[change.ElementID]; element != nil {
					processName = graph.ProcessNames[element.ProcessID]
					if processName != "" {
						break
					}
				}
			}
		}
	}
	weak := map[string]bool{}
	for _, d := range decisions {
		if d.Verdict == VerdictWeak {
			weak[candidateKey(d.File, d.Symbol)] = true
		}
	}
	roles, files := map[string]bool{}, map[string]bool{}
	var requests []ToolRequest
	for _, allowRepeatedRole := range []bool{false, true} {
		for _, c := range change.Candidates {
			role := lightFileRole(c.File, c.Module, c.Layer)
			if role == "" || files[c.File] || (!allowRepeatedRole && roles[role]) || (allowRepeatedRole && !weak[candidateKey(c.File, c.Symbol)]) {
				continue
			}
			parts := []string{c.Symbol, change.ElementName, processName}
			for _, branch := range change.BranchChanges {
				parts = append(parts, branch.NewCondition, branch.NewLabel)
			}
			parts = append(parts, change.ElementID)
			query := ""
			for _, word := range strings.Fields(strings.Join(parts, " ")) {
				if len(query)+len(word)+1 <= 200 {
					query = strings.TrimSpace(query + " " + word)
				}
			}
			if query == "" {
				continue
			}
			requests = append(requests, ToolRequest{Name: "inspect_source", File: c.File, Query: query})
			roles[role], files[c.File] = true, true
			if len(requests) == limit {
				return requests
			}
		}
	}
	return requests
}

func hasInspectedWeakTest(decisions []CandidateDecision, observations []ToolObservation) bool {
	hasImplementation := false
	weakTests := map[string]bool{}
	for _, decision := range decisions {
		switch lightFileRole(decision.File, "", "") {
		case "test":
			if decision.Verdict == VerdictWeak {
				weakTests[decision.File] = true
			}
		case "frontend", "backend":
			hasImplementation = hasImplementation || decision.Verdict == VerdictRelevant
		}
	}
	if !hasImplementation {
		return false
	}
	for _, observation := range observations {
		if observation.Request.Name == "inspect_source" && weakTests[observation.Request.File] && observation.Result.Error == "" {
			for _, candidate := range observation.Result.Candidates {
				if candidate.File == observation.Request.File && len(candidate.Evidence) > 0 {
					return true
				}
			}
		}
	}
	return false
}

// Light starts with three unique files and the best candidate for any missing
// implementation or test role. Other symbols and files remain available in
// rank order if the first draft leaves a gap.
func (e FocusedEngine) seedCandidates(candidates []Candidate) (initial, reserve []Candidate) {
	if !e.Light && !e.UltraLight {
		return candidates, nil
	}
	files := map[string]bool{}
	roles := map[string]bool{}
	selected := make([]bool, len(candidates))
	for i, c := range candidates {
		role := lightFileRole(c.File, c.Module, c.Layer)
		if !files[c.File] && (len(files) < 3 || role != "" && !roles[role]) {
			selected[i] = true
			files[c.File] = true
			roles[role] = true
		}
	}
	for i, c := range candidates {
		if selected[i] {
			initial = append(initial, c)
		} else {
			reserve = append(reserve, c)
		}
	}
	return initial, reserve
}

func lightEditRoles(plans []FilePlan) (roles map[string]bool, proposed bool) {
	roles = map[string]bool{}
	for _, p := range plans {
		switch p.Action {
		case domain.ActionAdd, domain.ActionModify, domain.ActionRemove:
			proposed = true
			roles[lightFileRole(p.File, p.Module, p.Layer)] = true
		}
	}
	return roles, proposed
}

// Make one supplemental batch from uncovered roles, with one new file per
// role. For unclassified repositories allow one ranked fallback if nothing
// actionable has been proposed.
func ultraLightReserve(initial, reserve []Candidate, plans []FilePlan) []Candidate {
	const maxCandidates = 3
	roles, proposed := lightEditRoles(plans)
	seen := make(map[string]bool, len(initial))
	for _, candidate := range initial {
		seen[candidate.File] = true
	}
	var selected []Candidate
	var fallback *Candidate
	for _, candidate := range reserve {
		role := lightFileRole(candidate.File, candidate.Module, candidate.Layer)
		if seen[candidate.File] || roles[role] {
			continue
		}
		if role == "" {
			if !proposed && fallback == nil {
				fallback = &candidate
			}
			continue
		}
		selected = append(selected, candidate)
		roles[role], seen[candidate.File] = true, true
		if len(selected) == maxCandidates {
			break
		}
	}
	if len(selected) < maxCandidates && fallback != nil {
		selected = append(selected, *fallback)
	}
	return selected
}

func lightNeedsMoreSource(candidates []Candidate, plans []FilePlan) bool {
	roles, proposed := lightEditRoles(plans)
	if !proposed {
		return true
	}
	for _, c := range candidates {
		role := lightFileRole(c.File, c.Module, c.Layer)
		if role != "" && !roles[role] {
			return true
		}
	}
	return false
}

func lightFileRole(file, module, layer string) string {
	file = strings.ToLower(strings.ReplaceAll(file, "\\", "/"))
	name := path.Base(file)
	switch {
	case layer == "test" || strings.HasSuffix(name, "_test.go") ||
		strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") ||
		strings.Contains("/"+file, "/test/") || strings.Contains("/"+file, "/tests/"):
		return "test"
	case layer == "documentation" || path.Ext(file) == ".md" || path.Ext(file) == ".sql":
		return ""
	case module == "frontend" || layer == "frontend" || strings.HasPrefix(file, "frontend/"):
		return "frontend"
	case module == "backend" || strings.HasPrefix(file, "backend/") || path.Ext(file) == ".go":
		return "backend"
	default:
		return ""
	}
}
