// Package agentjobs coordinates asynchronous, single-worker agent sessions.
// Model, source and persistence implementations are injected by composition root.
package agentjobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planstore"
)

var ErrBusy = errors.New("an agent session is already running")
var ErrMissing = errors.New("agent session not found")
var ErrBinding = errors.New("session does not belong to this analysis/change")

type Model interface {
	planning.AgentModel
	Prepare(context.Context) error
}
type Snapshot interface {
	planning.SourceTools
	CheckBundle(planning.EvidenceBundle) error
}

type View struct {
	ActiveMillis   *int64 `json:"active_ms,omitempty"`
	ChangeIndex    *int   `json:"change_index,omitempty"`
	runStarted     time.Time
	SourceActivity []SourceActivity                `json:"source_activity,omitempty"`
	Mode           string                          `json:"mode"`
	FocusedReview  *planning.FocusedReviewProgress `json:"focused_review,omitempty"`
	FullReview     *planning.FullReviewProgress    `json:"full_review,omitempty"`
	SessionID      string                          `json:"session_id"`
	AnalysisID     string                          `json:"analysis_id"`
	Status         string                          `json:"status"`
	Role           planning.AgentRole              `json:"role"`
	Turns          int                             `json:"turns"`
	Tools          int                             `json:"tools"`
	Revisions      int                             `json:"revisions"`
	Provider       planning.ProviderIdentity       `json:"provider"`
	Message        string                          `json:"message,omitempty"`
	Plan           *planning.ChangePlan            `json:"plan,omitempty"`
	Context        *planning.EvidenceBundle        `json:"context,omitempty"`
	Decisions      []planning.CandidateDecision    `json:"decisions,omitempty"`
	Events         []planning.AgentEvent           `json:"events,omitempty"`
	Issues         []string                        `json:"issues,omitempty"`
}

type Service struct {
	Configuration *Configuration
	Model         Model
	Sessions      planning.AgentSessionStore
	Plans         planstore.Store
	Source        func(context.Context, string) (Snapshot, error)
	Base          context.Context
	Timeout       time.Duration
	LogError      func(error)
	mu            sync.Mutex
	active        bool
	activeID      string
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	views         map[string]View
}

func (s *Service) Check(ctx context.Context) error {
	if s == nil {
		return errors.New("model disabled")
	}
	_, err := s.InspectModel(ctx)
	return err
}

