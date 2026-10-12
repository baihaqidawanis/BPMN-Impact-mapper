package configmapping_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/configmapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

func TestLoadAndAttachUsesOnlyIndexedTargets(t *testing.T) {
	mapper, err := configmapping.Load(filepath.Join("..", "..", "testdata", "config", "mapping.json"))
	if err != nil {
		t.Fatal(err)
	}
	changes := []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-VALIDATE", Type: "serviceTask"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}}
	artifacts := []domain.Artifact{{File: "internal/application/request.go", Module: "internal", Layer: "application", Symbol: domain.Symbol{Name: "ValidateRequest", Kind: "function"}}}
	attached, warnings, err := mapper.Attach(context.Background(), changes, artifacts, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || len(attached[0].Impacts) != 1 {
		t.Fatalf("impacts=%#v warnings=%#v", attached[0].Impacts, warnings)
	}
	impact := attached[0].Impacts[0]
	if impact.RequiredAction != domain.ActionAdd || impact.Evidence[0].Type != "CONFIG_MAPPING" {
		t.Fatalf("impact=%#v", impact)
	}
	attached, warnings, err = mapper.Attach(context.Background(), changes, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(attached[0].Impacts) != 0 || len(warnings) != 1 {
		t.Fatalf("missing target was invented: impacts=%#v warnings=%#v", attached[0].Impacts, warnings)
	}
}

func TestConfigValidationRejectsUnsafeAndAmbiguousRules(t *testing.T) {
	tests := []configmapping.Config{
		{Version: "2"},
		{Version: "1", Rules: []configmapping.Rule{{File: "file.go", Action: domain.ActionVerify, Rationale: "missing selector"}}},
		{Version: "1", Rules: []configmapping.Rule{{ElementID: "TASK", File: "../file.go", Action: domain.ActionVerify, Rationale: "unsafe path"}}},
		{Version: "1", Rules: []configmapping.Rule{{ElementID: "TASK", File: "file.go", Action: "INVALID", Rationale: "bad action"}}},
	}
	for index, config := range tests {
		if err := config.Validate(); err == nil {
			t.Errorf("case %d accepted invalid config", index)
		}
	}
}
