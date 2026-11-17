package llm

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func reviewAuditFixture(t planning.AgentTurn, status string, compared ...bool) map[string]any {
	comparison := len(compared) > 0 && compared[0]
	checks := map[string]any{}
	for i, p := range t.Draft {
		refs := []int{}
		if c := reviewSource(t, p); c != nil && len(c.Evidence) > 0 {
			refs = []int{0}
		}
		item := map[string]any{"evidence_indices": refs}
		if comparison {
			item["comparison_evidence_index"] = nil
			if c := reviewSource(t, p); c != nil && len(c.Evidence) > 0 {
				item["comparison_evidence_index"] = 0
			}
			item["implementation_index"] = 0
		}
		for _, key := range reviewCheckNames(comparison) {
			item[key] = map[string]any{"finding": "Fixture conclusion, not a live model judgment.", "status": status}
		}
		checks[strconv.Itoa(i)] = item
	}
	return map[string]any{"review": map[string]any{"checks": checks}}
}

func reviewTurnFixture() planning.AgentTurn {
	t := targetFixture()
	t.Role = planning.Reviewer
	for _, c := range t.Bundle.Changes[0].Candidates {
		t.Draft = append(t.Draft, planning.FilePlan{ChangeType: t.Bundle.Changes[0].ChangeType, ElementID: t.Bundle.Changes[0].ElementID, File: c.File, Symbol: c.Symbol, Steps: []string{"Preserve the existing input while adding action-specific validation."}})
	}
	return t
}

func TestReviewRequiresAllChecksAndBindsEvidence(t *testing.T) {
	turn := reviewTurnFixture()
	for _, status := range []string{"PASS", "REVISE", "NEED_EVIDENCE"} {
		raw, _ := json.Marshal(reviewAuditFixture(turn, status))
		content, err := restoreReviewAudit(string(raw), turn)
		var reply planning.AgentReply
		if err != nil || json.Unmarshal([]byte(content), &reply) != nil {
			t.Fatal(content, err)
		}
		r := reply.Review
		if r.Approved != (status == "PASS") || len(r.Findings) != 2 || r.Findings[1].File != "store.go" || r.Findings[1].EvidenceIDs[0] != "Store-proof" {
			t.Fatal("unbound or unearned approval", r)
		}
		if status == "NEED_EVIDENCE" && r.RevisionTarget != planning.Analyst || status == "REVISE" && r.RevisionTarget != planning.Planner {
			t.Fatal("wrong revision route", r)
		}
	}
	good, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
	for _, bad := range []string{
		`{"review":{"approved":true,"issues":[]}}`,
		strings.Replace(string(good), `"contracts":`, `"unknown":`, 1),
		strings.Replace(string(good), `"evidence_indices":[0]`, `"evidence_indices":[null]`, 1),
		strings.Replace(string(good), `"evidence_indices":[0]`, `"evidence_indices":[999]`, 1),
		strings.Replace(string(good), `"evidence_indices":[0]`, `"evidence_indices":[]`, 1),
		strings.Replace(string(good), `"PASS"`, `"UNKNOWN"`, 1),
		strings.Replace(string(good), `"status":"PASS"`, `"status":"REVISE","status":"PASS"`, 1),
		string(good) + `{}`,
	} {
		if _, err := restoreReviewAudit(bad, turn); err == nil {
			t.Fatal("invalid review accepted", bad)
		}
	}
}

func TestProposalOnlyReviewCannotInventSource(t *testing.T) {
	turn := reviewTurnFixture()
	turn.Bundle.Changes[0].Candidates = nil
	raw, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
	content, err := restoreReviewAudit(string(raw), turn)
	if err != nil || !strings.Contains(content, `"basis":"proposal_only"`) {
		t.Fatal(content, err)
	}
	bad := strings.Replace(string(raw), `"evidence_indices":[]`, `"evidence_indices":[0]`, 1)
	if _, err := restoreReviewAudit(bad, turn); err == nil {
		t.Fatal("proposal-only review invented source")
	}
}

func TestReviewComparisonRequiresActualEditAndBindsBothExcerpts(t *testing.T) {
	turn := reviewTurnFixture()
	turn.Draft[0].TargetBehavior = "Desired outcome only."
	turn.Draft[0].AcceptanceChecks = []string{"Desired outcome succeeds."}
	raw, _ := json.Marshal(reviewAuditFixture(turn, "PASS", true))
	content, err := restoreReviewAudit(string(raw), turn, true)
	var reply planning.AgentReply
	if err != nil || json.Unmarshal([]byte(content), &reply) != nil {
		t.Fatal(err, content)
	}
	f := reply.Review.Findings[0]
	if f.SourceExcerpt != turn.Bundle.Changes[0].Candidates[0].Evidence[0].Excerpt || f.ImplementationExcerpt != turn.Draft[0].Steps[0] {
		t.Fatal("comparison excerpts not bound to actual inputs", f)
	}
	for _, bad := range []string{
		strings.Replace(string(raw), `"implementation_index":0`, `"implementation_index":999`, 1),
		strings.Replace(string(raw), `"implementation_index":0`, `"implementation_index":null`, 1),
		strings.Replace(string(raw), `"comparison_evidence_index":0`, `"comparison_evidence_index":999`, 1),
		strings.Replace(string(raw), `"comparison_evidence_index":0`, `"comparison_evidence_index":null`, 1),
		strings.Replace(string(raw), `"valid_path":`, `"unknown_path":`, 1),
	} {
		if _, err := restoreReviewAudit(bad, turn, true); err == nil {
			t.Fatal("invented comparison accepted", bad)
		}
	}
	turn.Draft[0].Steps = nil
	if _, err := restoreReviewAudit(string(raw), turn, true); err == nil {
		t.Fatal("desired outcomes substituted for implementation edits")
	}
	turn.Draft[0].Action = "NO_ACTION"
	turn.Draft[0].CurrentBehavior = "Existing contract already supports this responsibility."
	if _, err := restoreReviewAudit(string(raw), turn, true); err != nil {
		t.Fatal("no-action comparison unavailable", err)
	}
}

func TestComparisonAuditIsExplicitlyOptIn(t *testing.T) {
	cfg := OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "local-test"}
	baseline, err := NewOllamaAgent(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ReviewComparisons = true
	experiment, err := NewOllamaAgent(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(baseline.Identity().PromptVersion, "ollama-agent-json-v20-") || !strings.HasPrefix(experiment.Identity().PromptVersion, "ollama-agent-json-v21-") || baseline.Identity() == experiment.Identity() {
		t.Fatal("audit policy identity not isolated")
	}
	turn := reviewTurnFixture()
	raw, _ := json.Marshal(reviewAuditFixture(turn, "PASS", true))
	if _, err := restoreReviewAudit(string(raw), turn); err == nil {
		t.Fatal("experimental audit accepted by baseline policy")
	}
	raw, _ = json.Marshal(reviewAuditFixture(turn, "PASS"))
	if _, err := restoreReviewAudit(string(raw), turn, true); err == nil {
		t.Fatal("missing experimental proof accepted")
	}
}
