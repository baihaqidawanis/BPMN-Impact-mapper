package index_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

func TestLiteralHTTPRouteLinks(t *testing.T) {
	inventory := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "backend/handler.go", Content: []byte("package backend\ntype Handler struct{}\nfunc (h Handler) Register(r interface{ Post(string, func()) }) { r.Post(\"/requests\", h.create) }\nfunc (h Handler) create() {}\n")},
		{RelativePath: "frontend/panel.tsx", Content: []byte("await api(\"/requests\", { method: \"POST\" });\nawait api(\"/requests\", { method: \"DELETE\" });\nawait api(`/requests/${id}`, { method: \"POST\" });\n// await api(\"/requests\", { method: \"POST\" });\nconst note = true; // await api(\"/requests\", { method: \"POST\" });\n")},
	}}
	got := index.DefaultRegistry().Build(inventory)
	var relations []domain.SourceRelation
	for _, artifact := range got.Artifacts {
		if artifact.File == "frontend/panel.tsx" {
			relations = artifact.Relations
		}
	}
	if len(relations) != 1 || relations[0].Kind != domain.RelationHTTPRoute || !strings.Contains(relations[0].TargetID, "method:create") || relations[0].Location != "frontend/panel.tsx:1" || relations[0].TargetLocation != "backend/handler.go:3" {
		t.Fatalf("unexpected route links: %+v", relations)
	}
	reversed := inventory
	reversed.Files = append([]ingest.File(nil), inventory.Files...)
	reversed.Files[0], reversed.Files[1] = reversed.Files[1], reversed.Files[0]
	if !reflect.DeepEqual(got, index.DefaultRegistry().Build(reversed)) {
		t.Fatal("route links changed with file order")
	}
}

func TestLiteralHTTPRouteAbstainsOnAmbiguousHandler(t *testing.T) {
	inventory := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "a/handler.go", Content: []byte("package a\ntype Handler struct{}\nfunc (h Handler) Register(r interface{ Post(string, func()) }) { r.Post(\"/requests\", h.create) }\nfunc (h Handler) create() {}")},
		{RelativePath: "b/handler.go", Content: []byte("package b\ntype Handler struct{}\nfunc (h Handler) Register(r interface{ Post(string, func()) }) { r.Post(\"/requests\", h.create) }\nfunc (h Handler) create() {}")},
		{RelativePath: "ui/panel.ts", Content: []byte("api(\"/requests\", { method: \"POST\" })")},
	}}
	for _, artifact := range index.DefaultRegistry().Build(inventory).Artifacts {
		if artifact.File == "ui/panel.ts" && len(artifact.Relations) > 0 {
			t.Fatalf("ambiguous route was linked: %+v", artifact.Relations)
		}
	}
}
