package filestore

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
	"os"
	"path/filepath"
	"testing"
)

func TestBPMNLibraryRejectsSymlinkAndBadNames(t *testing.T) {
	root := t.TempDir()
	s := BPMNLibrary{Root: root}
	if _, err := s.Get("../secret"); err == nil {
		t.Fatal("bad ID accepted")
	}
	if err := s.Put(bpmnlib.Entry{ID: "../secret"}); err == nil {
		t.Fatal("unsafe filename accepted")
	}
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	outside := filepath.Join(t.TempDir(), "outside.json")
	os.WriteFile(outside, []byte(`{}`), 0600)
	if err := os.Symlink(outside, filepath.Join(root, id+".json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := s.Get(id); err == nil {
		t.Fatal("symlink followed")
	}
}
