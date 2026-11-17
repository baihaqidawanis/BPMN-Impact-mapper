package llm

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"

// Planner cannot propose WEAK/rejected targets. Their source excerpts and long
// rationales need not compete with revision feedback and supported draft edits.
// Requirements, graphs and exact original target indices remain intact.
func plannerEvidenceBundle(bundle planning.EvidenceBundle, decisions []planning.CandidateDecision) planning.EvidenceBundle {
	out := bundle
	out.Changes = append([]planning.ChangeInput(nil), bundle.Changes...)
	for i, change := range out.Changes {
		out.Changes[i].Candidates = nil
		for _, c := range change.Candidates {
			for _, d := range decisions {
				if d.ChangeType == change.ChangeType && d.ElementID == change.ElementID && d.File == c.File && d.Symbol == c.Symbol && d.Verdict == planning.VerdictRelevant {
					out.Changes[i].Candidates = append(out.Changes[i].Candidates, c)
					break
				}
			}
		}
	}
	return out
}
