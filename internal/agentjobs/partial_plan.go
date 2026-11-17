package agentjobs

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// ExportDraft does not prepare models, inspect live source or resume inference.
func (s *Service) ExportDraft(ctx context.Context, id string) (planning.ChangePlan, error) {
	s.mu.Lock()
	if s.active && s.activeID == id {
		s.mu.Unlock()
		return planning.ChangePlan{}, ErrBusy
	}
	s.mu.Unlock()
	session, err := s.Sessions.Load(ctx, id)
	if err != nil {
		return planning.ChangePlan{}, err
	}
	p, err := session.PartialPlan()
	if err != nil {
		return planning.ChangePlan{}, err
	}
	if err = s.Plans.Save(ctx, p); err != nil {
		return planning.ChangePlan{}, err
	}
	s.mu.Lock()
	if s.views == nil {
		s.views = map[string]View{}
	}
	v, ok := s.views[id]
	if !ok {
		v = viewSession(session)
		if v.Status == "RUNNING" || v.Status == "STARTING" {
			v.Status = "PAUSED"
		}
	}
	v.Plan = &p
	s.views[id] = v
	s.mu.Unlock()
	return p, nil
}
