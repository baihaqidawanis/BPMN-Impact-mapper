package experiment

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reviewpack"
)

func RunPrepareReview(reportPaths []string, outputDirectory, reviewerA, reviewerB string) (reviewpack.Manifest, error) {
	if len(reportPaths) == 0 || len(reportPaths) > 16 {
		return reviewpack.Manifest{}, errors.New("provide between 1 and 16 --report files")
	}
	inputs := make([]reviewpack.Input, 0, len(reportPaths))
	for index, reportPath := range reportPaths {
		data, err := readRegular(reportPath, 64<<20, fmt.Sprintf("prediction report %d", index+1))
		if err != nil {
			return reviewpack.Manifest{}, err
		}
		var report domain.Report
		if err := decodeStrict(data, &report); err != nil {
			return reviewpack.Manifest{}, fmt.Errorf("decode prediction report %d: %w", index+1, err)
		}
		digest := sha256.Sum256(data)
		inputs = append(inputs, reviewpack.Input{Report: report, SHA256: hex.EncodeToString(digest[:])})
	}
	pack, err := reviewpack.Build(inputs, reviewerA, reviewerB)
	if err != nil {
		return reviewpack.Manifest{}, err
	}
	manifestJSON, err := reviewpack.JSON(pack.Manifest)
	if err != nil {
		return reviewpack.Manifest{}, err
	}
	reviewerAJSON, err := reviewpack.JSON(pack.ReviewerA)
	if err != nil {
		return reviewpack.Manifest{}, err
	}
	reviewerBJSON, err := reviewpack.JSON(pack.ReviewerB)
	if err != nil {
		return reviewpack.Manifest{}, err
	}
	artifacts := map[string][]byte{
		"coordinator-manifest.json": manifestJSON,
		"review-guide.md":           reviewpack.Guide(pack.Manifest),
		"reviewer-a.json":           reviewerAJSON,
		"reviewer-b.json":           reviewerBJSON,
	}
	if err := writeNewDirectory(outputDirectory, artifacts); err != nil {
		return reviewpack.Manifest{}, err
	}
	return pack.Manifest, nil
}

func writeNewDirectory(outputDirectory string, artifacts map[string][]byte) error {
	if strings.TrimSpace(outputDirectory) == "" {
		return errors.New("review pack output directory is required")
	}
	target, err := filepath.Abs(outputDirectory)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		return errors.New("review pack output directory already exists; choose a new directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".review-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	for _, name := range []string{"coordinator-manifest.json", "review-guide.md", "reviewer-a.json", "reviewer-b.json"} {
		if err := os.WriteFile(filepath.Join(staging, name), artifacts[name], 0o640); err != nil {
			return err
		}
	}
	return os.Rename(staging, target)
}
