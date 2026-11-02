package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/relationprobe"
)

const maxProbeReportBytes = 4 << 20

// probe-relations is an offline diagnostic. It neither rewrites the canonical
// report nor asserts that the repository still matches its input fingerprint.
func runProbeRelations(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("probe-relations", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repository := flags.String("repo", "", "local repository path, read only")
	reportPath := flags.String("report", "", "existing canonical impact-report.json")
	topSeeds := flags.Int("top-seeds", 5, "maximum displayed candidates per change used as relation seeds (1-20)")
	callers := flags.Bool("source-callers", false, "include bounded reverse source-caller associations (unranked)")
	inspect := flags.String("inspect-source", "", "comma-separated relative files; print indexed relation metadata only")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*repository) == "" || strings.TrimSpace(*reportPath) == "" || *topSeeds < 1 || *topSeeds > 20 {
		return errors.New("probe-relations requires --repo, --report, and --top-seeds between 1 and 20")
	}
	file, err := os.Open(*reportPath)
	if err != nil {
		return fmt.Errorf("open report: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProbeReportBytes+1))
	if err != nil || len(data) > maxProbeReportBytes {
		return errors.New("report cannot be read within 4 MiB limit")
	}
	var report domain.Report
	if err := json.Unmarshal(data, &report); err != nil || report.AnalysisID == "" || report.InputFingerprint == "" {
		return errors.New("report is not a canonical impact report")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	inventory, err := ingest.Build(ctx, *repository, ingest.Options{})
	if err != nil {
		return fmt.Errorf("inventory repository: %w", err)
	}
	registry := index.DefaultRegistry()
	if *callers {
		registry = index.SourceContextRegistry()
	}
	indexed := registry.Build(inventory)
	result := struct {
		AnalysisID      string                   `json:"analysis_id"`
		Warning         string                   `json:"warning"`
		Findings        []relationprobe.Finding  `json:"findings"`
		Proposals       []relationprobe.Proposal `json:"proposals"`
		CallerChanges   []domain.Change          `json:"caller_changes,omitempty"`
		SourceArtifacts []sourceArtifactInfo     `json:"source_artifacts,omitempty"`
	}{
		AnalysisID: report.AnalysisID,
		Warning:    "experimental only: repository snapshot equality with the saved report is not verified; proposed order is not a calibrated score or canonical report change",
		Findings:   relationprobe.Probe(report, indexed.Artifacts, *topSeeds),
	}
	result.Proposals = relationprobe.Propose(report, result.Findings)
	if *inspect != "" {
		selected := map[string]bool{}
		for _, f := range strings.Split(*inspect, ",") {
			selected[strings.TrimSpace(strings.ReplaceAll(f, "\\", "/"))] = true
		}
		for _, a := range indexed.Artifacts {
			if selected[a.File] {
				result.SourceArtifacts = append(result.SourceArtifacts, sourceArtifactInfo{a.ID, a.File, a.Symbol, a.Relations})
			}
		}
	}
	if *callers {
		for _, c := range report.Changes {
			files := relationprobe.CallerFiles(c, indexed.Artifacts)
			if len(files) == 0 {
				continue
			}
			result.CallerChanges = append(result.CallerChanges, domain.Change{ChangeType: c.ChangeType, OldElement: c.OldElement, NewElement: c.NewElement, RelatedFiles: files})
		}
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

type sourceArtifactInfo struct {
	ID        string                  `json:"id"`
	File      string                  `json:"file"`
	Symbol    domain.Symbol           `json:"symbol"`
	Relations []domain.SourceRelation `json:"relations"`
}
