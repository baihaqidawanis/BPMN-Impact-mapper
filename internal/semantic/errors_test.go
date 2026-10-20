package semantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMemoryErrorClassificationIncludesTracebackTail(t *testing.T) {
	if !errors.Is(runtimeFailure(strings.Repeat("traceback ", 200)+"MemoryError"), ErrRuntimeMemory) {
		t.Fatal("lost memory error at end of traceback")
	}
	if errors.Is(runtimeFailure("model is not cached"), ErrRuntimeMemory) {
		t.Fatal("misclassified model error")
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
