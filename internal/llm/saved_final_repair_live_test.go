//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Explicit, single-request Reviewer replay. Not an Analyst run or E2E benchmark.
func TestLiveSavedFinalFormatRepair(t *testing.T) {
	if os.Getenv("BPMN_SAVED_FINAL_REPAIR") != "1" {
		t.Skip("explicit local-lab replay required")
	}
	id := os.Getenv("BPMN_SAVED_FINAL_SESSION")
	modelName := os.Getenv("BPMN_PROBE_MODEL")
	if id == "" || modelName == "" {
		t.Fatal("explicit stored session and installed local model required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	root := filepath.Join("..", "..")
	s, err := (filestore.AgentSessionStore{Root: filepath.Join(root, ".data", "agent-sessions")}).Load(ctx, id)
	if err != nil || len(s.Draft) == 0 {
		t.Fatal("stored draft missing", err)
	}
	// Replay the same first-file source-backed check that failed in final review.
	draft := []planning.FilePlan{s.Draft[0]}
	b := s.Bundle
	b.Changes = append([]planning.ChangeInput(nil), s.Bundle.Changes...)
	for i, ch := range b.Changes {
		b.Changes[i].Candidates = nil
		for _, c := range ch.Candidates {
			if ch.ChangeType == draft[0].ChangeType && ch.ElementID == draft[0].ElementID && c.File == draft[0].File && c.Symbol == draft[0].Symbol {
				b.Changes[i].Candidates = append(b.Changes[i].Candidates, c)
			}
		}
	}
	b.ContextID, err = planning.ComputeContextID(b)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".data", "model-probes", "final-format-repair-"+time.Now().UTC().Format("20060102T150405"))
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("private diagnostic output", dir)
	var metrics []InferenceMetric
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: modelName, ContextSize: 32768, MaxOutput: 6144, Profile: "coding", Think: false, ReviewJSONMode: true, OnInference: func(v InferenceMetric) { metrics = append(metrics, v) }})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	turn := planning.AgentTurn{Role: planning.Reviewer, Bundle: b, Draft: draft, RelatedProposals: planning.BuildProposalContracts(s.Draft, draft), Feedback: []string{"FINAL FILE REVIEW: check the future proposal against supplied source and coordinated contracts. Do not require a new rule to exist before implementation.", "Model response rejected for reviewer: agent model output invalid or incomplete: duplicate review JSON field. Return every required criterion exactly once, with valid target evidence."}}
	started := time.Now()
	reply, runErr := m.Turn(ctx, turn)
	result := map[string]any{"scope": "one saved final file review; not end-to-end", "source_session": id, "provider": m.Identity(), "thinking": false, "seconds": time.Since(started).Seconds(), "metrics": metrics, "reply": reply, "validated": runErr == nil}
	if runErr != nil {
		result["error"] = runErr.Error()
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "result.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	if reply.Review == nil || len(reply.Review.Findings) != 1 {
		t.Fatal("review findings missing")
	}
	t.Logf("format validated; approval=%v; issues=%d; seconds=%.2f", reply.Review.Approved, len(reply.Review.Issues), time.Since(started).Seconds())
}
