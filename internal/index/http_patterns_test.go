package index_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"testing"
)

func patternRegistry() index.Registry {
	r := index.DefaultRegistry()
	r.Extractors = []index.RelationExtractor{index.GoCallExtractor{}, index.HTTPRouteExtractor{Parameterized: true}}
	return r
}
func TestParameterizedRouteLinksUniqueClientAndAbstainsOnAmbiguity(t *testing.T) {
	for _, duplicate := range []string{"", `r.Post("/docs/{key}/{verb}",h.other)`, `r.Post("/docs/{key}/{verb}",missing)`} {
		inventory := ingest.Inventory{Files: []ingest.File{
			{RelativePath: "backend/handler.go", Content: []byte("package b\ntype Handler struct{}\nfunc(h Handler) Register(r interface{Post(string,func())}) {r.Post(\"/docs/{id}/{action}\",h.run);" + duplicate + "}\nfunc(h Handler) run() {}\nfunc(h Handler) other() {}")},
			{RelativePath: "ui/panel.tsx", Content: []byte("await api(`/docs/${document.id}/${action}`, {method: 'POST'});\n")},
		}}
		result := patternRegistry().Build(inventory)
		for _, a := range result.Artifacts {
			if a.File == "ui/panel.tsx" {
				if duplicate != "" {
					if len(a.Relations) != 0 {
						t.Fatalf("ambiguous route linked: %+v", a.Relations)
					}
				} else if len(a.Relations) != 1 || a.Relations[0].Kind != domain.RelationHTTPPattern || a.Relations[0].Location != "ui/panel.tsx:1" {
					t.Fatalf("missing pattern provenance: %+v", a.Relations)
				}
			}
		}
		legacy := index.DefaultRegistry().Build(inventory)
		for _, a := range legacy.Artifacts {
			if a.File == "ui/panel.tsx" && len(a.Relations) != 0 {
				t.Fatal("changed legacy literal adapter")
			}
		}
	}
}

func TestParameterizedRoutesIgnoreCommentsStringsAndUnsupportedExpressions(t *testing.T) {
	code := "// api(`/docs/${id}/${action}`, {method:'POST'});\n/*\napi(`/docs/${id}/${action}`, {method:'POST'});\n*/\nconst note = \"api(`/docs/${id}/${action}`, {method:'POST'})\";\n"
	code += "api(`/docs/${encodeURIComponent(id)}/${action}`, {method:'POST'});\napi(`/${id}/${action}`, {method:'POST'});\napi(`/docs/${id}/${action}`, {method: verb});\n"
	result := patternRegistry().Build(ingest.Inventory{Files: []ingest.File{
		{RelativePath: "handler.go", Content: []byte("package b\ntype H struct{}\nfunc(h H) Register(r interface{Post(string,func())}){r.Post(\"/docs/{id}/{action}\",h.run)}\nfunc(h H)run(){}")},
		{RelativePath: "ui.ts", Content: []byte(code)},
	}})
	for _, a := range result.Artifacts {
		if a.File == "ui.ts" && len(a.Relations) != 0 {
			t.Fatalf("invented dynamic route: %+v", a.Relations)
		}
	}
}
