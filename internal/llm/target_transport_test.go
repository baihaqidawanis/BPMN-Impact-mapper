package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func targetFixture() planning.AgentTurn {
	return planning.AgentTurn{Role: planning.Analyst, Bundle: planning.EvidenceBundle{Changes: []planning.ChangeInput{{ChangeType: domain.AddTask, ElementID: "new-task", Candidates: []planning.Candidate{
		{File: "ui.ts", Symbol: "Panel", Module: "frontend", Layer: "frontend", Evidence: []planning.EvidenceItem{{ID: "UI-proof", Excerpt: "submit"}}},
		{File: "store.go", Symbol: "Store.Submit", Module: "backend", Layer: "adapter", Evidence: []planning.EvidenceItem{{ID: "Store-proof", Excerpt: "DRAFT"}}},
	}}}}}
}

func TestIndexedReferencesBindOriginalCandidateAndEvidence(t *testing.T) {
	turn := targetFixture()
	content, err := restoreTargetReferences(`{"verification":{"decisions":[{"target_index":1,"evidence_indices":[0],"verdict":"RELEVANT","rationale":"Guard applies here."}]}}`, turn)
	var reply planning.AgentReply
	if err != nil || json.Unmarshal([]byte(content), &reply) != nil {
		t.Fatal(content, err)
	}
	d := reply.Verification.Decisions[0]
	if d.File != "store.go" || d.Symbol != "Store.Submit" || d.ElementID != "new-task" || d.EvidenceIDs[0] != "Store-proof" {
		t.Fatal("wrong host binding", d)
	}
	for _, item := range []string{
		`{"target_index":5,"evidence_indices":[0]}`,
		`{"target_index":null,"evidence_indices":[0]}`,
		`{"target_index":0,"evidence_indices":[1]}`,
		`{"target_index":0,"evidence_indices":[]}`,
		`{"target_index":0,"evidence_indices":[null]}`,
		`{"target_index":-1,"evidence_indices":[0]}`,
		`{"target_index":0.5,"evidence_indices":[0]}`,
		`{"target_index":"0","evidence_indices":[0]}`,
		`{"target_index":true,"evidence_indices":[0]}`,
		`{"target_index":0,"evidence_indices":["0"]}`,
		`{"target_index":0,"evidence_indices":[-1]}`,
		`{"target_index":0,"evidence_indices":[0.5]}`,
		`{"target_index":0,"evidence_indices":[999]}`,
		`{"target_index":0,"file":"store.go","evidence_indices":[0]}`,
	} {
		if _, err := restoreTargetReferences(`{"verification":{"decisions":[`+item+`]}}`, turn); err == nil {
			t.Fatal("invalid reference accepted", item)
		}
	}
}

func TestIndexedInputKeepsOriginalDeltaFactsAndDoesNotMutate(t *testing.T) {
	turn := targetFixture()
	turn.Bundle.ProcessContext = &domain.ProcessContext{Before: &domain.Graph{Elements: map[string]*domain.Element{}, Flows: map[string]*domain.Flow{}}, After: &domain.Graph{Elements: map[string]*domain.Element{"new-task": {ID: "new-task", Name: "Validate reason", Type: "userTask"}}, Flows: map[string]*domain.Flow{}}}
	data, _ := json.Marshal(turn)
	annotated, err := annotateAgentInput(data, turn)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		ChangeFacts []struct {
			BeforePresent bool   `json:"before_present"`
			AfterPresent  bool   `json:"after_present"`
			ElementID     string `json:"element_id"`
		} `json:"change_facts"`
	}
	if json.Unmarshal(annotated, &wire) != nil || len(wire.ChangeFacts) != 1 || wire.ChangeFacts[0].BeforePresent || !wire.ChangeFacts[0].AfterPresent || wire.ChangeFacts[0].ElementID != "new-task" {
		t.Fatal(string(annotated))
	}
	if !strings.Contains(string(annotated), `"target_index":1`) || !strings.Contains(string(annotated), `"evidence_index":0`) {
		t.Fatal("indices absent")
	}
	after, _ := json.Marshal(turn)
	if string(after) != string(data) {
		t.Fatal("mutated original turn")
	}
	schema, _ := agentActionSchema(planning.Analyst)
	indexedTargetSchema(schema, "verification", "decisions", turn, false)
	encoded, _ := json.Marshal(schema)
	if !strings.Contains(string(encoded), "evidence_indices") || strings.Contains(string(encoded), "evidence_ids") {
		t.Fatal("schema still requires copying identity strings", string(encoded))
	}
}

