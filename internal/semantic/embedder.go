// Package semantic implements offline embedding-based candidate retrieval.
package semantic

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	DefaultModel       = "all-MiniLM-L6-v2"
	providerVersion    = "sentence-transformers-command-v1"
	maxEmbeddingTexts  = 5001
	maxEmbeddingOutput = 96 << 20
)

//go:embed embed.py
var pythonScript string

// A Python worker loads the full embedding model. Serialize workers inside one
// server process to avoid avoidable memory spikes under concurrent analyses.
var embeddingSlot = make(chan struct{}, 1)

var embeddingCache struct {
	sync.RWMutex
	key     [sha256.Size]byte
	vectors [][]float64
}

type Embedder interface {
	Identity() Identity
	Embed(context.Context, []string) ([][]float64, error)
}

type Identity struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type PythonEmbedder struct {
	Command string
	Model   string
}

func NewPythonEmbedderFromEnvironment() PythonEmbedder {
	command := strings.TrimSpace(os.Getenv("BPMN_IMPACT_PYTHON"))
	if command == "" {
		command = "python"
	}
	model := strings.TrimSpace(os.Getenv("BPMN_IMPACT_SEMANTIC_MODEL"))
	if model == "" {
		model = DefaultModel
	}
	return PythonEmbedder{Command: command, Model: model}
}

func (embedder PythonEmbedder) Identity() Identity {
	return Identity{Provider: providerVersion, Model: embedder.Model}
}

func (embedder PythonEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return [][]float64{}, nil
	}
	if len(texts) > maxEmbeddingTexts {
		return nil, fmt.Errorf("semantic embedding input exceeds %d texts", maxEmbeddingTexts)
	}
	if strings.TrimSpace(embedder.Command) == "" || strings.TrimSpace(embedder.Model) == "" {
		return nil, errors.New("semantic embedding command and model are required")
	}
	input, err := json.Marshal(struct {
		Texts []string `json:"texts"`
	}{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("encode semantic input: %w", err)
	}
	cacheKey := sha256.Sum256(append([]byte(embedder.Command+"\x00"+embedder.Model+"\x00"), input...))
	if cached := cachedVectors(cacheKey); cached != nil {
		return cached, nil
	}
	select {
	case embeddingSlot <- struct{}{}:
		defer func() { <-embeddingSlot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if cached := cachedVectors(cacheKey); cached != nil {
		return cached, nil
	}
	command := exec.CommandContext(ctx, embedder.Command, "-c", pythonScript, embedder.Model)
	command.Stdin = bytes.NewReader(input)
	command.Env = append(os.Environ(),
		"HF_HUB_OFFLINE=1",
		"TRANSFORMERS_OFFLINE=1",
		"TOKENIZERS_PARALLELISM=false",
		"PYTHONHASHSEED=0",
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedWriter{target: &stdout, remaining: maxEmbeddingOutput}
	command.Stderr = &limitedWriter{target: &stderr, remaining: 1 << 20}
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("offline embedding interrupted: %w", ctx.Err())
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("offline embedding runtime failed: %s", bounded(message, 800))
	}
	var response struct {
		Dimension int         `json:"dimension"`
		Vectors   [][]float64 `json:"vectors"`
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode semantic embeddings: %w", err)
	}
	if len(response.Vectors) != len(texts) || response.Dimension <= 0 {
		return nil, errors.New("semantic embedding runtime returned an invalid vector count or dimension")
	}
	for _, vector := range response.Vectors {
		if len(vector) != response.Dimension {
			return nil, errors.New("semantic embedding runtime returned inconsistent dimensions")
		}
	}
	storeCachedVectors(cacheKey, response.Vectors)
	return cloneVectors(response.Vectors), nil
}

func cachedVectors(key [sha256.Size]byte) [][]float64 {
	embeddingCache.RLock()
	defer embeddingCache.RUnlock()
	if embeddingCache.vectors == nil || embeddingCache.key != key {
		return nil
	}
	return cloneVectors(embeddingCache.vectors)
}

func storeCachedVectors(key [sha256.Size]byte, vectors [][]float64) {
	embeddingCache.Lock()
	defer embeddingCache.Unlock()
	embeddingCache.key = key
	embeddingCache.vectors = cloneVectors(vectors)
}

func cloneVectors(vectors [][]float64) [][]float64 {
	result := make([][]float64, len(vectors))
	for index := range vectors {
		result[index] = append([]float64(nil), vectors[index]...)
	}
	return result
}

type limitedWriter struct {
	target    *bytes.Buffer
	remaining int
}

func (writer *limitedWriter) Write(data []byte) (int, error) {
	if len(data) > writer.remaining {
		return 0, errors.New("semantic runtime output exceeds limit")
	}
	written, err := writer.target.Write(data)
	writer.remaining -= written
	return written, err
}

func bounded(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum-3]) + "..."
}
