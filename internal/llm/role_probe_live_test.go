//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type roleProbeResult struct {
	Model, Role, Variant, Scope, InputSHA256, Validation, Error string
	Seconds                                                     float64
	ParentAnalyst, Identity                                     planning.ProviderIdentity
	Reply                                                       planning.AgentReply
}

type roleProbeVariant struct {
	role    planning.AgentRole
	name    string
	draft   []planning.FilePlan
	scope   string
	related []planning.ProposalContract
}

type plannerReplyProbe struct {
	identity planning.ProviderIdentity
	reply    planning.AgentReply
}

func (m plannerReplyProbe) Identity() planning.ProviderIdentity { return m.identity }
func (m plannerReplyProbe) Turn(_ context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	if turn.Role == planning.Planner {
		return m.reply, nil
	}
	return planning.AgentReply{}, errors.New("probe stops before model review; no approval supplied")
}

// Explicit opt-in compares roles independently. Planner receives the same
// captured, validated Analyst decisions for every model. Reviewer receives two
// fixed source-authored drafts; expected judgments are never in model input.
// These partial-case diagnostics are not scenario precision/recall results.
func TestLiveRoleComparison(t *testing.T) {
	if os.Getenv("BPMN_ROLE_PROBE") != "1" {
		t.Skip("explicit lab role-comparison opt-in required")
	}
	data, err := os.ReadFile(os.Getenv("BPMN_PROBE_ANALYST_SESSION"))
	if err != nil {
		t.Fatal(err)
	}
	var stored struct{ Session planning.AgentSession }
	if err = json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	s := stored.Session
	if s.Role != planning.Planner || len(s.Decisions) == 0 {
		t.Fatal("validated Analyst checkpoint required")
	}
	bundle := plannerEvidenceBundle(s.Bundle, s.Decisions)
	plannerOnly := os.Getenv("BPMN_PROBE_ROLES") == "planner"
	if len(bundle.Changes) != 1 || len(bundle.Changes[0].Candidates) == 0 || (!plannerOnly && len(bundle.Changes[0].Candidates) != 2 && len(bundle.Changes[0].Candidates) != 3) {
		t.Fatal("this diagnostic requires one change; Reviewer controls require two or three fixed targets")
	}
	if !plannerOnly {
		expected := []string{"backend/internal/inventory/adapter/postgres/store.go", "backend/test/integration/operations_test.go", "frontend/features/inventory/inventory-adjustment-panel.tsx"}
		if bundle.Changes[0].ElementID != "ReasonReview" {
			t.Fatal("Reviewer controls are scoped to the frozen reason-validation case")
		}
		for i, c := range bundle.Changes[0].Candidates {
			if c.File != expected[i] {
				t.Fatal("Reviewer controls require the frozen backend/test/frontend target order")
			}
		}
	}
	bundle.ContextID, _ = planning.ComputeContextID(bundle)
	decisions := []planning.CandidateDecision{}
	for _, d := range s.Decisions {
		if d.Verdict == planning.VerdictRelevant {
			decisions = append(decisions, d)
		}
	}
	good := make([]planning.FilePlan, len(bundle.Changes[0].Candidates))
	var bad []planning.FilePlan
	if !plannerOnly {
		good = sourceAuthoredProbeDraft(bundle)
		bad = append([]planning.FilePlan(nil), good...)
		bad[0].ProposedChanges = []string{"Reject submission when the Go string reason parameter is nil; otherwise allow its value unchanged."}
		bad[0].Steps = []string{"Inside the submit action branch, compare the reason string to nil and return a validation error when nil, before changing the status."}
	}
	variants := []roleProbeVariant{{role: planning.Planner, name: "captured-decisions"}, {role: planning.Reviewer, name: "source-authored-proposal", draft: good}, {role: planning.Reviewer, name: "invalid-string-nil-guard", draft: bad}}
	if !plannerOnly && len(good) == 3 {
		missingPayload := append([]planning.FilePlan(nil), good...)
		missingPayload[2].ProposedChanges = []string{"Add action-specific non-whitespace validation and UI feedback for submit; leave the existing payload dispatch expression unchanged."}
		missingPayload[2].Steps = []string{"Before invoking action for submit, validate the existing reason state and block invalid input with feedback. Keep the existing actionReason conditional unchanged."}
		variants = append(variants, roleProbeVariant{role: planning.Reviewer, name: "missing-producer-payload-edit", draft: missingPayload})
		if os.Getenv("BPMN_PROBE_EXTRA_CONTROLS") == "1" {
			globalGuard := append([]planning.FilePlan(nil), good...)
			globalGuard[0].ProposedChanges = []string{"Require a non-whitespace reason before the action switch for every inventory transition, including submit, approve, reject, cancel and reverse."}
			globalGuard[0].Steps = []string{"Before dispatching any action in Store.TransitionAdjustment, return a validation error whenever strings.TrimSpace(reason) is empty. Keep all existing action branches after that global guard."}
			lostBranches := append([]planning.FilePlan(nil), good...)
			lostBranches[2].ProposedChanges = []string{"Replace the existing actionReason condition with name === \"submit\" ? reason : \"\" so only submission forwards the captured reason.", "Keep the proposed submit-specific non-whitespace validation and feedback."}
			lostBranches[2].Steps = []string{"Reuse existing reason state and input. Change actionReason to forward reason only for submit; reject, cancel and reverse must now use the empty fallback.", "Block invalid submit input before invoking the existing endpoint."}
			variants = append(variants, roleProbeVariant{role: planning.Reviewer, name: "unrelated-global-validation", draft: globalGuard}, roleProbeVariant{role: planning.Reviewer, name: "lost-unrelated-forwarding", draft: lostBranches})
		}
	}
	if os.Getenv("BPMN_PROBE_ROLES") == "reviewer" {
		variants = variants[1:]
	} else if plannerOnly {
		variants = variants[:1]
	}
	if requested := os.Getenv("BPMN_PROBE_VARIANTS"); requested != "" {
		selected := variants[:0]
		wanted := map[string]bool{}
		for _, name := range strings.Split(requested, ",") {
			wanted[strings.TrimSpace(name)] = true
		}
		for _, variant := range variants {
			if wanted[variant.name] {
				selected = append(selected, variant)
				delete(wanted, variant.name)
			}
		}
		if len(wanted) != 0 || len(selected) == 0 {
			t.Fatal("unknown or unavailable probe variant")
		}
		variants = selected
	}
	if os.Getenv("BPMN_PROBE_PER_FILE") == "1" {
		if os.Getenv("BPMN_PROBE_ROLES") != "reviewer" && !plannerOnly {
			t.Fatal("per-file diagnostic requires one explicit role")
		}
		var scoped []roleProbeVariant
		for _, variant := range variants {
			if plannerOnly {
				for _, candidate := range bundle.Changes[0].Candidates {
					scoped = append(scoped, roleProbeVariant{role: planning.Planner, name: variant.name + "@" + candidate.File, scope: candidate.File})
				}
				continue
			}
			for i, plan := range variant.draft {
				if variant.name == "invalid-string-nil-guard" && i != 0 {
					continue
				}
				if variant.name == "missing-producer-payload-edit" && i != 2 {
					continue
				}
				if variant.name == "unrelated-global-validation" && i != 0 || variant.name == "lost-unrelated-forwarding" && i != 2 {
					continue
				}
				local := []planning.FilePlan{plan}
				scoped = append(scoped, roleProbeVariant{role: variant.role, name: variant.name + "@" + plan.File, draft: local, scope: plan.File, related: planning.BuildProposalContracts(variant.draft, local)})
			}
		}
		variants = scoped
	}
	if file := os.Getenv("BPMN_PROBE_FILE"); file != "" {
		if os.Getenv("BPMN_PROBE_PER_FILE") != "1" {
			t.Fatal("file filter requires per-file review diagnostic")
		}
		filtered := variants[:0]
		for _, variant := range variants {
			if variant.scope == file {
				filtered = append(filtered, variant)
			}
		}
		if len(filtered) == 0 {
			t.Fatal("file filter selected no control")
		}
		variants = filtered
	}
	if ids := os.Getenv("BPMN_PROBE_CONTROL_IDS"); ids != "" {
		wanted := map[string]bool{}
		for _, id := range strings.Split(ids, ";") {
			wanted[strings.TrimSpace(id)] = true
		}
		filtered := variants[:0]
		for _, variant := range variants {
			if wanted[variant.name] {
				filtered = append(filtered, variant)
				delete(wanted, variant.name)
			}
		}
		if len(wanted) > 0 || len(filtered) == 0 {
			t.Fatal("control IDs must match available variants exactly")
		}
		variants = filtered
	}

	root := filepath.Join("..", "..", ".data", "model-probes", "roles-"+time.Now().UTC().Format("20060102T150405"))
	if err = os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("Private role diagnostic artifacts:", root)
	results := []roleProbeResult{}
	hashes := map[string]string{}
	for mi, name := range probeModelNames(t) {
		response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://127.0.0.1:11435/api/ps")
		if err != nil {
			t.Fatal(err)
		}
		var running struct{ Models []struct{ Name string } }
		err = json.NewDecoder(response.Body).Decode(&running)
		response.Body.Close()
		if err != nil || len(running.Models) > 1 || (len(running.Models) == 1 && running.Models[0].Name != name) {
			t.Fatal("idle lab runtime required", running.Models, err)
		}
		folder := filepath.Join(root, fmt.Sprintf("model-%d", mi))
		if err = os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		transport := &probeTransport{base: &http.Transport{Proxy: nil}, root: folder, releaseAt: len(variants) - 1}
		maxOutput := 4096
		think := os.Getenv("BPMN_PROBE_THINK") == "1"
		if think {
			maxOutput = 8192
		}
		if value := os.Getenv("BPMN_PROBE_OUTPUT"); value != "" {
			maxOutput, err = strconv.Atoi(value)
			if err != nil || maxOutput < 1024 || maxOutput > 8192 {
				t.Fatal("explicit diagnostic output budget must be 1024..8192")
			}
		}
		model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: name, ContextSize: 32768, MaxOutput: maxOutput, Think: think, ReviewComparisons: os.Getenv("BPMN_PROBE_REVIEW_COMPARISONS") == "1", ReviewJSONMode: os.Getenv("BPMN_PROBE_REVIEW_JSON") == "1", Profile: os.Getenv("BPMN_PROBE_PROFILE"), Client: &http.Client{Timeout: 150 * time.Second, Transport: transport}})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
		if err = model.Prepare(ctx); err != nil {
			cancel()
			t.Fatal(err)
		}
		t.Cleanup(func() { releaseProbeModel(t, name) })
		for _, v := range variants {
			start := time.Now()
			before := len(transport.hashes)
			turn := planning.AgentTurn{Role: v.role, Bundle: bundle, Decisions: decisions, Draft: v.draft, RelatedProposals: v.related}
			if v.scope != "" {
				turn.Bundle.Changes = append([]planning.ChangeInput(nil), bundle.Changes...)
				turn.Bundle.Changes[0].Candidates = nil
				for _, candidate := range bundle.Changes[0].Candidates {
					if candidate.File == v.scope {
						turn.Bundle.Changes[0].Candidates = append(turn.Bundle.Changes[0].Candidates, candidate)
					}
				}
				turn.Decisions = nil
				for _, decision := range decisions {
					if decision.File == v.scope {
						turn.Decisions = append(turn.Decisions, decision)
					}
				}
				turn.Bundle.ContextID, _ = planning.ComputeContextID(turn.Bundle)
			}
			reply, callErr := model.Turn(ctx, turn)
			r := roleProbeResult{Model: name, Role: string(v.role), Variant: v.name, Scope: v.scope, Seconds: time.Since(start).Seconds(), ParentAnalyst: s.Identity, Identity: model.Identity(), Reply: reply, Validation: "ADAPTER_ACCEPTED"}
			if callErr != nil {
				r.Validation, r.Error = "ADAPTER_REJECTED", callErr.Error()
			} else if v.role == planning.Planner {
				engine := planning.AgentEngine{Model: plannerReplyProbe{model.Identity(), reply}, Tools: noSourceTools{}, Store: filestore.AgentSessionStore{Root: filepath.Join(folder, "planner-validation")}, InitialDecisions: turn.Decisions, Budget: planning.AgentBudget{MaxTurns: 3}}
				// Each scoped target has its own immutable validation checkpoint.
				validationID := fmt.Sprintf("probe-validation-%d", len(results))
				checked, _ := engine.Run(context.Background(), validationID, turn.Bundle)
				if checked.Role == planning.Reviewer && len(checked.Draft) == len(turn.Decisions) {
					r.Validation = "ENGINE_VALID_DRAFT_NOT_APPROVED"
				} else {
					r.Validation, r.Error = "ENGINE_REJECTED", fmt.Sprint(checked.Feedback)
				}
			}
			if len(transport.hashes) > before {
				r.InputSHA256 = transport.hashes[before]
			}
			if mi == 0 {
				hashes[v.name] = r.InputSHA256
			} else if r.InputSHA256 != hashes[v.name] {
				t.Error("role comparison input differs", v.name)
			}
			results = append(results, r)
			raw, _ := json.MarshalIndent(results, "", "  ")
			if err = os.WriteFile(filepath.Join(root, "results.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			t.Logf("MODEL=%s role=%s variant=%s validation=%s seconds=%.2f error=%s", name, v.role, v.name, r.Validation, r.Seconds, r.Error)
			if reply.Review != nil {
				t.Logf("REVIEW approved=%t issues=%v", reply.Review.Approved, reply.Review.Issues)
			}
		}
		cancel()
		releaseProbeModel(t, name)
	}
}