func TestOllamaIndexedDecisionRoundTrip(t *testing.T) {
	turn := targetFixture()
	chats := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			chats++
			var body struct {
				Messages []struct{ Content string }
				Format   map[string]any
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if !strings.Contains(body.Messages[1].Content, `"target_index":1`) {
				t.Error("target indices not sent")
			}
			var content string
			if turn.Role == planning.Planner {
				content = `{"plan":{"plans":{"1":{"evidence_indices":[0],"summary":"Require reason at submit","current_behavior":"Submit checks status","target_behavior":"Submit validates reason","design_rationale":"Existing guard location","steps":["Add action-specific validation"],"proposed_changes":["Reject empty submit reason"],"acceptance_checks":["Empty reason leaves state unchanged"]}}}}`
			} else {
				content = `{"verification":{"decisions":[{"target_index":1,"evidence_indices":[0],"verdict":"RELEVANT","rationale":"Existing submit guard"}]}}`
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": content}})
		}
	}))
	defer srv.Close()
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", Client: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	decision, err := m.Turn(context.Background(), turn)
	if err != nil || decision.Verification == nil {
		t.Fatal(decision, err)
	}
	turn.Decisions = decision.Verification.Decisions
	turn.Role = planning.Planner
	result, err := m.Turn(context.Background(), turn)
	if err != nil || result.Plan == nil {
		t.Fatal(result, err)
	}
	p := result.Plan.Plans[0]
	if chats != 2 || p.File != "store.go" || p.Symbol != "Store.Submit" || p.Module != "backend" || p.Layer != "adapter" || p.EvidenceIDs[0] != "Store-proof" || p.Action != domain.ActionVerify {
		t.Fatal("indexed plan incorrectly bound", p)
	}
}

func TestIndexedPlanActionIsHostBound(t *testing.T) {
	turn := targetFixture()
	turn.Role = planning.Planner
	content, err := restoreTargetReferences(`{"plan":{"plans":[{"target_index":1,"evidence_indices":[0]}]}}`, turn)
	var reply planning.AgentReply
	if err != nil || json.Unmarshal([]byte(content), &reply) != nil || reply.Plan.Plans[0].Action != domain.ActionVerify {
		t.Fatal(content, err)
	}
	for _, action := range []string{"modify", "ADD_TASK", "add"} {
		raw := `{"plan":{"plans":[{"target_index":1,"evidence_indices":[0],"action":"` + action + `"}]}}`
		if _, err := restoreTargetReferences(raw, turn); err == nil {
			t.Fatal("invalid legacy action accepted", action)
		}
	}
	schema, _ := agentActionSchema(planning.Planner)
	indexedTargetSchema(schema, "plan", "plans", turn, true)
	b, _ := json.Marshal(schema)
	var tree map[string]any
	json.Unmarshal(b, &tree)
	var inspect func(map[string]any)
	inspect = func(s map[string]any) {
		if choices, ok := s["oneOf"].([]any); ok {
			for _, c := range choices {
				inspect(c.(map[string]any))
			}
			return
		}
		props, _ := s["properties"].(map[string]any)
		plan, ok := props["plan"].(map[string]any)
		if !ok {
			return
		}
		slots := plan["properties"].(map[string]any)["plans"].(map[string]any)
		if slots["type"] != "object" {
			t.Fatal("planner targets are not fixed slots")
		}
		for _, v := range slots["properties"].(map[string]any) {
			if _, ok := v.(map[string]any)["properties"].(map[string]any)["action"]; ok {
				t.Fatal("model still chooses unrestricted action")
			}
		}
	}
	inspect(tree)
}

