package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// PreparedAgent inspects installed metadata without loading weights or sending
// repository source. Routing belongs to the adapter composition, not planning.
type PreparedAgent interface {
	planning.AgentModel
	Prepare(context.Context) error
}

// RoleRouter fixes one model per role for the session. It never chooses a model
// based on reference answers or silently retries using a different model.
type RoleRouter struct {
	models map[planning.AgentRole]PreparedAgent
	gate   chan struct{}
}

var agentRoles = []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer}

type OllamaRoleOverride struct {
	Model   string
	Profile string
	Think   *bool // nil inherits the shared setting; false is an explicit override
}

// NewOllamaRoleRouter composes fixed role settings without loading models.
// Inference remains serialized by RoleRouter; no role silently falls back
// after a configured model or capability fails validation.
func NewOllamaRoleRouter(base OllamaAgentConfig, overrides map[planning.AgentRole]OllamaRoleOverride) (*RoleRouter, error) {
	for role := range overrides {
		if role != planning.Analyst && role != planning.Planner && role != planning.Reviewer {
			return nil, errors.New("unknown Ollama role override")
		}
	}
	fallback, err := NewOllamaAgent(base)
	if err != nil {
		return nil, err
	}
	routes := map[planning.AgentRole]PreparedAgent{}
	for _, role := range agentRoles {
		config := base
		override := overrides[role]
		if override.Model != "" {
			config.Model = override.Model
		}
		if override.Profile != "" {
			config.Profile = override.Profile
		}
		if override.Think != nil {
			config.Think = *override.Think
		}
		routes[role] = fallback
		if config.Model != base.Model || config.Profile != base.Profile || config.Think != base.Think {
			routes[role], err = NewOllamaAgent(config)
			if err != nil {
				return nil, err
			}
		}
	}
	return NewRoleRouter(routes)
}

func NewRoleRouter(models map[planning.AgentRole]PreparedAgent) (*RoleRouter, error) {
	if len(models) != len(agentRoles) {
		return nil, errors.New("exactly Analyst, Planner and Reviewer model routes required")
	}
	copy := map[planning.AgentRole]PreparedAgent{}
	for _, role := range agentRoles {
		if models[role] == nil {
			return nil, errors.New("missing model for agent role")
		}
		copy[role] = models[role]
	}
	return &RoleRouter{models: copy, gate: make(chan struct{}, 1)}, nil
}

func (r *RoleRouter) lock(ctx context.Context) error {
	select {
	case r.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-r.gate
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *RoleRouter) Prepare(ctx context.Context) error {
	if err := r.lock(ctx); err != nil {
		return err
	}
	defer func() { <-r.gate }()
	for _, role := range agentRoles {
		if err := r.models[role].Prepare(ctx); err != nil {
			return err
		}
	}
	return nil
}

// RoleIdentities preserves the actual aliases/profiles behind the composite
// identity. Callers receive a copy; changing it cannot mutate session routes.
func (r *RoleRouter) RoleIdentities() map[planning.AgentRole]planning.ProviderIdentity {
	identities := map[planning.AgentRole]planning.ProviderIdentity{}
	for _, role := range agentRoles {
		identities[role] = r.models[role].Identity()
	}
	return identities
}

func (r *RoleRouter) PreferFileReview() bool {
	policy, ok := r.models[planning.Reviewer].(planning.FileReviewPolicy)
	return ok && policy.PreferFileReview()
}

func (r *RoleRouter) Identity() planning.ProviderIdentity {
	identities := r.RoleIdentities()
	names := []string{}
	for _, role := range agentRoles {
		names = append(names, string(role)+"="+identities[role].Model)
	}
	raw, _ := json.Marshal([]any{"role-router-v1", identities})
	hash := sha256.Sum256(raw)
	return planning.ProviderIdentity{Provider: "ollama-local-role-router", Model: strings.Join(names, "; "), PromptVersion: "role-router-v1-" + hex.EncodeToString(hash[:])[:24]}
}

func (r *RoleRouter) Turn(ctx context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	model, ok := r.models[turn.Role]
	if !ok {
		return planning.AgentReply{}, errors.New("unknown agent role; no inference sent")
	}
	if err := r.lock(ctx); err != nil {
		return planning.AgentReply{}, err
	}
	defer func() { <-r.gate }()
	return model.Turn(ctx, turn)
}
