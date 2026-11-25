package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestInvalidPathKeepsItsOwnGatewayAndSibling(t *testing.T) {
	g := &domain.Graph{Elements: map[string]*domain.Element{
		"valid": {Type: "exclusiveGateway", DefaultFlow: "fail"}, "delta": {Type: "exclusiveGateway"},
	}, Flows: map[string]*domain.Flow{
		"fail":   {SourceRef: "valid", TargetRef: "invalid", Name: "Validation failed"},
		"pass":   {SourceRef: "valid", TargetRef: "inspect", Condition: "valid version"},
		"submit": {SourceRef: "delta", TargetRef: "queue", Condition: "nonzero"},
	}}
	turn := planning.AgentTurn{Role: planning.Planner, Bundle: planning.EvidenceBundle{
		ProcessContext: &domain.ProcessContext{After: g}, Changes: []planning.ChangeInput{{ElementID: "invalid"}},
	}}
	input := []byte(`{"bundle":{}}`)
	for _, role := range []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer} {
		turn.Role = role
		payload := withLocalRoutes(input, turn, 4096)
		var wire struct {
			Routes []gatewayRoute `json:"local_after_routes"`
		}
		if err := json.Unmarshal(payload, &wire); err != nil {
			t.Fatal(err)
		}
		want := []gatewayRoute{{From: "valid", To: "invalid", Label: "Validation failed", Default: true}, {From: "valid", To: "inspect", Condition: "valid version"}}
		if !reflect.DeepEqual(wire.Routes, want) {
			t.Fatalf("%s routes: %+v", role, wire.Routes)
		}
		if got := withLocalRoutes(input, turn, len(payload)-1); string(got) != string(input) {
			t.Fatal("route index displaced original evidence")
		}
	}
}

func TestLocalGatewayRoutesIncludesElementNames(t *testing.T) {
	g := &domain.Graph{Elements: map[string]*domain.Element{
		"check":   {ID: "check", Type: "serviceTask", Name: "Lock adjustment DRAFT"},
		"valid":   {ID: "valid", Type: "exclusiveGateway", Name: "Syarat terpenuhi?"},
		"invalid": {ID: "invalid", Type: "serviceTask", Name: "Tolak submit"},
	}, Flows: map[string]*domain.Flow{
		"in":   {SourceRef: "check", TargetRef: "valid"},
		"fail": {SourceRef: "valid", TargetRef: "invalid", Name: "Tidak"},
	}}
	routes := localGatewayRoutes(g, "invalid")
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %+v", routes)
	}
	if routes[0].FromName != "Syarat terpenuhi?" || routes[0].ToName != "Tolak submit" {
		t.Fatalf("expected named endpoints on fail edge, got %+v", routes[0])
	}
	if routes[1].FromName != "Lock adjustment DRAFT" || routes[1].ToName != "Syarat terpenuhi?" {
		t.Fatalf("expected named endpoints on incoming check edge, got %+v", routes[1])
	}
}

func TestSkipSingleTargetReviewFits32KTransportAndIncludesLocalRoutes(t *testing.T) {
	base := filepath.Join("..", "..", "experiments", "simple-erp", "poc-patterns", "skip")
	beforeBytes, err := os.ReadFile(filepath.Join(base, "before.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	afterBytes, err := os.ReadFile(filepath.Join(base, "after.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := bpmn.Parse(beforeBytes, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := bpmn.Parse(afterBytes, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	pc := &domain.ProcessContext{Before: before, After: after}
	changes := diff.Compare(before, after)
	bundle := planning.BuildAgentBundle(domain.Report{AnalysisID: strings.Repeat("a", 32), InputFingerprint: strings.Repeat("b", 32), Changes: changes, ProcessContext: pc}, planning.DefaultLimits())
	if len(bundle.Changes) == 0 {
		t.Fatal("expected Skip diff changes")
	}
	// Simulate scoped single-target review bundle (1 change, no README, realistic source excerpt).
	single := bundle
	single.RepositoryOverview = nil
	targetChange := bundle.Changes[0]
	targetChange.Candidates = []planning.Candidate{{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Symbol:          "Store.TransitionAdjustment",
		Module:          "backend",
		Layer:           "adapter",
		SuggestedAction: domain.ActionVerify,
		Evidence: []planning.EvidenceItem{{
			ID:       "src-1",
			Type:     "SOURCE_TOOL",
			Location: "backend/internal/inventory/adapter/postgres/store.go:120-195",
			Excerpt:  strings.Repeat("x", 3000),
		}},
	}}
	single.Changes = []planning.ChangeInput{targetChange}
	draft := []planning.FilePlan{{
		ChangeType:       targetChange.ChangeType,
		ElementID:        targetChange.ElementID,
		File:             targetChange.Candidates[0].File,
		Symbol:           targetChange.Candidates[0].Symbol,
		Module:           "backend",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		Summary:          "Evaluate per-line deltas and persist selected status before commit.",
		CurrentBehavior:  "TransitionAdjustment transitions DRAFT to SUBMITTED.",
		TargetBehavior:   "If all lines have zero delta close as NO_ADJUSTMENT; otherwise SUBMITTED.",
		DesignRationale:  "Enforce rule inside transaction.",
		ProposedChanges:  []string{"Compare each line against snapshot and set NO_ADJUSTMENT or SUBMITTED."},
		Steps:            []string{"Lock DRAFT row, check version, compare each line, update status, audit, commit."},
		AcceptanceChecks: []string{"Zero delta sets NO_ADJUSTMENT; +2 and -2 sets SUBMITTED."},
		EvidenceIDs:      []string{"src-1"},
	}}
	var sawRoutes bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			var body struct{ Messages []struct{ Content string } }
			json.NewDecoder(r.Body).Decode(&body)
			sawRoutes = strings.Contains(body.Messages[1].Content, `"local_after_routes":`)
			resp := map[string]any{"review": map[string]any{
				"checks": map[string]any{
					"0": map[string]any{
						"evidence_indices": []int{0},
						"requirement":      map[string]any{"status": "PASS", "finding": "Per-line check matches after BPMN."},
						"contracts":        map[string]any{"status": "PASS", "finding": "Signature matches TransitionAdjustment."},
						"scope":            map[string]any{"status": "PASS", "finding": "Scoped to submit branch."},
						"failure_tests":    map[string]any{"status": "PASS", "finding": "Covers zero and opposite nonzero deltas."},
					},
				},
			}}
			content, _ := json.Marshal(resp)
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]string{"role": "assistant", "content": string(content)}})
		}
	}))
	defer srv.Close()
	model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, MaxOutput: 6144, ReviewJSONMode: true, Client: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = model.Turn(context.Background(), planning.AgentTurn{
		Role:     planning.Reviewer,
		Bundle:   single,
		Draft:    draft,
		Feedback: []string{"FINAL TARGET SOURCE REVIEW: inspect this complete proposal against its source and all BPMN requirements."},
	})
	if err != nil {
		t.Fatalf("scoped single-target Skip review failed transport budget: %v", err)
	}
	if !sawRoutes {
		t.Fatal("expected local_after_routes to be attached for scoped single-target Skip review")
	}
}

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
