package diff

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func Compare(oldGraph, newGraph *domain.Graph) []domain.Change {
	changes := make([]domain.Change, 0)
	matchedOld, matchedNew := map[string]string{}, map[string]string{}
	for id := range oldGraph.Elements {
		if newGraph.Elements[id] != nil {
			matchedOld[id], matchedNew[id] = id, id
		}
	}
	changes = append(changes, structuralMatches(oldGraph, newGraph, matchedOld, matchedNew)...)

	oldIDs := sortedElementIDs(oldGraph)
	for _, oldID := range oldIDs {
		newID, matched := matchedOld[oldID]
		if !matched {
			changes = append(changes, domain.Change{ChangeType: removalType(oldGraph.Elements[oldID]), OldElement: clone(oldGraph.Elements[oldID]), Impacts: []domain.Impact{}, UnresolvedImpact: []string{}})
			continue
		}
		if newID == "" {
			continue
		}
		changes = append(changes, compareElement(oldGraph, newGraph, oldGraph.Elements[oldID], newGraph.Elements[newID])...)
	}
	for _, newID := range sortedElementIDs(newGraph) {
		if _, matched := matchedNew[newID]; !matched {
			changes = append(changes, domain.Change{ChangeType: additionType(newGraph.Elements[newID]), NewElement: clone(newGraph.Elements[newID]), Impacts: []domain.Impact{}, UnresolvedImpact: []string{}})
		}
	}
	changes = append(changes, compareFlows(oldGraph, newGraph)...)
	annotateChangeContexts(oldGraph, newGraph, changes)
	sort.SliceStable(changes, func(i, j int) bool {
		ki, kj := changeKey(changes[i]), changeKey(changes[j])
		return ki < kj
	})
	return changes
}

func annotateChangeContexts(oldGraph, newGraph *domain.Graph, changes []domain.Change) {
	moved := movedTaskContexts(oldGraph, newGraph)
	for index := range changes {
		change := &changes[index]
		switch change.ChangeType {
		case domain.AddTask:
			if change.NewElement != nil && isTask(change.NewElement.Type) {
				change.Context = insertionContext(oldGraph, newGraph, change.NewElement.ID)
			}
		case domain.RemoveTask:
			if change.OldElement != nil && isTask(change.OldElement.Type) {
				change.Context = removalContext(oldGraph, newGraph, change.OldElement.ID)
			}
		case domain.AddGateway:
			if change.NewElement != nil {
				change.Context = gatewayBranchContext(newGraph, change.NewElement.ID)
			}
		case domain.RemoveGateway:
			if change.OldElement != nil {
				change.Context = removedGatewayBranchContext(oldGraph, newGraph, change.OldElement.ID)
			}
		case domain.ModifySequenceFlow:
			change.Context = movedFlowContext(change, moved)
		}
	}
}

func movedTaskContexts(oldGraph, newGraph *domain.Graph) map[string]*domain.ChangeContext {
	result := map[string]*domain.ChangeContext{}
	for _, id := range sortedElementIDs(oldGraph) {
		oldElement, newElement := oldGraph.Elements[id], newGraph.Elements[id]
		if oldElement == nil || newElement == nil || !isTask(oldElement.Type) || oldElement.Type != newElement.Type || !sameScope(oldElement, newElement) {
			continue
		}
		oldPredecessors, oldIncoming := neighbors(oldGraph, id, true)
		oldSuccessors, oldOutgoing := neighbors(oldGraph, id, false)
		newPredecessors, newIncoming := neighbors(newGraph, id, true)
		newSuccessors, newOutgoing := neighbors(newGraph, id, false)
		if contextIDs(oldPredecessors) == contextIDs(newPredecessors) && contextIDs(oldSuccessors) == contextIDs(newSuccessors) {
			continue
		}
		result[id] = &domain.ChangeContext{Pattern: domain.PatternMoveTask, Predecessors: newPredecessors, Successors: newSuccessors, RelatedFlowIDs: sortedUnique(append(append(append(oldIncoming, oldOutgoing...), newIncoming...), newOutgoing...))}
	}
	return result
}

func movedFlowContext(change *domain.Change, moved map[string]*domain.ChangeContext) *domain.ChangeContext {
	ids := []string{}
	for _, element := range []*domain.Element{change.OldElement, change.NewElement} {
		if element != nil {
			ids = append(ids, element.Incoming...)
			ids = append(ids, element.Outgoing...)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if context := moved[id]; context != nil {
			copy := *context
			copy.Predecessors = append([]domain.ElementContext(nil), context.Predecessors...)
			copy.Successors = append([]domain.ElementContext(nil), context.Successors...)
			copy.RelatedFlowIDs = append([]string(nil), context.RelatedFlowIDs...)
			return &copy
		}
	}
	return nil
}

func contextIDs(values []domain.ElementContext) string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	sort.Strings(ids)
	return strings.Join(ids, "|")
}

