package diff

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// AnnotateConditionDelta records lexical differences per stable outgoing flow.
// It is opt-in and never interprets operators, code dependencies or required actions.
// Operator-only changes retain an empty delta so queries can use full context.
func AnnotateConditionDelta(oldGraph, newGraph *domain.Graph, changes []domain.Change) {
	for i := range changes {
		c := &changes[i]
		if c.ChangeType != domain.ModifyGatewayCondition || c.Context == nil || c.OldElement == nil || c.NewElement == nil || c.OldElement.ID != c.NewElement.ID {
			continue
		}
		id := c.NewElement.ID
		_, before := neighbors(oldGraph, id, false)
		_, after := neighbors(newGraph, id, false)
		delta := []string{}
		fallback := false
		for _, fid := range sortedUnique(append(before, after...)) {
			old, next := oldGraph.Flows[fid], newGraph.Flows[fid]
			if old == nil || next == nil || old.SourceRef != id || next.SourceRef != id || old.TargetRef != next.TargetRef {
				fallback = true // Added/removed branches need full structural context.
				break
			}
			left, right := map[string]bool{}, map[string]bool{}
			for _, token := range textutil.Tokenize(branchCondition(old)) {
				left[token] = true
			}
			for _, token := range textutil.Tokenize(branchCondition(next)) {
				right[token] = true
			}
			flowDelta := []string{}
			for token := range left {
				if !right[token] {
					flowDelta = append(flowDelta, token)
				}
			}
			for token := range right {
				if !left[token] {
					flowDelta = append(flowDelta, token)
				}
			}
			if branchCondition(old) != branchCondition(next) && len(flowDelta) == 0 {
				fallback = true
				break
			}
			logicalOnly := len(flowDelta) > 0
			for _, token := range flowDelta {
				if token != "and" && token != "or" && token != "not" && token != "true" && token != "false" {
					logicalOnly = false
				}
			}
			if logicalOnly {
				fallback = true
				break
			}
			delta = append(delta, flowDelta...)
		}
		if fallback {
			delta = nil
		}
		copyContext := *c.Context
		copyContext.ConditionDelta = textutil.UniqueSorted(delta)
		c.Context = &copyContext
	}
}
