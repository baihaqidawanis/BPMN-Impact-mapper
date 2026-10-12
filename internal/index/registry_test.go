package index_test

import (
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"reflect"
	"testing"
)

func TestCapabilitiesReflectFallbackAndMixedLanguages(t *testing.T) {
	inventory := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "valid.go", Content: []byte("package p\nfunc Run(){}")},
		{RelativePath: "broken.go", Content: []byte("package p\nfunc (")},
		{RelativePath: "ui.ts", Content: []byte("export function run() {}")},
	}}
	result := index.DefaultRegistry().Build(inventory)
	if len(result.Files) != 3 || len(result.Warnings) != 1 {
		t.Fatalf("result=%+v", result)
	}
	for _, f := range result.Files {
		switch f.File {
		case "valid.go":
			if !reflect.DeepEqual(f.Capabilities, []domain.Capability{domain.ASTSymbols, domain.StaticCalls, domain.TypeRelations}) {
				t.Fatalf("Go capabilities=%v", f)
			}
		case "broken.go":
			if !f.Fallback || f.Adapter != "generic-text" || !reflect.DeepEqual(f.Capabilities, []domain.Capability{domain.TextOnly}) {
				t.Fatalf("false AST claim: %+v", f)
			}
		case "ui.ts":
			if f.Fallback || f.Adapter != "generic-text" {
				t.Fatalf("text is baseline, not failed AST: %+v", f)
			}
		}
	}
	reversed := inventory
	reversed.Files = append([]ingest.File(nil), inventory.Files...)
	reversed.Files[0], reversed.Files[2] = reversed.Files[2], reversed.Files[0]
	if !reflect.DeepEqual(result, index.DefaultRegistry().Build(reversed)) {
		t.Fatal("inventory order changed result")
	}
}

type customAdapter struct{ index.TextAdapter }

func (customAdapter) Name() string { return "custom-test-adapter" }

type failedExtractor struct{}

func (failedExtractor) Name() string                      { return "test-failure" }
func (failedExtractor) Supports(domain.FileAnalysis) bool { return true }
func (failedExtractor) Capabilities() []domain.Capability {
	return []domain.Capability{domain.StaticCalls}
}
func (failedExtractor) Extract(ingest.Inventory, []domain.Artifact) (map[string][]domain.SourceRelation, error) {
	return nil, errors.New("private path must not escape")
}

func TestCustomAdapterAndFailedRelationExtraction(t *testing.T) {
	registry := index.Registry{Adapters: []index.SourceAdapter{customAdapter{}}, Extractors: []index.RelationExtractor{failedExtractor{}}}
	result := registry.Build(ingest.Inventory{Files: []ingest.File{{RelativePath: "odd.custom", Content: []byte("hello")}}})
	if len(result.Artifacts) != 1 || result.Files[0].Adapter != "custom-test-adapter" || len(result.Files[0].Capabilities) != 1 || len(result.Warnings) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Warnings[0] != "relation extractor test-failure failed; source index retained" {
		t.Fatal("raw adapter error leaked")
	}
}
