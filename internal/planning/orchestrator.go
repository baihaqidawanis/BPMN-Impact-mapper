package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type Orchestrator struct {
	Provider Provider
	Limits   Limits
	Policy   Policy
}

func (orchestrator Orchestrator) Run(ctx context.Context, report domain.Report) (ChangePlan, error) {
	if orchestrator.Provider == nil {
		return ChangePlan{}, errors.New("planning provider is required")
	}
	bundle := BuildEvidenceBundle(report, orchestrator.Limits)
	identity := orchestrator.Provider.Identity()
	result := ChangePlan{
		Version: BundleVersion, AnalysisID: report.AnalysisID, InputFingerprint: report.InputFingerprint,
		ContextID: bundle.ContextID, Provider: identity, Status: StatusDisabled, Plans: []FilePlan{}, ReviewIssues: []string{},
	}
	if !orchestrator.Provider.Enabled() {
		return finalize(result)
	}
	if strings.TrimSpace(identity.Provider) == "" || strings.TrimSpace(identity.Model) == "" || strings.TrimSpace(identity.PromptVersion) == "" {
		return ChangePlan{}, errors.New("enabled planning provider requires provider, model, and prompt version identity")
	}
	if err := validateBundle(bundle); err != nil {
		return ChangePlan{}, err
	}
	policy := normalizedPolicy(orchestrator.Policy)
	decisions := make([]CandidateDecision, 0)
	for _, change := range bundle.Changes {
		if err := ctx.Err(); err != nil {
			return ChangePlan{}, err
		}
		roleCtx, cancel := context.WithTimeout(ctx, policy.VerifyTimeout)
		verified, err := orchestrator.Provider.VerifyCandidates(roleCtx, VerificationRequest{Change: change})
		cancel()
		if err != nil {
			return ChangePlan{}, fmt.Errorf("verify %s/%s: %w", change.ChangeType, change.ElementID, err)
		}
		if err := validateDecisions(change, verified.Decisions); err != nil {
			return ChangePlan{}, fmt.Errorf("verify %s/%s: %w", change.ChangeType, change.ElementID, err)
		}
		decisions = append(decisions, verified.Decisions...)
	}
	layers := decisionLayers(bundle, decisions)
	draft := make([]FilePlan, 0)
	for _, layer := range layers {
		layerDecisions := decisionsForLayer(bundle, decisions, layer)
		roleCtx, cancel := context.WithTimeout(ctx, policy.LayerTimeout)
		planned, err := orchestrator.Provider.PlanLayer(roleCtx, LayerPlanningRequest{Layer: layer, Bundle: bundle, Decisions: layerDecisions})
		cancel()
		if err != nil {
			return ChangePlan{}, fmt.Errorf("plan layer %s: %w", layer, err)
		}
		if err := validatePlans(bundle, decisions, layer, planned.Plans); err != nil {
			return ChangePlan{}, fmt.Errorf("plan layer %s: %w", layer, err)
		}
		draft = append(draft, planned.Plans...)
	}
	sortPlans(draft)
	roleCtx, cancel := context.WithTimeout(ctx, policy.ReviewTimeout)
	review, err := orchestrator.Provider.ReviewPlan(roleCtx, ReviewRequest{Bundle: bundle, Draft: draft})
	cancel()
	if err != nil {
		return ChangePlan{}, fmt.Errorf("review plan: %w", err)
	}
	finalPlans := draft
	if len(review.Revised) > 0 {
		if err := validatePlans(bundle, decisions, "", review.Revised); err != nil {
			return ChangePlan{}, fmt.Errorf("review plan: %w", err)
		}
		finalPlans = append([]FilePlan(nil), review.Revised...)
		sortPlans(finalPlans)
	}
	result.Plans = finalPlans
	result.ReviewIssues = cleanStrings(review.Issues)
	result.Status = StatusNeedsReview
	if review.Approved {
		result.Status = StatusComplete
	}
	return finalize(result)
}

