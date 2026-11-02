package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type syntheticTrace struct {
	*OllamaAgent
	path string
}

func (m syntheticTrace) Turn(ctx context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	reply, err := m.OllamaAgent.Turn(ctx, turn)
	data, _ := json.Marshal(struct {
		Role   planning.AgentRole
		Reply  planning.AgentReply
		Failed bool
		Error  string
	}{turn.Role, reply, err != nil, func() string {
		if err != nil {
			return err.Error()
		}
		return ""
	}()})
	f, e := os.OpenFile(m.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e == nil {
		_, _ = f.Write(append(data, '\n'))
		_ = f.Close()
	}
	return reply, err
}

// Explicit opt-in: sends only this small synthetic fixture through the lab
// tunnel, never source ERP, ground truth or full repository contents.
func TestOllamaAgentLiveSynthetic(t *testing.T) {
	if os.Getenv("BPMN_IMPACT_LAB_SMOKE") != "1" {
		t.Skip("lab model test requires explicit opt-in")
	}
	root := filepath.Join("..", "..", "outputs", "lab-agent-smoke-"+time.Now().UTC().Format("20060102T150405"))
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	source := []byte("package fixture\nfunc Submit(amount int) int { return amount }\n")
	if err := os.WriteFile(filepath.Join(repo, "submit.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	tools, err := agenttools.New(ctx, repo, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "qwen3.5:9b-q8_0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	b := planning.EvidenceBundle{Version: planning.BundleVersion, AnalysisID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", InputFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Changes: []planning.ChangeInput{{ChangeType: domain.AddTask, ElementID: "validate-positive", ElementName: "Validate positive amount before submit", Reason: "A new validation task must reject zero and negative amounts before submitting.", Candidates: []planning.Candidate{{File: "submit.go", Symbol: "Submit", Module: "core", Layer: "application", SuggestedAction: domain.ActionVerify, Evidence: []planning.EvidenceItem{{ID: "SYNTHETIC-E1", Type: "SOURCE", Location: "submit.go:2", Excerpt: "func Submit(amount int) int { return amount }"}}}}}}}
	b.ContextID, _ = planning.ComputeContextID(b)
	data, _ := json.MarshalIndent(b, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "context.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	engine := planning.AgentEngine{Model: syntheticTrace{model, filepath.Join(root, "synthetic-actions.jsonl")}, Tools: tools, Store: filestore.AgentSessionStore{Root: filepath.Join(root, "sessions")}}
	s, err := engine.Run(ctx, "live-synthetic", b)
	t.Logf("Artifact folder %s; status=%s turns=%d tools=%d revisions=%d", root, s.Status, s.Turns, s.Tools, s.Revisions)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[planning.AgentRole]bool{}
	for _, e := range s.Events {
		if e.Kind == "TURN_STARTED" {
			seen[e.Role] = true
		}
	}
	if !seen[planning.Analyst] || !seen[planning.Planner] || !seen[planning.Reviewer] {
		t.Fatal("not all roles reached")
	}
	plan, err := s.ChangePlan()
	if err != nil {
		t.Fatal(err)
	}
	data, err = planning.JSON(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "change-plan.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "change-plan.md"), planning.Markdown(plan), 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(repo, "submit.go")); err != nil || string(data) != string(source) {
		t.Fatal("synthetic source modified")
	}
	if s.Status != "COMPLETE" && s.Status != "NEEDS_REVIEW" {
		t.Fatal("session not final")
	}
}
