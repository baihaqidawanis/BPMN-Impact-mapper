package planning

import (
	"context"
	"fmt"
)

// Prefer one combined audit; partition only when the real context guard fails.
type combinedFirstModel struct{ AgentModel }

func (combinedFirstModel) PreferFileReview() bool { return false }

// Correct source-bound failing targets once. Generic issues and missing source
// remain visible; they do not restart Analyst or trigger another revision loop.
func (e FocusedEngine) reviseFocusedPlan(ctx context.Context, id string, s *AgentSession, review AgentReview) (bool, error) {
	if len(review.Findings) == 0 {
		return false, nil
	}
	input := s.Bundle
	input.Changes = nil
	var decisions []CandidateDecision
	const ultraLightCorrectionLimit = 2
	for _, change := range s.Bundle.Changes {
		next := change
		next.Candidates = nil
		for _, candidate := range change.Candidates {
			if e.UltraLight && len(decisions) >= ultraLightCorrectionLimit {
				break
			}
			failed := false
			needsEvidence := false
			for _, f := range review.Findings {
				if f.ChangeType != string(change.ChangeType) || f.ElementID != change.ElementID || f.File != candidate.File || f.Symbol != candidate.Symbol {
					continue
				}
				for _, check := range f.Checks {
					if check.Status == "NEED_EVIDENCE" {
						needsEvidence = true
					}
					if check.Status == "REVISE" {
						failed = true
					}
				}
			}
			if !failed || needsEvidence {
				continue
			}
			for _, d := range s.Decisions {
				if d.ChangeType == change.ChangeType && d.ElementID == change.ElementID && d.File == candidate.File && d.Symbol == candidate.Symbol && d.Verdict == VerdictRelevant {
					next.Candidates = append(next.Candidates, candidate)
					decisions = append(decisions, d)
					break
				}
			}
		}
		if len(next.Candidates) > 0 {
			input.Changes = append(input.Changes, next)
		}
	}
	if len(decisions) == 0 {
		return false, nil
	}
	input.ContextID, _ = ComputeContextID(input)
	feedback := []string{"FINAL TARGETED CORRECTION: revise only the supplied targets using the review findings. Other files are outside this correction. Return NO_CHANGE with empty edits when this file can stay as-is under the coordinated plan. For any remaining required edit, give concrete changes and observable checks."}
	for _, f := range review.Findings {
		for _, d := range decisions {
			if f.ChangeType == string(d.ChangeType) && f.ElementID == d.ElementID && f.File == d.File && f.Symbol == d.Symbol {
				for _, check := range f.Checks {
					if check.Status == "REVISE" {
						feedback = append(feedback, fmt.Sprintf("%s/%s %s %s: %s: %s", f.ChangeType, f.ElementID, f.File, f.Symbol, check.Criterion, check.Finding))
					}
				}
			}
		}
	}
	var prior []FilePlan
	for _, p := range s.Draft {
		for _, d := range decisions {
			if p.ChangeType == d.ChangeType && p.ElementID == d.ElementID && p.File == d.File && p.Symbol == d.Symbol {
				prior = append(prior, p)
				break
			}
		}
	}
	engine := AgentEngine{Model: e.Model, Tools: e.Tools, Store: e.Store, InitialDecisions: decisions, InitialFeedback: feedback, InitialDraft: prior, RelatedProposals: BuildProposalContracts(s.Draft, prior), DeferReview: true, Budget: AgentBudget{MaxTurns: 4, MaxTools: 0, MaxRevisions: 0, MaxDiscoveries: 0}}
	child, err := engine.Run(ctx, "fix-"+agentDigest([]any{id, input.ContextID, feedback})[:48], input)
	s.Turns += child.Turns
	if err != nil && child.Status != "NEEDS_REVIEW" {
		return false, err
	}
	if !child.AwaitingCombinedReview {
		return false, nil
	}
	if _, err := child.ChangePlan(); err != nil {
		return false, nil
	}
	for _, p := range child.Draft {
		for i, old := range s.Draft {
			if hasFocusedPlan([]FilePlan{old}, p) {
				s.Draft[i] = p
				break
			}
		}
	}
	s.Revisions++
	s.Role = Reviewer
	s.ProgressMessage = "Checking the targeted correction; no additional revision loop is allowed."
	if err := e.Store.Save(ctx, *s); err != nil {
		return false, err
	}
	return true, nil
}
