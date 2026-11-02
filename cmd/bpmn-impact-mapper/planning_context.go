package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Export from the immutable report only; never reload source, rerank or call a model.
func runPlanningContext(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("planning-context", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("report", "", "canonical impact-report.json")
	output := flags.String("output", "", "new context JSON file; omitted writes stdout")
	defaults := planning.DefaultLimits()
	candidates := flags.Int("candidates", defaults.CandidatesPerChange, "ranked candidates per change")
	related := flags.Int("related", defaults.RelatedPerChange, "additional source connections per change")
	evidence := flags.Int("evidence", defaults.EvidencePerCandidate, "evidence items per candidate")
	excerpt := flags.Int("excerpt", defaults.ExcerptRunes, "maximum excerpt runes")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *input == "" || flags.NArg() != 0 {
		return errors.New("planning-context requires --report")
	}
	if *candidates < 1 || *candidates > 20 || *related < 0 || *related > 3 || *evidence < 1 || *evidence > 8 || *excerpt < 32 || *excerpt > 240 {
		return errors.New("planning context limits outside supported bounds")
	}
	info, err := os.Lstat(*input)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return errors.New("report must be a regular file within 4 MiB")
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 4<<20))
	decoder.DisallowUnknownFields()
	var report domain.Report
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if report.AnalysisID == "" || report.InputFingerprint == "" || !report.Method.Valid() {
		return errors.New("input is not a canonical analysis report")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("report contains trailing data")
	}
	data, err := planning.BundleJSON(planning.BuildEvidenceBundle(report, planning.Limits{CandidatesPerChange: *candidates, RelatedPerChange: *related, EvidencePerCandidate: *evidence, ExcerptRunes: *excerpt}))
	if err != nil {
		return err
	}
	if *output == "" {
		_, err = stdout.Write(data)
		return err
	}
	destination, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = destination.Write(data)
	return err
}
