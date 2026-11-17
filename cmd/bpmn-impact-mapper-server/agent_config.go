package main

import (
	"context"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Parse explicit role options before creating directories or starting services.
func agentRoleOverrides(models, profiles, thinking map[planning.AgentRole]string) (map[planning.AgentRole]llm.OllamaRoleOverride, error) {
	result := map[planning.AgentRole]llm.OllamaRoleOverride{}
	for _, role := range []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer} {
		override := llm.OllamaRoleOverride{Model: models[role], Profile: profiles[role]}
		switch thinking[role] {
		case "", "inherit":
		case "on", "off":
			value := thinking[role] == "on"
			override.Think = &value
		default:
			return nil, fmt.Errorf("%s thinking must be inherit, on or off", role)
		}
		if override.Model != "" || override.Profile != "" || override.Think != nil {
			result[role] = override
		}
	}
	return result, nil
}

// The HTTP layer can change generation options but never the server-owned URL.
func webAgentConfiguration(base llm.OllamaAgentConfig, overrides map[planning.AgentRole]llm.OllamaRoleOverride) *agentjobs.Configuration {
	settings := agentjobs.Settings{Context: base.ContextSize, Output: base.MaxOutput}
	slots := []*agentjobs.RoleSettings{&settings.Analyst, &settings.Planner, &settings.Reviewer}
	for i, role := range []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer} {
		v := agentjobs.RoleSettings{Model: base.Model, Thinking: base.Think, Profile: base.Profile}
		o := overrides[role]
		if o.Model != "" {
			v.Model = o.Model
		}
		if o.Profile != "" {
			v.Profile = o.Profile
		}
		if o.Think != nil {
			v.Thinking = *o.Think
		}
		*slots[i] = v
	}
	return &agentjobs.Configuration{Current: settings, List: func(ctx context.Context) ([]string, error) {
		catalog, err := llm.NewOllamaAgent(base)
		if err != nil {
			return nil, err
		}
		return catalog.InstalledModelNames(ctx)
	}, Build: func(v agentjobs.Settings) (agentjobs.Model, error) {
		if err := v.Validate(); err != nil {
			return nil, err
		}
		config := base
		config.ContextSize = v.Context
		config.MaxOutput = v.Output
		routes := map[planning.AgentRole]llm.OllamaRoleOverride{}
		for role, option := range map[planning.AgentRole]agentjobs.RoleSettings{planning.Analyst: v.Analyst, planning.Planner: v.Planner, planning.Reviewer: v.Reviewer} {
			think := option.Thinking
			routes[role] = llm.OllamaRoleOverride{Model: option.Model, Profile: option.Profile, Think: &think}
		}
		return llm.NewOllamaRoleRouter(config, routes)
	}}
}
