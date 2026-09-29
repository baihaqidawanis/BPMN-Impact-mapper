// Package filestore implements local filesystem persistence for canonical reports.
package filestore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
)

var validAnalysisID = regexp.MustCompile(`^[a-f0-9]{32}$`)

type ReportStore struct {
	Root string
}

func (store ReportStore) Save(ctx context.Context, value domain.Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validAnalysisID.MatchString(value.AnalysisID) {
		return errors.New("invalid analysis ID")
	}
	jsonData, err := report.JSON(value)
	if err != nil {
		return fmt.Errorf("render canonical JSON: %w", err)
	}
	markdownData := report.Markdown(value)
	directory, err := store.analysisDirectory(value.AnalysisID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return fmt.Errorf("create analysis directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("analysis directory must be a real directory")
	}
	if err := writeAtomic(filepath.Join(directory, "impact-report.json"), jsonData); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(directory, "impact-report.md"), markdownData); err != nil {
		return err
	}
	return nil
}

func (store ReportStore) Get(ctx context.Context, analysisID string) (domain.Report, error) {
	if err := ctx.Err(); err != nil {
		return domain.Report{}, err
	}
	directory, err := store.analysisDirectory(analysisID)
	if err != nil {
		return domain.Report{}, reportstore.ErrNotFound
	}
	data, err := os.ReadFile(filepath.Join(directory, "impact-report.json"))
	if errors.Is(err, os.ErrNotExist) {
		return domain.Report{}, reportstore.ErrNotFound
	}
	if err != nil {
		return domain.Report{}, fmt.Errorf("read analysis report: %w", err)
	}
	var value domain.Report
	if err := json.Unmarshal(data, &value); err != nil {
		return domain.Report{}, fmt.Errorf("decode analysis report: %w", err)
	}
	if value.AnalysisID != analysisID {
		return domain.Report{}, errors.New("stored analysis ID mismatch")
	}
	return value, nil
}

func (store ReportStore) analysisDirectory(analysisID string) (string, error) {
	if store.Root == "" {
		return "", errors.New("report store root is required")
	}
	if !validAnalysisID.MatchString(analysisID) {
		return "", errors.New("invalid analysis ID")
	}
	root, err := filepath.Abs(store.Root)
	if err != nil {
		return "", fmt.Errorf("resolve report store root: %w", err)
	}
	directory := filepath.Join(root, analysisID)
	if filepath.Dir(directory) != root {
		return "", errors.New("analysis directory escapes store root")
	}
	return directory, nil
}

func writeAtomic(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, data) {
			return nil
		}
		return errors.New("refusing to overwrite a different report artifact")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing report: %w", err)
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".report-*")
	if err != nil {
		return fmt.Errorf("create temporary report: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary report: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary report: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary report: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish report: %w", err)
	}
	return nil
}
