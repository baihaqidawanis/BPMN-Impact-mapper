package llm

import (
	"context"
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCombinedScenarioFullProcessFitsScanTransport(t *testing.T) {
	base := filepath.Join("..", "..", "experiments", "simple-erp", "current-staff-v2", "combined")
	a, err := os.ReadFile(filepath.Join(base, "before.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(base, "after.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := bpmn.Parse(a, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := bpmn.Parse(b, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	bundle := planning.BuildEvidenceBundle(domain.Report{AnalysisID: strings.Repeat("a", 32), InputFingerprint: strings.Repeat("b", 32), Changes: diff.Compare(before, after), ProcessContext: &domain.ProcessContext{Before: before, After: after}}, planning.DefaultLimits())
	bundle.RepositoryOverview = &planning.RepositoryOverview{File: "README.md", Summary: strings.Repeat("x", 1600)}
	for i := range bundle.Changes {
		bundle.Changes[i].Candidates = []planning.Candidate{{File: "backend/internal/inventory/store.go", Module: "backend", Layer: "adapter", SuggestedAction: domain.ActionVerify, Evidence: []planning.EvidenceItem{{ID: "chunk-" + bundle.Changes[i].ElementID, Type: "FULL_SOURCE_CHUNK", Location: "backend/internal/inventory/store.go@bytes:0-4096", Excerpt: strings.Repeat("x", 240)}}}}
	}
	sent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			sent = true
			var body struct{ Messages []struct{ Content string } }
			json.NewDecoder(r.Body).Decode(&body)
			var turn planning.AgentTurn
			json.Unmarshal([]byte(body.Messages[1].Content), &turn)
			if len(turn.Bundle.Changes) != len(bundle.Changes) || len(turn.Bundle.ProcessContext.After.Elements) != len(after.Elements) || len(turn.Bundle.ProcessContext.After.Flows) != len(after.Flows) {
				t.Error("process requirements omitted")
			}
			items := []map[string]any{}
			for i := range turn.Bundle.Changes {
				items = append(items, map[string]any{"change_index": i, "verdict": "WEAK", "rationale": "Offline fixture"})
			}
			content, _ := json.Marshal(map[string]any{"verification": map[string]any{"decisions": items}})
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]string{"role": "assistant", "content": string(content)}})
		}
	}))
	defer srv.Close()
	model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, Client: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = model.Turn(context.Background(), planning.AgentTurn{Role: planning.Analyst, Bundle: bundle, RepositoryChunk: &planning.RepositoryChunk{HybridLeads: []string{strings.Repeat("hint", 500)}, HybridRank: 1, File: "backend/internal/inventory/store.go", StartByte: 0, EndByte: 4096, FileBytes: 4096, Content: strings.Repeat("x", 4096)}})
	if err != nil || !sent {
		t.Fatal("combined scenario cannot reach model adapter", err)
	}
	if before.Incoming == nil || after.Outgoing == nil {
		t.Fatal("transport modified original graphs")
	}
}

func TestRoutingGuidanceFollowsParsedChanges(t *testing.T) {
	base := filepath.Join("..", "..", "experiments", "simple-erp")
	for _, tc := range []struct {
		dir      string
		element  string
		expanded bool
		facts    []string
	}{
		{"poc-patterns/skip", "CloseWithoutAdjustment", true, []string{"semua selisih nol", "ada baris dengan selisih nonnol", "NO_ADJUSTMENT"}},
		{"poc-patterns/redo", "BackToReview", true, []string{"REVISION_REQUIRED", "Kirim respons setelah commit", "ID yang sama", "ReviewEntryJoin"}},
		{"poc-patterns/switch", "SwitchToRejection", true, []string{"semua produk PR aktif", "ada produk PR yang nonaktif", "manual tetap memakai alasan reviewer"}},
		{"poc-patterns/non-free-choice", "LoadLockedApprovalRoute", true, []string{"LockApprovalRoute", "Pada request approve maupun reject", "jalur tersimpan = STANDARD", "jalur tersimpan = ESCALATED", "sebelum mutasi"}},
		{"current-staff-v2/add-task", "ReasonReview", false, nil},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			read := func(version string) *domain.Graph {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(base, tc.dir, version+".bpmn"))
				if err != nil {
					t.Fatal(err)
				}
				graph, err := bpmn.Parse(data, bpmn.Limits{})
				if err != nil {
					t.Fatal(err)
				}
				return graph
			}
			pc := &domain.ProcessContext{Before: read("before"), After: read("after")}
			report := domain.Report{Changes: diff.Compare(pc.Before, pc.After), ProcessContext: pc}
			bundle := planning.BuildAgentBundle(report, planning.DefaultLimits())
			for _, change := range bundle.Changes {
				if change.ElementID != tc.element {
					continue
				}
				bundle.Changes = []planning.ChangeInput{change}
				for role, want := range map[planning.AgentRole]string{
					planning.Analyst: routingAnalystInstruction, planning.Planner: routingPlannerInstruction,
					planning.Reviewer: routingReviewerInstruction,
				} {
					got := expandedDecisionInstruction(planning.AgentTurn{Role: role, Bundle: bundle})
					if tc.expanded && got != want || !tc.expanded && got != "" {
						t.Fatalf("wrong guidance for %s: %q", role, got)
					}
				}
				bundle.ProcessContext = &domain.ProcessContext{Before: compactProcessGraph(pc.Before), After: compactProcessGraph(pc.After)}
				input, err := json.Marshal(planning.AgentTurn{Role: planning.Planner, Bundle: bundle})
				if err != nil {
					t.Fatal(err)
				}
				wire, err := sharedProcessInput(input, bundle.ProcessContext)
				if err != nil {
					t.Fatal(err)
				}
				for _, fact := range tc.facts {
					if !strings.Contains(string(wire), fact) {
						t.Errorf("single-change review lost process requirement %q", fact)
					}
				}
				return
			}
			t.Fatalf("expected change %s missing", tc.element)
		})
	}
}