func TestPlanSchemaRequiresRelevantButAllowsWeakOmission(t *testing.T) {
	schema, _ := agentActionSchema(planning.Planner)
	constrainActionRange(schema, "plan", "plans", 1, 2)
	var inspect func(map[string]any)
	found := false
	inspect = func(s map[string]any) {
		if choices, ok := s["oneOf"].([]any); ok {
			for _, c := range choices {
				inspect(c.(map[string]any))
			}
			return
		}
		props, _ := s["properties"].(map[string]any)
		plan, ok := props["plan"].(map[string]any)
		if !ok {
			return
		}
		array := plan["properties"].(map[string]any)["plans"].(map[string]any)
		if array["minItems"] != 1 || array["maxItems"] != 2 {
			t.Fatal("weak targets made mandatory", array)
		}
		found = true
	}
	inspect(schema)
	if !found {
		t.Fatal("plan schema absent")
	}
}

func TestPlannerSlotsBindUniqueRelevantTargetsAndExcludeWeak(t *testing.T) {
	turn := targetFixture()
	turn.Role = planning.Planner
	for i, target := range turnTargets(turn) {
		verdict := planning.VerdictRelevant
		if i == 0 {
			verdict = planning.VerdictWeak
		}
		turn.Decisions = append(turn.Decisions, planning.CandidateDecision{ChangeType: target.change.ChangeType, ElementID: target.change.ElementID, File: target.candidate.File, Symbol: target.candidate.Symbol, Verdict: verdict})
	}
	schema := map[string]any{"properties": map[string]any{"plan": map[string]any{"properties": map[string]any{"plans": map[string]any{}}}}}
	indexedTargetSchema(schema, "plan", "plans", turn, true)
	slots := schema["properties"].(map[string]any)["plan"].(map[string]any)["properties"].(map[string]any)["plans"].(map[string]any)
	if slots["type"] != "object" || len(slots["required"].([]string)) != 1 || slots["required"].([]string)[0] != "1" || len(slots["properties"].(map[string]any)) != 1 {
		t.Fatal(slots)
	}
	content, err := restoreTargetReferences(`{"plan":{"plans":{"1":{"evidence_indices":[0]},"0":{"evidence_indices":[0]}}}}`, turn)
	if err != nil {
		t.Fatal(err)
	}
	var out planning.AgentReply
	json.Unmarshal([]byte(content), &out)
	if len(out.Plan.Plans) != 2 || out.Plan.Plans[0].File != "ui.ts" || out.Plan.Plans[1].File != "store.go" || out.Plan.Plans[1].EvidenceIDs[0] != "Store-proof" {
		t.Fatal(out)
	}
	for _, raw := range []string{
		`{"1":{"evidence_indices":[0]},"1":{"evidence_indices":[0]}}`,
		`{"99":{"evidence_indices":[0]}}`, `{"01":{"evidence_indices":[0]}}`,
		`{"1":null}`, `{"1":{"target_index":0,"evidence_indices":[0]}}`,
		`{"1":{"file":"ui.ts","evidence_indices":[0]}}`, `{"1":{"evidence_indices":[1]}}`,
	} {
		if _, err := restoreTargetReferences(`{"plan":{"plans":`+raw+`}}`, turn); err == nil {
			t.Fatal("invalid keyed plan accepted", raw)
		}
	}
}

