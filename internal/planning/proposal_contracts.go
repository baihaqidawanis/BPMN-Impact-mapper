package planning

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

// ProposalContract is planned coordination context, never observed source.
// It deliberately carries no source excerpts or citable evidence identities.
type ProposalContract struct {
	ChangeType      domain.ChangeType     `json:"change_type"`
	ElementID       string                `json:"element_id"`
	File            string                `json:"file"`
	Symbol          string                `json:"symbol,omitempty"`
	Action          domain.RequiredAction `json:"action"`
	TargetBehavior  string                `json:"target_behavior"`
	ProposedChanges []string              `json:"proposed_changes"`
	Assumptions     []string              `json:"assumptions,omitempty"`
}

// BuildProposalContracts retains other targets, with the latest version of an
// identical target replacing an earlier revision in place. Input plans are not
// modified and no plan text is promoted to source evidence.
func BuildProposalContracts(all, local []FilePlan) []ProposalContract {
	key := func(p FilePlan) string {
		return string(p.ChangeType) + "\x00" + p.ElementID + "\x00" + p.File + "\x00" + p.Symbol
	}
	excluded := map[string]bool{}
	for _, p := range local {
		excluded[key(p)] = true
	}
	positions := map[string]int{}
	var result []ProposalContract
	for _, p := range all {
		id := key(p)
		if excluded[id] {
			continue
		}
		contract := ProposalContract{ChangeType: p.ChangeType, ElementID: p.ElementID, File: p.File, Symbol: p.Symbol, Action: p.Action, TargetBehavior: p.TargetBehavior, ProposedChanges: append([]string{}, p.ProposedChanges...), Assumptions: append([]string{}, p.Assumptions...)}
		if i, ok := positions[id]; ok {
			result[i] = contract
		} else {
			positions[id] = len(result)
			result = append(result, contract)
		}
	}
	return result
}
