package planning

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type scriptedAgent struct {
	fn func(AgentTurn) (AgentReply, error)
}

func (m scriptedAgent) Identity() ProviderIdentity {
	return ProviderIdentity{"scripted", "fixture", "v1"}
}
func (m scriptedAgent) Turn(_ context.Context, t AgentTurn) (AgentReply, error) { return m.fn(t) }

type memorySessions map[string]AgentSession

func (m memorySessions) Load(_ context.Context, id string) (AgentSession, error) {
	s, ok := m[id]
	if !ok {
		return s, ErrAgentSessionMissing
	}
	data, _ := json.Marshal(s)
	var out AgentSession
	_ = json.Unmarshal(data, &out)
	return out, nil
}
func (m memorySessions) Save(_ context.Context, s AgentSession) error {
	data, _ := json.Marshal(s)
	var out AgentSession
	_ = json.Unmarshal(data, &out)
	m[s.ID] = out
	return nil
}

type fixtureTools struct{ id string }

func (t fixtureTools) Identity() string { return t.id }
func (t fixtureTools) Execute(context.Context, ToolRequest) (ToolResult, error) {
	return ToolResult{Candidates: []Candidate{{File: "ui.ts", Symbol: "SubmitForm", Module: "ui", Layer: "frontend", SuggestedAction: domain.ActionVerify, Origin: "agent-source-tool", Evidence: []EvidenceItem{{ID: "ui-proof", Type: "SOURCE_TOOL", Location: "ui.ts:1", Excerpt: "submit request"}}}}}, nil
}

func agentBundle() EvidenceBundle {
	b := EvidenceBundle{Version: BundleVersion, AnalysisID: "analysis", InputFingerprint: "fingerprint", Changes: []ChangeInput{{ChangeType: domain.AddTask, ElementID: "task", Candidates: []Candidate{{File: "store.go", Symbol: "Submit", Module: "core", Layer: "application", SuggestedAction: domain.ActionVerify, Evidence: []EvidenceItem{{ID: "proof", Type: "TEXT", Location: "store.go:1", Excerpt: "submit"}}}}}}}
	b.ContextID, _ = ComputeContextID(b)
	return b
}

func goodReply(t AgentTurn) (AgentReply, error) {
	switch t.Role {
	case Analyst:
		out := VerificationResponse{}
		for _, change := range t.Bundle.Changes {
			for _, c := range change.Candidates {
				out.Decisions = append(out.Decisions, CandidateDecision{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Verdict: VerdictRelevant, Rationale: "Existing submit behavior is relevant; confirm requirement.", EvidenceIDs: []string{c.Evidence[0].ID}})
			}
		}
		return AgentReply{Verification: &out}, nil
	case Planner:
		out := LayerPlanningResponse{}
		for _, change := range t.Bundle.Changes {
			for _, c := range change.Candidates {
				out.Plans = append(out.Plans, FilePlan{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: domain.ActionVerify, Summary: "Review submit requirement", ProposedChanges: []string{"Require the new validation before submission while preserving unrelated actions."}, AcceptanceChecks: []string{"Invalid input is rejected before state mutation; valid input preserves the existing transition."}, Steps: []string{"Verify validation before status mutation."}, EvidenceIDs: []string{c.Evidence[0].ID}})
			}
		}
		return AgentReply{Plan: &out}, nil
	default:
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}
}

func TestAgentsDiscoveryRevisionAndCanonicalPlan(t *testing.T) {
	store := memorySessions{}
	calls, reviews := 0, 0
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if calls == 1 {
			return AgentReply{Tool: &ToolRequest{Name: "search_source", Query: "submit", ChangeIndex: 0}}, nil
		}
		if turn.Role == Reviewer {
			reviews++
			if reviews == 1 {
				return AgentReply{Review: &AgentReview{Issues: []string{"Explain failure behavior."}, RevisionTarget: Planner}}, nil
			}
		}
		if turn.Role == Planner && reviews == 1 && len(turn.Feedback) == 0 {
			t.Fatal("revision feedback missing")
		}
		return goodReply(turn)
	}}
	original := agentBundle()
	engine := AgentEngine{Model: m, Tools: fixtureTools{"snapshot1"}, Store: store}
	s, err := engine.Run(context.Background(), "demo", original)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != "COMPLETE" || s.Tools != 1 || s.Revisions != 1 || len(s.Draft) != 2 || len(original.Changes[0].Candidates) != 1 {
		t.Fatalf("unexpected result: %+v", s)
	}
	p, err := s.ChangePlan()
	if err != nil || p.PlanID == "" || len(Markdown(p)) == 0 {
		t.Fatalf("plan export: %v", err)
	}
	before := calls
	if _, err := engine.Run(context.Background(), "demo", original); err != nil || calls != before {
		t.Fatal("completed session reran model")
	}
	engine.Tools = fixtureTools{"changed-snapshot"}
	if _, err := engine.Run(context.Background(), "demo", original); err == nil {
		t.Fatal("accepted source drift")
	}
}

