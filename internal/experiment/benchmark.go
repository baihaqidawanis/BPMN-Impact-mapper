package experiment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	impactreport "github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
)

func RunBenchmark(ctx context.Context, manifestPath, outputDirectory string, analyzer benchmark.Analyzer) (benchmark.Report, error) {
	manifestBytes, err := readRegular(manifestPath, 1<<20, "benchmark manifest")
	if err != nil {
		return benchmark.Report{}, err
	}
	var manifest benchmark.Manifest
	if err := decodeStrict(manifestBytes, &manifest); err != nil {
		return benchmark.Report{}, fmt.Errorf("decode benchmark manifest: %w", err)
	}
	store, err := NewLocalFiles(outputDirectory)
	if err != nil {
		return benchmark.Report{}, err
	}
	runner := benchmark.Runner{Analyzer: analyzer, Files: store, Predictions: predictionRenderer{}}
	report, timing, err := runner.Run(ctx, manifest)
	if err != nil {
		return benchmark.Report{}, err
	}
	reportJSON, err := benchmark.JSON(report)
	if err != nil {
		return benchmark.Report{}, err
	}
	timingJSON, err := benchmark.TimingJSON(timing)
	if err != nil {
		return benchmark.Report{}, err
	}
	for name, data := range map[string][]byte{"benchmark-report.json": reportJSON, "benchmark-report.md": benchmark.Markdown(report), "benchmark-timing.json": timingJSON} {
		if err := store.Write(name, data); err != nil {
			return benchmark.Report{}, err
		}
	}
	return report, nil
}

type predictionRenderer struct{}

func (predictionRenderer) JSON(value domain.Report) ([]byte, error) { return impactreport.JSON(value) }
func (predictionRenderer) Markdown(value domain.Report) []byte      { return impactreport.Markdown(value) }

type LocalFiles struct{ root string }

func NewLocalFiles(outputDirectory string) (*LocalFiles, error) {
	if strings.TrimSpace(outputDirectory) == "" {
		return nil, errors.New("benchmark output directory is required")
	}
	abs, err := filepath.Abs(outputDirectory)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &LocalFiles{root: filepath.Clean(abs)}, nil
}

func (files *LocalFiles) Read(path string, maximumBytes int64) ([]byte, error) {
	return readRegular(path, maximumBytes, "benchmark input")
}

func (files *LocalFiles) Write(relativePath string, data []byte) error {
	if filepath.IsAbs(relativePath) || strings.Contains(relativePath, "..") {
		return errors.New("benchmark artifact path must be contained and relative")
	}
	target := filepath.Join(files.root, filepath.FromSlash(relativePath))
	relative, err := filepath.Rel(files.root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("benchmark artifact escaped output directory")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	return atomicWrite(target, data)
}
