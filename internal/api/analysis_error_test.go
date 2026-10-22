package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
	"strings"
	"testing"
)

func TestFailureMessagesDoNotExposePathsOrSource(t *testing.T) {
	for _, prefix := range []string{"inventory repository:", "parse old BPMN:", "parse new BPMN:", "apply hybrid mapping:"} {
		got := analysisFailureMessage(errors.New(prefix + " C:/private/secret.go credential=hidden"))
		if strings.Contains(got, "private") || strings.Contains(got, "credential") {
			t.Fatal(got)
		}
	}
	if !strings.Contains(analysisFailureMessage(context.DeadlineExceeded), "timed out") {
		t.Fatal("missing timeout classification")
	}
}

func TestSemanticResourceErrorIsActionableAndRedacted(t *testing.T) {
	err := fmt.Errorf("apply hybrid: %w: C:/private/model", semantic.ErrRuntimeMemory)
	got := analysisFailureMessage(err)
	if !strings.Contains(got, "memory") || strings.Contains(got, "private") {
		t.Fatal(got)
	}
	if !strings.Contains(analysisFailureMessage(semantic.ErrInputLimit), "Top-K") {
		t.Fatal("missing corpus guidance")
	}
	workerMessage := analysisFailureMessage(fmt.Errorf("apply hybrid: %w: C:/private/model", semantic.ErrRuntimeWorker))
	if !strings.Contains(workerMessage, "worker stopped") || strings.Contains(workerMessage, "private") {
		t.Fatal(workerMessage)
	}
}

func TestWebRequestTrimsPastedRepositoryPath(t *testing.T) {
	request, err := webRequest(map[string]string{"repo_path": "  C:/example  "}, map[string]string{"old_bpmn": "old", "new_bpmn": "new"})
	if err != nil || request.RepositoryPath != "C:/example" {
		t.Fatalf("%+v %v", request, err)
	}
}
