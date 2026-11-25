package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Append a redundant index only when it fits alongside all primary evidence.
// The input is a compact JSON object produced by the host.
func withLocalRoutes(input []byte, turn planning.AgentTurn, byteBudget int) []byte {
	pc := turn.Bundle.ProcessContext
	if pc == nil || pc.After == nil || (turn.Role != planning.Analyst && turn.Role != planning.Planner && turn.Role != planning.Reviewer) {
		return input
	}
	elementID := ""
	switch {
	case len(turn.Bundle.Changes) == 1:
		elementID = turn.Bundle.Changes[0].ElementID
	case turn.Role == planning.Reviewer && len(turn.Draft) == 1:
		elementID = turn.Draft[0].ElementID
	default:
		return input
	}
	if elementID == "" {
		return input
	}
	routes := localGatewayRoutes(pc.After, elementID)
	if len(routes) == 0 {
		return input
	}
	data, _ := json.Marshal(routes)
	const field = `,"local_after_routes":`
	if len(input)+len(field)+len(data) > byteBudget {
		return input
	}
	out := make([]byte, 0, len(input)+len(field)+len(data))
	out = append(out, input[:len(input)-1]...)
	out = append(out, field...)
	out = append(out, data...)
	return append(out, '}')
}

type gatewayRoute struct {
	From      string `json:"from"`
	FromName  string `json:"from_name,omitempty"`
	Condition string `json:"condition,omitempty"`
	Label     string `json:"label,omitempty"`
	Default   bool   `json:"default,omitempty"`
	To        string `json:"to"`
	ToName    string `json:"to_name,omitempty"`
}

// A compact reading aid for the full after graph. Keep each gateway's
// alternatives together, including the distinct inputs of a join.
func localGatewayRoutes(g *domain.Graph, elementID string) []gatewayRoute {
	nodes := map[string]bool{elementID: true}
	if flow := g.Flows[elementID]; flow != nil {
		nodes = map[string]bool{flow.SourceRef: true, flow.TargetRef: true}
	}
	gateways := map[string]bool{}
	for _, flow := range g.Flows {
		if flow == nil || (!nodes[flow.SourceRef] && !nodes[flow.TargetRef]) {
			continue
		}
		for _, id := range []string{flow.SourceRef, flow.TargetRef} {
			if node := g.Elements[id]; node != nil && strings.HasSuffix(node.Type, "Gateway") {
				gateways[id] = true
			}
		}
	}
	var ids []string
	for id, flow := range g.Flows {
		if flow != nil && (gateways[flow.SourceRef] || gateways[flow.TargetRef]) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var routes []gatewayRoute
	for _, id := range ids {
		f := g.Flows[id]
		source := g.Elements[f.SourceRef]
		target := g.Elements[f.TargetRef]
		r := gatewayRoute{From: f.SourceRef, Condition: f.Condition, Default: source != nil && source.DefaultFlow == id, To: f.TargetRef}
		if source != nil && strings.TrimSpace(source.Name) != "" && source.Name != f.SourceRef {
			r.FromName = strings.TrimSpace(source.Name)
		}
		if target != nil && strings.TrimSpace(target.Name) != "" && target.Name != f.TargetRef {
			r.ToName = strings.TrimSpace(target.Name)
		}
		if f.Name != f.Condition {
			r.Label = f.Name
		}
		routes = append(routes, r)
	}
	return routes
}

const sharedProcessInstructions = " Reconstruct each process graph: merge shared_elements/shared_flows into before/after Elements/Flows. Resolve @ keys/values using id_aliases; emit ORIGINAL BPMN IDs. All nodes/edges remain available. Evidence locations starting ':' use candidate.file as prefix."

// Transport-only factoring of identical before/after values. No source evidence,
// condition, actor, element or edge is dropped. Saved context remains unchanged.
func sharedProcessInput(input []byte, pc *domain.ProcessContext) ([]byte, error) {
	if pc.Before == nil || pc.After == nil {
		return input, nil
	}
	before, after := *pc.Before, *pc.After
	before.Elements = make(map[string]*domain.Element)
	after.Elements = make(map[string]*domain.Element)
	before.Flows = make(map[string]*domain.Flow)
	after.Flows = make(map[string]*domain.Flow)
	elements := make(map[string]*domain.Element)
	flows := make(map[string]*domain.Flow)
	for id, value := range pc.Before.Elements {
		if other, ok := pc.After.Elements[id]; ok && reflect.DeepEqual(value, other) {
			elements[id] = value
		} else {
			before.Elements[id] = value
		}
	}
	for id, value := range pc.After.Elements {
		if _, ok := elements[id]; !ok {
			after.Elements[id] = value
		}
	}
	for id, value := range pc.Before.Flows {
		if other, ok := pc.After.Flows[id]; ok && reflect.DeepEqual(value, other) {
			flows[id] = value
		} else {
			before.Flows[id] = value
		}
	}
	for id, value := range pc.After.Flows {
		if _, ok := flows[id]; !ok {
			after.Flows[id] = value
		}
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(input, &wire); err != nil {
		return nil, err
	}
	var bundle map[string]json.RawMessage
	if err := json.Unmarshal(wire["bundle"], &bundle); err != nil {
		return nil, err
	}
	context, err := json.Marshal(map[string]any{"before": before, "after": after, "shared_elements": elements, "shared_flows": flows})
	if err != nil {
		return nil, err
	}
	// Repeated stable IDs dominate small graphs. Alias only known graph IDs,
	// with a complete dictionary, rather than dropping nodes or shortening text.
	ids := map[string]bool{}
	for _, g := range []*domain.Graph{pc.Before, pc.After} {
		for id := range g.Elements {
			if len(id) > 24 {
				ids[id] = true
			}
		}
		for id := range g.Flows {
			if len(id) > 24 {
				ids[id] = true
			}
		}
	}
	ordered := []string{}
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	aliases := map[string]string{}
	reverse := map[string]string{}
	for i, id := range ordered {
		alias := fmt.Sprintf("@%d", i)
		if _, ok := pc.Before.Elements[alias]; ok {
			return input, nil
		}
		if _, ok := pc.After.Elements[alias]; ok {
			return input, nil
		}
		if _, ok := pc.Before.Flows[alias]; ok {
			return input, nil
		}
		if _, ok := pc.After.Flows[alias]; ok {
			return input, nil
		}
		aliases[alias] = id
		reverse[id] = alias
	}
	var tree any
	if err := json.Unmarshal(context, &tree); err != nil {
		return nil, err
	}
	var factor func(any) any
	factor = func(v any) any {
		switch x := v.(type) {
		case string:
			if a, ok := reverse[x]; ok {
				return a
			}
		case []any:
			for i := range x {
				x[i] = factor(x[i])
			}
		case map[string]any:
			y := map[string]any{}
			for k, val := range x {
				key := k
				if a, ok := reverse[k]; ok {
					key = a
				}
				y[key] = factor(val)
			}
			return y
		}
		return v
	}
	value := factor(tree).(map[string]any)
	value["id_aliases"] = aliases
	context, err = json.Marshal(value)
	if err != nil {
		return nil, err
	}
	bundle["process_context"] = context
	wire["bundle"], err = json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire)
}
