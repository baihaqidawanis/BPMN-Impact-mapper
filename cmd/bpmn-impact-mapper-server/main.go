package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("bpmn-impact-mapper-server", flag.ContinueOnError)
	address := flags.String("addr", "127.0.0.1:8080", "HTTP listen address")
	dataDirectory := flags.String("data-dir", ".data", "local report and temporary artifact directory")
	analysisTimeout := flags.Duration("analysis-timeout", 10*time.Minute, "maximum duration for one HTTP analysis")
	maxRequestBytes := flags.Int64("max-request-bytes", 20<<20, "maximum multipart request bytes")
	maxBPMNBytes := flags.Int64("max-bpmn-bytes", 8<<20, "maximum bytes per uploaded BPMN file")
	agentModel := flags.String("agent-model", "", "opt-in local Ollama model for Planning")
	analystModel := flags.String("analyst-model", "", "optional fixed Analyst model; falls back to --agent-model")
	plannerModel := flags.String("planner-model", "", "optional fixed Planner model; falls back to --agent-model")
	reviewerModel := flags.String("reviewer-model", "", "optional fixed Reviewer model; falls back to --agent-model")
	agentURL := flags.String("agent-url", "http://127.0.0.1:11435", "local Ollama tunnel URL")
	agentContext := flags.Int("agent-context", 32768, "agent context tokens (maximum 32768; only used with --agent-model)")
	agentThink := flags.Bool("agent-think", false, "opt-in thinking for a local model advertising thinking capability; requires a fresh agent session")
	agentReviewComparisons := flags.Bool("agent-review-comparisons", false, "experimental evidence/implementation comparison audit; not validated for automatic sign-off")
	agentReviewJSON := flags.Bool("agent-review-json", false, "experimental explicit-schema JSON Reviewer with file reviews plus final contract check")
	agentOutput := flags.Int("agent-output", 4096, "maximum agent output tokens, including thinking (32-8192)")
	agentProfile := flags.String("agent-profile", "baseline", "generation profile: baseline or coding (explicit sampling options)")
	analystProfile := flags.String("analyst-profile", "", "optional Analyst profile; falls back to --agent-profile")
	plannerProfile := flags.String("planner-profile", "", "optional Planner profile; falls back to --agent-profile")
	reviewerProfile := flags.String("reviewer-profile", "", "optional Reviewer profile; falls back to --agent-profile")
	analystThinking := flags.String("analyst-thinking", "inherit", "Analyst reasoning mode: inherit, on or off")
	plannerThinking := flags.String("planner-thinking", "inherit", "Planner reasoning mode: inherit, on or off")
	reviewerThinking := flags.String("reviewer-thinking", "inherit", "Reviewer reasoning mode: inherit, on or off")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *analysisTimeout <= 0 || *maxRequestBytes <= 0 || *maxBPMNBytes <= 0 {
		return errors.New("timeouts and byte limits must be positive")
	}
	overrides, err := agentRoleOverrides(
		map[planning.AgentRole]string{planning.Analyst: *analystModel, planning.Planner: *plannerModel, planning.Reviewer: *reviewerModel},
		map[planning.AgentRole]string{planning.Analyst: *analystProfile, planning.Planner: *plannerProfile, planning.Reviewer: *reviewerProfile},
		map[planning.AgentRole]string{planning.Analyst: *analystThinking, planning.Planner: *plannerThinking, planning.Reviewer: *reviewerThinking},
	)
	if err != nil {
		return err
	}
	hasRoleOverrides := len(overrides) > 0
	if *agentModel == "" && hasRoleOverrides {
		return errors.New("role model overrides require an explicit --agent-model fallback")
	}
	root, err := filepath.Abs(*dataDirectory)
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	embedder := semantic.NewPythonEmbedderFromEnvironment()
	base, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var agents *agentjobs.Service
	if *agentModel != "" {
		config := llm.OllamaAgentConfig{BaseURL: *agentURL, Model: *agentModel, ContextSize: *agentContext, Think: *agentThink, ReviewComparisons: *agentReviewComparisons, ReviewJSONMode: *agentReviewJSON, MaxOutput: *agentOutput, Profile: *agentProfile}
		config.OnInference = func(m llm.InferenceMetric) {
			log.Printf("agent inference role=%s model=%q profile=%s thinking=%t outcome=%s duration_ms=%d load_ms=%d input_tokens=%d generated_tokens=%d", m.Role, m.Model, m.Profile, m.Think, m.Outcome, m.Milliseconds, m.LoadMilliseconds, m.PromptTokens, m.GeneratedTokens)
		}
		var model llm.PreparedAgent
		if hasRoleOverrides {
			model, err = llm.NewOllamaRoleRouter(config, overrides)
		} else {
			model, err = llm.NewOllamaAgent(config)
		}
		if err != nil {
			return err
		}
		agents = &agentjobs.Service{Configuration: webAgentConfiguration(config, overrides), Model: model, Sessions: filestore.AgentSessionStore{Root: filepath.Join(root, "agent-sessions")}, Plans: filestore.PlanStore{Root: filepath.Join(root, "plans")}, Base: base, Source: func(ctx context.Context, repo string) (agentjobs.Snapshot, error) {
			if err := agenttools.CheckOutputBoundary(repo, root); err != nil {
				return nil, err
			}
			return agenttools.New(ctx, repo, ingest.Options{})
		}}
		agents.LogError = func(err error) { log.Printf("agent session error: %v", err) }
	}
	var semanticProbe struct {
		sync.Mutex
		ready bool
	}
	handler := api.Handler{
		Library:        filestore.BPMNLibrary{Root: filepath.Join(root, "bpmn-library")},
		Agents:         agents,
		Analyzer:       wiring.NewAnalyzerWithSemanticEmbedder(embedder),
		Store:          filestore.ReportStore{Root: filepath.Join(root, "analyses")},
		Planner:        planning.Orchestrator{Provider: llm.NoLLMProvider{}},
		PlanStore:      filestore.PlanStore{Root: filepath.Join(root, "plans")},
		TempRoot:       filepath.Join(root, "tmp"),
		MaxRequestSize: *maxRequestBytes,
		MaxBPMNSize:    *maxBPMNBytes,
		LogError:       func(err error) { log.Printf("request error: %v", err) },
		SemanticCheck: func(ctx context.Context) (string, string, error) {
			identity := embedder.Identity()
			semanticProbe.Lock()
			defer semanticProbe.Unlock()
			if semanticProbe.ready {
				return identity.Provider, identity.Model, nil
			}
			_, err := embedder.Embed(ctx, []string{"semantic runtime readiness"})
			semanticProbe.ready = err == nil
			return identity.Provider, identity.Model, err
		},
	}
	server := &http.Server{
		Addr:              *address,
		Handler:           withTimeout(handler, *analysisTimeout),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      *analysisTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownComplete := make(chan struct{})
	go func() {
		<-base.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown: %v", err)
		}
		if agents != nil {
			if err := agents.Wait(shutdownContext); err != nil {
				log.Printf("agent shutdown: %v", err)
			}
		}
		close(shutdownComplete)
	}()

	log.Printf("BPMN Impact Mapper API listening on http://%s", *address)
	err = server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-shutdownComplete
	return nil
}

func withTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), timeout)
		defer cancel()
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