func sourceAuthoredProbeDraft(bundle planning.EvidenceBundle) []planning.FilePlan {
	change := bundle.Changes[0]
	plans := []planning.FilePlan{}
	for i, c := range change.Candidates {
		p := planning.FilePlan{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: c.SuggestedAction}
		for _, e := range c.Evidence {
			p.EvidenceIDs = append(p.EvidenceIDs, e.ID)
		}
		if i == 0 {
			p.CurrentBehavior = "Store.TransitionAdjustment already accepts reason as a Go string, locks the adjustment row, checks DRAFT in the submit branch, and changes status to SUBMITTED without a reason guard in that branch."
			p.TargetBehavior = "The new reason-entry and validation task must finish before submission. Invalid reasons must not change the adjustment state."
			p.DesignRationale = "The inspected submit branch owns the affected state transition. Reuse the existing reason parameter and persistence transaction rather than proposing a new endpoint or schema."
			p.Summary = "Validate the existing reason input specifically during adjustment submission."
			p.ProposedChanges = []string{"Require the existing reason string to contain non-whitespace text in the submit action, before assigning SUBMITTED; return a validation failure without persisting state on invalid input.", "Preserve the existing DRAFT state check and unrelated approve, cancel and other actions; retain the existing transaction and row-lock ordering."}
			p.Steps = []string{"Inside Store.TransitionAdjustment's submit branch, preserve the DRAFT guard and add the action-specific non-whitespace reason guard before the status assignment.", "Use an existing suitable validation-error convention if confirmed; otherwise introduce a proposed error locally following the existing error pattern. Do not change the parameter type or signature.", "Keep transaction rollback on failure and existing successful transition behavior; validate the proposal with submit success and failure regression cases."}
			p.Assumptions = []string{"The exact validation error name and HTTP translation are not established by these excerpts; confirm before implementation."}
			if len(change.Candidates) == 2 {
				p.Assumptions = append(p.Assumptions, "Frontend changes are outside this two-file diagnostic, not considered covered.")
			}
			p.AcceptanceChecks = []string{"An empty or whitespace-only reason fails submission and leaves the adjustment in DRAFT.", "A non-whitespace reason preserves the successful DRAFT-to-SUBMITTED transition.", "Non-submit actions retain their current behavior and row-lock ordering."}
		} else if i == 1 {
			p.CurrentBehavior = "Existing inventory integration calls use the service Transition method with submit and an empty reason, then assert SUBMITTED; cancel calls already include a reason."
			p.TargetBehavior = "Inventory submit regressions must distinguish a valid reason from empty and whitespace-only inputs."
			p.DesignRationale = "The observed service-level integration calls exercise the submission path changed by the new BPMN requirement."
			p.Summary = "Update inventory submit integration inputs and add rejection cases."
			p.ProposedChanges = []string{"Pass a meaningful reason in inventory success-path submit calls to the existing service Transition entry point.", "Add new inventory submission cases for empty and whitespace-only reasons, asserting failure and unchanged DRAFT state; preserve unrelated purchasing, sales and cancel inputs."}
			p.Steps = []string{"Update the observed inventory submit calls that currently pass an empty reason while keeping their service entry point and argument structure.", "Propose new negative integration cases using the same inventory setup and state assertion approach; submit with empty and whitespace-only reasons and check the error and DRAFT state.", "Retain and run the valid-reason submission and existing cancel-path regressions when implementation is available."}
			p.AcceptanceChecks = []string{"Valid-reason inventory submission still reaches SUBMITTED.", "Empty and whitespace-only reason submissions fail and remain DRAFT.", "Purchasing, sales and unrelated inventory transitions keep their existing behavior."}
			p.Assumptions = []string{"These are proposed tests; no repository test execution or passing result is claimed."}
		} else {
			p.CurrentBehavior = "The component already has reason state. Its actionReason conditional forwards that state for reject/cancel/reverse and sends empty text for submit."
			p.TargetBehavior = "Submit must capture and validate non-whitespace reason, forward that value to the existing endpoint, and preserve unrelated action behavior."
			p.DesignRationale = "Validation at the consumer cannot succeed on a valid UI submission while its producer still sends an empty default. Reuse the existing state and request field."
			p.Summary = "Forward and validate the existing reason state specifically for submission."
			p.ProposedChanges = []string{"Extend the actionReason dispatch condition to include submit, so the existing reason value reaches the payload for this action.", "Add submit-specific non-whitespace validation and required-field feedback, preserving current reject/cancel/reverse constraints and other actions."}
			p.Steps = []string{"Reuse the reason state and input; update its label to include submission.", "Include submit in the existing actionReason conditional instead of sending its empty fallback.", "Before submit dispatch, block an empty/whitespace-only reason with visible feedback; retain unrelated action behavior."}
			p.AcceptanceChecks = []string{"A valid submit sends the captured reason text in the existing request field.", "Empty and whitespace-only submit reasons block the request and show feedback.", "Reject/cancel/reverse retain their existing reason handling and other actions are unchanged."}
		}
		plans = append(plans, p)
	}
	return plans
}
