package planning

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

// BuildAgentBundle preserves the report's evidence IDs while assigning an
// explicitly subordinate flow to its owning requirement. Gateway grouping
// carries the branch details and source leads into the owner.
func BuildAgentBundle(report domain.Report, limits Limits) EvidenceBundle {
	b := buildEvidenceBundle(report, limits, true)
	owners := map[string][]int{}
	for i, c := range report.Changes {
		if c.PartOf == "" && b.Changes[i].ElementID != "" {
			id := b.Changes[i].ElementID
			owners[id] = append(owners[id], i)
		}
	}
	grouped := make(map[int]bool)
	for i, c := range report.Changes {
		parent := owners[c.PartOf]
		if c.ChangeType != domain.ModifySequenceFlow || len(parent) != 1 || b.Changes[i].ElementID == "" || len(c.Impacts) != 0 || len(c.RelatedFiles) != 0 {
			continue
		}
		owner := &b.Changes[parent[0]]
		owner.SupportingFlowIDs = append(owner.SupportingFlowIDs, b.Changes[i].ElementID)
		grouped[i] = true
	}
	if limits.GroupGatewayChanges && report.ProcessContext != nil && report.ProcessContext.Before != nil && report.ProcessContext.After != nil {
		groupJoinWiring(report, &b, grouped)
		groupGatewayLabels(report, &b, grouped)
		if limits.GroupGatewayBranches {
			groupGatewayBranches(report, &b, grouped)
			groupNewGatewayBranches(report, &b, grouped)
		}
	}
	if len(grouped) > 0 {
		units := make([]ChangeInput, 0, len(b.Changes)-len(grouped))
		for i, c := range b.Changes {
			if !grouped[i] {
				units = append(units, c)
			}
		}
		b.Changes = units
		b.ContextID, _ = ComputeContextID(b)
	}
	return b
}

// A new split's branch context already describes its new outgoing edges.
// An unconditional link from a new activity is also wiring for the split.
// Rewired edges and separately guarded incoming edges stay reviewable.
func groupNewGatewayBranches(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	before, after := report.ProcessContext.Before, report.ProcessContext.After
	owners := make(map[string]int)
	for i, change := range bundle.Changes {
		if change.ChangeType != domain.AddGateway || change.ContextTruncated || len(change.BranchChanges) < 2 {
			continue
		}
		for _, branch := range change.BranchChanges {
			if flow := after.Flows[branch.FlowID]; before.Flows[branch.FlowID] == nil && flow != nil && flow.SourceRef == change.ElementID {
				owners[branch.FlowID] = i
			}
		}
		for _, id := range after.Incoming[change.ElementID] {
			flow := after.Flows[id]
			if flow != nil && before.Flows[id] == nil && flow.Name == "" && flow.Condition == "" &&
				before.Elements[flow.SourceRef] == nil {
				owners[id] = i
			}
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.PartOf != "" || change.ChangeType != domain.ModifySequenceFlow ||
			change.OldElement != nil || change.NewElement == nil {
			continue
		}
		id := change.NewElement.ID
		parent, ok := owners[id]
		if !ok {
			continue
		}
		unit := &bundle.Changes[parent]
		unit.SupportingFlowIDs = append(unit.SupportingFlowIDs, id)
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

// A branch already described by a condition change shares that review unit.
// A flow moved between gateways or absent from either version stays separate.
func groupGatewayBranches(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	before, after := report.ProcessContext.Before, report.ProcessContext.After
	owners := make(map[string]int)
	for i, change := range bundle.Changes {
		if change.ChangeType != domain.ModifyGatewayCondition || change.ContextTruncated || change.ContextIncomplete {
			continue
		}
		for _, branch := range change.BranchChanges {
			old, next := before.Flows[branch.FlowID], after.Flows[branch.FlowID]
			if old != nil && next != nil && old.SourceRef == change.ElementID && next.SourceRef == change.ElementID {
				owners[branch.FlowID] = i
			}
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.ChangeType != domain.ModifySequenceFlow || change.NewElement == nil {
			continue
		}
		id := change.NewElement.ID
		parent, ok := owners[id]
		if !ok || (change.PartOf != "" && change.PartOf != bundle.Changes[parent].ElementID) {
			continue
		}
		unit := &bundle.Changes[parent]
		unit.SupportingFlowIDs = append(unit.SupportingFlowIDs, id)
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

// The condition review already has the before and after gateway labels.
func groupGatewayLabels(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	conditions := map[string]int{}
	for i, change := range report.Changes {
		if change.ChangeType == domain.ModifyGatewayCondition && change.NewElement != nil {
			conditions[change.NewElement.ID] = i
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.PartOf != "" || change.ChangeType != domain.ModifyTaskLabel || change.NewElement == nil {
			continue
		}
		owner, ok := conditions[change.NewElement.ID]
		if !ok {
			continue
		}
		unit := &bundle.Changes[owner]
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

// Group only unconditional sequence flows incident to a newly added XOR join.
// Conditional flows and distinct activity changes retain their own review.
func groupJoinWiring(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	before, after := report.ProcessContext.Before, report.ProcessContext.After
	joins := map[string]int{}
	for i, change := range report.Changes {
		e := change.NewElement
		if change.ChangeType == domain.AddGateway && e != nil && e.Type == "exclusiveGateway" &&
			len(after.Incoming[e.ID]) >= 2 && len(after.Outgoing[e.ID]) == 1 {
			joins[e.ID] = i
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.PartOf != "" || change.ChangeType != domain.ModifySequenceFlow || change.NewElement == nil {
			continue
		}
		id := change.NewElement.ID
		flow := after.Flows[id]
		if flow == nil || flow.Name != "" || flow.Condition != "" {
			continue
		}
		if old := before.Flows[id]; old != nil && (old.Name != "" || old.Condition != "") {
			continue
		}
		source, fromJoin := joins[flow.SourceRef]
		target, toJoin := joins[flow.TargetRef]
		if fromJoin == toJoin {
			continue
		}
		owner := source
		if toJoin {
			owner = target
		}
		unit := &bundle.Changes[owner]
		unit.SupportingFlowIDs = append(unit.SupportingFlowIDs, id)
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

func mergeAgentLead(unit *ChangeInput, lead Candidate) {
	for i := range unit.Candidates {
		c := &unit.Candidates[i]
		if c.File != lead.File || c.Symbol != lead.Symbol {
			continue
		}
		if c.SuggestedAction != lead.SuggestedAction {
			c.SuggestedAction = domain.ActionVerify
		}
		for _, evidence := range lead.Evidence {
			duplicate := false
			for _, prior := range c.Evidence {
				if prior.Location == evidence.Location && prior.Excerpt == evidence.Excerpt {
					duplicate = true
					break
				}
			}
			if !duplicate {
				c.Evidence = append(c.Evidence, evidence)
			}
		}
		return
	}
	unit.Candidates = append(unit.Candidates, lead)
}
