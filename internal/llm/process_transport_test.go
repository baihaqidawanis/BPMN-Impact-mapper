package llm

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"reflect"
	"testing"
)

func TestSharedProcessRoundTripPreservesVersions(t *testing.T) {
	id := "long_stable_task_identifier_for_shared_before_after"
	pc := &domain.ProcessContext{Before: &domain.Graph{Elements: map[string]*domain.Element{id: {ID: id, Type: "userTask", Actor: "Staff"}}, Flows: map[string]*domain.Flow{"edge": {ID: "edge", SourceRef: id, TargetRef: id, Condition: "old"}}}, After: &domain.Graph{Elements: map[string]*domain.Element{id: {ID: id, Type: "userTask", Actor: "Staff"}}, Flows: map[string]*domain.Flow{"edge": {ID: "edge", SourceRef: id, TargetRef: id, Condition: "new"}}}}
	before, _ := json.Marshal(pc)
	input, _ := json.Marshal(map[string]any{"bundle": map[string]any{"process_context": pc}})
	encoded, err := sharedProcessInput(input, pc)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	json.Unmarshal(encoded, &wire)
	ctx := wire["bundle"].(map[string]any)["process_context"].(map[string]any)
	aliases := ctx["id_aliases"].(map[string]any)
	var expand func(any) any
	expand = func(v any) any {
		switch x := v.(type) {
		case string:
			if id, ok := aliases[x]; ok {
				return id
			}
		case []any:
			for i := range x {
				x[i] = expand(x[i])
			}
		case map[string]any:
			y := map[string]any{}
			for k, val := range x {
				if id, ok := aliases[k]; ok {
					k = id.(string)
				}
				y[k] = expand(val)
			}
			return y
		}
		return v
	}
	ctx = expand(ctx).(map[string]any)
	for _, name := range []string{"before", "after"} {
		g := ctx[name].(map[string]any)
		for _, pair := range [][2]string{{"Elements", "shared_elements"}, {"Flows", "shared_flows"}} {
			for id, value := range ctx[pair[1]].(map[string]any) {
				g[pair[0]].(map[string]any)[id] = value
			}
		}
	}
	var reconstructed domain.ProcessContext
	data, _ := json.Marshal(ctx)
	json.Unmarshal(data, &reconstructed)
	if !reflect.DeepEqual(pc, &reconstructed) {
		t.Fatal("lost graph state during shared transport")
	}
	after, _ := json.Marshal(pc)
	if string(before) != string(after) {
		t.Fatal("mutated original process context")
	}
}
