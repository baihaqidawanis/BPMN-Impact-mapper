package agentjobs

import (
	"context"
	"errors"
	"strings"
)

var ErrConfiguration = errors.New("invalid agent configuration")

type RoleSettings struct {
	Model    string `json:"model"`
	Thinking bool   `json:"thinking"`
	Profile  string `json:"profile"`
}

// Settings contains only generation options. Endpoint and credentials remain
// server-owned; applying settings never installs or loads a model.
type Settings struct {
	Analyst  RoleSettings `json:"analyst"`
	Planner  RoleSettings `json:"planner"`
	Reviewer RoleSettings `json:"reviewer"`
	Context  int          `json:"context"`
	Output   int          `json:"output"`
}

func (v Settings) Validate() error {
	if v.Context < 4096 || v.Context > 32768 || v.Output < 32 || v.Output > 8192 || v.Output >= v.Context {
		return ErrConfiguration
	}
	for _, role := range []RoleSettings{v.Analyst, v.Planner, v.Reviewer} {
		if role.Model == "" || strings.TrimSpace(role.Model) != role.Model || len(role.Model) > 200 || strings.ContainsAny(role.Model, "\r\n\x00") || (role.Profile != "coding" && role.Profile != "baseline") {
			return ErrConfiguration
		}
	}
	return nil
}

type Configuration struct {
	Current Settings
	Build   func(Settings) (Model, error)
	List    func(context.Context) ([]string, error)
}

func (s *Service) AvailableModels(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return nil, ErrBusy
	}
	if s.Configuration == nil || s.Configuration.List == nil {
		return nil, ErrConfiguration
	}
	return s.Configuration.List(ctx)
}

func (s *Service) Settings() (Settings, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Configuration == nil {
		return Settings{}, false
	}
	return s.Configuration.Current, true
}

// Configure is atomic with Start/Check. Prepare validates installed metadata
// only. Failure preserves the previous model and settings.
func (s *Service) Configure(ctx context.Context, v Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return ErrBusy
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Configuration == nil || s.Configuration.Build == nil {
		return ErrConfiguration
	}
	if err := v.Validate(); err != nil {
		return err
	}
	model, err := s.Configuration.Build(v)
	if err != nil {
		return err
	}
	if model == nil {
		return ErrConfiguration
	}
	if err = model.Prepare(ctx); err != nil {
		return err
	}
	s.Model = model
	s.Configuration.Current = v
	return nil
}

func (s *Service) InspectModel(ctx context.Context) (Model, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	model := s.Model
	if model == nil {
		return nil, errors.New("model disabled")
	}
	if s.active {
		return model, ErrBusy
	}
	return model, model.Prepare(ctx)
}
