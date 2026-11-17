package agenttools_test

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRepositoryChunksCoverAllEligibleBytesAndRemainImmutable(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"flow.go": "package fixture\n" + strings.Repeat("// 中文🙂 repeated source\n", 700), "empty.txt": "", "view.tsx": strings.Repeat("component;", 900)}
	for file, content := range files {
		if err := os.WriteFile(filepath.Join(root, file), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=excluded"), 0600)
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	chunks := snapshot.RepositoryChunks()
	covered := map[string][]bool{}
	last := map[string]int{}
	for file, text := range files {
		covered[file] = make([]bool, len(text))
	}
	for _, c := range chunks {
		text, ok := files[c.File]
		if !ok {
			t.Fatal("excluded source leaked", c.File)
		}
		if len(c.Content) > 4096 || !utf8.ValidString(c.Content) || text[c.StartByte:c.EndByte] != c.Content {
			t.Fatal("invalid range", c)
		}
		for i := c.StartByte; i < c.EndByte; i++ {
			covered[c.File][i] = true
		}
		if c.LastInFile {
			last[c.File]++
		}
	}
	for file, bits := range covered {
		if last[file] != 1 {
			t.Fatal("missing/duplicate last chunk", file)
		}
		for _, bit := range bits {
			if !bit {
				t.Fatal("byte omitted", file)
			}
		}
	}
	os.WriteFile(filepath.Join(root, "view.tsx"), []byte("changed"), 0600)
	if !reflect.DeepEqual(chunks, snapshot.RepositoryChunks()) {
		t.Fatal("snapshot changed after live file write")
	}
}

func TestBulkScanSkipsProseAndDependencyManifests(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"README.md", "notes.mdx", "guide.rst", "package.json", "package-lock.json", "go.mod", "go.sum", "service.go", "schema.sql", "settings.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("allowed textual fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, c := range snapshot.RepositoryChunks() {
		names[c.File] = true
	}
	for _, name := range []string{"README.md", "notes.mdx", "guide.rst", "package.json", "package-lock.json", "go.mod", "go.sum"} {
		if names[name] {
			t.Fatal("unnecessary bulk input", name)
		}
	}
	for _, name := range []string{"service.go", "schema.sql", "settings.json"} {
		if !names[name] {
			t.Fatal("implementation excluded", name)
		}
	}
}

func TestRootReadmeIsBoundedAndOptional(t *testing.T) {
	root := t.TempDir()
	text := strings.Repeat("overview ", 1500)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	readme := snapshot.RootReadme()
	if readme == nil || !readme.Truncated || len(readme.Content) > 8192 || !utf8.ValidString(readme.Content) {
		t.Fatal("unbounded README", readme)
	}
	for _, chunk := range snapshot.RepositoryChunks() {
		if chunk.File == "README.md" {
			t.Fatal("README in exhaustive scan")
		}
	}
	empty, err := agenttools.New(context.Background(), t.TempDir(), ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if empty.RootReadme() != nil {
		t.Fatal("invented README")
	}
}