func normalizedPolicy(policy Policy) Policy {
	defaults := DefaultPolicy()
	if policy.VerifyTimeout <= 0 {
		policy.VerifyTimeout = defaults.VerifyTimeout
	}
	if policy.LayerTimeout <= 0 {
		policy.LayerTimeout = defaults.LayerTimeout
	}
	if policy.ReviewTimeout <= 0 {
		policy.ReviewTimeout = defaults.ReviewTimeout
	}
	return policy
}

func validateBundle(bundle EvidenceBundle) error {
	if bundle.AnalysisID == "" || bundle.InputFingerprint == "" {
		return errors.New("analysis identity is required")
	}
	expectedContextID, err := ComputeContextID(bundle)
	if err != nil || bundle.ContextID == "" || bundle.ContextID != expectedContextID {
		return errors.New("planning context identity is invalid")
	}
	for _, change := range bundle.Changes {
		if change.ChangeType == "" || change.ElementID == "" {
			return errors.New("every planning change requires type and element ID")
		}
		seen := map[string]bool{}
		for _, candidate := range change.Candidates {
			key := candidateKey(candidate.File, candidate.Symbol)
			if candidate.File == "" || seen[key] || len(candidate.Evidence) == 0 {
				return fmt.Errorf("change %s/%s contains invalid candidate %q", change.ChangeType, change.ElementID, key)
			}
			seen[key] = true
		}
	}
	return nil
}

func validateDecisions(change ChangeInput, decisions []CandidateDecision) error {
	allowed := map[string]Candidate{}
	for _, candidate := range change.Candidates {
		allowed[candidateKey(candidate.File, candidate.Symbol)] = candidate
	}
	if len(decisions) != len(allowed) {
		return fmt.Errorf("provider must decide all %d candidates, got %d", len(allowed), len(decisions))
	}
	seen := map[string]bool{}
	for _, decision := range decisions {
		key := candidateKey(decision.File, decision.Symbol)
		candidate, ok := allowed[key]
		if !ok || seen[key] {
			return fmt.Errorf("provider referenced unknown or duplicate candidate %q", key)
		}
		seen[key] = true
		if decision.ChangeType != change.ChangeType || decision.ElementID != change.ElementID {
			return fmt.Errorf("provider changed BPMN identity for %q", key)
		}
		switch decision.Verdict {
		case VerdictRelevant, VerdictWeak, VerdictReject:
		default:
			return fmt.Errorf("candidate %q has invalid verdict %q", key, decision.Verdict)
		}
		if strings.TrimSpace(decision.Rationale) == "" {
			return fmt.Errorf("candidate %q requires rationale", key)
		}
		if err := validateEvidenceIDs(candidate, decision.EvidenceIDs); err != nil {
			return fmt.Errorf("candidate %q: %w", key, err)
		}
	}
	return nil
}

func validatePlans(bundle EvidenceBundle, decisions []CandidateDecision, requiredLayer string, plans []FilePlan) error {
	allowed := allowedPlanningTargets(bundle, decisions)
	seen := map[string]bool{}
	for _, plan := range plans {
		key := planKey(plan.ChangeType, plan.ElementID, plan.File, plan.Symbol)
		candidate, ok := allowed[key]
		if !ok || seen[key] {
			return fmt.Errorf("provider planned unknown, rejected, or duplicate target %q", key)
		}
		seen[key] = true
		if requiredLayer != "" && candidate.Layer != requiredLayer {
			return fmt.Errorf("target %q belongs to layer %q, not %q", key, candidate.Layer, requiredLayer)
		}
		if plan.Layer != candidate.Layer || plan.Module != candidate.Module {
			return fmt.Errorf("provider changed module/layer for %q", key)
		}
		if !validAction(plan.Action) || strings.TrimSpace(plan.Summary) == "" || len(cleanStrings(plan.Steps)) == 0 {
			return fmt.Errorf("target %q requires valid action, summary, and steps", key)
		}
		if err := validateEvidenceIDs(candidate, plan.EvidenceIDs); err != nil {
			return fmt.Errorf("target %q: %w", key, err)
		}
	}
	for _, decision := range decisions {
		if decision.Verdict != VerdictRelevant {
			continue
		}
		key := planKey(decision.ChangeType, decision.ElementID, decision.File, decision.Symbol)
		candidate, ok := allowed[key]
		if !ok || (requiredLayer != "" && candidate.Layer != requiredLayer) {
			continue
		}
		if !seen[key] {
			return fmt.Errorf("relevant candidate %q has no file plan", key)
		}
	}
	return nil
}

