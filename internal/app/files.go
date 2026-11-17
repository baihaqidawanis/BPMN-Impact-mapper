package app

// Filesystem compatibility helpers for the path-based CLI/application facade.
// Pure planning and mapping packages do not depend on these helpers.
import (
	"errors"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func WriteArtifacts(outputDirectory string, value domain.Report) error {
	if strings.TrimSpace(outputDirectory) == "" {
		return errors.New("output directory is required")
	}
	abs, err := filepath.Abs(outputDirectory)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	jsonData, err := report.JSON(value)
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(abs, "impact-report.json"), jsonData); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(abs, "impact-report.md"), report.Markdown(value)); err != nil {
		return err
	}
	return nil
}

func readBPMN(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("BPMN path must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("BPMN path must be a regular file")
	}
	if info.Size() > bpmn.DefaultMaxBytes {
		return nil, fmt.Errorf("BPMN exceeds byte limit %d", bpmn.DefaultMaxBytes)
	}
	return os.ReadFile(path)
}

func appendInputExclusions(root string, patterns []string, inputPaths ...string) []string {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return patterns
	}
	for _, input := range inputPaths {
		if strings.TrimSpace(input) == "" {
			continue
		}
		absolute, err := filepath.Abs(input)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(rootAbsolute, absolute)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			patterns = append(patterns, filepath.ToSlash(relative))
		}
	}
	sort.Strings(patterns)
	return patterns
}

func atomicWrite(target string, data []byte) error {
	directory := filepath.Dir(target)
	temporary, err := os.CreateTemp(directory, ".impact-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
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
	return os.Rename(temporaryName, target)
}
