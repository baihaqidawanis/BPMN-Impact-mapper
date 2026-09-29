package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
)

type Request struct {
	RepositoryPath    string                `json:"repository_path"`
	OldBPMNPath       string                `json:"old_bpmn_path"`
	NewBPMNPath       string                `json:"new_bpmn_path"`
	RepositoryName    string                `json:"repository_name,omitempty"`
	Revision          string                `json:"revision,omitempty"`
	Method            domain.AnalysisMethod `json:"method,omitempty"`
	MappingConfigPath string                `json:"mapping_config_path,omitempty"`
	Ingest            ingest.Options        `json:"ingest,omitempty"`
	Retrieval         mapping.Options       `json:"retrieval,omitempty"`
}

type Analyzer struct{ Resolver mapping.Resolver }

func (analyzer Analyzer) Analyze(ctx context.Context, request Request) (domain.Report, error) {
	if err := validateRequest(request); err != nil {
		return domain.Report{}, err
	}
	method := request.Method
	if method == "" {
		method = domain.MethodHybrid
	}
	request.Method = method
	if analyzer.Resolver == nil {
		return domain.Report{}, errors.New("analysis strategy resolver is not configured")
	}
	strategy, methodConfig, err := analyzer.Resolver.Resolve(method, request.MappingConfigPath)
	if err != nil {
		return domain.Report{}, err
	}
	ingestOptions := request.Ingest
	ingestOptions.Ignore = append([]string(nil), request.Ingest.Ignore...)
	ingestOptions.Ignore = appendInputExclusions(request.RepositoryPath, ingestOptions.Ignore, request.OldBPMNPath, request.NewBPMNPath, request.MappingConfigPath)
	inventory, err := ingest.Build(ctx, request.RepositoryPath, ingestOptions)
	if err != nil {
		return domain.Report{}, fmt.Errorf("inventory repository: %w", err)
	}
	oldData, err := readBPMN(request.OldBPMNPath)
	if err != nil {
		return domain.Report{}, fmt.Errorf("read old BPMN: %w", err)
	}
	newData, err := readBPMN(request.NewBPMNPath)
	if err != nil {
		return domain.Report{}, fmt.Errorf("read new BPMN: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return domain.Report{}, err
	}
	oldGraph, err := bpmn.Parse(oldData, bpmn.Limits{})
	if err != nil {
		return domain.Report{}, fmt.Errorf("parse old BPMN: %w", err)
	}
	newGraph, err := bpmn.Parse(newData, bpmn.Limits{})
	if err != nil {
		return domain.Report{}, fmt.Errorf("parse new BPMN: %w", err)
	}
	artifacts, indexWarnings := index.Build(inventory)
	changes := diff.Compare(oldGraph, newGraph)
	changes, retrievalWarnings := strategy.Attach(changes, artifacts, request.Retrieval)
	name := strings.TrimSpace(request.RepositoryName)
	if name == "" {
		name = filepath.Base(inventory.Root)
	}
	revision := strings.TrimSpace(request.Revision)
	if revision == "" {
		revision = "unspecified"
	}
	warnings := make([]domain.Warning, 0, len(inventory.Warnings)+len(indexWarnings)+len(retrievalWarnings))
	for _, warning := range inventory.Warnings {
		warnings = append(warnings, domain.Warning{Code: "INGEST_SKIPPED", Message: warning})
	}
	for _, warning := range indexWarnings {
		warnings = append(warnings, domain.Warning{Code: "INDEX_FALLBACK", Message: warning})
	}
	warnings = append(warnings, retrievalWarnings...)
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Code != warnings[j].Code {
			return warnings[i].Code < warnings[j].Code
		}
		return warnings[i].Message < warnings[j].Message
	})
	fingerprint := inputFingerprint(name, revision, oldData, newData, inventory, request.Ingest)
	return domain.Report{
		AnalysisID:       analysisID(fingerprint, request, methodConfig),
		InputFingerprint: fingerprint,
		Method:           method,
		Repository:       domain.Repository{Name: name, Revision: revision},
		Changes:          changes, Warnings: warnings,
	}, nil
}

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

func validateRequest(request Request) error {
	if strings.TrimSpace(request.RepositoryPath) == "" || strings.TrimSpace(request.OldBPMNPath) == "" || strings.TrimSpace(request.NewBPMNPath) == "" {
		return errors.New("repository, old BPMN, and new BPMN paths are required")
	}
	method := request.Method
	if method == "" {
		method = domain.MethodHybrid
	}
	if !method.Valid() {
		return fmt.Errorf("unsupported analysis method %q", method)
	}
	if method == domain.MethodConfiguration && strings.TrimSpace(request.MappingConfigPath) == "" {
		return errors.New("mapping configuration path is required for configuration method")
	}
	if method != domain.MethodConfiguration && strings.TrimSpace(request.MappingConfigPath) != "" {
		return errors.New("mapping configuration is only valid for configuration method")
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

func inputFingerprint(name, revision string, oldData, newData []byte, inventory ingest.Inventory, options ingest.Options) string {
	hash := sha256.New()
	writeHash(hash, []byte(name))
	writeHash(hash, []byte(revision))
	writeHash(hash, oldData)
	writeHash(hash, newData)
	for _, file := range inventory.Files {
		writeHash(hash, []byte(file.RelativePath))
		writeHash(hash, file.Content)
	}
	options.Ignore = append([]string(nil), options.Ignore...)
	sort.Strings(options.Ignore)
	encoded, _ := json.Marshal(options)
	writeHash(hash, encoded)
	return hex.EncodeToString(hash.Sum(nil)[:16])
}

func analysisID(fingerprint string, request Request, methodConfig []byte) string {
	hash := sha256.New()
	writeHash(hash, []byte(fingerprint))
	writeHash(hash, []byte(request.Method))
	writeHash(hash, methodConfig)
	encoded, _ := json.Marshal(request.Retrieval)
	writeHash(hash, encoded)
	return hex.EncodeToString(hash.Sum(nil)[:16])
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

type hashWriter interface{ Write([]byte) (int, error) }

func writeHash(hash hashWriter, value []byte) {
	_, _ = hash.Write([]byte(fmt.Sprintf("%d:", len(value))))
	_, _ = hash.Write(value)
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
