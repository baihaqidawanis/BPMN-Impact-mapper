package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planstore"
)

type PlanStore struct {
	Root string
}

func (store PlanStore) Save(ctx context.Context, value planning.ChangePlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validAnalysisID.MatchString(value.PlanID) || !validAnalysisID.MatchString(value.AnalysisID) || !validAnalysisID.MatchString(value.ContextID) {
		return errors.New("invalid plan identity")
	}
	expected, err := planning.ComputePlanID(value)
	if err != nil || expected != value.PlanID {
		return errors.New("plan identity mismatch")
	}
	jsonData, err := planning.JSON(value)
	if err != nil {
		return fmt.Errorf("render plan JSON: %w", err)
	}
	directory, err := store.planDirectory(value.PlanID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return fmt.Errorf("create plan directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("plan directory must be a real directory")
	}
	if err := writeAtomic(filepath.Join(directory, "change-plan.json"), jsonData); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(directory, "change-plan.md"), planning.Markdown(value))
}

func (store PlanStore) Get(ctx context.Context, planID string) (planning.ChangePlan, error) {
	if err := ctx.Err(); err != nil {
		return planning.ChangePlan{}, err
	}
	directory, err := store.planDirectory(planID)
	if err != nil {
		return planning.ChangePlan{}, planstore.ErrNotFound
	}
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) || err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return planning.ChangePlan{}, planstore.ErrNotFound
	}
	if err != nil {
		return planning.ChangePlan{}, fmt.Errorf("inspect plan directory: %w", err)
	}
	path := filepath.Join(directory, "change-plan.json")
	fileInfo, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) || err == nil && (fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular()) {
		return planning.ChangePlan{}, planstore.ErrNotFound
	}
	if err != nil {
		return planning.ChangePlan{}, fmt.Errorf("inspect change plan: %w", err)
	}
	if fileInfo.Size() > 16<<20 {
		return planning.ChangePlan{}, errors.New("stored change plan exceeds size limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return planning.ChangePlan{}, fmt.Errorf("read change plan: %w", err)
	}
	var value planning.ChangePlan
	if err := json.Unmarshal(data, &value); err != nil {
		return planning.ChangePlan{}, fmt.Errorf("decode change plan: %w", err)
	}
	expected, err := planning.ComputePlanID(value)
	if err != nil || value.PlanID != planID || expected != planID {
		return planning.ChangePlan{}, errors.New("stored plan identity mismatch")
	}
	return value, nil
}

func (store PlanStore) ListByAnalysis(ctx context.Context, analysisID string) ([]planning.PlanSummary, error) {
	if !validAnalysisID.MatchString(analysisID) {
		return nil, errors.New("invalid analysis ID")
	}
	root, err := filepath.Abs(store.Root)
	if err != nil || store.Root == "" {
		return nil, errors.New("plan store root is required")
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []planning.PlanSummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 1000 {
		return nil, errors.New("plan store entry limit exceeded")
	}
	result := make([]planning.PlanSummary, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() || !validAnalysisID.MatchString(entry.Name()) {
			continue
		}
		value, err := store.Get(ctx, entry.Name())
		if errors.Is(err, planstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if value.AnalysisID == analysisID {
			result = append(result, planning.Summarize(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PlanID < result[j].PlanID })
	return result, nil
}

func (store PlanStore) planDirectory(planID string) (string, error) {
	if store.Root == "" {
		return "", errors.New("plan store root is required")
	}
	if !validAnalysisID.MatchString(planID) {
		return "", errors.New("invalid plan ID")
	}
	root, err := filepath.Abs(store.Root)
	if err != nil {
		return "", fmt.Errorf("resolve plan store root: %w", err)
	}
	directory := filepath.Join(root, planID)
	if filepath.Dir(directory) != root {
		return "", errors.New("plan directory escapes store root")
	}
	return directory, nil
}
