// Opt-in agent runner. The normal server still starts without a generative model.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("bpmn-impact-agents", flag.ContinueOnError)
	repo := flags.String("repo", "", "read-only source repository")
	input := flags.String("context", "", "exported planning context JSON")
	fromSession := flags.String("from-session", "", "revise a finalized session's edit proposals with Planner only")
	element := flags.String("element", "", "BPMN element ID whose edit targets are locked for revision")
	reviewNotes := flags.String("review-notes", "", "local text file with review findings for the Planner")
	testCalls := flags.String("test-calls", "", "require locked test call coverage: file:package.Constructor:Method:position=literal[:position=literal]")
	var sourceWindows []string
	flags.Func("source-window", "additional locked file evidence as relative/path:line[-last-line] (repeatable)", func(value string) error {
		sourceWindows = append(sourceWindows, value)
		return nil
	})
	id := flags.String("session", "", "session ID; reuse to resume the same inputs")
	endpoint := flags.String("endpoint", "http://127.0.0.1:11435", "local Ollama tunnel URL")
	model := flags.String("model", "", "explicit installed local Ollama model")
	contextSize := flags.Int("context-size", 0, "model context window in tokens")
	maxOutput := flags.Int("max-output", 0, "maximum output tokens")
	profile := flags.String("profile", "", "model generation profile")
	dataRoot := flags.String("data-dir", ".data", "local checkpoint/plan directory, outside repo")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall run timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *id == "" || *model == "" || flags.NArg() != 0 || *timeout <= 0 {
		return errors.New("requires --repo, --session and --model")
	}
	if (*input == "") == (*fromSession == "") || (*element != "") != (*fromSession != "") || *fromSession == *id {
		return errors.New("supply --context or --from-session with --element and a different --session ID")
	}
	if len(sourceWindows) > 8 || (len(sourceWindows) > 0 && *fromSession == "") {
		return errors.New("up to eight --source-window values are allowed with --from-session")
	}
	if *reviewNotes != "" && *fromSession == "" {
		return errors.New("--review-notes requires --from-session")
	}
	if *testCalls != "" && *fromSession == "" {
		return errors.New("--test-calls requires --from-session")
	}
	repoPath, err := filepath.Abs(*repo)
	if err != nil {
		return err
	}
	repoPath, err = filepath.EvalSymlinks(repoPath)
	if err != nil {
		return err
	}
	root, err := filepath.Abs(*dataRoot)
	if err != nil {
		return err
	}
	// Evaluate existing ancestors to prevent a data path aliasing the source repo.
	probe := root
	var suffix []string
	for {
		if _, e := os.Lstat(probe); e == nil {
			resolved, e := filepath.EvalSymlinks(probe)
			if e != nil {
				return e
			}
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			root = resolved
			break
		} else if !errors.Is(e, os.ErrNotExist) {
			return e
		}
		suffix = append(suffix, filepath.Base(probe))
		parent := filepath.Dir(probe)
		if parent == probe {
			return errors.New("cannot resolve data directory")
		}
		probe = parent
	}
	rel, err := filepath.Rel(repoPath, root)
	if err != nil {
		return err
	}
	if rel != ".." && !startsParent(rel) {
		return errors.New("data directory must stay outside analyzed repository")
	}
	base, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(base, *timeout)
	defer cancel()
	sessions := filestore.AgentSessionStore{Root: filepath.Join(root, "agent-sessions")}
	engine := planning.AgentEngine{Store: sessions}
	var bundle planning.EvidenceBundle
	var revision lockedPlan
	if *fromSession != "" {
		source, err := sessions.Load(ctx, *fromSession)
		if err != nil {
			return err
		}
		revision, err = lockPlan(source, *element)
		if err != nil {
			return err
		}
		bundle = revision.bundle
		engine.InitialDecisions, engine.InitialDraft = revision.decisions, revision.draft
		engine.DeferReview = true
		engine.Budget = planning.AgentBudget{MaxTurns: 12}
	} else {
		bundle, err = readContext(*input)
		if err != nil {
			return err
		}
	}
	tools, err := agenttools.New(ctx, repoPath, ingest.Options{})
	if err != nil {
		return err
	}
	if err := tools.CheckBundle(bundle); err != nil {
		return err
	}
	focusedEvidence := make(map[string][]string)
	if len(sourceWindows) > 0 {
		bundle, err = addLockedSourceWindows(ctx, tools, bundle, sourceWindows, focusedEvidence)
		if err != nil {
			return err
		}
	}
	var testAudit testCallAudit
	if *testCalls != "" {
		bundle, testAudit, err = addLockedTestCalls(ctx, tools, bundle, *testCalls)
		if err != nil {
			return err
		}
		for _, site := range testAudit.sites {
			fmt.Printf("Required test call %s (%s)\n", site.Reference, site.Test)
			focusedEvidence[testAudit.file] = append(focusedEvidence[testAudit.file], site.Evidence.ID)
		}
	}
	notes, err := readRevisionNotes(*reviewNotes)
	if err != nil {
		return err
	}
	provider, err := llm.NewOllamaAgent(llm.OllamaAgentConfig{BaseURL: *endpoint, Model: *model, ContextSize: *contextSize, MaxOutput: *maxOutput, Profile: *profile})
	if err != nil {
		return err
	}
	if err := provider.Prepare(ctx); err != nil {
		return err
	}
	engine.Model, engine.Tools = provider, tools
	if *fromSession != "" {
		if len(testAudit.sites) == 0 {
			focusedEvidence = nil
		}
		engine.Model = lockedPlanner{AgentModel: provider, actions: revision.actions, notes: notes, testFile: testAudit.file, callSites: testAudit.sites, focusedEvidence: focusedEvidence}
	}
	session, err := engine.Run(ctx, *id, bundle)
	fmt.Printf("Session %s: %s; turns=%d tools=%d revisions=%d\n", session.ID, session.Status, session.Turns, session.Tools, session.Revisions)
	if err != nil {
		return err
	}
	if *fromSession != "" {
		if !session.AwaitingCombinedReview {
			return errors.New("Planner did not return a complete revision; no revised plan was exported")
		}
		session.Feedback = []string{fmt.Sprintf("Planner-only revision from session %s. The %d edit targets were retained. The model received %d additional source windows. Review the steps and tests before implementation.", *fromSession, len(revision.actions), len(sourceWindows))}
		if notes != "" {
			session.Feedback = append(session.Feedback, "The Planner received operator review findings about the source and prior draft.")
		}
		if len(testAudit.sites) > 0 {
			session.Feedback = append(session.Feedback, fmt.Sprintf("The revision references and cites all %d required source test call sites. Review whether the proposed edits preserve their original outcomes.", len(testAudit.sites)))
		}
	}
	plan, err := session.ChangePlan()
	if err != nil {
		return err
	}
	store := filestore.PlanStore{Root: filepath.Join(root, "agent-plans")}
	if err := store.Save(ctx, plan); err != nil {
		return err
	}
	fmt.Printf("Plan %s; see %s\n", plan.PlanID, filepath.Join(*dataRoot, "agent-plans", plan.PlanID))
	return nil
}

func readContext(path string) (planning.EvidenceBundle, error) {
	var bundle planning.EvidenceBundle
	info, err := os.Lstat(path)
	if err != nil {
		return bundle, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return bundle, errors.New("context must be a regular JSON file within 1 MiB")
	}
	f, err := os.Open(path)
	if err != nil {
		return bundle, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, (1<<20)+1))
	d.DisallowUnknownFields()
	if err := d.Decode(&bundle); err != nil {
		return bundle, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return bundle, errors.New("trailing context data")
	}
	return bundle, nil
}

func startsParent(rel string) bool { return len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator) }
