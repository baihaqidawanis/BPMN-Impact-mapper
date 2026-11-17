package main

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestWebSettingsKeepCLIDefaultsAndBindThinkingToIdentity(t *testing.T) {
	on := true
	c := webAgentConfiguration(llm.OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "analyst-fixture", ContextSize: 32768, MaxOutput: 6144, Profile: "coding"}, map[planning.AgentRole]llm.OllamaRoleOverride{planning.Planner: {Model: "planner-fixture"}, planning.Reviewer: {Think: &on}})
	if c.Current.Planner.Model != "planner-fixture" || !c.Current.Reviewer.Thinking || c.Current.Analyst.Thinking {
		t.Fatal("CLI role defaults were lost")
	}
	a, err := c.Build(c.Current)
	if err != nil {
		t.Fatal(err)
	}
	next := c.Current
	next.Reviewer.Thinking = false
	b, err := c.Build(next)
	if err != nil {
		t.Fatal(err)
	}
	if a.Identity() == b.Identity() {
		t.Fatal("thinking change did not alter session identity")
	}
}

func TestRoleThinkingDoesNotForceOtherRolesOrLoseExplicitFalse(t *testing.T) {
	roles, err := agentRoleOverrides(nil, nil, map[planning.AgentRole]string{planning.Reviewer: "on"})
	if err != nil || len(roles) != 1 || roles[planning.Reviewer].Think == nil || !*roles[planning.Reviewer].Think {
		t.Fatal(roles, err)
	}
	roles, err = agentRoleOverrides(nil, nil, map[planning.AgentRole]string{planning.Reviewer: "off", planning.Analyst: "inherit"})
	if err != nil || len(roles) != 1 || roles[planning.Reviewer].Think == nil || *roles[planning.Reviewer].Think {
		t.Fatal(roles, err)
	}
	if _, err = agentRoleOverrides(nil, nil, map[planning.AgentRole]string{planning.Reviewer: "auto"}); err == nil {
		t.Fatal("ambiguous thinking option accepted")
	}
}
