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
	sort.SliceStable(changes, func(i, j int) bool {
		ki, kj := changeKey(changes[i]), changeKey(changes[j])
		return ki < kj
	})
	return changes
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
	score := 0.0
	if textutil.Normalize(oldElement.Name) != "" && textutil.Normalize(oldElement.Name) == textutil.Normalize(newElement.Name) {
		score += 0.4
	}
	if textutil.Normalize(oldElement.Actor) != "" && textutil.Normalize(oldElement.Actor) == textutil.Normalize(newElement.Actor) {
		score += 0.2
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
