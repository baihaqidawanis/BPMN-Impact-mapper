package planning

import (
	"context"
	"errors"
)

var errBatchReviewBudget = errors.New("partitioned review exhausted remaining agent turns")

// Optional adapter policy for slower reasoning models. File reviews still
// include all process requirements and are followed by a cross-file check.
type FileReviewPolicy interface {
	PreferFileReview() bool
}

func prefersFileReview(model AgentModel, draft []FilePlan) bool {
	policy, ok := model.(FileReviewPolicy)
	if !ok || !policy.PreferFileReview() {
		return false
	}
	files := map[string]bool{}
	for _, plan := range draft {
		files[plan.File] = true
	}
	return len(files) > 1
}

// Additional review calls consume and persist the same session budget. Cached
// completed file reviews do not consume another turn when a session is resumed.
type batchReviewModel struct {
	engine  AgentEngine
	session *AgentSession
}

func (m batchReviewModel) Identity() ProviderIdentity { return m.engine.Model.Identity() }
func (m batchReviewModel) PreferFileReview() bool {
	policy, ok := m.engine.Model.(FileReviewPolicy)
	return ok && policy.PreferFileReview()
}
func (m batchReviewModel) Turn(ctx context.Context, turn AgentTurn) (AgentReply, error) {
	if m.session.Turns >= m.session.Budget.MaxTurns {
		return AgentReply{}, errBatchReviewBudget
	}
	m.session.Turns++
	kind := "PARTITIONED_REVIEW_STARTED"
	if turn.Role == Planner {
		kind = "PARTITIONED_PLANNING_STARTED"
	}
	saved, err := m.engine.save(ctx, *m.session, kind)
	if err != nil {
		return AgentReply{}, err
	}
	*m.session = saved
	return m.engine.Model.Turn(ctx, turn)
}
func (engine AgentEngine) reviewOversizedBatch(ctx context.Context, s *AgentSession) (AgentReply, error) {
	// Bind receipts to exact draft/feedback, independently of the public session ID.
	id := agentDigest([]any{s.ID, s.Bundle.ContextID, s.Draft, s.Feedback})[:32]
	review, _, err := (FullRepositoryEngine{Model: batchReviewModel{engine, s}, Store: engine.Store}).reviewCombinedPlan(ctx, id, *s)
	if err != nil {
		return AgentReply{}, err
	}
	return AgentReply{Review: &review}, nil
}
