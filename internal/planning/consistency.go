package planning

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

var errConsistencyAttempts = errors.New("final consistency review attempts exhausted")

const gatewayConsistencyInstruction = " For a changed gateway condition, compare the proposed valid and rejected paths across all supplied proposals for that operation. Trace the input sent by the caller, the backend predicate, and the resulting response and state. Flag disagreement between proposed edits or between an edit and its acceptance checks. When source is present, verify the actual owner of the operation. Cite the specific conflicting file, branch or check in the finding."

func canPartitionReview(err error) bool {
	return errors.Is(err, ErrAgentContextLimit) || errors.Is(err, ErrAgentOutputLimit) || errors.Is(err, errConsistencyAttempts)
}

// Try the full proposal first. On context overflow, review detailed file groups
// followed by an explicit compact cross-file contract review. No file is omitted.
func (e FullRepositoryEngine) reviewCombinedPlan(ctx context.Context, id string, s AgentSession) (AgentReview, int, error) {
	if len(s.Draft) == 0 {
		return AgentReview{Issues: []string{"No validated file proposals are available for final consistency review; resolve planning failures first."}}, 0, nil
	}
	var review AgentReview
	// Review the proposed targets, not the entire retrieval inventory again.
	// Keep every BPMN requirement and all evidence for those exact targets.
	s.Bundle = consistencyBundle(s.Bundle, s.Draft, false)
	s.Decisions = consistencyDecisions(s.Decisions, s.Draft)
	var turns int
	var err error
	if !prefersFileReview(e.Model, s.Draft) {
		review, turns, err = e.reviewConsistencyPart(ctx, id+"-consistency", s, "FINAL CROSS-BATCH REVIEW: inspect the entire assembled plan across all changes and files. Detect contradictory requirements, duplicate or incompatible proposals for the same file, missing BPMN change coverage, and unsupported evidence. Approve only if these are consistent. Return actionable issues when not approved. No source modification or test execution is permitted.")
		if !canPartitionReview(err) {
			return review, turns, err
		}
	}
	groups := map[string][]FilePlan{}
	for _, p := range s.Draft {
		groups[p.File] = append(groups[p.File], p)
	}
	files := make([]string, 0, len(groups))
	for file := range groups {
		files = append(files, file)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return review, turns, err
	}
	var issues []string
	var findings []FileReviewFinding
	var revisionTarget AgentRole
	for i, file := range files {
		part := s
		part.Draft = groups[file]
		part.Bundle = consistencyBundle(s.Bundle, part.Draft, false)
		part.Decisions = consistencyDecisions(s.Decisions, part.Draft)
		contracts := BuildProposalContracts(s.Draft, part.Draft)
		fileInstruction := "FINAL FILE REVIEW: all proposals for this file are supplied together with their source evidence and process graphs. Check this file's responsibility within the coordinated proposal, action scope and interfaces. Related proposals are planned contracts only, not source proof. Do not require a test file to implement the backend behavior it tests. No source tools are available. Return review only."
		r, n, partErr := e.reviewConsistencyPart(ctx, fmt.Sprintf("%s-cf-%d", id, i), part, fileInstruction, contracts)
		turns += n
		if errors.Is(partErr, ErrAgentContextLimit) && len(part.Draft) == 1 && len(s.Bundle.Changes) > 1 {
			// Single-proposal file hit context budget because of unrelated
			// change entries or README background; keep full process graphs and
			// retry with only this proposal's change entry.
			scoped := part
			scoped.Bundle = scopedConsistencyBundle(s.Bundle, part.Draft)
			var scopedTurns int
			r, scopedTurns, partErr = e.reviewConsistencyPart(ctx, fmt.Sprintf("%s-cf-%d-scoped", id, i), scoped, fileInstruction, contracts)
			turns += scopedTurns
		}
		if canPartitionReview(partErr) && len(part.Draft) > 1 {
			// A single file can have many change/function proposals. Review
			// each in full before the mandatory cross-file contract pass.
			r = AgentReview{Approved: true}
			partErr = nil
			targetInstruction := "FINAL TARGET SOURCE REVIEW: inspect this complete proposal against its source and all BPMN requirements. Cross-file coordination is checked separately afterward. Do not require this file to implement another file's responsibility. Verify function owner, exact operation, existing types and schema constraints. A missing excerpt is NEED_EVIDENCE, never proof of existing behavior. Check current behavior, proposed edits and acceptance checks for contradictions. Return review only."
			for j, proposal := range part.Draft {
				one := part
				one.Draft = []FilePlan{proposal}
				one.Bundle = consistencyBundle(s.Bundle, one.Draft, false)
				one.Decisions = nil
				check, count, checkErr := e.reviewConsistencyPart(ctx, fmt.Sprintf("%s-target-%d-%d", id, i, j), one, targetInstruction)
				turns += count
				if errors.Is(checkErr, ErrAgentContextLimit) && len(s.Bundle.Changes) > 1 {
					one.Bundle = scopedConsistencyBundle(s.Bundle, one.Draft)
					var scopedCount int
					check, scopedCount, checkErr = e.reviewConsistencyPart(ctx, fmt.Sprintf("%s-target-%d-%d-scoped", id, i, j), one, targetInstruction)
					turns += scopedCount
				}
				if canPartitionReview(checkErr) {
					r.Approved = false
					r.Issues = append(r.Issues, fmt.Sprintf("Final review could not validate proposal %d for %s.", j+1, file))
					if errors.Is(checkErr, ErrAgentModelOutput) {
						r.Issues = append(r.Issues, modelOutputFeedback(Reviewer, checkErr))
					}
					continue
				}
				if checkErr != nil {
					partErr = checkErr
					break
				}
				r.Approved = r.Approved && check.Approved
				r.Issues = append(r.Issues, check.Issues...)
				r.Findings = append(r.Findings, check.Findings...)
				if check.RevisionTarget == Analyst || (r.RevisionTarget == "" && check.RevisionTarget == Planner) {
					r.RevisionTarget = check.RevisionTarget
				}
			}
		}
		if canPartitionReview(partErr) {
			issues = append(issues, fmt.Sprintf("Final review could not validate every proposal for %s. Inspect this file before applying the plan.", file))
			if errors.Is(partErr, ErrAgentModelOutput) {
				issues = append(issues, modelOutputFeedback(Reviewer, partErr))
			}
		} else if partErr != nil {
			return AgentReview{}, turns, partErr
		}
		issues = append(issues, r.Issues...)
		findings = append(findings, r.Findings...)
		if r.RevisionTarget == Analyst || (revisionTarget == "" && r.RevisionTarget == Planner) {
			revisionTarget = r.RevisionTarget
		}
	}
	if len(issues) > 0 {
		return AgentReview{Issues: cleanStrings(issues), RevisionTarget: revisionTarget, Findings: findings}, turns, nil
	}
	// Detailed steps and evidence were reviewed above. This turn reviews only
	// cross-file contracts, retaining every plan's requirements and proposed edits.
	compact := s
	compact.Draft = append([]FilePlan(nil), s.Draft...)
	for i := range compact.Draft {
		if compact.Draft[i].Action != "NO_ACTION" {
			compact.Draft[i].CurrentBehavior = ""
		}
		compact.Draft[i].DesignRationale = ""
		compact.Draft[i].Steps = nil
		if compact.Draft[i].ChangeType != domain.ModifyGatewayCondition {
			compact.Draft[i].AcceptanceChecks = nil
		}
	}
	compact.Bundle = consistencyBundle(s.Bundle, s.Draft, true)
	compact.Decisions = nil
	r, n, finalErr := e.reviewConsistencyPart(ctx, id+"-contracts", compact, "FINAL CONTRACT REVIEW: every file proposal is present with target behavior, summary, proposed edits and assumptions. Detailed steps and source evidence were checked separately. Gateway condition acceptance checks are included to compare outcomes across files. Check cross-file contracts and BPMN coverage. If a proposal makes a value required on every write but BPMN requires it at one transition, mark scope REVISE. Check that proposed tests exercise the entry point that implements the rule. Do not infer source facts. Return review only.")
	r.Findings = append(findings, r.Findings...)
	if canPartitionReview(finalErr) {
		r.Issues = []string{"File reviews finished, but the combined contract review could not be validated. Check consistency across files before applying the plan."}
		return r, turns + n, nil
	}
	return r, turns + n, finalErr
}

