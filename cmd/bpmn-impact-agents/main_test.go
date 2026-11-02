package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectDataDirectoryInsideRepoBeforeNetwork(t *testing.T) {
	repo := t.TempDir()
	err := run([]string{"--repo", repo, "--context", "missing.json", "--session", "safe", "--model", "fixture", "--data-dir", filepath.Join(repo, "nested", "data")})
	if err == nil || !strings.Contains(err.Error(), "outside analyzed repository") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "nested")); !os.IsNotExist(err) {
		t.Fatal("runner wrote into source repo")
	}
}

func TestRequiredArgumentsAndParentPath(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("missing arguments accepted")
	}
	if !startsParent(".."+string(filepath.Separator)+"safe") || startsParent("..not-parent") {
		t.Fatal("invalid containment test")
	}
}
