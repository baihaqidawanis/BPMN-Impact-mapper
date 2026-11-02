package planning

import (
	"context"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

const BundleVersion = "1"

type Verdict string

const (
	VerdictRelevant Verdict = "RELEVANT"
	VerdictWeak     Verdict = "WEAK"
	VerdictReject   Verdict = "REJECT"
)

type Status string

const (
	StatusDisabled    Status = "DISABLED"
	StatusComplete    Status = "COMPLETE"
	StatusNeedsReview Status = "NEEDS_REVIEW"
)

type Limits struct {
	CandidatesPerChange  int
	RelatedPerChange     int
	EvidencePerCandidate int
	ExcerptRunes         int
}

func DefaultLimits() Limits {
	return Limits{CandidatesPerChange: 10, RelatedPerChange: 3, EvidencePerCandidate: 8, ExcerptRunes: 240}
}

type EvidenceItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Excerpt  string `json:"excerpt"`
}

type Candidate struct {
	File            string                `json:"file"`
	Symbol          string                `json:"symbol,omitempty"`
	Module          string                `json:"module"`
	Layer           string                `json:"layer"`
	SuggestedAction domain.RequiredAction `json:"suggested_action"`
	RetrievalScore  *float64              `json:"retrieval_score,omitempty"`
	Origin          string                `json:"origin,omitempty"`
	Evidence        []EvidenceItem        `json:"evidence"`
}

type ChangeInput struct {
	Before            *ElementInput         `json:"before,omitempty"`
	After             *ElementInput         `json:"after,omitempty"`
	BranchChanges     []domain.BranchChange `json:"branch_changes,omitempty"`
	ContextTruncated  bool                  `json:"context_truncated,omitempty"`
	ContextIncomplete bool                  `json:"context_incomplete,omitempty"`
	ChangeType        domain.ChangeType     `json:"change_type"`
	ElementID         string                `json:"element_id"`
	ElementName       string                `json:"element_name,omitempty"`
	Reason            string                `json:"reason,omitempty"`
	Candidates        []Candidate           `json:"candidates"`
}

type ElementInput struct {
	GatewayDirection string   `json:"gateway_direction,omitempty"`
	DefaultFlow      string   `json:"default_flow,omitempty"`
	AttachedToRef    string   `json:"attached_to_ref,omitempty"`
	EventDefinitions []string `json:"event_definitions,omitempty"`
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type"`
	Actor            string   `json:"actor,omitempty"`
	State            string   `json:"state,omitempty"`
	Documentation    string   `json:"documentation,omitempty"`
	DataRequirements []string `json:"data_requirements,omitempty"`
}

type EvidenceBundle struct {
	Warnings          []domain.Warning      `json:"warnings,omitempty"`
	WarningsTruncated bool                  `json:"warnings_truncated,omitempty"`
	Version           string                `json:"version"`
	ContextID         string                `json:"context_id"`
	AnalysisID        string                `json:"analysis_id"`
	InputFingerprint  string                `json:"input_fingerprint"`
	Method            domain.AnalysisMethod `json:"method"`
	Repository        domain.Repository     `json:"repository"`
	Changes           []ChangeInput         `json:"changes"`
}

type ProviderIdentity struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	PromptVersion string `json:"prompt_version"`
}

type Policy struct {
	VerifyTimeout time.Duration
	LayerTimeout  time.Duration
	ReviewTimeout time.Duration
}

func DefaultPolicy() Policy {
	return Policy{VerifyTimeout: 45 * time.Second, LayerTimeout: 90 * time.Second, ReviewTimeout: 60 * time.Second}
}

type CandidateDecision struct {
	ChangeType  domain.ChangeType `json:"change_type"`
	ElementID   string            `json:"element_id"`
	File        string            `json:"file"`
	Symbol      string            `json:"symbol,omitempty"`
	Verdict     Verdict           `json:"verdict"`
	Rationale   string            `json:"rationale"`
	EvidenceIDs []string          `json:"evidence_ids"`
}

type VerificationRequest struct {
	Change ChangeInput `json:"change"`
}

type VerificationResponse struct {
	Decisions []CandidateDecision `json:"decisions"`
}

type LayerPlanningRequest struct {
	Layer     string              `json:"layer"`
	Bundle    EvidenceBundle      `json:"bundle"`
	Decisions []CandidateDecision `json:"decisions"`
}

type FilePlan struct {
	ChangeType       domain.ChangeType     `json:"change_type"`
	ElementID        string                `json:"element_id"`
	File             string                `json:"file"`
	Symbol           string                `json:"symbol,omitempty"`
	Module           string                `json:"module"`
	Layer            string                `json:"layer"`
	Action           domain.RequiredAction `json:"action"`
	Summary          string                `json:"summary"`
	Steps            []string              `json:"steps"`
	ProposedChanges  []string              `json:"proposed_changes,omitempty"`
	AcceptanceChecks []string              `json:"acceptance_checks,omitempty"`
	EvidenceIDs      []string              `json:"evidence_ids"`
}

type LayerPlanningResponse struct {
	Plans []FilePlan `json:"plans"`
}

type ReviewRequest struct {
	Bundle EvidenceBundle `json:"bundle"`
	Draft  []FilePlan     `json:"draft"`
}

type ReviewResponse struct {
	Approved bool       `json:"approved"`
	Issues   []string   `json:"issues"`
	Revised  []FilePlan `json:"revised,omitempty"`
}

type ChangePlan struct {
	FullReview       *FullReviewProgress `json:"full_review,omitempty"`
	Version          string              `json:"version"`
	PlanID           string              `json:"plan_id"`
	AnalysisID       string              `json:"analysis_id"`
	InputFingerprint string              `json:"input_fingerprint"`
	ContextID        string              `json:"context_id"`
	Provider         ProviderIdentity    `json:"provider"`
	Status           Status              `json:"status"`
	Plans            []FilePlan          `json:"plans"`
	ReviewIssues     []string            `json:"review_issues"`
}

type PlanSummary struct {
	PlanID     string           `json:"plan_id"`
	AnalysisID string           `json:"analysis_id"`
	ContextID  string           `json:"context_id"`
	Provider   ProviderIdentity `json:"provider"`
	Status     Status           `json:"status"`
	Files      int              `json:"files"`
}

func Summarize(plan ChangePlan) PlanSummary {
	return PlanSummary{
		PlanID: plan.PlanID, AnalysisID: plan.AnalysisID, ContextID: plan.ContextID,
		Provider: plan.Provider, Status: plan.Status, Files: len(plan.Plans),
	}
}

type Provider interface {
	Identity() ProviderIdentity
	Enabled() bool
	VerifyCandidates(context.Context, VerificationRequest) (VerificationResponse, error)
	PlanLayer(context.Context, LayerPlanningRequest) (LayerPlanningResponse, error)
	ReviewPlan(context.Context, ReviewRequest) (ReviewResponse, error)
}
