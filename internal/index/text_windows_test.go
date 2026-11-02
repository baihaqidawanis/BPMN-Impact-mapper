package index_test

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

func TestTextWindowsCoverUnicodeSourceWithBoundedCountAndRealLines(t *testing.T) {
	text := strings.Repeat("αβγ 文字\n", 3000)
	file := ingest.File{RelativePath: "src/screen.unknown", Content: []byte(text)}
	a := index.TextWindowAdapter{}
	got, err := a.Index(file)
	if err != nil || len(got) != 9 || got[0].Content != text {
		t.Fatalf("coverage/count: %d %v", len(got), err)
	}
	var joined strings.Builder
	line := 1
	for _, w := range got[1:] {
		if !utf8.ValidString(w.Content) || w.Symbol.Name != "" || w.Symbol.Kind != "text_window" || w.Symbol.StartLine != line {
			t.Fatalf("invented code name or wrong source line: %+v", w.Symbol)
		}
		joined.WriteString(w.Content)
		line += strings.Count(w.Content, "\n")
		if w.Symbol.EndLine != line {
			t.Fatal("wrong end line")
		}
	}
	if joined.String() != text {
		t.Fatal("lost/overlapping source text")
	}
	again, _ := a.Index(file)
	if !reflect.DeepEqual(got, again) {
		t.Fatal("nondeterministic windows")
	}
}

func TestTextWindowsKeepNativeGoAndDoNotInventASTCapabilities(t *testing.T) {
	a := index.TextWindowAdapter{}
	if a.Supports(ingest.File{RelativePath: "source.go"}) || !reflect.DeepEqual(a.Capabilities(), []domain.Capability{domain.TextOnly}) {
		t.Fatal("false AST support")
	}
	got, _ := a.Index(ingest.File{RelativePath: "short.py", Content: []byte("print('hello')")})
	if len(got) != 1 {
		t.Fatal("duplicated short source")
	}
	r := index.DefaultRegistry()
	identity := r.Identity()
	r.Adapters = append(r.Adapters, a)
	if r.Identity() == identity {
		t.Fatal("index policy identity unchanged")
	}
	result := r.Build(ingest.Inventory{Files: []ingest.File{{RelativePath: "native.go", Content: []byte("package p\nfunc F() {}")}, {RelativePath: "other.custom", Content: []byte(strings.Repeat("item ", 500))}}})
	if result.Files[0].Adapter != "go-syntax" || result.Files[1].Adapter != a.Name() || len(result.Files[1].Capabilities) != 1 || result.Files[1].Capabilities[0] != domain.TextOnly {
		t.Fatal("adapter routing/capability regression")
	}
}

func TestCodeWindowsScopeLeavesContractsAndUnknownLanguagesOnFallback(t *testing.T) {
	a := index.TextWindowAdapter{CodeOnly: true}
	for _, f := range []string{"source.tsx", "src/module.java", "src/service.py", "src/controller.cs", "src/run.js"} {
		if !a.Supports(ingest.File{RelativePath: f}) {
			t.Fatal("missing generic code support", f)
		}
	}
	for _, f := range []string{"api/contract.yaml", "schema.sql", "guide.md", "other.unknown", "native.go"} {
		if a.Supports(ingest.File{RelativePath: f}) {
			t.Fatal("changed unrelated adapter", f)
		}
	}
	if a.Name() == (index.TextWindowAdapter{}).Name() {
		t.Fatal("missing scope identity")
	}
}

func TestTextWindowRegistryRetainsLiteralRouteEvidenceOnParentFile(t *testing.T) {
	r := index.DefaultRegistry()
	r.Adapters = append(r.Adapters, index.TextWindowAdapter{})
	result := r.Build(ingest.Inventory{Files: []ingest.File{
		{RelativePath: "backend/handler.go", Content: []byte("package backend\ntype Handler struct{}\nfunc (h Handler) Register(r interface{ Post(string, func()) }) { r.Post(\"/claims\", h.create) }\nfunc (h Handler) create() {}\n")},
		{RelativePath: "ui/panel.tsx", Content: []byte(strings.Repeat("// padding\n", 150) + "await api(\"/claims\", { method: \"POST\" });\n")},
	}})
	found := false
	for _, artifact := range result.Artifacts {
		if artifact.ID != "ui/panel.tsx" {
			continue
		}
		if len(artifact.Relations) != 1 || artifact.Relations[0].Location != "ui/panel.tsx:151" {
			t.Fatalf("lost original route line: %+v", artifact.Relations)
		}
		found = true
	}
	if !found {
		t.Fatal("lost parent source artifact")
	}
}
