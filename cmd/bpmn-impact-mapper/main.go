package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/experiment"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("usage: bpmn-impact-mapper <analyze|evaluate|benchmark|reconcile|prepare-review|mutate> [flags]")
	}
	switch arguments[0] {
	case "analyze":
		return runAnalyze(arguments[1:], stdout, stderr)
	case "evaluate":
		return runEvaluate(arguments[1:], stdout, stderr)
	case "benchmark":
		return runBenchmark(arguments[1:], stdout, stderr)
	case "reconcile":
		return runReconcile(arguments[1:], stdout, stderr)
	case "prepare-review":
		return runPrepareReview(arguments[1:], stdout, stderr)
	case "mutate":
		return runMutate(arguments[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q; expected analyze, evaluate, benchmark, reconcile, prepare-review, or mutate", arguments[0])
	}
}

func runPrepareReview(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("prepare-review", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var reports stringList
	flags.Var(&reports, "report", "canonical impact-report.json; repeatable")
	output := flags.String("output", "", "new review-pack output directory")
	reviewerA := flags.String("reviewer-a", "reviewer-a", "first independent reviewer identifier")
	reviewerB := flags.String("reviewer-b", "reviewer-b", "second independent reviewer identifier")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	manifest, err := experiment.RunPrepareReview(reports, *output, *reviewerA, *reviewerB)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "review pack %s: wrote blind templates, guide, and coordinator manifest\n", manifest.PackID)
	return nil
}

func runMutate(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("mutate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	spec := flags.String("spec", "", "versioned mutation specification JSON")
	baseline := flags.String("baseline", "", "frozen baseline BPMN file")
	output := flags.String("output", "", "new BPMN output file; must not already exist")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	result, err := experiment.RunMutation(*spec, *baseline, *output)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "scenario %s: wrote generated BPMN\n", result.ScenarioID)
	return nil
}

func runReconcile(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	reviewerA := flags.String("reviewer-a", "", "reviewer A annotation JSON")
	reviewerB := flags.String("reviewer-b", "", "reviewer B annotation JSON")
	manifest := flags.String("manifest", "", "optional coordinator manifest binding both completed reviews to a frozen snapshot")
	output := flags.String("output", "", "reconciliation output directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	result, err := experiment.RunReconcileWithManifest(*reviewerA, *reviewerB, *manifest, *output)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "reconciliation: %d agreed case(s), %d disagreement(s)\n", len(result.Agreed.Cases), len(result.Disagreements))
	return nil
}

func runBenchmark(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifest := flags.String("manifest", "", "frozen benchmark manifest JSON")
	output := flags.String("output", "", "benchmark output directory")
	timeout := flags.Duration("timeout", 10*time.Minute, "benchmark timeout")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := experiment.RunBenchmark(ctx, *manifest, *output, wiring.NewAnalyzer())
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "benchmark %s: wrote benchmark-report.json, benchmark-report.md, benchmark-report.csv, and benchmark-timing.json\n", result.BenchmarkID)
	return nil
}

func runAnalyze(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("analyze", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var ignores stringList
	repository := flags.String("repo", "", "local repository path")
	oldBPMN := flags.String("old-bpmn", "", "baseline BPMN file")
	newBPMN := flags.String("new-bpmn", "", "new BPMN file")
	output := flags.String("output", "", "output directory")
	name := flags.String("repository-name", "", "logical repository name")
	revision := flags.String("revision", "unspecified", "logical repository revision")
	timeout := flags.Duration("timeout", 2*time.Minute, "analysis timeout")
	maxFileSize := flags.Int64("max-file-bytes", 1<<20, "maximum indexed file bytes")
	maxTotalSize := flags.Int64("max-total-bytes", 64<<20, "maximum indexed repository bytes")
	maxFiles := flags.Int("max-files", 10000, "maximum indexed file count")
	topK := flags.Int("top-k", 10, "maximum candidates per change")
	method := flags.String("method", string(domain.MethodHybrid), "mapping method: hybrid, fusion, bm25, semantic, or configuration")
	mappingConfig := flags.String("mapping-config", "", "frozen JSON mapping configuration (configuration method only)")
	flags.Var(&ignores, "ignore", "additional slash-normalized glob; repeatable")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	base, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(base, *timeout)
	defer cancel()
	request := app.Request{
		RepositoryPath: *repository, OldBPMNPath: *oldBPMN, NewBPMNPath: *newBPMN,
		RepositoryName: *name, Revision: *revision,
		Method: domain.AnalysisMethod(*method), MappingConfigPath: *mappingConfig,
		Ingest:    ingest.Options{MaxFileSize: *maxFileSize, MaxTotalSize: *maxTotalSize, MaxFiles: *maxFiles, Ignore: ignores},
		Retrieval: mapping.Options{TopK: *topK},
	}
	value, err := wiring.NewAnalyzer().Analyze(ctx, request)
	if err != nil {
		return err
	}
	if err := app.WriteArtifacts(*output, value); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "analysis %s: wrote impact-report.json and impact-report.md\n", value.AnalysisID)
	return nil
}

func runEvaluate(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	prediction := flags.String("report", "", "canonical impact-report.json")
	groundTruth := flags.String("ground-truth", "", "hidden ground-truth JSON")
	output := flags.String("output", "", "evaluation output directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	result, err := experiment.Run(*prediction, *groundTruth, *output)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "evaluation %s: wrote evaluation-report.json and evaluation-report.md\n", result.EvaluationID)
	return nil
}

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}
