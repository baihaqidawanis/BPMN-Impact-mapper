package llm

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"

// InferenceMetric contains operational metadata only. It never includes
// prompts, repository paths, source excerpts, model prose or thinking content.
type InferenceMetric struct {
	Role             planning.AgentRole `json:"role"`
	Model            string             `json:"model"`
	Profile          string             `json:"profile"`
	Think            bool               `json:"thinking"`
	Outcome          string             `json:"outcome"`
	Milliseconds     int64              `json:"milliseconds"`
	LoadMilliseconds int64              `json:"load_milliseconds"`
	PromptTokens     int                `json:"prompt_tokens"`
	GeneratedTokens  int                `json:"generated_tokens"`
}
