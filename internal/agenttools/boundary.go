package agenttools

import (
	"errors"
	"path/filepath"
	"strings"
)

// CheckOutputBoundary requires existing roots; server startup creates its data
// root before accepting jobs. Never persist model outputs inside analyzed source.
func CheckOutputBoundary(repo, dataRoot string) error {
	r, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	r, err = filepath.EvalSymlinks(r)
	if err != nil {
		return err
	}
	d, err := filepath.Abs(dataRoot)
	if err != nil {
		return err
	}
	d, err = filepath.EvalSymlinks(d)
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.VolumeName(r), filepath.VolumeName(d)) {
		return nil
	}
	rel, err := filepath.Rel(r, d)
	if err != nil {
		return err
	}
	if rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("agent data directory must stay outside analyzed source")
	}
	return nil
}
