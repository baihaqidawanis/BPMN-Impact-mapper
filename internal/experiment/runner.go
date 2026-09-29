// Package experiment is the filesystem adapter for the evaluation use case.
// It reads completed prediction and hidden-ground-truth artifacts only; it is
// not reachable from the inference pipeline.
package experiment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

func Run(predictionPath, groundTruthPath, outputDirectory string) (evaluation.Report, error) {
	predictionBytes, err := readRegular(predictionPath, 64<<20, "prediction report")
	if err != nil {
		return evaluation.Report{}, err
	}
	truthBytes, err := readRegular(groundTruthPath, 8<<20, "ground truth")
	if err != nil {
		return evaluation.Report{}, err
	}
	var prediction domain.Report
	if err := decodeStrict(predictionBytes, &prediction); err != nil {
		return evaluation.Report{}, fmt.Errorf("decode prediction report: %w", err)
	}
	if prediction.AnalysisID == "" || !prediction.Method.Valid() {
		return evaluation.Report{}, errors.New("prediction report requires analysis_id and valid method")
	}
	var truth evaluation.GroundTruth
	if err := decodeStrict(truthBytes, &truth); err != nil {
		return evaluation.Report{}, fmt.Errorf("decode ground truth: %w", err)
	}
	result, err := evaluation.Evaluate(prediction, predictionBytes, truth, truthBytes)
	if err != nil {
		return evaluation.Report{}, err
	}
	jsonData, err := evaluation.JSON(result)
	if err != nil {
		return evaluation.Report{}, err
	}
	if strings.TrimSpace(outputDirectory) == "" {
		return evaluation.Report{}, errors.New("evaluation output directory is required")
	}
	if err := os.MkdirAll(outputDirectory, 0o750); err != nil {
		return evaluation.Report{}, err
	}
	if err := atomicWrite(filepath.Join(outputDirectory, "evaluation-report.json"), jsonData); err != nil {
		return evaluation.Report{}, err
	}
	if err := atomicWrite(filepath.Join(outputDirectory, "evaluation-report.md"), evaluation.Markdown(result)); err != nil {
		return evaluation.Report{}, err
	}
	return result, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func readRegular(path string, limit int64, label string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", label)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds byte limit %d", label, limit)
	}
	return os.ReadFile(path)
}

func atomicWrite(target string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".evaluation-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(name, target)
}