func (e FullRepositoryEngine) reviewConsistencyPart(ctx context.Context, childID string, s AgentSession, instruction string, related ...[]ProposalContract) (AgentReview, int, error) {
	for _, change := range s.Bundle.Changes {
		if change.ChangeType == domain.ModifyGatewayCondition {
			instruction += gatewayConsistencyInstruction
			break
		}
	}
	parts := []any{"consistency-v5-scoped-target-reviews", instruction, s.Binding, s.Bundle.ContextID, s.Draft, s.Decisions}
	var contracts []ProposalContract
	if len(related) > 0 && len(related[0]) > 0 {
		contracts = related[0]
		childID += "-related-" + agentDigest(contracts)[:12]
		parts = append(parts, "related-proposals-v1", contracts)
	}
	if len(s.Feedback) > 0 {
		// A repair must reach the model, not replay the same failed prompt or
		// reuse a successful receipt produced under different feedback.
		childID += "-fb-" + agentDigest(s.Feedback)[:12]
		parts = append(parts, "forward-feedback-v1", s.Feedback)
	}
	// Compound source/feedback receipts must respect the filesystem store's
	// 64-character ID limit. Hash the entire derived name, never truncate it.
	if len(childID) > 64 {
		childID = "review-" + agentDigest([]any{"review-receipt-v3", childID})[:48]
	}
	binding := agentDigest(parts)
	receipt, err := e.Store.Load(ctx, childID)
	if errors.Is(err, ErrAgentSessionMissing) {
		receipt = AgentSession{Version: "1", ID: childID, Binding: binding, Identity: s.Identity, Bundle: s.Bundle, Role: Reviewer, Status: "RUNNING"}
	} else if err != nil {
		return AgentReview{}, 0, err
	}
	if receipt.Binding != binding {
		return AgentReview{}, receipt.Turns, errors.New("combined review binding changed")
	}
	if receipt.Status == "CONTEXT_LIMIT" {
		return AgentReview{}, receipt.Turns, ErrAgentContextLimit
	}
	if receipt.Status == "OUTPUT_LIMIT" {
		return AgentReview{}, receipt.Turns, ErrAgentOutputLimit
	}
	if receipt.Status == "COMPLETE" && receipt.ConsistencyReview != nil {
		return *receipt.ConsistencyReview, receipt.Turns, nil
	}
	for receipt.Turns < 3 {
		receipt.Turns++
		if err := e.Store.Save(ctx, receipt); err != nil {
			return AgentReview{}, receipt.Turns, err
		}
		callCtx, cancel := context.WithTimeout(ctx, AgentTurnTimeout)
		feedback := append(append([]string{instruction}, s.Feedback...), receipt.Feedback...)
		reply, callErr := e.Model.Turn(callCtx, AgentTurn{Role: Reviewer, Bundle: s.Bundle, Draft: s.Draft, Decisions: s.Decisions, RelatedProposals: contracts, RemainingTools: 0, Feedback: feedback})
		cancel()
		if callErr == nil {
			if reply.Review == nil || reply.Plan != nil || reply.Tool != nil || reply.Verification != nil || reply.NeedEvidence != nil {
				callErr = fmt.Errorf("%w: final check requires one review action", ErrAgentModelOutput)
			} else {
				review := *reply.Review
				review.Issues = cleanStrings(review.Issues)
				if (review.Approved && (len(review.Issues) > 0 || review.RevisionTarget != "")) || (!review.Approved && len(review.Issues) == 0) || len(review.Issues) > 24 {
					callErr = fmt.Errorf("%w: contradictory or missing final review conclusion", ErrAgentModelOutput)
				}
				for _, issue := range review.Issues {
					if len([]rune(issue)) > 2000 {
						callErr = fmt.Errorf("%w: review issue too long", ErrAgentModelOutput)
					}
				}
				if callErr == nil {
					receipt.ConsistencyReview = &review
					receipt.Status = "COMPLETE"
					receipt.Feedback = nil
					if err := e.Store.Save(ctx, receipt); err != nil {
						return AgentReview{}, receipt.Turns, err
					}
					return review, receipt.Turns, nil
				}
			}
		}
		receipt.Status = "PAUSED"
		receipt.Feedback = []string{"Final review request interrupted; no approval was inferred."}
		invalid := !errors.Is(callErr, ErrAgentOutputLimit) && (errors.Is(callErr, ErrAgentModelOutput) || errors.Is(callErr, ErrAgentPlanIncomplete))
		if invalid {
			receipt.Feedback = []string{modelOutputFeedback(Reviewer, callErr)}
			if receipt.Turns >= 3 {
				receipt.Status = "NEEDS_REVIEW"
			}
		}
		if errors.Is(callErr, ErrAgentContextLimit) {
			receipt.Status = "CONTEXT_LIMIT"
			receipt.Feedback = []string{"Final review exceeded model context; source was not silently truncated."}
		}
		if errors.Is(callErr, ErrAgentOutputLimit) {
			receipt.Status = "OUTPUT_LIMIT"
			receipt.Feedback = []string{"Output budget exhausted; the incomplete response was not accepted."}
		}
		checkpoint, stop := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		saveErr := e.Store.Save(checkpoint, receipt)
		stop()
		if saveErr != nil {
			return AgentReview{}, receipt.Turns, saveErr
		}
		// A per-batch AgentEngine already owns one bounded output repair and
		// forwards its feedback to all coordinated file checks. Do not multiply
		// that repair budget with another local retry loop.
		_, parentManaged := e.Model.(batchReviewModel)
		if invalid && parentManaged {
			return AgentReview{}, receipt.Turns, callErr
		}
		if !invalid {
			return AgentReview{}, receipt.Turns, callErr
		}
		if receipt.Turns >= 3 {
			return AgentReview{}, receipt.Turns, errors.Join(errConsistencyAttempts, callErr)
		}
		// Retry only invalid model output, with persisted, specific feedback.
		// Same receipt and turn budget; never retry network failures blindly.
	}
	return AgentReview{}, receipt.Turns, errConsistencyAttempts
}
func summarizeChangeCoverage(bundle EvidenceBundle, decisions []CandidateDecision, plans []FilePlan) []ChangeCoverage {
	result := make([]ChangeCoverage, 0, len(bundle.Changes))
	for _, change := range bundle.Changes {
		item := ChangeCoverage{ElementName: change.ElementName, ChangeType: change.ChangeType, ElementID: change.ElementID, Status: "UNRESOLVED"}
		noChange := 0
		for _, d := range decisions {
			if d.ChangeType == change.ChangeType && d.ElementID == change.ElementID {
				switch d.Verdict {
				case VerdictRelevant:
					item.Relevant++
				case VerdictWeak:
					item.Weak++
				case VerdictReject:
					item.Rejected++
				}
			}
		}
		for _, p := range plans {
			if p.ChangeType == change.ChangeType && p.ElementID == change.ElementID {
				if p.Action == "NO_ACTION" {
					noChange++
				} else {
					item.Plans++
				}
			}
		}
		if item.Plans > 0 {
			item.Status = "PLANNED"
		} else if noChange > 0 {
			item.Status = "NO_CHANGE_PROPOSED"
		}
		result = append(result, item)
	}
	return result
}

