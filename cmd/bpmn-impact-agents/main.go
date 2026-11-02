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
	id := flags.String("session", "", "session ID; reuse to resume the same inputs")
	endpoint := flags.String("endpoint", "http://127.0.0.1:11435", "local Ollama tunnel URL")
	model := flags.String("model", "", "explicit installed local Ollama model")
	dataRoot := flags.String("data-dir", ".data", "local checkpoint/plan directory, outside repo")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall run timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *input == "" || *id == "" || *model == "" || flags.NArg() != 0 || *timeout <= 0 {
		return errors.New("requires --repo, --context, --session and --model")
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
	info, err := os.Lstat(*input)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return errors.New("context must be a regular JSON file within 1 MiB")
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, (1<<20)+1))
	d.DisallowUnknownFields()
	var bundle planning.EvidenceBundle
	if err := d.Decode(&bundle); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing context data")
	}
	base, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(base, *timeout)
	defer cancel()
	tools, err := agenttools.New(ctx, repoPath, ingest.Options{})
	if err != nil {
		return err
	}
	if err := tools.CheckBundle(bundle); err != nil {
		return err
	}
	provider, err := llm.NewOllamaAgent(llm.OllamaAgentConfig{BaseURL: *endpoint, Model: *model})
	if err != nil {
		return err
	}
	if err := provider.Prepare(ctx); err != nil {
		return err
	}
	engine := planning.AgentEngine{Model: provider, Tools: tools, Store: filestore.AgentSessionStore{Root: filepath.Join(root, "agent-sessions")}}
	session, err := engine.Run(ctx, *id, bundle)
	fmt.Printf("Session %s: %s; turns=%d tools=%d revisions=%d\n", session.ID, session.Status, session.Turns, session.Tools, session.Revisions)
	if err != nil {
		return err
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

func startsParent(rel string) bool { return len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator) }
