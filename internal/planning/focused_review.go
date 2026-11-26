package planning

import (
	"context"
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
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

// polishDeterministicPlanIssues runs at most one lightweight single-proposal
// Planner pass per draft item that still has deterministic schema/table/routing
// issues after all batches complete. Any error, invalid output, action flip, or
// non-decreasing issue count safely falls back to the existing draft item.
func (e FocusedEngine) polishDeterministicPlanIssues(ctx context.Context, id string, s *AgentSession) {
	if len(s.Draft) == 0 {
		return
	}
	var headContracts []ProposalContract
	if catalog, ok := e.Tools.(MigrationHeadCatalog); ok {
		for _, headFile := range catalog.MigrationHeads() {
			headContracts = append(headContracts, ProposalContract{
				File:           headFile,
				Action:         domain.ActionNoAction,
				TargetBehavior: "Highest existing numbered migration in this directory; any new forward migration must use a higher number.",
			})
		}
	}
	var schemaFacts []SchemaFact
	if sc, ok := e.Tools.(SchemaFactCatalog); ok {
		schemaFacts = sc.SchemaFacts()
	}
	changeMap := make(map[string]ChangeInput, len(s.Bundle.Changes))
	for _, ch := range s.Bundle.Changes {
		changeMap[planKey(ch.ChangeType, ch.ElementID, "", "")] = ch
	}
	evalIssues := func(p FilePlan) []string {
		rel := append([]ProposalContract(nil), headContracts...)
		rel = append(rel, schemaFactContractsForPlan(s.Bundle.ProcessContext, p, schemaFacts)...)
		var out []string
		out = append(out, migrationPrerequisiteIssuesWithRelated(s.Bundle, []FilePlan{p}, rel)...)
		out = append(out, rollbackRowStrategyIssuesWithRelated(s.Bundle, []FilePlan{p}, rel)...)
		if err := validateRoutingPlanWithContext(s.Bundle.ProcessContext, changeMap[planKey(p.ChangeType, p.ElementID, "", "")], p); err != nil {
			out = append(out, err.Error())
		}
		return cleanStrings(out)
	}
	for idx, p := range s.Draft {
		if ctx.Err() != nil {
			return
		}
		if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "documentation" {
			continue
		}
		oldIssues := evalIssues(p)
		if len(oldIssues) == 0 {
			continue
		}
		ch, ok := changeMap[planKey(p.ChangeType, p.ElementID, "", "")]
		if !ok {
			continue
		}
		var matchedCand *Candidate
		for ci := range ch.Candidates {
			if ch.Candidates[ci].File == p.File && ch.Candidates[ci].Symbol == p.Symbol {
				c := ch.Candidates[ci]
				matchedCand = &c
				break
			}
		}
		var matchedDec *CandidateDecision
		for di := range s.Decisions {
			d := s.Decisions[di]
			if d.ChangeType == p.ChangeType && d.ElementID == p.ElementID && d.File == p.File && d.Symbol == p.Symbol && d.Verdict == VerdictRelevant {
				matchedDec = &d
				break
			}
		}
		if matchedCand == nil || matchedDec == nil {
			continue
		}
		singleChange := ch
		singleChange.Candidates = []Candidate{*matchedCand}
		input := s.Bundle
		input.Changes = []ChangeInput{singleChange}
		input.ContextID, _ = ComputeContextID(input)
		feedback := make([]string, 0, len(oldIssues)+1)
		feedback = append(feedback, fmt.Sprintf("POST-BATCH DETERMINISTIC POLISH: keep action=%s on %s and preserve all valid branches, forward migration numbers, and acceptance checks from your draft while fixing ONLY the specific schema/source issues below (also remove any inaccurate claim in current_behavior or design_rationale that pre-change code already queries a non-zero-only ledger table):", p.Action, p.File))
		feedback = append(feedback, oldIssues...)
		childID := "polish-" + agentDigest([]any{id, idx, input.ContextID, oldIssues})[:44]
		child, err := (AgentEngine{
			Model:            e.Model,
			Tools:            e.Tools,
			Store:            e.Store,
			InitialDecisions: []CandidateDecision{*matchedDec},
			InitialFeedback:  feedback,
			InitialDraft:     []FilePlan{p},
			RelatedProposals: headContracts,
			DeferReview:      true,
			Budget:           AgentBudget{MaxTurns: 3, MaxTools: 0, MaxRevisions: 0, MaxDiscoveries: 0},
		}).Run(ctx, childID, input)
		s.Turns += child.Turns
		if err != nil || !child.AwaitingCombinedReview || len(child.Draft) != 1 {
			continue
		}
		if _, err := child.ChangePlan(); err != nil {
			continue
		}
		polished := child.Draft[0]
		if polished.File != p.File || polished.Symbol != p.Symbol || polished.Action != p.Action {
			continue
		}
		newIssues := evalIssues(polished)
		if len(newIssues) < len(oldIssues) {
			s.Draft[idx] = polished
			s.Revisions++
		}
	}
}
