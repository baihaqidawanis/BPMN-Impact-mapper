package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const FocusedProtocolVersion = "focused-batches-v12-output-capacity"

// FocusedEngine reviews every change with bounded seed input per model session,
// then assembles one evidence-bound plan. It never scans the full source corpus.
type FocusedEngine struct {
	Model      AgentModel
	Tools      SourceTools
	Store      AgentSessionStore
	Light      bool
	UltraLight bool
}

// Group by payload size as well as count; never discard a seed. The model's
// actual context guard still applies to graphs, observations and output reserve.
func focusedSeedBatches(candidates []Candidate) [][]Candidate {
	batches := [][]Candidate{}
	for start := 0; start < len(candidates); {
		stop := start + 1
		for stop < len(candidates) && stop-start < 10 {
			data, _ := json.Marshal(candidates[start : stop+1])
			if len(data) > 12000 {
				break
			}
			stop++
		}
		batches = append(batches, candidates[start:stop])
		start = stop
	}
	if len(batches) == 0 {
		return [][]Candidate{nil}
	}
	return batches
}

func focusedChildID(id string, change int, seeds []Candidate) string {
	return fmt.Sprintf("%s-fc-%d-%s", id, change, agentDigest(seeds)[:8])
}

func capacityLimitedChild(child AgentSession, err error) bool {
	if errors.Is(err, ErrAgentContextLimit) || errors.Is(err, ErrAgentOutputLimit) || errors.Is(err, errBatchReviewBudget) {
		return true
	}
	if len(child.Events) == 0 {
		return false
	}
	switch child.Events[len(child.Events)-1].Kind {
	case "CONTEXT_LIMIT", "BATCH_BUDGET_EXHAUSTED", "BUDGET_EXHAUSTED":
		return true
	default:
		return false
	}
}

