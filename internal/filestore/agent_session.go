package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// AgentSessionStore is local, single-writer checkpoint storage. Callers serialize
// runs per ID. Checksums detect accidental corruption, not malicious local edits.
type AgentSessionStore struct{ Root string }

var agentSessionID = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

type agentEnvelope struct {
	SHA256  string                `json:"sha256"`
	Session planning.AgentSession `json:"session"`
}

func (store AgentSessionStore) sessionPath(id string, create bool) (string, error) {
	if !agentSessionID.MatchString(id) {
		return "", errors.New("invalid agent session ID")
	}
	root, err := filepath.Abs(store.Root)
	if err != nil || store.Root == "" {
		return "", errors.New("session root required")
	}
	// Check every existing ancestor before mkdir and again afterwards. This is a
	// local single-user store, not a defense against concurrent filesystem swaps.
	for p := root; ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return "", errors.New("session root must not traverse symlinks")
		}
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return "", e
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	if create {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return "", err
		}
	}
	path := filepath.Join(root, id+".json")
	if info, e := os.Lstat(path); e == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return "", errors.New("session file must be regular")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	return path, nil
}

func (store AgentSessionStore) Save(ctx context.Context, s planning.AgentSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := store.sessionPath(s.ID, true)
	if err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	wrapped, err := json.MarshalIndent(agentEnvelope{hex.EncodeToString(hash[:]), s}, "", "  ")
	if err != nil {
		return err
	}
	limit := 2 << 20
	if s.FullReview != nil {
		limit = 64 << 20
	}
	if len(wrapped) > limit {
		return errors.New("agent checkpoint exceeds size limit")
	}
	// Reports are immutable, but a session is a mutable checkpoint. Write a
	// complete temporary file then replace; never truncate the current checkpoint.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-checkpoint-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(append(wrapped, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return replaceAgentFile(ctx, tmp.Name(), path, os.Rename)
}

// Windows indexers/AV/readers can briefly deny replacement. Retry permission
// errors within one second without deleting or truncating the old checkpoint.
func replaceAgentFile(ctx context.Context, from, to string, rename func(string, string) error) error {
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		err = rename(from, to)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrPermission) {
			return err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func (store AgentSessionStore) Load(ctx context.Context, id string) (planning.AgentSession, error) {
	var out planning.AgentSession
	if err := ctx.Err(); err != nil {
		return out, err
	}
	path, err := store.sessionPath(id, false)
	if err != nil {
		return out, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, planning.ErrAgentSessionMissing
	}
	if err != nil {
		return out, err
	}
	if info.Size() > 64<<20 {
		return out, errors.New("agent checkpoint too large")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	var envelope agentEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return out, errors.New("invalid agent checkpoint JSON")
	}
	if envelope.Session.FullReview == nil && len(data) > 2<<20 {
		return out, errors.New("focused checkpoint too large")
	}
	canonical, err := json.Marshal(envelope.Session)
	if err != nil {
		return out, err
	}
	hash := sha256.Sum256(canonical)
	if envelope.Session.ID != id || hex.EncodeToString(hash[:]) != envelope.SHA256 {
		return out, errors.New("agent checkpoint integrity mismatch")
	}
	return envelope.Session, nil
}