// Keep all changed requirements and the process context; only candidate evidence
// outside this file review is omitted. Compact contract review carries no source
// excerpts and must not pretend to revalidate source facts.
func consistencyBundle(original EvidenceBundle, plans []FilePlan, contracts bool) EvidenceBundle {
	b := original
	b.Changes = append([]ChangeInput(nil), original.Changes...)
	for i, c := range b.Changes {
		b.Changes[i].Candidates = nil
		if contracts {
			continue
		}
		for _, candidate := range c.Candidates {
			for _, p := range plans {
				if p.ChangeType == c.ChangeType && p.ElementID == c.ElementID && p.File == candidate.File && p.Symbol == candidate.Symbol {
					b.Changes[i].Candidates = append(b.Changes[i].Candidates, candidate)
					break
				}
			}
		}
	}
	return b
}

// Narrow source-backed file/target reviews to the changes referenced by plans
// while retaining full process graphs. This avoids paying context budget for
// unrelated changes or README background and keeps single-change local routes active.
func scopedConsistencyBundle(original EvidenceBundle, plans []FilePlan) EvidenceBundle {
	b := consistencyBundle(original, plans, false)
	b.RepositoryOverview = nil
	wanted := map[string]bool{}
	for _, p := range plans {
		if p.ChangeType != "" || p.ElementID != "" {
			wanted[planKey(p.ChangeType, p.ElementID, "", "")] = true
		}
	}
	if len(wanted) > 0 {
		filtered := make([]ChangeInput, 0, len(wanted))
		for _, c := range b.Changes {
			if wanted[planKey(c.ChangeType, c.ElementID, "", "")] {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			b.Changes = filtered
		}
	}
	b.ContextID, _ = ComputeContextID(b)
	return b
}
func consistencyDecisions(original []CandidateDecision, plans []FilePlan) []CandidateDecision {
	var result []CandidateDecision
	for _, d := range original {
		for _, p := range plans {
			if d.ChangeType == p.ChangeType && d.ElementID == p.ElementID && d.File == p.File && d.Symbol == p.Symbol {
				result = append(result, d)
				break
			}
		}
	}
	return result
}
