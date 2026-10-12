package index

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

// SourceAdapter produces normalized source artifacts. First matching adapter wins.
type SourceAdapter interface {
	Adapter
	Name() string
	Capabilities() []domain.Capability
}

// RelationExtractor operates only on successfully indexed files it supports.
// It returns outgoing relations by artifact ID without mutating the input.
type RelationExtractor interface {
	Name() string
	Supports(domain.FileAnalysis) bool
	Capabilities() []domain.Capability
	Extract(ingest.Inventory, []domain.Artifact) (map[string][]domain.SourceRelation, error)
}

type Registry struct {
	Adapters   []SourceAdapter
	Extractors []RelationExtractor
}

type Result struct {
	Artifacts []domain.Artifact
	Files     []domain.FileAnalysis
	Warnings  []string
}

// Adapter/extractor names must include a version when their behavior changes.
// Registration order is material because first matching adapter wins.
func (registry Registry) Identity() string {
	parts := []string{"generic-text-v1"}
	for _, a := range registry.Adapters {
		parts = append(parts, "adapter:"+a.Name())
	}
	for _, e := range registry.Extractors {
		parts = append(parts, "extractor:"+e.Name())
	}
	data, _ := json.Marshal(parts)
	return string(data)
}

func DefaultRegistry() Registry {
	return Registry{Adapters: []SourceAdapter{GoAdapter{}}, Extractors: []RelationExtractor{GoCallExtractor{}}}
}

func (TextAdapter) Name() string                      { return "generic-text" }
func (TextAdapter) Capabilities() []domain.Capability { return []domain.Capability{domain.TextOnly} }
func (GoAdapter) Name() string                        { return "go-syntax" }
func (GoAdapter) Capabilities() []domain.Capability   { return []domain.Capability{domain.ASTSymbols} }

func (registry Registry) Build(inventory ingest.Inventory) Result {
	result := Result{Artifacts: []domain.Artifact{}, Files: []domain.FileAnalysis{}, Warnings: []string{}}
	files := append([]ingest.File(nil), inventory.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	for _, file := range files {
		var selected SourceAdapter = TextAdapter{}
		for _, adapter := range registry.Adapters {
			if adapter.Supports(file) {
				selected = adapter
				break
			}
		}
		artifacts, err := selected.Index(file)
		fallback := err != nil
		if err != nil {
			// Avoid leaking an adapter error containing absolute paths or file contents.
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: adapter %s failed; using generic text", file.RelativePath, selected.Name()))
			selected = TextAdapter{}
			artifacts, _ = selected.Index(file)
		}
		result.Artifacts = append(result.Artifacts, artifacts...)
		capabilities := append([]domain.Capability(nil), selected.Capabilities()...)
		sort.Slice(capabilities, func(i, j int) bool { return capabilities[i] < capabilities[j] })
		result.Files = append(result.Files, domain.FileAnalysis{File: file.RelativePath, Language: language(file.RelativePath), Adapter: selected.Name(), Capabilities: capabilities, Fallback: fallback})
	}
	sort.Slice(result.Artifacts, func(i, j int) bool { return result.Artifacts[i].ID < result.Artifacts[j].ID })
	for _, extractor := range registry.Extractors {
		eligible := map[string]bool{}
		for _, file := range result.Files {
			if extractor.Supports(file) {
				eligible[file.File] = true
			}
		}
		if len(eligible) == 0 {
			continue
		}
		selected := []domain.Artifact{}
		for _, a := range result.Artifacts {
			if eligible[a.File] {
				selected = append(selected, a)
			}
		}
		relations, err := extractor.Extract(inventory, selected)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("relation extractor %s failed; source index retained", extractor.Name()))
			continue
		}
		ids := map[string]bool{}
		for _, a := range result.Artifacts {
			ids[a.ID] = true
		}
		for i := range result.Artifacts {
			a := &result.Artifacts[i]
			if !eligible[a.File] {
				continue
			}
			for _, edge := range relations[a.ID] {
				if ids[edge.TargetID] && edge.Location != "" && edge.Excerpt != "" {
					a.Relations = append(a.Relations, edge)
				}
			}
			sort.Slice(a.Relations, func(i, j int) bool {
				x, y := a.Relations[i], a.Relations[j]
				if x.Kind != y.Kind {
					return x.Kind < y.Kind
				}
				if x.TargetID != y.TargetID {
					return x.TargetID < y.TargetID
				}
				if x.Location != y.Location {
					return x.Location < y.Location
				}
				return x.Excerpt < y.Excerpt
			})
		}
		for i := range result.Files {
			if eligible[result.Files[i].File] {
				result.Files[i].Capabilities = append(result.Files[i].Capabilities, extractor.Capabilities()...)
				sort.Slice(result.Files[i].Capabilities, func(a, b int) bool {
					return result.Files[i].Capabilities[a] < result.Files[i].Capabilities[b]
				})
			}
		}
	}
	sort.Strings(result.Warnings)
	return result
}

type GoCallExtractor struct{}

func (GoCallExtractor) Name() string { return "go-static-calls-v2" }
func (GoCallExtractor) Capabilities() []domain.Capability {
	return []domain.Capability{domain.StaticCalls, domain.TypeRelations}
}
func (GoCallExtractor) Supports(file domain.FileAnalysis) bool {
	return file.Adapter == "go-syntax" && !file.Fallback
}
func (GoCallExtractor) Extract(inventory ingest.Inventory, artifacts []domain.Artifact) (map[string][]domain.SourceRelation, error) {
	copies := append([]domain.Artifact(nil), artifacts...)
	for i := range copies {
		copies[i].Relations = nil
	}
	attachCalls(inventory, copies)
	result := map[string][]domain.SourceRelation{}
	for _, a := range copies {
		result[a.ID] = a.Relations
	}
	return result, nil
}