func (e FocusedEngine) Run(ctx context.Context, id string, original EvidenceBundle) (AgentSession, error) {
	var s AgentSession
	if e.Model == nil || e.Tools == nil || e.Store == nil || !validAgentID(id) || len(id) > 40 {
		return s, errors.New("focused dependencies required")
	}
	if err := validateBundle(original); err != nil {
		return s, err
	}
	protocol := FocusedProtocolVersion
	if e.Light {
		protocol = "focused-light-v2-source-roles"
	}
	budget := DefaultAgentBudget()
	if e.UltraLight {
		protocol = "focused-ultra-light-v7-schema-evidence"
		budget.MaxTools = 4
		budget.MaxDiscoveries = 2
		budget.MaxRevisions = 0
	}
	binding := agentDigest([]any{protocol, original.ContextID, e.Tools.Identity(), e.Model.Identity()})
	saved, err := e.Store.Load(ctx, id)
	if err == nil {
		if saved.Binding != binding {
			return s, errors.New("focused source/model/context changed; start a new session")
		}
		if saved.Status == "COMPLETE" || saved.Status == "NEEDS_REVIEW" {
			return saved, nil
		}
	} else if !errors.Is(err, ErrAgentSessionMissing) {
		return s, err
	}
	bundle := original
	bundle.Changes = append([]ChangeInput(nil), original.Changes...)
	s = AgentSession{Version: "1", ID: id, Binding: binding, Identity: e.Model.Identity(), Bundle: bundle, Role: Analyst, Status: "RUNNING"}
	pause := func(message string, cause error, details ...string) (AgentSession, error) {
		s.Status = "PAUSED"
		s.Feedback = append([]string{message}, details...)
		checkpoint, cancel := context.WithCancel(context.WithoutCancel(ctx))
		defer cancel()
		if err := e.Store.Save(checkpoint, s); err != nil {
			return s, err
		}
		return s, cause
	}
	var issues []string
	overlappingFiles := make(map[string]bool)
	s.FocusedReview = &FocusedReviewProgress{Light: e.Light, UltraLight: e.UltraLight, OutcomeCountsAvailable: true, CombinedReviewOnly: true}
	for _, c := range original.Changes {
		s.FocusedReview.CandidatesTotal += len(c.Candidates)
		seeds, _ := e.seedCandidates(c.Candidates)
		s.FocusedReview.BatchesTotal += len(focusedSeedBatches(seeds))
	}
	batchIndex := 0
	for i, change := range original.Changes {
		initial, reserve := e.seedCandidates(change.Candidates)
		batches := focusedSeedBatches(initial)
		draftStart := len(s.Draft)
		ultraSupplement := false
		for position := 0; position < len(batches); position++ {
			seeds := batches[position]
			if err := ctx.Err(); err != nil {
				return pause("Focused review interrupted; completed candidate batches are preserved.", err)
			}
			s.Role = Analyst
			s.ProgressMessage = fmt.Sprintf("Focused batch %d/%d, BPMN change %d/%d: inspect source, plan, review.", batchIndex+1, s.FocusedReview.BatchesTotal, i+1, len(original.Changes))
			s.ActiveChildID = focusedChildID(id, i, seeds)
			if err := e.Store.Save(ctx, s); err != nil {
				return s, err
			}
			input := original
			input.Changes = []ChangeInput{change}
			input.Changes[0].Candidates = append([]Candidate(nil), seeds...)
			input.ContextID, _ = ComputeContextID(input)
			// Earlier proposals for these files are coordination context only.
			// Reconstructing them from completed children keeps resume stable.
			// Also carry forward historical-migration constraint notes so storage
			// batches know about blocking schema constraints discovered earlier.
			var companion []FilePlan
			hasImplSeed := false
			for _, seed := range seeds {
				if !IsHistoricalMigration(seed.File) && seed.Layer != "documentation" {
					hasImplSeed = true
					break
				}
			}
			for draftIndex, p := range s.Draft {
				if (ultraSupplement || len(batches) > 1) && draftIndex >= draftStart {
					companion = append(companion, p)
					continue
				}
				if hasImplSeed && IsHistoricalMigration(p.File) {
					companion = append(companion, p)
					continue
				}
				for _, seed := range seeds {
					if p.File == seed.File {
						companion = append(companion, p)
						break
					}
				}
			}
			child, err := (AgentEngine{Model: e.Model, Tools: e.Tools, Store: e.Store, Budget: budget, DeferReview: true, InspectWeak: e.UltraLight && len(seeds) > 0, RelatedProposals: BuildProposalContracts(companion, nil)}).Run(ctx, s.ActiveChildID, input)
			s.Turns += child.Turns
			s.Tools += child.Tools
			s.Revisions += child.Revisions
			for _, o := range child.Observations {
				o.Request.ChangeIndex = i
				s.Observations = append(s.Observations, o)
			}
			if capacityLimitedChild(child, err) && len(seeds) > 1 {
				middle := len(seeds) / 2
				next := [][]Candidate{seeds[:middle], seeds[middle:]}
				batches = append(append(batches[:position:position], next...), batches[position+1:]...)
				s.FocusedReview.BatchesTotal++
				position--
				continue
			}
			if err != nil && child.Status != "NEEDS_REVIEW" {
				s.Role = child.Role
				return pause(fmt.Sprintf("Focused candidate batch %d interrupted; resume to continue.", batchIndex+1), err, child.Feedback...)
			}
			if len(child.Bundle.Changes) == 1 {
				for _, candidate := range child.Bundle.Changes[0].Candidates {
					mergeFocusedCandidate(&s.Bundle.Changes[i], candidate)
				}
			}
			if _, err := child.ChangePlan(); err != nil {
				issues = append(issues, fmt.Sprintf("Focused batch %d has no valid plan: %s", batchIndex+1, err))
				issues = append(issues, child.Feedback...)
			} else {
				for _, d := range child.Decisions {
					if !hasFocusedDecision(s.Decisions, d) {
						s.Decisions = append(s.Decisions, d)
					}
				}
				if !child.AwaitingCombinedReview && child.Status != "COMPLETE" && len(child.Draft) > 0 {
					issues = append(issues, fmt.Sprintf("Focused batch %d draft withheld because reviewer approval is incomplete; inspect its checkpoint.", batchIndex+1))
				}
				allowed := allowedPlanningTargets(s.Bundle, s.Decisions)
				for _, p := range child.Draft {
					if !child.AwaitingCombinedReview && child.Status != "COMPLETE" {
						continue
					}
					if _, ok := allowed[planKey(p.ChangeType, p.ElementID, p.File, p.Symbol)]; !ok {
						issues = append(issues, fmt.Sprintf("Withheld proposal for %s/%s: %s (%s) conflicts with the stored target decision.", p.ChangeType, p.ElementID, p.File, p.Symbol))
						continue
					}
					if !hasFocusedPlan(s.Draft, p) {
						s.Draft = append(s.Draft, p)
					} else if !overlappingFiles[p.File] {
						overlappingFiles[p.File] = true
						issues = append(issues, fmt.Sprintf("Overlapping focused batches proposed changes to %s. Check for conflicting recommendations.", p.File))
					}
				}
				if !child.AwaitingCombinedReview && child.Status != "COMPLETE" {
					issues = append(issues, child.Feedback...)
					if len(child.Feedback) == 0 {
						issues = append(issues, "A focused candidate batch still needs review.")
					}
				}
			}
			batchIndex++
			s.FocusedReview.BatchesReviewed = batchIndex
			if child.AwaitingCombinedReview {
				s.FocusedReview.BatchesPrepared++
			} else {
				s.FocusedReview.BatchesNeedingReview++
			}
			s.FocusedReview.CandidatesProcessed += len(seeds)
			if position == len(batches)-1 && len(reserve) > 0 && lightNeedsMoreSource(change.Candidates, s.Draft[draftStart:]) {
				if e.UltraLight {
					reserve = ultraLightReserve(initial, reserve, s.Draft[draftStart:])
				}
				if len(reserve) == 0 {
					continue
				}
				extra := focusedSeedBatches(reserve)
				if e.UltraLight {
					extra = extra[:1]
					ultraSupplement = true
				}
				batches = append(batches, extra...)
				s.FocusedReview.BatchesTotal += len(extra)
				reserve = nil
			}
		}
	}

	s.Bundle.ContextID, _ = ComputeContextID(s.Bundle)
	s.Role = Reviewer
	s.ActiveChildID = ""
	s.ProgressMessage = "Checking the combined focused plan."
	if err := e.Store.Save(ctx, s); err != nil {
		return s, err
	}
	auditor := FullRepositoryEngine{Model: combinedFirstModel{e.Model}, Store: e.Store}
	review, turns, err := auditor.reviewCombinedPlan(ctx, id, s)
	s.Turns += turns
	if err == nil && !review.Approved {
		revised, revisionErr := e.reviseFocusedPlan(ctx, id, &s, review)
		if revisionErr != nil {
			return pause("Focused plan correction interrupted; prepared drafts are preserved.", revisionErr)
		}
		if revised {
			review, turns, err = auditor.reviewCombinedPlan(ctx, id+"-revised", s)
			s.Turns += turns
		}
	}
	if e.UltraLight {
		for _, change := range original.Changes {
			_, reserve := e.seedCandidates(change.Candidates)
			if len(reserve) == 0 {
				continue
			}
			var plans []FilePlan
			for _, plan := range s.Draft {
				if plan.ChangeType == change.ChangeType && plan.ElementID == change.ElementID {
					plans = append(plans, plan)
				}
			}
			if lightNeedsMoreSource(change.Candidates, plans) {
				issues = append(issues, fmt.Sprintf("Ultra Light left source coverage unresolved for %s/%s. Consider reviewing the remaining mapped candidates.", change.ChangeType, change.ElementID))
			}
		}
	}
	if err != nil {
		if errors.Is(err, ErrAgentContextLimit) || errors.Is(err, errConsistencyAttempts) {
			issues = append(issues, "Combined focused-plan review exceeded its context or retry budget; inspect all file recommendations.")
			if errors.Is(err, ErrAgentModelOutput) {
				issues = append(issues, modelOutputFeedback(Reviewer, err))
			}
		} else {
			return pause("Final focused review interrupted; completed change batches are preserved.", err)
		}
	} else {
		s.ConsistencyReview = &review
		if !review.Approved {
			issues = append(issues, review.Issues...)
		}
	}
	for _, c := range summarizeChangeCoverage(s.Bundle, s.Decisions, s.Draft) {
		if c.Plans == 0 {
			issues = append(issues, fmt.Sprintf("%s/%s has no evidence-backed file plan.", c.ChangeType, c.ElementID))
		}
	}
	issues = append(issues, implementationCoverageIssues(s.Bundle, s.Draft)...)
	issues = append(issues, combinedPlanConflictIssues(s.Bundle, s.Draft)...)
	s.Status = "COMPLETE"
	s.ProgressMessage = "Focused review finished."
	s.Feedback = cleanStrings(issues)
	if len(issues) > 0 || agentIncomplete(s.Bundle) {
		s.Status = "NEEDS_REVIEW"
	}
	if err := e.Store.Save(ctx, s); err != nil {
		return s, err
	}
	return s, nil
}

func mergeFocusedCandidate(change *ChangeInput, c Candidate) {
	for i, v := range change.Candidates {
		if v.File == c.File && v.Symbol == c.Symbol {
			for _, item := range c.Evidence {
				seen := false
				for _, old := range change.Candidates[i].Evidence {
					if old.ID == item.ID {
						seen = true
						break
					}
				}
				if !seen {
					change.Candidates[i].Evidence = append(change.Candidates[i].Evidence, item)
				}
			}
			return
		}
	}
	change.Candidates = append(change.Candidates, c)
}
func hasFocusedDecision(items []CandidateDecision, v CandidateDecision) bool {
	for _, x := range items {
		if x.ChangeType == v.ChangeType && x.ElementID == v.ElementID && x.File == v.File && x.Symbol == v.Symbol {
			return true
		}
	}
	return false
}
func hasFocusedPlan(items []FilePlan, v FilePlan) bool {
	for _, x := range items {
		if x.ChangeType == v.ChangeType && x.ElementID == v.ElementID && x.File == v.File && x.Symbol == v.Symbol {
			return true
		}
	}
	return false
}
