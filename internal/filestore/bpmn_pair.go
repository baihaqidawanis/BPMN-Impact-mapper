package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
)

func parsePair(pair domain.BPMNPair) (*domain.ProcessContext, error) {
	before, err := bpmn.Parse([]byte(pair.BeforeXML), bpmn.Limits{})
	if err != nil {
		return nil, err
	}
	after, err := bpmn.Parse([]byte(pair.AfterXML), bpmn.Limits{})
	if err != nil {
		return nil, err
	}
	return &domain.ProcessContext{Before: before, After: after}, nil
}
func (store ReportStore) pairPath(id string) (string, error) {
	dir, err := store.analysisDirectory(id)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("invalid analysis directory")
	}
	path := filepath.Join(dir, "bpmn-inputs.json")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("invalid BPMN artifact")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return path, nil
}
func (store ReportStore) SaveBPMN(ctx context.Context, pair domain.BPMNPair) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := parsePair(pair); err != nil {
		return err
	}
	report, err := store.Get(ctx, pair.AnalysisID)
	if err != nil {
		return err
	}
	if report.InputFingerprint != pair.InputFingerprint {
		return errors.New("BPMN input identity mismatch")
	}
	path, err := store.pairPath(pair.AnalysisID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(pair)
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}
func (store ReportStore) GetBPMN(ctx context.Context, id string) (domain.BPMNPair, error) {
	var pair domain.BPMNPair
	if err := ctx.Err(); err != nil {
		return pair, err
	}
	path, err := store.pairPath(id)
	if errors.Is(err, os.ErrNotExist) {
		return pair, reportstore.ErrNotFound
	}
	if err != nil {
		return pair, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return pair, reportstore.ErrNotFound
	}
	if err != nil {
		return pair, err
	}
	if info.Size() > 100<<20 {
		return pair, errors.New("BPMN input artifact too large")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return pair, err
	}
	if err = json.Unmarshal(data, &pair); err != nil {
		return pair, err
	}
	if pair.AnalysisID != id {
		return pair, errors.New("BPMN input identity mismatch")
	}
	if _, err := parsePair(pair); err != nil {
		return pair, err
	}
	return pair, nil
}
