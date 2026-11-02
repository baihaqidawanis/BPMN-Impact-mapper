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
