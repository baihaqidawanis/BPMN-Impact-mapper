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
	"sort"

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
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) || err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return domain.Report{}, reportstore.ErrNotFound
	}
	if err != nil {
		return domain.Report{}, fmt.Errorf("inspect analysis directory: %w", err)
	}
	reportPath := filepath.Join(directory, "impact-report.json")
	fileInfo, err := os.Lstat(reportPath)
	if errors.Is(err, os.ErrNotExist) || err == nil && (fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular()) {
		return domain.Report{}, reportstore.ErrNotFound
	}
	if err != nil {
		return domain.Report{}, fmt.Errorf("inspect analysis report: %w", err)
	}
	if fileInfo.Size() > 64<<20 {
		return domain.Report{}, errors.New("stored analysis report exceeds size limit")
	}
	data, err := os.ReadFile(reportPath)
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

func (store ReportStore) List(ctx context.Context) ([]domain.ReportSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(store.Root)
	if err != nil || store.Root == "" {
		return nil, errors.New("report store root is required")
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.ReportSummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 1000 {
		return nil, errors.New("report store entry limit exceeded")
	}
	result := make([]domain.ReportSummary, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() || !validAnalysisID.MatchString(entry.Name()) {
			continue
		}
		value, err := store.Get(ctx, entry.Name())
		if errors.Is(err, reportstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		summary := domain.ReportSummary{AnalysisID: value.AnalysisID, InputFingerprint: value.InputFingerprint, Method: value.Method, Repository: value.Repository, Changes: len(value.Changes), Warnings: len(value.Warnings)}
		for _, change := range value.Changes {
			summary.Candidates += len(change.Impacts)
			if len(change.Impacts) == 0 || len(change.UnresolvedImpact) > 0 {
				summary.Unresolved++
			}
		}
		result = append(result, summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AnalysisID < result[j].AnalysisID })
	return result, nil
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
