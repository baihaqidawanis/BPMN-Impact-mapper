package planning

import (
	"context"
	"errors"
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// A large Planner turn is split by immutable candidate target, never truncated.
// Every requirement/graph remains supplied. Completed parts are bound to the
// exact draft, feedback, decisions, model and source snapshot and can be reused.
// The ordinary combined Reviewer still checks cross-file consistency afterward.
func (e AgentEngine) planOversizedBatch(ctx context.Context, s *AgentSession) (AgentReply, error) {
	base := agentDigest([]any{"partitioned-planner-v2-related", s.ID, s.Bundle.ContextID, s.Draft, s.Decisions, s.Feedback, s.Identity, e.Tools.Identity()})[:32]
	var plans []FilePlan
	for ci, change := range s.Bundle.Changes {
		for pi, candidate := range change.Candidates {
			var selected []CandidateDecision
			for _, d := range s.Decisions {
				if d.ChangeType == change.ChangeType && d.ElementID == change.ElementID && d.File == candidate.File && d.Symbol == candidate.Symbol && d.Verdict == VerdictRelevant {
					selected = append(selected, d)
				}
			}
			if len(selected) == 0 {
				continue
			}
			part := s.Bundle
			part.Changes = append([]ChangeInput(nil), s.Bundle.Changes...)
			for i := range part.Changes {
				part.Changes[i].Candidates = nil
			}
			part.Changes[ci].Candidates = []Candidate{candidate}
			var err error
			part.ContextID, err = ComputeContextID(part)
			if err != nil {
				return AgentReply{}, err
			}
			var draft []FilePlan
			for _, p := range s.Draft {
				if p.ChangeType == change.ChangeType && p.ElementID == change.ElementID && p.File == candidate.File && p.Symbol == candidate.Symbol {
					draft = append(draft, p)
				}
			}
			id := fmt.Sprintf("%s-pp-%d-%d", base, ci, pi)
			related := BuildProposalContracts(append(append([]FilePlan{}, s.Draft...), plans...), draft)
			binding := agentDigest([]any{"planning-part-v2-related", base, part.ContextID, selected, draft, related})
			receipt, err := e.Store.Load(ctx, id)
			if errors.Is(err, ErrAgentSessionMissing) {
				receipt = AgentSession{Version: "1", ID: id, Binding: binding, Identity: s.Identity, Bundle: part, Role: Planner, Status: "RUNNING"}
			} else if err != nil {
				return AgentReply{}, err
			}
			if receipt.Binding != binding || receipt.Identity != s.Identity {
				return AgentReply{}, errors.New("partitioned planning binding changed")
			}
			if receipt.Status == "PLANNED" {
				if err := validatePlans(part, selected, "", receipt.Draft); err != nil {
					return AgentReply{}, err
				}
				plans = append(plans, receipt.Draft...)
				continue
			}
			if err := ctx.Err(); err != nil {
				return AgentReply{}, err
			}
			if s.Turns >= s.Budget.MaxTurns {
				return AgentReply{}, errBatchReviewBudget
			}
			receipt.Turns++
			if err := e.Store.Save(ctx, receipt); err != nil {
				return AgentReply{}, err
			}
			callCtx, cancel := context.WithTimeout(ctx, AgentTurnTimeout)
			reply, err := (batchReviewModel{e, s}).Turn(callCtx, AgentTurn{Role: Planner, Bundle: part, Decisions: selected, Draft: draft, RelatedProposals: related, Feedback: s.Feedback})
			cancel()
			if err != nil {
				return AgentReply{}, err
			}
			if reply.Plan == nil || reply.Tool != nil || reply.Review != nil || reply.Verification != nil || reply.NeedEvidence != nil {
				return AgentReply{}, fmt.Errorf("%w: partitioned Planner must return the selected file proposal", ErrAgentModelOutput)
			}
			if err := validatePlans(part, selected, "", reply.Plan.Plans); err != nil {
				return AgentReply{}, fmt.Errorf("%w: %v", ErrAgentModelOutput, err)
			}
			for _, p := range reply.Plan.Plans {
				if p.Action == domain.ActionNoAction {
					continue
				}
				if len(cleanStrings(p.ProposedChanges)) == 0 || len(cleanStrings(p.AcceptanceChecks)) == 0 {
					return AgentReply{}, ErrAgentPlanIncomplete
				}
			}
			receipt.Draft, receipt.Status = reply.Plan.Plans, "PLANNED"
			if err := e.Store.Save(ctx, receipt); err != nil {
				return AgentReply{}, err
			}
			plans = append(plans, reply.Plan.Plans...)
		}
	}
	if len(plans) == 0 {
		return AgentReply{}, ErrAgentContextLimit
	}
	return AgentReply{Plan: &LayerPlanningResponse{Plans: plans}}, nil
}
