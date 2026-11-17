package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

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