func (s *Service) Start(ctx context.Context, report domain.Report, repo string, changeIndex int, id string, modes ...string) (View, error) {
	mode := "focused"
	if len(modes) > 0 && modes[0] != "" {
		mode = modes[0]
	}
	if mode != "focused" && mode != "full" && mode != "light" && mode != "ultra-light" {
		return View{}, ErrBinding
	}
	if s.Sessions == nil || s.Plans == nil || s.Source == nil {
		return View{}, errors.New("agent service incomplete")
	}
	if changeIndex < -1 || changeIndex >= len(report.Changes) || len(report.Changes) == 0 || repo == "" {
		return View{}, ErrBinding
	}
	if err := ctx.Err(); err != nil {
		return View{}, err
	}
	if s.Base != nil {
		if err := s.Base.Err(); err != nil {
			return View{}, err
		}
	}
	selected := report
	if changeIndex >= 0 {
		selected.Changes = []domain.Change{report.Changes[changeIndex]}
	}
	limit := 3
	related := 1
	if mode != "full" {
		limit = 1
		related = -1
		for _, c := range selected.Changes {
			if len(c.Impacts) > limit {
				limit = len(c.Impacts)
			}
		}
	}
	limits := planning.Limits{CandidatesPerChange: limit, RelatedPerChange: related, EvidencePerCandidate: 2, ExcerptRunes: 320}
	limits.GroupGatewayChanges = mode != "full"
	limits.GroupGatewayBranches = mode != "full"
	bundle := planning.BuildAgentBundle(selected, limits)
	var resumeIdentity *planning.ProviderIdentity
	var elapsedAtStart int64
	previousMillis := &elapsedAtStart
	if id != "" {
		stored, err := s.Sessions.Load(ctx, id)
		if err != nil {
			return View{}, err
		}
		if stored.ChangeIndex != nil && *stored.ChangeIndex != changeIndex {
			return View{}, ErrBinding
		}
		// Rebuild older bundle shapes for resume. The engine still verifies
		// the exact source, context and model binding before using a checkpoint.
		if !sameChanges(stored.Bundle.Changes, bundle.Changes) {
			limits.GroupGatewayBranches = false
			bundle = planning.BuildAgentBundle(selected, limits)
		}
		if !sameChanges(stored.Bundle.Changes, bundle.Changes) {
			limits.GroupGatewayChanges = false
			bundle = planning.BuildAgentBundle(selected, limits)
		}
		if !sameChanges(stored.Bundle.Changes, bundle.Changes) {
			bundle = planning.BuildEvidenceBundle(selected, limits)
		}
		if stored.Bundle.AnalysisID != report.AnalysisID || stored.Bundle.InputFingerprint != bundle.InputFingerprint || stored.Bundle.Method != bundle.Method || !sameChanges(stored.Bundle.Changes, bundle.Changes) {
			return View{}, ErrBinding
		}
		identity := stored.Identity
		previousMillis = stored.ActiveMillis
		resumeIdentity = &identity
		storedMode := sessionMode(stored)
		if len(modes) == 0 || modes[0] == "" {
			mode = storedMode
		} else if mode != storedMode {
			return View{}, ErrBinding
		}
	} else {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return View{}, err
		}
		id = hex.EncodeToString(raw)
	}
	s.mu.Lock()
	if s.Model == nil {
		s.mu.Unlock()
		return View{}, errors.New("model disabled")
	}
	if resumeIdentity != nil && *resumeIdentity != s.Model.Identity() {
		s.mu.Unlock()
		return View{}, ErrBinding
	}
	if s.active {
		s.mu.Unlock()
		return View{}, ErrBusy
	}
	if s.views == nil {
		s.views = map[string]View{}
	}
	// Only active/recent transient views are retained; durable checkpoints live in
	// Sessions. Local single-worker service does not queue GPU requests.
	if len(s.views) >= 100 {
		s.views = map[string]View{}
	}
	v := View{SessionID: id, AnalysisID: report.AnalysisID, Status: "STARTING", Role: planning.Analyst, Provider: s.Model.Identity(), Message: "Preparing approved source and model metadata."}
	v.Mode = mode
	v.ChangeIndex = &changeIndex
	v.ActiveMillis = previousMillis
	v.runStarted = time.Now()
	s.views[id] = v
	s.active = true
	runBase, runCancel := context.WithCancel(safeBase(s.Base))
	s.cancel = runCancel
	s.activeID = id
	s.wg.Add(1)
	s.mu.Unlock()
	base := runBase
	if base == nil {
		base = context.Background()
	}
	timeout := s.Timeout
	// Whole jobs are cancellable/resumable; each model turn still has a timeout.
	go func() {
		defer s.wg.Done()
		defer runCancel()
		runCtx, cancel := context.WithCancel(base)
		if timeout > 0 {
			cancel()
			runCtx, cancel = context.WithTimeout(base, timeout)
		}
		defer cancel()
		s.execute(runCtx, v, repo, bundle)
		s.mu.Lock()
		s.active = false
		s.activeID = ""
		s.cancel = nil
		s.mu.Unlock()
	}()
	return v, nil
}