func insertionContext(oldGraph, newGraph *domain.Graph, id string) *domain.ChangeContext {
	predecessors, incomingFlows := neighbors(newGraph, id, true)
	successors, outgoingFlows := neighbors(newGraph, id, false)
	if len(predecessors) != 1 || len(successors) != 1 || oldGraph.Elements[predecessors[0].ID] == nil || oldGraph.Elements[successors[0].ID] == nil {
		return nil
	}
	if !sameScope(newGraph.Elements[id], newGraph.Elements[predecessors[0].ID], newGraph.Elements[successors[0].ID]) {
		return nil
	}
	bridged := directFlowIDs(oldGraph, predecessors[0].ID, successors[0].ID)
	if len(bridged) == 0 {
		return nil
	}
	return &domain.ChangeContext{
		Pattern: domain.PatternInsertTask, Predecessors: predecessors, Successors: successors,
		RelatedFlowIDs: sortedUnique(append(append(incomingFlows, outgoingFlows...), bridged...)),
	}
}

func removalContext(oldGraph, newGraph *domain.Graph, id string) *domain.ChangeContext {
	predecessors, incomingFlows := neighbors(oldGraph, id, true)
	successors, outgoingFlows := neighbors(oldGraph, id, false)
	if len(predecessors) != 1 || len(successors) != 1 || newGraph.Elements[predecessors[0].ID] == nil || newGraph.Elements[successors[0].ID] == nil {
		return nil
	}
	if !sameScope(oldGraph.Elements[id], oldGraph.Elements[predecessors[0].ID], oldGraph.Elements[successors[0].ID]) {
		return nil
	}
	bridged := directFlowIDs(newGraph, predecessors[0].ID, successors[0].ID)
	if len(bridged) == 0 {
		return nil
	}
	return &domain.ChangeContext{
		Pattern: domain.PatternRemoveTaskWithRewire, Predecessors: predecessors, Successors: successors,
		RelatedFlowIDs: sortedUnique(append(append(incomingFlows, outgoingFlows...), bridged...)),
	}
}

func gatewayBranchContext(graph *domain.Graph, id string) *domain.ChangeContext {
	predecessors, incomingFlows := neighbors(graph, id, true)
	successors, outgoingFlows := neighbors(graph, id, false)
	if len(predecessors) < 2 && len(successors) < 2 {
		return nil
	}
	return &domain.ChangeContext{
		Pattern: domain.PatternAddGatewayBranch, Predecessors: predecessors, Successors: successors,
		RelatedFlowIDs: sortedUnique(append(incomingFlows, outgoingFlows...)),
	}
}

func removedGatewayBranchContext(oldGraph, newGraph *domain.Graph, id string) *domain.ChangeContext {
	predecessors, incomingFlows := neighbors(oldGraph, id, true)
	successors, outgoingFlows := neighbors(oldGraph, id, false)
	if len(predecessors) < 2 && len(successors) < 2 {
		return nil
	}
	preservedNeighbor := false
	for _, neighbor := range append(append([]domain.ElementContext(nil), predecessors...), successors...) {
		if newGraph.Elements[neighbor.ID] != nil {
			preservedNeighbor = true
			break
		}
	}
	if !preservedNeighbor {
		return nil
	}
	return &domain.ChangeContext{Pattern: domain.PatternRemoveGatewayBranch, Predecessors: predecessors, Successors: successors, RelatedFlowIDs: sortedUnique(append(incomingFlows, outgoingFlows...))}
}

func sameScope(elements ...*domain.Element) bool {
	if len(elements) < 2 || elements[0] == nil {
		return false
	}
	for _, element := range elements[1:] {
		if element == nil || element.ProcessID != elements[0].ProcessID || element.ParentID != elements[0].ParentID {
			return false
		}
	}
	return true
}

func neighbors(graph *domain.Graph, id string, incoming bool) ([]domain.ElementContext, []string) {
	flowIDs := graph.Outgoing[id]
	if incoming {
		flowIDs = graph.Incoming[id]
	}
	flowIDs = append([]string(nil), flowIDs...)
	sort.Strings(flowIDs)
	result := make([]domain.ElementContext, 0, len(flowIDs))
	for _, flowID := range flowIDs {
		flow := graph.Flows[flowID]
		if flow == nil {
			continue
		}
		neighborID := flow.TargetRef
		if incoming {
			neighborID = flow.SourceRef
		}
		if element := graph.Elements[neighborID]; element != nil {
			result = append(result, domain.ElementContext{ID: element.ID, Name: element.Name, Type: element.Type})
		}
	}
	return result, flowIDs
}