func TestSearchAttemptDoesNotForceFailedQueryLoop(t *testing.T) {
	turn := targetFixture()
	turn.Bundle.Changes[0].ElementName = strings.Repeat("word ", 20)
	query := agentInspectionQuery(turn, 0)
	if len(strings.Fields(query)) > 12 || len(query) > 200 {
		t.Fatal(query)
	}
	turn.Observations = []planning.ToolObservation{{Request: planning.ToolRequest{Name: "search_source", Query: query}, Result: planning.ToolResult{Error: "rejected"}}}
	if !agentSearchedRepository(turn) {
		t.Fatal("failed search forced repeated identical lookup")
	}
	if file, _ := unreadAgentSource(turn); file != "ui.ts" {
		t.Fatal("failed search counted as source inspection", file)
	}
}

func TestInitialSearchFitsServiceLimitsAndFailureFallsThroughToInspection(t *testing.T) {
	turn := targetFixture()
	turn.RemainingTools = 8
	turn.Bundle.Changes[0].ElementName = strings.Repeat("behavior ", 20)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			var body struct{ Format map[string]any }
			json.NewDecoder(r.Body).Decode(&body)
			tool := body.Format["properties"].(map[string]any)["tool"].(map[string]any)
			props := tool["properties"].(map[string]any)
			name := props["name"].(map[string]any)["enum"].([]any)[0].(string)
			query := props["query"].(map[string]any)["enum"].([]any)[0].(string)
			if len(strings.Fields(query)) > 12 || len(query) > 200 {
				t.Error("search tool limits not honored", query)
			}
			if calls == 0 && name != "search_source" || calls == 1 && name != "inspect_source" {
				t.Error("source inspection starved", name)
			}
			request := map[string]any{"name": name, "query": query, "change_index": 0}
			if name == "inspect_source" {
				request["file"] = "ui.ts"
			}
			content, _ := json.Marshal(map[string]any{"tool": request})
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "message": map[string]string{"role": "assistant", "content": string(content)}})
			calls++
		}
	}))
	defer srv.Close()
	m, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", Client: srv.Client()})
	if err := m.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	reply, err := m.Turn(context.Background(), turn)
	if err != nil || reply.Tool == nil {
		t.Fatal(reply, err)
	}
	turn.Observations = []planning.ToolObservation{{Request: *reply.Tool, Result: planning.ToolResult{Error: "lookup rejected"}}}
	if _, err = m.Turn(context.Background(), turn); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

type noSourceTools struct{}

func (noSourceTools) Identity() string { return "immutable-fixture" }
func (noSourceTools) Execute(context.Context, planning.ToolRequest) (planning.ToolResult, error) {
	panic("no source lookup permitted by this zero-tool fixture")
}

