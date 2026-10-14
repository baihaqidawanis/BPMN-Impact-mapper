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
	"syscall"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
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
	analysisTimeout := flags.Duration("analysis-timeout", 2*time.Minute, "maximum duration for one HTTP analysis")
	maxRequestBytes := flags.Int64("max-request-bytes", 20<<20, "maximum multipart request bytes")
	maxBPMNBytes := flags.Int64("max-bpmn-bytes", 8<<20, "maximum bytes per uploaded BPMN file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *analysisTimeout <= 0 || *maxRequestBytes <= 0 || *maxBPMNBytes <= 0 {
		return errors.New("timeouts and byte limits must be positive")
	}
	root, err := filepath.Abs(*dataDirectory)
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	handler := api.Handler{
		Analyzer:       wiring.NewAnalyzer(),
		Store:          filestore.ReportStore{Root: filepath.Join(root, "analyses")},
		Planner:        planning.Orchestrator{Provider: llm.NoLLMProvider{}},
		PlanStore:      filestore.PlanStore{Root: filepath.Join(root, "plans")},
		TempRoot:       filepath.Join(root, "tmp"),
		MaxRequestSize: *maxRequestBytes,
		MaxBPMNSize:    *maxBPMNBytes,
		LogError:       func(err error) { log.Printf("request error: %v", err) },
	}
	server := &http.Server{
		Addr:              *address,
		Handler:           withTimeout(handler, *analysisTimeout),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      *analysisTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	base, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownComplete := make(chan struct{})
	go func() {
		<-base.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown: %v", err)
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
