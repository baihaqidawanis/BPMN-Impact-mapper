package planning

import "fmt"

// PartialPlan exposes stored proposals for inspection without granting approval,
// changing checkpoints or inventing plans for unresolved targets. Every emitted
// target still passes source identity/evidence/text validation.
func (s AgentSession) PartialPlan() (ChangePlan, error) {
	preview := s
	preview.Status = "NEEDS_REVIEW"
	preview.Draft = nil
	preview.Feedback = append([]string{}, s.Feedback...)
	preview.Feedback = append(preview.Feedback, "PARTIAL CHECKPOINT EXPORT: combined agent review is incomplete. Missing or withheld file proposals remain unresolved. This is not an approved implementation plan.")
	var firstInvalid error
	for _, plan := range s.Draft {
		one := []FilePlan{plan}
		err := validatePlans(s.Bundle, consistencyDecisions(s.Decisions, one), "", one)
		if err == nil && hasFocusedPlan(preview.Draft, plan) {
			err = fmt.Errorf("duplicate file proposal")
		}
		if err != nil {
			if firstInvalid == nil {
				firstInvalid = err
			}
			preview.Feedback = append(preview.Feedback, fmt.Sprintf("Withheld invalid proposal for %s/%s: %s (%s). %s", plan.ChangeType, plan.ElementID, plan.File, plan.Symbol, err))
			continue
		}
		preview.Draft = append(preview.Draft, plan)
	}
	if len(preview.Draft) == 0 && firstInvalid != nil {
		return ChangePlan{}, firstInvalid
	}
	preview.Decisions = consistencyDecisions(s.Decisions, preview.Draft)
	p, err := preview.ChangePlan()
	if err != nil {
		return ChangePlan{}, err
	}
	p.ChangeCoverage = summarizeChangeCoverage(s.Bundle, s.Decisions, preview.Draft)
	for _, d := range s.Decisions {
		if d.Verdict != VerdictRelevant {
			continue
		}
		found := false
		for _, plan := range preview.Draft {
			if planKey(d.ChangeType, d.ElementID, d.File, d.Symbol) == planKey(plan.ChangeType, plan.ElementID, plan.File, plan.Symbol) {
				found = true
				break
			}
		}
		if !found {
			p.ReviewIssues = append(p.ReviewIssues, fmt.Sprintf("Unresolved target %s/%s: %s has no exported file proposal.", d.ChangeType, d.ElementID, d.File))
			for i := range p.ChangeCoverage {
				if p.ChangeCoverage[i].ChangeType == d.ChangeType && p.ChangeCoverage[i].ElementID == d.ElementID {
					p.ChangeCoverage[i].Status = "PARTIAL"
				}
			}
		}
	}
	return finalize(p)
}