func (s *Service) execute(ctx context.Context, v View, repo string, bundle planning.EvidenceBundle) {
	clock := timedCheckpoints{AgentSessionStore: s.Sessions, id: v.SessionID, start: v.runStarted, previousMillis: v.ActiveMillis, changeIndex: v.ChangeIndex}
	fail := func(status, message string) { v.Status = status; v.Message = message; s.put(v) }
	source, err := s.Source(ctx, repo)
	if err != nil {
		fail("FAILED", "Source could not be prepared. Check the repository path, ignore warnings and data-directory boundary.")
		return
	}
	if err := source.CheckBundle(bundle); err != nil {
		fail("FAILED", "A candidate file/function is missing from the current source. Run mapping again against this repository.")
		return
	}
	if err := s.Model.Prepare(ctx); err != nil {
		fail("FAILED", "Local model unavailable. Check VPN, SSH tunnel, model name and model metadata.")
		return
	}
	v.Status = "RUNNING"
	v.Message = "Agent session running."
	s.put(v)
	var session planning.AgentSession
	var runErr error
	if v.Mode == "full" {
		fullSource, ok := source.(planning.RepositorySource)
		if !ok {
			fail("FAILED", "Source adapter does not support full repository review.")
			return
		}
		session, runErr = (planning.FullRepositoryEngine{Model: s.Model, Source: fullSource, Store: clock}).Run(ctx, v.SessionID, bundle)
	} else {
		session, runErr = (planning.FocusedEngine{Model: s.Model, Tools: source, Store: clock, Light: v.Mode == "light", UltraLight: v.Mode == "ultra-light"}).Run(ctx, v.SessionID, bundle)
	}
	if session.ID != "" {
		v = viewSession(session)
		v.ActiveMillis = clock.elapsed()
		v.ChangeIndex = clock.changeIndex
	}
	if runErr != nil {
		if s.LogError != nil {
			s.LogError(runErr)
		}
		if errors.Is(runErr, planning.ErrAgentOutputLimit) {
			v.Message = "Model output budget was exhausted before JSON completed. The incomplete response was not accepted. Try Thinking Off or a larger output budget in a new session."
		} else if errors.Is(runErr, planning.ErrAgentContextLimit) {
			v.Message = "Evidence exceeds the configured model context. Use a smaller candidate report or increase --agent-context, then start a new session. Source was not silently truncated."
		} else if errors.Is(runErr, planning.ErrAgentModelOutput) {
			v.Message = "Model response could not be validated. Inspect Session issues for the failed role and reason before retrying; consumed turns remain charged."
		} else if errors.Is(runErr, context.DeadlineExceeded) {
			v.Message = "A model turn exceeded its time budget while loading or generating. The checkpoint is saved; a timeout alone does not prove VPN/tunnel failure."
		} else if errors.Is(runErr, context.Canceled) {
			v.Message = "Review was cancelled. Completed checkpoints remain saved."
		} else if session.Status == "PAUSED" {
			v.Message = "A model request was interrupted. The checkpoint is saved. Inspect the backend log and model runtime before resuming; this message does not establish a VPN or GPU-memory failure."
		} else {
			v.Message = "Session could not continue or its evidence failed validation. Inspect the checkpoint; start a new session if source/model changed."
		}
		if v.Status == "" || v.Status == "RUNNING" || v.Status == "STARTING" {
			v.Status = "FAILED"
		}
		s.put(v)
		return
	}
	plan, err := session.ChangePlan()
	if err != nil && session.Status == "NEEDS_REVIEW" {
		plan, err = session.PartialPlan()
	}
	if err != nil {
		if s.LogError != nil {
			s.LogError(err)
		}
		v.Status = "NEEDS_REVIEW"
		v.Message = "No validated plan can be exported from this session."
		s.put(v)
		return
	}
	if err := s.Plans.Save(ctx, plan); err != nil {
		if s.LogError != nil {
			s.LogError(err)
		}
		v.Status = "FAILED"
		v.Message = "Session finished but plan could not be saved. Resume the session to retry saving."
		s.put(v)
		return
	}
	v.Plan = &plan
	v.Message = "Review finished. Model approval is not independent validation."
	s.put(v)
}

func (s *Service) put(v View) {
	if v.Status != "RUNNING" && v.Status != "STARTING" {
		v = v.withLiveElapsed()
	}
	s.mu.Lock()
	s.views[v.SessionID] = v
	s.mu.Unlock()
}

func viewSession(session planning.AgentSession) View {
	b := session.Bundle
	return View{ActiveMillis: session.ActiveMillis, ChangeIndex: session.ChangeIndex, FocusedReview: session.FocusedReview, SourceActivity: sourceActivity(session.Observations), Message: session.ProgressMessage, Mode: sessionMode(session), FullReview: session.FullReview, SessionID: session.ID, AnalysisID: b.AnalysisID, Status: session.Status, Role: session.Role, Turns: session.Turns, Tools: session.Tools, Revisions: session.Revisions, Provider: session.Identity, Context: &b, Decisions: session.Decisions, Events: session.Events, Issues: session.Feedback}
}

