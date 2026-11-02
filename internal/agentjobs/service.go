// Package agentjobs coordinates asynchronous, single-worker agent sessions.
// Model, source and persistence implementations are injected by composition root.
package agentjobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	Mode       string                       `json:"mode"`
	FullReview *planning.FullReviewProgress `json:"full_review,omitempty"`
	SessionID  string                       `json:"session_id"`
	AnalysisID string                       `json:"analysis_id"`
	Status     string                       `json:"status"`
	Role       planning.AgentRole           `json:"role"`
	Turns      int                          `json:"turns"`
	Tools      int                          `json:"tools"`
	Revisions  int                          `json:"revisions"`
	Provider   planning.ProviderIdentity    `json:"provider"`
	Message    string                       `json:"message,omitempty"`
	Plan       *planning.ChangePlan         `json:"plan,omitempty"`
	Context    *planning.EvidenceBundle     `json:"context,omitempty"`
	Decisions  []planning.CandidateDecision `json:"decisions,omitempty"`
	Events     []planning.AgentEvent        `json:"events,omitempty"`
	Issues     []string                     `json:"issues,omitempty"`
}

type Service struct {
	Model    Model
	Sessions planning.AgentSessionStore
	Plans    planstore.Store
	Source   func(context.Context, string) (Snapshot, error)
	Base     context.Context
	Timeout  time.Duration
	LogError func(error)
	mu       sync.Mutex
	active   bool
	activeID string
	wg       sync.WaitGroup
	views    map[string]View
}

func (s *Service) Check(ctx context.Context) error {
	if s == nil || s.Model == nil {
		return errors.New("model disabled")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return ErrBusy
	}
	return s.Model.Prepare(ctx)
}

func (s *Service) Start(ctx context.Context, report domain.Report, repo string, changeIndex int, id string, modes ...string) (View, error) {
	mode := "focused"
	if len(modes) > 0 && modes[0] != "" {
		mode = modes[0]
	}
	if mode != "focused" && mode != "full" {
		return View{}, ErrBinding
	}
	if s.Model == nil || s.Sessions == nil || s.Plans == nil || s.Source == nil {
		return View{}, errors.New("agent service incomplete")
	}
	if changeIndex < 0 || changeIndex >= len(report.Changes) || repo == "" {
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
	selected.Changes = []domain.Change{report.Changes[changeIndex]}
	bundle := planning.BuildEvidenceBundle(selected, planning.Limits{CandidatesPerChange: 3, RelatedPerChange: 1, EvidencePerCandidate: 2, ExcerptRunes: 120})
	if id != "" {
		stored, err := s.Sessions.Load(ctx, id)
		if err != nil {
			return View{}, err
		}
		if stored.Bundle.AnalysisID != report.AnalysisID || len(stored.Bundle.Changes) != 1 || stored.Bundle.Changes[0].ChangeType != bundle.Changes[0].ChangeType || stored.Bundle.Changes[0].ElementID != bundle.Changes[0].ElementID {
			return View{}, ErrBinding
		}
		storedMode := "focused"
		if stored.FullReview != nil {
			storedMode = "full"
		}
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
	s.views[id] = v
	s.active = true
	s.activeID = id
	s.wg.Add(1)
	s.mu.Unlock()
	base := s.Base
	if base == nil {
		base = context.Background()
	}
	timeout := s.Timeout
	if timeout <= 0 && mode != "full" {
		timeout = 10 * time.Minute
	}
	go func() {
		defer s.wg.Done()
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
		s.mu.Unlock()
	}()
	return v, nil
}

func (s *Service) execute(ctx context.Context, v View, repo string, bundle planning.EvidenceBundle) {
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
		session, runErr = (planning.FullRepositoryEngine{Model: s.Model, Source: fullSource, Store: s.Sessions}).Run(ctx, v.SessionID, bundle)
	} else {
		engine := planning.AgentEngine{Model: s.Model, Tools: source, Store: s.Sessions}
		session, runErr = engine.Run(ctx, v.SessionID, bundle)
	}
	if session.ID != "" {
		v = viewSession(session)
	}
	if runErr != nil {
		if s.LogError != nil {
			s.LogError(runErr)
		}
		if errors.Is(runErr, planning.ErrAgentContextLimit) {
			v.Message = "Evidence exceeds the configured model context. Use a smaller candidate report or increase --agent-context, then start a new session. Source was not silently truncated."
		} else if errors.Is(runErr, planning.ErrAgentModelOutput) {
			v.Message = "Model output was incomplete or invalid. Resume to retry; consumed turns remain charged."
		} else if session.Status == "PAUSED" {
			v.Message = "Model connection interrupted. Restore VPN/tunnel and resume."
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

func (s *Service) put(v View) { s.mu.Lock(); s.views[v.SessionID] = v; s.mu.Unlock() }

func viewSession(session planning.AgentSession) View {
	b := session.Bundle
	mode := "focused"
	if session.FullReview != nil {
		mode = "full"
	}
	return View{Mode: mode, FullReview: session.FullReview, SessionID: session.ID, AnalysisID: b.AnalysisID, Status: session.Status, Role: session.Role, Turns: session.Turns, Tools: session.Tools, Revisions: session.Revisions, Provider: session.Identity, Context: &b, Decisions: session.Decisions, Events: session.Events, Issues: session.Feedback}
}

func (s *Service) Get(ctx context.Context, id string) (View, error) {
	s.mu.Lock()
	v, known := s.views[id]
	active := s.activeID == id
	s.mu.Unlock()
	if known && v.Status == "STARTING" {
		return v, nil
	}
	if known && v.Status != "RUNNING" && v.Status != "STARTING" {
		return v, nil
	}
	session, err := s.Sessions.Load(ctx, id)
	if errors.Is(err, planning.ErrAgentSessionMissing) {
		if known {
			return v, nil
		}
		return View{}, ErrMissing
	}
	if err != nil {
		return View{}, err
	}
	result := viewSession(session)
	if active && (result.Status == "PAUSED" || result.Status == "RUNNING") {
		result.Status = "RUNNING"
	}
	if result.Status == "RUNNING" && !active {
		result.Status = "PAUSED"
		result.Message = "Server restarted. Re-enter the source path and resume this checkpoint."
	}
	if result.Status == "COMPLETE" || result.Status == "NEEDS_REVIEW" {
		if plan, e := session.ChangePlan(); e == nil {
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
