package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestAgentCheckpointUpdateAndIntegrity(t *testing.T) {
	store := AgentSessionStore{Root: t.TempDir()}
	ctx := context.Background()
	s := planning.AgentSession{ID: "session", Version: "1", Status: "RUNNING"}
	if err := store.Save(ctx, s); err != nil {
		t.Fatal(err)
	}
	s.Turns = 2
	s.Status = "PAUSED"
	if err := store.Save(ctx, s); err != nil {
		t.Fatal("checkpoint update:", err)
	}
	loaded, err := store.Load(ctx, "session")
	if err != nil || loaded.Turns != 2 || loaded.Status != "PAUSED" {
		t.Fatal("checkpoint lost", err)
	}
	file := filepath.Join(store.Root, "session.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		if data[i] == '2' {
			data[i] = '3'
			break
		}
	}
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx, "session"); err == nil {
		t.Fatal("corrupt checkpoint accepted")
	}
	if _, err := store.Load(ctx, "../escape"); err == nil {
		t.Fatal("path escape accepted")
	}
}

func TestAgentCheckpointRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "sessions")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlink privilege unavailable")
	}
	if err := (AgentSessionStore{Root: link}).Save(context.Background(), planning.AgentSession{ID: "safe"}); err == nil {
		t.Fatal("linked root accepted")
	}
}

func TestCheckpointReplacementRetriesWithoutDeletingOldFile(t *testing.T) {
	root := t.TempDir()
	from, to := filepath.Join(root, "temporary"), filepath.Join(root, "checkpoint")
	if err := os.WriteFile(from, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := replaceAgentFile(context.Background(), from, to, func(a, b string) error {
		calls++
		if calls == 1 {
			data, e := os.ReadFile(b)
			if e != nil || string(data) != "old" {
				t.Fatal("old checkpoint lost")
			}
			return os.ErrPermission
		}
		return os.Rename(a, b)
	})
	if err != nil || calls != 2 {
		t.Fatal("transient replacement not recovered", err)
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "new" {
		t.Fatal("new checkpoint missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := replaceAgentFile(ctx, from, to, os.Rename); err != context.Canceled {
		t.Fatal("cancellation ignored")
	}
}