func directFlowIDs(graph *domain.Graph, sourceID, targetID string) []string {
	var result []string
	for _, flowID := range graph.Outgoing[sourceID] {
		if flow := graph.Flows[flowID]; flow != nil && flow.TargetRef == targetID {
			result = append(result, flowID)
		}
	}
	sort.Strings(result)
	return result
}

func sortedUnique(values []string) []string {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = true
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func structuralMatches(oldGraph, newGraph *domain.Graph, matchedOld, matchedNew map[string]string) []domain.Change {
	var unresolved []domain.Change
	for _, oldID := range sortedElementIDs(oldGraph) {
		if _, ok := matchedOld[oldID]; ok {
			continue
		}
		oldElement := oldGraph.Elements[oldID]
		best, bestScore, tied := "", 0.0, false
		for _, newID := range sortedElementIDs(newGraph) {
			if _, used := matchedNew[newID]; used || oldElement.Type != newGraph.Elements[newID].Type {
				continue
			}
			score := similarity(oldGraph, newGraph, oldElement, newGraph.Elements[newID])
			if score > bestScore+0.0001 {
				best, bestScore, tied = newID, score, false
			} else if math.Abs(score-bestScore) <= 0.0001 && score > 0 {
				tied = true
			}
		}
		if bestScore < 0.65 {
			continue
		}
		if tied {
			reason := fmt.Sprintf("ambiguous structural match for %s at score %.2f", oldID, bestScore)
			unresolved = append(unresolved, domain.Change{ChangeType: domain.UnresolvedMatch, OldElement: clone(oldElement), Impacts: []domain.Impact{}, UnresolvedImpact: []string{reason}, Reason: reason})
			matchedOld[oldID] = ""
			continue
		}
		matchedOld[oldID], matchedNew[best] = best, oldID
	}
	return unresolved
}

func similarity(oldGraph, newGraph *domain.Graph, oldElement, newElement *domain.Element) float64 {
	if oldElement.ProcessID != "" && newElement.ProcessID != "" && oldElement.ProcessID != newElement.ProcessID {
		return 0
	}
	if oldElement.ParentID != "" && newElement.ParentID != "" && oldElement.ParentID != newElement.ParentID {
		return 0
	}
	score := 0.0
	if textutil.Normalize(oldElement.Name) != "" && textutil.Normalize(oldElement.Name) == textutil.Normalize(newElement.Name) {
		score += 0.35
	}
	if textutil.Normalize(oldElement.Actor) != "" && textutil.Normalize(oldElement.Actor) == textutil.Normalize(newElement.Actor) {
		score += 0.15
	}
	if oldElement.ProcessID != "" && oldElement.ProcessID == newElement.ProcessID && oldElement.ParentID == newElement.ParentID {
		score += 0.1
	}
	if len(oldElement.Incoming) == len(newElement.Incoming) && len(oldElement.Outgoing) == len(newElement.Outgoing) {
		score += 0.2
	}
	if neighborSignature(oldGraph, oldElement) == neighborSignature(newGraph, newElement) && neighborSignature(oldGraph, oldElement) != "" {
		score += 0.2
	}
	return score
}

func neighborSignature(graph *domain.Graph, element *domain.Element) string {
	var names []string
	for _, flowID := range element.Incoming {
		if flow := graph.Flows[flowID]; flow != nil && graph.Elements[flow.SourceRef] != nil {
			names = append(names, textutil.Normalize(graph.Elements[flow.SourceRef].Name))
		}
	}
	for _, flowID := range element.Outgoing {
		if flow := graph.Flows[flowID]; flow != nil && graph.Elements[flow.TargetRef] != nil {
			names = append(names, textutil.Normalize(graph.Elements[flow.TargetRef].Name))
		}
	}
	sort.Strings(names)
	return strings.Join(names, "|")
}

func compareElement(oldGraph, newGraph *domain.Graph, oldElement, newElement *domain.Element) []domain.Change {
	var changes []domain.Change
	base := func(kind domain.ChangeType, reason string) domain.Change {
		return domain.Change{ChangeType: kind, OldElement: clone(oldElement), NewElement: clone(newElement), Impacts: []domain.Impact{}, UnresolvedImpact: []string{}, Reason: reason}
	}
	if oldElement.Name != newElement.Name {
		kind := domain.ModifyTaskLabel
		if isTerminal(oldGraph, oldElement) || isTerminal(newGraph, newElement) {
			kind = domain.ModifyTerminalOutcome
		}
		changes = append(changes, base(kind, "element label changed"))
	}
	if oldElement.Actor != newElement.Actor {
		changes = append(changes, base(domain.ModifyActorRole, "lane or actor changed"))
	}
	oldData, newData := stringSet(oldElement.DataRequirements), stringSet(newElement.DataRequirements)
	if hasAdded(oldData, newData) {
		changes = append(changes, base(domain.AddDataRequirement, "data requirement added"))
	}
	if hasAdded(newData, oldData) {
		changes = append(changes, base(domain.RemoveDataRequirement, "data requirement removed"))
	}
	if oldElement.State == "" && newElement.State != "" {
		changes = append(changes, base(domain.AddState, "data state added"))
	} else if oldElement.State != newElement.State && (oldElement.State != "" || newElement.State != "") {
		changes = append(changes, base(domain.AddState, "data state changed"))
	}
	if isGateway(oldElement.Type) && outgoingConditions(oldGraph, oldElement.ID) != outgoingConditions(newGraph, newElement.ID) {
		changes = append(changes, base(domain.ModifyGatewayCondition, "gateway outgoing condition changed"))
	}
	return changes
}

func compareFlows(oldGraph, newGraph *domain.Graph) []domain.Change {
	var changes []domain.Change
	ids := map[string]struct{}{}
	for id := range oldGraph.Flows {
		ids[id] = struct{}{}
	}
	for id := range newGraph.Flows {
		ids[id] = struct{}{}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	for _, id := range ordered {
		oldFlow, newFlow := oldGraph.Flows[id], newGraph.Flows[id]
		if oldFlow != nil && newFlow != nil && oldFlow.SourceRef == newFlow.SourceRef && oldFlow.TargetRef == newFlow.TargetRef && oldFlow.Name == newFlow.Name && oldFlow.Condition == newFlow.Condition {
			continue
		}
		var oldElement, newElement *domain.Element
		if oldFlow != nil {
			oldElement = flowElement(oldFlow)
		}
		if newFlow != nil {
			newElement = flowElement(newFlow)
		}
		changes = append(changes, domain.Change{ChangeType: domain.ModifySequenceFlow, OldElement: oldElement, NewElement: newElement, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}, Reason: "sequence flow added, removed, or modified"})
	}
	return changes
}

func flowElement(flow *domain.Flow) *domain.Element {
	return &domain.Element{ID: flow.ID, Name: flow.Name, Type: "sequenceFlow", Documentation: flow.Condition, DataRequirements: []string{}, Incoming: []string{flow.SourceRef}, Outgoing: []string{flow.TargetRef}}
}

func additionType(element *domain.Element) domain.ChangeType {
	if isGateway(element.Type) {
		return domain.AddGateway
	}
	if element.Type == "dataObjectReference" || element.Type == "dataStoreReference" {
		return domain.AddDataRequirement
	}
	return domain.AddTask
}

func removalType(element *domain.Element) domain.ChangeType {
	if isGateway(element.Type) {
		return domain.RemoveGateway
	}
	if element.Type == "dataObjectReference" || element.Type == "dataStoreReference" {
		return domain.RemoveDataRequirement
	}
	return domain.RemoveTask
}

func isGateway(kind string) bool { return strings.HasSuffix(kind, "Gateway") }

func isTask(kind string) bool {
	return kind == "task" || strings.HasSuffix(kind, "Task") || kind == "callActivity" || kind == "subProcess"
}

func isTerminal(graph *domain.Graph, element *domain.Element) bool {
	return element.Type == "endEvent" || len(graph.Outgoing[element.ID]) == 0
}

func outgoingConditions(graph *domain.Graph, id string) string {
	var conditions []string
	for _, flowID := range graph.Outgoing[id] {
		if flow := graph.Flows[flowID]; flow != nil {
			conditions = append(conditions, flow.ID+":"+flow.Name+":"+flow.Condition)
		}
	}
	sort.Strings(conditions)
	return strings.Join(conditions, "|")
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func hasAdded(oldValues, newValues map[string]bool) bool {
	for value := range newValues {
		if !oldValues[value] {
			return true
		}
	}
	return false
}

func clone(element *domain.Element) *domain.Element {
	if element == nil {
		return nil
	}
	copy := *element
	copy.EventDefinitions = append([]string{}, element.EventDefinitions...)
	copy.DataRequirements = append([]string{}, element.DataRequirements...)
	copy.Incoming = append([]string{}, element.Incoming...)
	copy.Outgoing = append([]string{}, element.Outgoing...)
	return &copy
}

func sortedElementIDs(graph *domain.Graph) []string {
	ids := make([]string, 0, len(graph.Elements))
	for id := range graph.Elements {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func changeKey(change domain.Change) string {
	id := ""
	if change.NewElement != nil {
		id = change.NewElement.ID
	} else if change.OldElement != nil {
		id = change.OldElement.ID
	}
	return id + "|" + string(change.ChangeType) + "|" + change.Reason
}