func TestKeyedPlannerCompletesRealThreeRoleEngineWithExactTargets(t *testing.T) {
	turn := targetFixture()
	// File-level and function-level targets are distinct and must not overwrite
	// each other even when they refer to the same file.
	turn.Bundle.Changes[0].Candidates[0].File = "store.go"
	turn.Bundle.Changes[0].Candidates[0].Symbol = ""
	turn.Bundle.Version = planning.BundleVersion
	turn.Bundle.AnalysisID = "fixture-analysis"
	turn.Bundle.InputFingerprint = "fixture-input"
	turn.Bundle.ContextID, _ = planning.ComputeContextID(turn.Bundle)
	roles := []planning.AgentRole{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			var body struct {
				Messages []struct{ Content string }
				Format   map[string]any
			}
			json.NewDecoder(r.Body).Decode(&body)
			var input planning.AgentTurn
			json.Unmarshal([]byte(body.Messages[1].Content), &input)
			roles = append(roles, input.Role)
			var action any
			switch input.Role {
			case planning.Analyst:
				decisions := []map[string]any{}
				for i := range turnTargets(input) {
					decisions = append(decisions, map[string]any{"target_index": i, "evidence_indices": []int{0}, "verdict": "RELEVANT", "rationale": "Existing submit entry point carries the affected behavior."})
				}
				action = map[string]any{"verification": map[string]any{"decisions": decisions}}
			case planning.Planner:
				plans := map[string]any{}
				for i := range turnTargets(input) {
					plans[strconv.Itoa(i)] = map[string]any{"evidence_indices": []int{0}, "summary": "Enforce the new submit requirement", "current_behavior": "Submit accepts the existing request.", "target_behavior": "Reject invalid submit input before mutation.", "design_rationale": "Use the observed entry point.", "steps": []string{"Add an action-specific input guard before mutation."}, "proposed_changes": []string{"Reject invalid submit input while retaining other actions."}, "acceptance_checks": []string{"Invalid input leaves the original state unchanged."}}
				}
				action = map[string]any{"plan": map[string]any{"plans": plans}}
			case planning.Reviewer:
				action = reviewAuditFixture(input, "PASS")
			}
			content, _ := json.Marshal(action)
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "message": map[string]string{"role": "assistant", "content": string(content)}})
		}
	}))
	defer srv.Close()
	model, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", Client: srv.Client()})
	if err := model.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	engine := planning.AgentEngine{Model: model, Tools: noSourceTools{}, Store: filestore.AgentSessionStore{Root: t.TempDir()}, Budget: planning.AgentBudget{MaxTurns: 6, MaxRevisions: 1}}
	result, err := engine.Run(context.Background(), "keyed-fixture", turn.Bundle)
	if err != nil || result.Status != "COMPLETE" || len(result.Draft) != 2 {
		t.Fatal(result.Status, result.Feedback, err)
	}
	if len(roles) != 3 || roles[0] != planning.Analyst || roles[1] != planning.Planner || roles[2] != planning.Reviewer {
		t.Fatal(roles)
	}
	if result.Draft[0].Symbol != "" || result.Draft[1].Symbol != "Store.Submit" {
		t.Fatal("targets were merged", result.Draft)
	}
}

func TestFlowSearchUsesEndpointBehaviorInsteadOfGeneratedEdgeID(t *testing.T) {
	turn := targetFixture()
	turn.Bundle.Changes[0].ElementID = "generated-edge-123"
	turn.Bundle.Changes[0].ElementName = ""
	turn.Bundle.ProcessContext = &domain.ProcessContext{After: &domain.Graph{Elements: map[string]*domain.Element{"a": {Name: "Validate adjustment reason"}, "b": {Name: "Submit stock adjustment"}}, Flows: map[string]*domain.Flow{"generated-edge-123": {SourceRef: "a", TargetRef: "b"}}}}
	query := agentInspectionQuery(turn, 0)
	if !strings.Contains(query, "reason") || !strings.Contains(query, "Submit") || strings.Contains(query, "generated-edge") {
		t.Fatal(query)
	}
}

func TestVerificationVerdictsAreGrammarConstrainedToEngineValues(t *testing.T) {
	turn := targetFixture()
	schema := map[string]any{"properties": map[string]any{"verification": map[string]any{"properties": map[string]any{"decisions": map[string]any{}}}}}
	indexedTargetSchema(schema, "verification", "decisions", turn, false)
	array := schema["properties"].(map[string]any)["verification"].(map[string]any)["properties"].(map[string]any)["decisions"].(map[string]any)
	verdict := array["properties"].(map[string]any)["0"].(map[string]any)["properties"].(map[string]any)["verdict"].(map[string]any)
	got := verdict["enum"].([]string)
	if strings.Join(got, ",") != "RELEVANT,WEAK,REJECT" {
		t.Fatal("invalid verdicts still grammar-valid", verdict)
	}
}

func TestPlannerContextPrunesIneligibleEvidenceWithoutRenumberingTargets(t *testing.T) {
	turn := targetFixture()
	turn.Role = planning.Planner
	target := turnTargets(turn)[1]
	turn.Decisions = []planning.CandidateDecision{{ChangeType: target.change.ChangeType, ElementID: target.change.ElementID, File: target.candidate.File, Symbol: target.candidate.Symbol, Verdict: planning.VerdictRelevant}}
	pruned := turn
	pruned.Bundle = plannerEvidenceBundle(turn.Bundle, turn.Decisions)
	if len(pruned.Bundle.Changes[0].Candidates) != 1 || len(turn.Bundle.Changes[0].Candidates) != 2 {
		t.Fatal("planner pruning mutated source")
	}
	raw, _ := json.Marshal(pruned)
	wire, err := annotateAgentInput(raw, turn)
	if err != nil || !strings.Contains(string(wire), `"target_index":1`) || strings.Contains(string(wire), `"target_index":0`) {
		t.Fatal("eligible target renumbered", string(wire), err)
	}
	if !strings.Contains(string(wire), "Store-proof") || strings.Contains(string(wire), "UI-proof") {
		t.Fatal("wrong source evidence retained")
	}
}

