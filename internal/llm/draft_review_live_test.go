//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// A bounded fixed-target Planner/Reviewer trial, not a fresh Analyst scan or
// complete saved-analysis benchmark. Source and decisions come from an explicit
// frozen checkpoint; no reference answer is sent and no repository is executed.
func TestLiveReasoningDraftReview(t *testing.T) {
	if os.Getenv("BPMN_DRAFT_REVIEW_PROBE") != "1" {
		t.Skip("explicit fixed-target live trial required")
	}
	modelName := os.Getenv("BPMN_PROBE_MODEL")
	if modelName == "" {
		t.Fatal("explicit installed local model required")
	}
	data, err := os.ReadFile(os.Getenv("BPMN_PROBE_ANALYST_SESSION"))
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint struct{ Session planning.AgentSession }
	if err := json.Unmarshal(data, &checkpoint); err != nil || checkpoint.Session.Role != planning.Planner {
		t.Fatal("validated frozen Analyst checkpoint required", err)
	}
	s := checkpoint.Session
	bundle := plannerEvidenceBundle(s.Bundle, s.Decisions)
	var decisions []planning.CandidateDecision
	for _, d := range s.Decisions {
		if d.Verdict == planning.VerdictRelevant {
			decisions = append(decisions, d)
		}
	}
	if len(bundle.Changes) != 1 || len(decisions) != 3 || len(bundle.Changes[0].Candidates) != 3 {
		t.Fatal("this bounded trial requires the frozen three-target change")
	}
	bundle.ContextID, err = planning.ComputeContextID(bundle)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", ".data", "model-probes", "draft-review-"+time.Now().UTC().Format("20060102T150405"))
	if err = os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("Private fixed-target trial artifacts:", root)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://127.0.0.1:11435/api/ps")
	if err != nil {
		t.Fatal(err)
	}
	var running struct{ Models []struct{ Name string } }
	err = json.NewDecoder(response.Body).Decode(&running)
	response.Body.Close()
	if err != nil || len(running.Models) > 0 {
		t.Fatal("idle lab runtime required", err)
	}
	var metrics []InferenceMetric
	config := OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: modelName, ContextSize: 32768, MaxOutput: 6144, Profile: "coding", ReviewJSONMode: true, OnInference: func(m InferenceMetric) {
		metrics = append(metrics, m)
		t.Logf("role=%s thinking=%t outcome=%s seconds=%.2f", m.Role, m.Think, m.Outcome, float64(m.Milliseconds)/1000)
	}}
	think := true
	switch os.Getenv("BPMN_DRAFT_REVIEW_THINK") {
	case "", "on":
	case "off":
		think = false
	default:
		t.Fatal("review thinking must be on or off")
	}
	reviewerName := os.Getenv("BPMN_DRAFT_REVIEW_REVIEWER_MODEL")
	if reviewerName == "" {
		reviewerName = modelName
	}
	model, err := NewOllamaRoleRouter(config, map[planning.AgentRole]OllamaRoleOverride{planning.Reviewer: {Model: reviewerName, Think: &think}})
	if err != nil {
		t.Fatal(err)
	}
	minutes := 6
	if value := os.Getenv("BPMN_DRAFT_REVIEW_MINUTES"); value != "" {
		minutes, err = strconv.Atoi(value)
		if err != nil || minutes < 1 || minutes > 12 {
			t.Fatal("explicit diagnostic duration must be 1-12 minutes")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(minutes)*time.Minute)
	defer cancel()
	if err := model.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	defer releaseProbeModel(t, modelName)
	if reviewerName != modelName {
		defer releaseProbeModel(t, reviewerName)
	}
	started := time.Now()
	write := func(name string, value any) {
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), append(encoded, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if repairPath := os.Getenv("BPMN_DRAFT_REVIEW_REPAIR_SESSION"); repairPath != "" {
		// Replay the failed file as a NEW diagnostic with its real repair
		// feedback. The old session/counters/checkpoints remain untouched.
		raw, err := os.ReadFile(repairPath)
		if err != nil {
			t.Fatal(err)
		}
		var prior planning.AgentSession
		if err := json.Unmarshal(raw, &prior); err != nil || prior.Bundle.ContextID != bundle.ContextID || len(prior.Feedback) == 0 {
			t.Fatal("matching failed review and feedback required", err)
		}
		file := os.Getenv("BPMN_PROBE_FILE")
		if file == "" {
			t.Fatal("explicit failed file required")
		}
		turn := planning.AgentTurn{Role: planning.Reviewer, Bundle: prior.Bundle, Feedback: append([]string{"FINAL FILE REVIEW: review the proposed changes for this file against its approved source and BPMN requirements."}, prior.Feedback...)}
		for _, p := range prior.Draft {
			if p.File == file {
				turn.Draft = append(turn.Draft, p)
			}
		}
		if len(turn.Draft) != 1 {
			t.Fatal("one matching failed proposal required")
		}
		turn.Bundle.Changes = append([]planning.ChangeInput(nil), prior.Bundle.Changes...)
		for i, ch := range turn.Bundle.Changes {
			turn.Bundle.Changes[i].Candidates = nil
			for _, c := range ch.Candidates {
				if c.File == file {
					turn.Bundle.Changes[i].Candidates = append(turn.Bundle.Changes[i].Candidates, c)
				}
			}
		}
		turn.Bundle.ContextID, _ = planning.ComputeContextID(turn.Bundle)
		// Restore the warm-weight condition of the original retry. This
		// explicit load request has no prompt/source and is reported separately.
		prepared := model.models[planning.Reviewer].(*OllamaAgent)
		var loaded any
		loadStarted := time.Now()
		if err := prepared.request(ctx, http.MethodPost, "/api/generate", map[string]any{"model": reviewerName, "keep_alive": "2m", "stream": false}, &loaded); err != nil {
			t.Fatal(err)
		}
		loadSeconds := time.Since(loadStarted).Seconds()
		reply, callErr := model.Turn(ctx, turn)
		write("review.json", reply)
		write("inferences.json", metrics)
		summary := map[string]any{"scope": "one failed file with real output-repair feedback; fresh diagnostic; not a complete plan", "identity": model.Identity(), "role_identities": model.RoleIdentities(), "warmup_seconds": loadSeconds, "seconds": time.Since(started).Seconds(), "prior_session": prior.ID, "approved": reply.Review != nil && reply.Review.Approved}
		if callErr != nil {
			summary["error"] = callErr.Error()
		}
		write("summary.json", summary)
		t.Logf("repair file approved=%t error=%v", reply.Review != nil && reply.Review.Approved, callErr)
		return
	}
	engine := planning.AgentEngine{Model: model, Tools: noSourceTools{}, Store: filestore.AgentSessionStore{Root: filepath.Join(root, "sessions")}, InitialDecisions: decisions, Budget: planning.AgentBudget{MaxTurns: 12, MaxTools: 0, MaxRevisions: 1, MaxDiscoveries: 0}}
	id := "fixed-reason-input"
	resumeFrom := os.Getenv("BPMN_DRAFT_REVIEW_RESUME_DIR")
	warmupSeconds := 0.0
	if resumeFrom != "" {
		// Work on a copy: previous experiment bytes/counters stay available for
		// comparison. AgentEngine still enforces source/model/budget bindings.
		priorStore := filestore.AgentSessionStore{Root: filepath.Join(resumeFrom, "sessions")}
		prior, err := priorStore.Load(ctx, id)
		if err != nil || prior.Bundle.ContextID != bundle.ContextID || prior.Identity != model.Identity() || prior.Status != "PAUSED" {
			t.Fatal("matching paused checkpoint required; no identity/budget reset allowed", err)
		}
		entries, err := os.ReadDir(priorStore.Root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".json" {
				childID := strings.TrimSuffix(entry.Name(), ".json")
				child, err := priorStore.Load(ctx, childID)
				if err != nil {
					t.Fatal(err)
				}
				if err := engine.Store.Save(ctx, child); err != nil {
					t.Fatal(err)
				}
			}
		}
		// The original interruption occurred with resident weights. Restore
		// that condition explicitly; this load sends no prompt or source.
		prepared := model.models[prior.Role].(*OllamaAgent)
		var loaded any
		loadStarted := time.Now()
		if err := prepared.request(ctx, http.MethodPost, "/api/generate", map[string]any{"model": prepared.model, "keep_alive": "2m", "stream": false}, &loaded); err != nil {
			t.Fatal(err)
		}
		warmupSeconds = time.Since(loadStarted).Seconds()
		t.Logf("Resuming role=%s consumed_turns=%d repairs=%d; previous artifacts preserved", prior.Role, prior.Turns, prior.OutputRepairs)
	}
	result, runErr := engine.Run(ctx, id, bundle)
	write("session.json", result)
	write("inferences.json", metrics)
	summary := map[string]any{"scope": "fixed-target Planner and Reviewer only; captured Analyst decisions; not full scenario accuracy", "status": result.Status, "seconds": time.Since(started).Seconds(), "diagnostic_minutes": minutes, "identity": model.Identity(), "role_identities": model.RoleIdentities(), "turns": result.Turns, "revisions": result.Revisions}
	if resumeFrom != "" {
		summary["resumed_from"] = resumeFrom
		summary["warmup_seconds"] = warmupSeconds
	}
	if runErr != nil {
		summary["error"] = runErr.Error()
	}
	if plan, err := result.ChangePlan(); err == nil {
		write("plan.json", plan)
		if err := os.WriteFile(filepath.Join(root, "plan.md"), planning.Markdown(plan), 0600); err != nil {
			t.Fatal(err)
		}
		summary["exported"] = true
	} else {
		summary["exported"] = false
	}
	write("summary.json", summary)
	t.Logf("status=%s turns=%d revisions=%d run_error=%v", result.Status, result.Turns, result.Revisions, runErr)
}