func sessionMode(session planning.AgentSession) string {
	if session.FullReview != nil {
		return "full"
	}
	if progress := session.FocusedReview; progress != nil {
		if progress.UltraLight {
			return "ultra-light"
		}
		if progress.Light {
			return "light"
		}
	}
	return "focused"
}

func (s *Service) Get(ctx context.Context, id string) (View, error) {
	s.mu.Lock()
	v, known := s.views[id]
	active := s.activeID == id
	s.mu.Unlock()
	if known && v.Status == "STARTING" {
		return v.withLiveElapsed(), nil
	}
	if known && v.Status != "RUNNING" && v.Status != "STARTING" {
		return v, nil
	}
	session, err := s.Sessions.Load(ctx, id)
	if errors.Is(err, planning.ErrAgentSessionMissing) {
		if known {
			return v.withLiveElapsed(), nil
		}
		return View{}, ErrMissing
	}
	if err != nil {
		return View{}, err
	}
	result := viewSession(session)
	// Current parents already store observations with parent change indices.
	// Only older sessions need reconstruction from numbered child checkpoints.
	if session.FullReview == nil && len(session.Observations) == 0 && (session.FocusedReview == nil || !session.FocusedReview.CombinedReviewOnly) && (len(session.Bundle.Changes) > 1 || session.FocusedReview != nil) {
		batches := len(session.Bundle.Changes)
		if session.FocusedReview != nil {
			batches = session.FocusedReview.BatchesTotal
		}
		for i := 0; i < batches; i++ {
			child, err := s.Sessions.Load(ctx, fmt.Sprintf("%s-focused-%d", session.ID, i))
			if err != nil {
				continue
			}
			result.SourceActivity = append(result.SourceActivity, childActivity(session, child, i)...)
		}
	}
	if active && session.Status == "RUNNING" && session.ActiveChildID != "" {
		if child, err := s.Sessions.Load(ctx, session.ActiveChildID); err == nil {
			result.Role = child.Role
			result.Turns += child.Turns
			result.Tools += child.Tools
			result.Revisions += child.Revisions
			if session.FocusedReview != nil && session.FocusedReview.CombinedReviewOnly {
				result.SourceActivity = append(result.SourceActivity, childActivity(session, child, 0)...)
			}
		}
	}
	if active && (result.Status == "PAUSED" || result.Status == "RUNNING") {
		result.Status = "RUNNING"
		if known {
			result.ActiveMillis = v.withLiveElapsed().ActiveMillis
		}
	}
	if result.Status == "RUNNING" && !active {
		result.Status = "PAUSED"
		result.Message = "Server restarted. Re-enter the source path and resume this checkpoint."
	}
	if result.Status == "COMPLETE" || result.Status == "NEEDS_REVIEW" {
		// Reopen the exact immutable export before applying current draft policy.
		// A missing archive still goes through the normal draft validation path.
		if id, e := session.ArchivedPlanID(); e == nil {
			if saved, e := s.Plans.Get(ctx, id); e == nil {
				result.Plan = &saved
				return result, nil
			}
		}
		plan, e := session.ChangePlan()
		if e != nil && session.Status == "NEEDS_REVIEW" {
			plan, e = session.PartialPlan()
		}
		if e == nil {
			if saved, e := s.Plans.Get(ctx, plan.PlanID); e == nil {
				result.Plan = &saved
			} else {
				result.Status = "FAILED"
				result.Message = "Plan export is missing. Re-enter the source path and resume to retry saving this final session."
			}
		}
	}
	return result, nil
}

func (s *Service) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func safeBase(base context.Context) context.Context {
	if base == nil {
		return context.Background()
	}
	return base
}
func (s *Service) Stop(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeID != id || s.cancel == nil {
		return ErrMissing
	}
	s.cancel()
	return nil
}

// Runtime bundles gain discovered evidence and README context. Engines bind the
// immutable initial context separately; resume must not compare enriched hashes.
func sameChanges(a, b []planning.ChangeInput) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ChangeType != b[i].ChangeType || a[i].ElementID != b[i].ElementID {
			return false
		}
	}
	return true
}
