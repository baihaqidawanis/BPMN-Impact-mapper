package ingest_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

func TestBuildHonorsSecurityAndIgnoreRules(t *testing.T) {
	root := t.TempDir()
	write(t, root, "z.go", "package sample")
	write(t, root, "a.txt", "allowed")
	write(t, root, ".env", "SECRET=value")
	write(t, root, "private.key", "key")
	write(t, root, "node_modules/library.js", "excluded")
	write(t, root, "docs/PROCESS-MAPPING.md", "leakage")
	write(t, root, "skip/ignored.txt", "ignored")
	writeBytes(t, root, "image.bin", []byte{0, 1, 2})
	if err := os.Symlink(filepath.Join(root, "a.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Logf("symlink creation unavailable on this platform: %v", err)
	}
	result, err := ingest.Build(context.Background(), root, ingest.Options{Ignore: []string{"skip/**"}})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, file := range result.Files {
		paths = append(paths, file.RelativePath)
	}
	if !reflect.DeepEqual(paths, []string{"a.txt", "z.go"}) {
		t.Fatalf("paths=%v", paths)
	}
	second, err := ingest.Build(context.Background(), root, ingest.Options{Ignore: []string{"skip/**"}})
	if err != nil || !reflect.DeepEqual(result.Files, second.Files) {
		t.Fatalf("inventory is not deterministic: %v", err)
	}
}

func TestBuildRejectsLimitsAndCancellation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "one.txt", "12345")
	write(t, root, "two.txt", "67890")
	if _, err := ingest.Build(context.Background(), root, ingest.Options{MaxFiles: 1}); err == nil {
		t.Fatal("expected file limit error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ingest.Build(ctx, root, ingest.Options{}); err == nil {
		t.Fatal("expected cancellation")
	}
}

func TestBuildRejectsSymlinkRootWhenAvailable(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := ingest.Build(context.Background(), link, ingest.Options{}); err == nil {
		t.Fatal("expected symlink root rejection")
	}
}

func write(t *testing.T, root, relative, value string) {
	t.Helper()
	writeBytes(t, root, relative, []byte(value))
}

func writeBytes(t *testing.T, root, relative string, value []byte) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, value, 0o600); err != nil {
		t.Fatal(err)
	}
}
