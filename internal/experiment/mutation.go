package experiment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/scenario"
)

// RunMutation is the filesystem adapter for deterministic BPMN experiment input generation.
// It refuses to overwrite either the baseline or any existing output.
func RunMutation(specPath, baselinePath, outputPath string) (scenario.Spec, error) {
	if specPath == "" || baselinePath == "" || outputPath == "" {
		return scenario.Spec{}, errors.New("spec, baseline, and output paths are required")
	}
	baselineAbs, err := filepath.Abs(baselinePath)
	if err != nil {
		return scenario.Spec{}, fmt.Errorf("resolve baseline path: %w", err)
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return scenario.Spec{}, fmt.Errorf("resolve output path: %w", err)
	}
	if filepath.Clean(baselineAbs) == filepath.Clean(outputAbs) {
		return scenario.Spec{}, errors.New("output must differ from baseline")
	}
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return scenario.Spec{}, fmt.Errorf("read mutation specification: %w", err)
	}
	spec, err := scenario.DecodeSpec(specData)
	if err != nil {
		return scenario.Spec{}, err
	}
	baseline, err := os.ReadFile(baselineAbs)
	if err != nil {
		return scenario.Spec{}, fmt.Errorf("read baseline BPMN: %w", err)
	}
	generated, err := scenario.Mutate(baseline, spec)
	if err != nil {
		return scenario.Spec{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outputAbs), 0o750); err != nil {
		return scenario.Spec{}, fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.OpenFile(outputAbs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return scenario.Spec{}, fmt.Errorf("create output without overwrite: %w", err)
	}
	writeErr := func() error {
		if _, err := file.Write(generated); err != nil {
			return err
		}
		return file.Sync()
	}()
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(outputAbs)
		return scenario.Spec{}, errors.Join(writeErr, closeErr)
	}
	return spec, nil
}
