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
	maxEmbeddingInput  = 16 << 20
	maxCacheBytes      = 64 << 20
	maxCacheEntries    = 4
)

//go:embed embed.py
var pythonScript string

// A Python worker loads the full embedding model. Serialize workers inside one
// server process to avoid avoidable memory spikes under concurrent analyses.
var embeddingSlot = make(chan struct{}, 1)

var embeddingCache struct {
	sync.RWMutex
	entries map[[sha256.Size]byte]cacheEntry
	order   [][sha256.Size]byte
	bytes   int
}

type cacheEntry struct {
	vectors [][]float64
	runtime Identity
	bytes   int
}

type Embedder interface {
	Identity() Identity
	Embed(context.Context, []string) ([][]float64, error)
}

type Identity struct {
	Provider                    string `json:"provider"`
	Model                       string `json:"model"`
	AdapterSHA256               string `json:"adapter_sha256"`
	PythonVersion               string `json:"python_version,omitempty"`
	SentenceTransformersVersion string `json:"sentence_transformers_version,omitempty"`
	ModelArtifactSHA256         string `json:"model_artifact_sha256,omitempty"`
	EmbeddingDimension          int    `json:"embedding_dimension,omitempty"`
}

type PythonEmbedder struct {
	Command string
	Model   string
	state   *identityState
}

type identityState struct {
	sync.RWMutex
	runtime Identity
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
	return PythonEmbedder{Command: command, Model: model, state: &identityState{}}
}

func (embedder PythonEmbedder) Identity() Identity {
	adapterHash := sha256.Sum256([]byte(pythonScript))
	identity := Identity{Provider: providerVersion, Model: embedder.Model, AdapterSHA256: fmt.Sprintf("%x", adapterHash)}
	if embedder.state == nil {
		return identity
	}
	embedder.state.RLock()
	defer embedder.state.RUnlock()
	identity.PythonVersion = embedder.state.runtime.PythonVersion
	identity.SentenceTransformersVersion = embedder.state.runtime.SentenceTransformersVersion
	identity.ModelArtifactSHA256 = embedder.state.runtime.ModelArtifactSHA256
	identity.EmbeddingDimension = embedder.state.runtime.EmbeddingDimension
	return identity
}

func (embedder PythonEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return [][]float64{}, nil
	}
	if len(texts) > maxEmbeddingTexts {
		return nil, fmt.Errorf("%w: maximum %d texts", ErrInputLimit, maxEmbeddingTexts)
	}
	inputBytes := 0
	for _, text := range texts {
		inputBytes += len(text)
		if inputBytes > maxEmbeddingInput {
			return nil, ErrInputLimit
		}
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
	if cached, identity := cachedEmbedding(cacheKey); cached != nil {
		embedder.storeRuntimeIdentity(identity)
		return cached, nil
	}
	select {
	case embeddingSlot <- struct{}{}:
		defer func() { <-embeddingSlot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if cached, identity := cachedEmbedding(cacheKey); cached != nil {
		embedder.storeRuntimeIdentity(identity)
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
		return nil, runtimeFailure(message)
	}
	var response struct {
		Dimension int         `json:"dimension"`
		Vectors   [][]float64 `json:"vectors"`
		Runtime   struct {
			PythonVersion               string `json:"python_version"`
			SentenceTransformersVersion string `json:"sentence_transformers_version"`
			ModelArtifactSHA256         string `json:"model_artifact_sha256"`
		} `json:"runtime"`
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
	runtimeIdentity := Identity{
		PythonVersion: response.Runtime.PythonVersion, SentenceTransformersVersion: response.Runtime.SentenceTransformersVersion,
		ModelArtifactSHA256: response.Runtime.ModelArtifactSHA256, EmbeddingDimension: response.Dimension,
	}
	if runtimeIdentity.PythonVersion == "" || runtimeIdentity.SentenceTransformersVersion == "" || len(runtimeIdentity.ModelArtifactSHA256) != 64 {
		return nil, errors.New("semantic embedding runtime returned incomplete reproducibility metadata")
	}
	embedder.storeRuntimeIdentity(runtimeIdentity)
	storeCachedVectors(cacheKey, response.Vectors, runtimeIdentity)
	return cloneVectors(response.Vectors), nil
}

// Read vectors and their identity under one lock so concurrent cache replacement
// cannot attach another model's metadata to an otherwise valid cache hit.
func cachedEmbedding(key [sha256.Size]byte) ([][]float64, Identity) {
	embeddingCache.RLock()
	defer embeddingCache.RUnlock()
	entry, ok := embeddingCache.entries[key]
	if !ok {
		return nil, Identity{}
	}
	return cloneVectors(entry.vectors), entry.runtime
}

func (embedder PythonEmbedder) storeRuntimeIdentity(identity Identity) {
	if embedder.state == nil || identity.PythonVersion == "" {
		return
	}
	embedder.state.Lock()
	embedder.state.runtime = identity
	embedder.state.Unlock()
}

func cachedVectors(key [sha256.Size]byte) [][]float64 {
	vectors, _ := cachedEmbedding(key)
	return vectors
}

func cachedRuntimeIdentity(key [sha256.Size]byte) Identity {
	_, identity := cachedEmbedding(key)
	return identity
}

func storeCachedVectors(key [sha256.Size]byte, vectors [][]float64, runtime Identity) {
	cost := 0
	for _, vector := range vectors {
		cost += len(vector) * 8
	}
	if cost > maxCacheBytes {
		return
	}
	embeddingCache.Lock()
	defer embeddingCache.Unlock()
	if embeddingCache.entries == nil {
		embeddingCache.entries = make(map[[sha256.Size]byte]cacheEntry)
	}
	if old, ok := embeddingCache.entries[key]; ok {
		embeddingCache.bytes -= old.bytes
		for i, item := range embeddingCache.order {
			if item == key {
				embeddingCache.order = append(embeddingCache.order[:i], embeddingCache.order[i+1:]...)
				break
			}
		}
	}
	for len(embeddingCache.order) >= maxCacheEntries || embeddingCache.bytes+cost > maxCacheBytes {
		oldest := embeddingCache.order[0]
		embeddingCache.bytes -= embeddingCache.entries[oldest].bytes
		delete(embeddingCache.entries, oldest)
		embeddingCache.order = embeddingCache.order[1:]
	}
	embeddingCache.entries[key] = cacheEntry{vectors: cloneVectors(vectors), runtime: runtime, bytes: cost}
	embeddingCache.order = append(embeddingCache.order, key)
	embeddingCache.bytes += cost
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
