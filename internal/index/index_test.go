package index_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

func TestBuildUsesNativeGoAndGenericAdapters(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "repository")
	inventory, err := ingest.Build(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	artifacts, warnings := index.Build(inventory)
	if len(warnings) != 0 {
		t.Fatalf("warnings=%v", warnings)
	}
	foundMethod, foundTypeScript := false, false
	for _, artifact := range artifacts {
		if artifact.Symbol.Name == "ValidateRequest" && artifact.Symbol.Kind == "function" && artifact.Language == "Go" {
			foundMethod = true
		}
		if artifact.File == "web/request-form.ts" && artifact.Language == "TypeScript" {
			foundTypeScript = true
		}
	}
	if !foundMethod || !foundTypeScript {
		t.Fatalf("native=%v generic=%v artifacts=%#v", foundMethod, foundTypeScript, artifacts)
	}
}
