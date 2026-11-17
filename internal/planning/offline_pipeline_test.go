package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// These fixtures exercise orchestration contracts, not model intelligence.
func TestOfflineInvalidPlannerNeverReachesReviewer(t *testing.T) {
	for _, mode := range []string{"empty", "unknown-evidence", "unknown-file", "blank-checks"} {
		t.Run(mode, func(t *testing.T) {
			reviews := 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				r, err := goodReply(turn)
				if turn.Role == Reviewer {
					reviews++
				}
				if turn.Role == Planner {
					switch mode {
					case "empty":
						r.Plan.Plans = nil
					case "unknown-evidence":
						r.Plan.Plans[0].EvidenceIDs = []string{"invented-proof"}
					case "unknown-file":
						r.Plan.Plans[0].File = "unverified.go"
					case "blank-checks":
						r.Plan.Plans[0].AcceptanceChecks = []string{" \t\n"}
					}
				}
				return r, err
			}}
			s, err := (AgentEngine{Model: model, Tools: fixtureTools{"offline-source"}, Store: memorySessions{}}).Run(context.Background(), "offline-"+mode, agentBundle())
			if err == nil || s.Status == "COMPLETE" || reviews != 0 || len(s.Draft) != 0 {
				t.Fatalf("invalid planner escaped validation: status=%s reviews=%d draft=%d error=%v", s.Status, reviews, len(s.Draft), err)
			}
			if s.OutputRepairs > 1 {
				t.Fatal("invalid output caused unbounded retries")
			}
		})
	}
}

func TestOfflinePlanExportKeepsTechnicalContentFromCanonicalJSON(t *testing.T) {
	p := ChangePlan{Status: StatusNeedsReview, Plans: []FilePlan{{
		File: "service.go", Summary: "Validate input before submit.",
		CurrentBehavior:  "Submit currently accepts an empty reason.",
		TargetBehavior:   "Reject an empty reason before changing status.",
		DesignRationale:  "Keep validation at the transition boundary.",
		Assumptions:      []string{"Minimum length is an open product decision."},
		Steps:            []string{"Normalize whitespace before validation."},
		ProposedChanges:  []string{"Add an action-specific input guard."},
		AcceptanceChecks: []string{"An invalid reason leaves the stored status unchanged."},
		EvidenceIDs:      []string{"source-proof"},
	}}, ReviewIssues: []string{"Confirm the validation rule with source evidence."}}
	encoded, err := JSON(p)
	if err != nil {
		t.Fatal(err)
	}
	var restored ChangePlan
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	md := Markdown(restored)
	if !bytes.Equal(md, Markdown(p)) || !bytes.Equal(md, Markdown(restored)) {
		t.Fatal("export changed across JSON roundtrip or repeated render")
	}
	for _, section := range []string{"Unapproved draft", "Review issues", "Current behavior", "Required behavior", "Design rationale", "Assumptions and open decisions", "Implementation steps", "Proposed changes", "Acceptance checks"} {
		if !strings.Contains(string(md), section) {
			t.Fatalf("missing section %q", section)
		}
	}
	f := restored.Plans[0]
	for _, text := range []string{f.File, f.Summary, f.CurrentBehavior, f.TargetBehavior, f.DesignRationale, f.Assumptions[0], f.Steps[0], f.ProposedChanges[0], f.AcceptanceChecks[0], f.EvidenceIDs[0], restored.ReviewIssues[0]} {
		if !strings.Contains(string(md), text) {
			t.Fatalf("canonical content omitted: %q", text)
		}
	}
}
