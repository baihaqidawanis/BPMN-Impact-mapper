package diff

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

// Keep the sequence flow as the change owner when a gateway branch is
// redirected. Include both tasks so retrieval can find the old and new paths.
func gatewayRerouteContext(before, after *domain.Graph, change domain.Change) *domain.ChangeContext {
	if change.OldElement == nil || change.NewElement == nil {
		return nil
	}
	oldFlow := before.Flows[change.OldElement.ID]
	newFlow := after.Flows[change.NewElement.ID]
	if oldFlow == nil || newFlow == nil || oldFlow.SourceRef != newFlow.SourceRef || oldFlow.TargetRef == newFlow.TargetRef {
		return nil
	}
	oldGateway, newGateway := before.Elements[oldFlow.SourceRef], after.Elements[newFlow.SourceRef]
	if oldGateway == nil || newGateway == nil || !isGateway(oldGateway.Type) || !isGateway(newGateway.Type) {
		return nil
	}
	oldTarget, newTarget := before.Elements[oldFlow.TargetRef], after.Elements[newFlow.TargetRef]
	if oldTarget == nil || newTarget == nil {
		return nil
	}
	return &domain.ChangeContext{
		Pattern:     domain.PatternModifyGatewayBranch,
		ProcessName: after.ProcessNames[newGateway.ProcessID],
		Predecessors: []domain.ElementContext{
			{ID: newGateway.ID, Name: newGateway.Name, Type: newGateway.Type},
		},
		Successors: []domain.ElementContext{
			{ID: oldTarget.ID, Name: oldTarget.Name, Type: oldTarget.Type},
			{ID: newTarget.ID, Name: newTarget.Name, Type: newTarget.Type},
		},
		BranchChanges:  []domain.BranchChange{branchTransition(newFlow.ID, oldFlow, newFlow)},
		BranchTerms:    sortedUnique([]string{oldFlow.Name, oldFlow.Condition, newFlow.Name, newFlow.Condition}),
		RelatedFlowIDs: []string{newFlow.ID},
	}
}

func branchTransition(id string, oldFlow, newFlow *domain.Flow) domain.BranchChange {
	route := domain.BranchChange{FlowID: id}
	if oldFlow != nil {
		route.OldLabel, route.OldCondition, route.OldTarget = oldFlow.Name, oldFlow.Condition, oldFlow.TargetRef
	}
	if newFlow != nil {
		route.NewLabel, route.NewCondition, route.NewTarget = newFlow.Name, newFlow.Condition, newFlow.TargetRef
	}
	return route
}