func TestNoChangeDispositionIsHostBoundAndDoesNotRenameTarget(t *testing.T) {
	turn := targetFixture()
	turn.Role = planning.Planner
	content, err := restoreTargetReferences(`{"plan":{"plans":{"1":{"disposition":"NO_CHANGE","evidence_indices":[0],"summary":"Existing contract is adequate","current_behavior":"Forwards the existing field","target_behavior":"Preserve the field for validation","design_rationale":"This file already fulfills the forwarding contract","steps":[],"proposed_changes":[],"acceptance_checks":["Existing field is forwarded"]}}}}`, turn)
	var reply planning.AgentReply
	if err != nil || json.Unmarshal([]byte(content), &reply) != nil {
		t.Fatal(content, err)
	}
	p := reply.Plan.Plans[0]
	if p.File != "store.go" || p.Action != domain.ActionNoAction || p.Symbol != "Store.Submit" || len(p.ProposedChanges) != 0 {
		t.Fatal(p)
	}
	if _, err := restoreTargetReferences(`{"plan":{"plans":{"1":{"disposition":"NO_CHANGE","action":"VERIFY","evidence_indices":[0]}}}}`, turn); err == nil {
		t.Fatal("contradictory no-change action accepted")
	}
}

func TestAnalystSlotsConstrainEvidencePerTarget(t *testing.T) {
	turn := targetFixture()
	for i := 1; i < 8; i++ {
		turn.Bundle.Changes[0].Candidates[0].Evidence = append(turn.Bundle.Changes[0].Candidates[0].Evidence, planning.EvidenceItem{ID: strconv.Itoa(i)})
	}
	for i := 1; i < 7; i++ {
		turn.Bundle.Changes[0].Candidates[1].Evidence = append(turn.Bundle.Changes[0].Candidates[1].Evidence, planning.EvidenceItem{ID: "store-" + strconv.Itoa(i)})
	}
	schema, _ := agentActionSchema(planning.Analyst)
	indexedTargetSchema(schema, "verification", "decisions", turn, false)
	var check func(map[string]any)
	check = func(s map[string]any) {
		if choices, ok := s["oneOf"].([]any); ok {
			for _, v := range choices {
				check(v.(map[string]any))
			}
			return
		}
		props := s["properties"].(map[string]any)
		v, ok := props["verification"].(map[string]any)
		if !ok {
			return
		}
		slots := v["properties"].(map[string]any)["decisions"].(map[string]any)["properties"].(map[string]any)
		for key, want := range map[string]int{"0": 7, "1": 6} {
			max := slots[key].(map[string]any)["properties"].(map[string]any)["evidence_indices"].(map[string]any)["items"].(map[string]any)["maximum"]
			if max != want {
				t.Fatal("wrong target-specific bound", key, max, want)
			}
		}
	}
	check(schema)
	good := `{"verification":{"decisions":{"0":{"verdict":"RELEVANT","rationale":"Current handler","evidence_indices":[7]},"1":{"verdict":"WEAK","rationale":"Uncertain path","evidence_indices":[6]}}}}`
	if _, err := restoreTargetReferences(good, turn); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(good, `"evidence_indices":[6]`, `"evidence_indices":[7]`, 1)
	if _, err := restoreTargetReferences(bad, turn); err == nil {
		t.Fatal("out-of-target proof accepted")
	}
}