func allowedPlanningTargets(bundle EvidenceBundle, decisions []CandidateDecision) map[string]Candidate {
	accepted := map[string]bool{}
	for _, decision := range decisions {
		if decision.Verdict != VerdictReject {
			accepted[planKey(decision.ChangeType, decision.ElementID, decision.File, decision.Symbol)] = true
		}
	}
	result := map[string]Candidate{}
	for _, change := range bundle.Changes {
		for _, candidate := range change.Candidates {
			key := planKey(change.ChangeType, change.ElementID, candidate.File, candidate.Symbol)
			if accepted[key] {
				result[key] = candidate
			}
		}
	}
	return result
}

func validateEvidenceIDs(candidate Candidate, values []string) error {
	allowed := map[string]bool{}
	for _, evidence := range candidate.Evidence {
		allowed[evidence.ID] = true
	}
	if len(values) == 0 {
		return errors.New("at least one evidence reference is required")
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !allowed[value] || seen[value] {
			return fmt.Errorf("unknown or duplicate evidence reference %q", value)
		}
		seen[value] = true
	}
	return nil
}

func decisionLayers(bundle EvidenceBundle, decisions []CandidateDecision) []string {
	allowed := allowedPlanningTargets(bundle, decisions)
	set := map[string]bool{}
	for _, candidate := range allowed {
		set[candidate.Layer] = true
	}
	result := make([]string, 0, len(set))
	for layer := range set {
		result = append(result, layer)
	}
	sort.Strings(result)
	return result
}

func decisionsForLayer(bundle EvidenceBundle, decisions []CandidateDecision, layer string) []CandidateDecision {
	targets := allowedPlanningTargets(bundle, decisions)
	result := make([]CandidateDecision, 0)
	for _, decision := range decisions {
		candidate, ok := targets[planKey(decision.ChangeType, decision.ElementID, decision.File, decision.Symbol)]
		if ok && candidate.Layer == layer {
			result = append(result, decision)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return planKey(result[i].ChangeType, result[i].ElementID, result[i].File, result[i].Symbol) < planKey(result[j].ChangeType, result[j].ElementID, result[j].File, result[j].Symbol)
	})
	return result
}

func validAction(action domain.RequiredAction) bool {
	switch action {
	case domain.ActionAdd, domain.ActionModify, domain.ActionRemove, domain.ActionVerify, domain.ActionNoAction, domain.ActionUnresolved:
		return true
	default:
		return false
	}
}

func sortPlans(plans []FilePlan) {
	for index := range plans {
		plans[index].Steps = cleanStrings(plans[index].Steps)
		plans[index].EvidenceIDs = cleanStrings(plans[index].EvidenceIDs)
	}
	sort.Slice(plans, func(i, j int) bool {
		return planKey(plans[i].ChangeType, plans[i].ElementID, plans[i].File, plans[i].Symbol) < planKey(plans[j].ChangeType, plans[j].ElementID, plans[j].File, plans[j].Symbol)
	})
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func candidateKey(file, symbol string) string { return file + "\x00" + symbol }
func planKey(change domain.ChangeType, elementID, file, symbol string) string {
	return string(change) + "\x00" + elementID + "\x00" + candidateKey(file, symbol)
}

func finalize(plan ChangePlan) (ChangePlan, error) {
	plan.ReviewIssues = cleanStrings(plan.ReviewIssues)
	planID, err := ComputePlanID(plan)
	if err != nil {
		return ChangePlan{}, err
	}
	plan.PlanID = planID
	return plan, nil
}

func ComputePlanID(plan ChangePlan) (string, error) {
	plan.PlanID = ""
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])[:32], nil
}