func TestAgentsResumeAfterModelFailure(t *testing.T) {
	store := memorySessions{}
	first := true
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if first {
			first = false
			return AgentReply{}, errors.New("connection failed with sensitive detail")
		}
		return goodReply(turn)
	}}
	e := AgentEngine{Model: m, Tools: fixtureTools{"snapshot"}, Store: store}
	s, err := e.Run(context.Background(), "resume", agentBundle())
	if err == nil || s.Status != "PAUSED" || s.Turns != 1 {
		t.Fatal("failure not checkpointed")
	}
	s, err = e.Run(context.Background(), "resume", agentBundle())
	if err != nil || s.Status != "COMPLETE" || s.Turns != 4 {
		t.Fatalf("resume failed: %+v %v", s, err)
	}
}

func TestAgentsRejectFabricatedEvidenceAndRoleTools(t *testing.T) {
	for _, kind := range []string{"evidence", "file", "shell", "mixed", "missing-plan"} {
		t.Run(kind, func(t *testing.T) {
			m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				r, _ := goodReply(turn)
				if kind == "missing-plan" {
					if r.Plan != nil {
						r.Plan.Plans = nil
					}
					return r, nil
				}
				if kind == "shell" {
					return AgentReply{Tool: &ToolRequest{Name: "shell"}}, nil
				}
				if kind == "mixed" {
					r.Tool = &ToolRequest{Name: "search_source"}
					return r, nil
				}
				if r.Verification != nil {
					if kind == "file" {
						r.Verification.Decisions[0].File = "invented.go"
					} else {
						r.Verification.Decisions[0].EvidenceIDs = []string{"fake"}
					}
				}
				return r, nil
			}}
			s, err := (AgentEngine{Model: m, Tools: fixtureTools{"s"}, Store: memorySessions{}}).Run(context.Background(), "invalid", agentBundle())
			if err == nil || s.Status != "NEEDS_REVIEW" {
				t.Fatal("invalid action accepted")
			}
		})
	}
}

func TestAgentsBudgetAndPartialContext(t *testing.T) {
	for _, partial := range []bool{false, true} {
		b := agentBundle()
		if partial {
			b.Changes[0].ContextIncomplete = true
			b.ContextID, _ = ComputeContextID(b)
		}
		m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
			if !partial && turn.Role == Reviewer {
				return AgentReply{Review: &AgentReview{Issues: []string{"Need more evidence."}, RevisionTarget: Analyst}}, nil
			}
			return goodReply(turn)
		}}
		s, err := (AgentEngine{Model: m, Tools: fixtureTools{"s"}, Store: memorySessions{}}).Run(context.Background(), "bounded", b)
		if err != nil || s.Status != "NEEDS_REVIEW" || s.Turns > 18 {
			t.Fatalf("unbounded or partial approved: %+v %v", s, err)
		}
	}
}

func TestAgentsProviderCannotMutateSession(t *testing.T) {
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		r, _ := goodReply(turn)
		turn.Bundle.Changes[0].Candidates[0].File = "invented.go"
		return r, nil
	}}
	s, err := (AgentEngine{Model: m, Tools: fixtureTools{"s"}, Store: memorySessions{}}).Run(context.Background(), "detached", agentBundle())
	if err != nil || s.Bundle.Changes[0].Candidates[0].File != "store.go" {
		t.Fatal("provider mutated trusted state")
	}
}
