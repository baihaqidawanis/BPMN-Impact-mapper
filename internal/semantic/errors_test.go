package semantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMemoryErrorClassificationIncludesTracebackTail(t *testing.T) {
	message := strings.Repeat("traceback frame\n", 200) + "MemoryError: failed to allocate"
	if !errors.Is(runtimeFailure(message), ErrRuntimeMemory) {
		t.Fatal("lost memory error at end of traceback")
	}
	if got := runtimeFailure(message).Error(); !strings.Contains(got, "MemoryError: failed to allocate") || strings.Contains(got, "traceback frame") {
		t.Fatalf("missing actionable traceback tail: %s", got)
	}
	if errors.Is(runtimeFailure("model is not cached"), ErrRuntimeMemory) {
		t.Fatal("misclassified model error")
	}
}

func TestRuntimeErrorDetailPreservesFinalException(t *testing.T) {
	got := runtimeFailure("Loading weights: 100%\nTraceback (most recent call last):\n  File \"private.py\", line 42\nValueError: unsupported input").Error()
	if !strings.HasSuffix(got, "ValueError: unsupported input") || strings.Contains(got, "private.py") {
		t.Fatalf("unexpected runtime error: %s", got)
	}
}

func TestWorkerExitWithoutTracebackHasOwnClassification(t *testing.T) {
	err := runtimeFailure("Loading weights: 100%\nworker exited: exit status 1")
	if !errors.Is(err, ErrRuntimeWorker) || !strings.Contains(err.Error(), "exit status 1") {
		t.Fatalf("unexpected worker failure: %v", err)
	}
}

func TestEmbeddingRejectsOversizeBeforeWorker(t *testing.T) {
	_, err := (PythonEmbedder{Command: "unused", Model: "unused"}).Embed(context.Background(), []string{strings.Repeat("x", maxEmbeddingInput+1)})
	if !errors.Is(err, ErrInputLimit) {
		t.Fatal(err)
	}
}

func TestEmbeddingQueueHonorsCancellation(t *testing.T) {
	embeddingSlot <- struct{}{}
	defer func() { <-embeddingSlot }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := (PythonEmbedder{Command: "unused", Model: "unused"}).Embed(ctx, []string{"queue cancellation fixture"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
