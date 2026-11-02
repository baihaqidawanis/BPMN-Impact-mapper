package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type profileRecorder struct {
	model string
	texts []string
}

func (p *profileRecorder) Identity() Identity { return Identity{Model: p.model} }
func (p *profileRecorder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	p.texts = append([]string(nil), texts...)
	result := make([][]float64, len(texts))
	for i := range result {
		result[i] = []float64{1, 0}
	}
	return result, nil
}

func TestModelProfilePrefixesQueriesAndEveryPassage(t *testing.T) {
	for _, model := range []string{DefaultModel, "intfloat/multilingual-e5-small"} {
		t.Run(model, func(t *testing.T) {
			embedder := &profileRecorder{model: model}
			strategy := Strategy{Embedder: embedder}
			changes, warnings, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{Name: "Unggah dokumen", Type: "userTask"}}}, []domain.Artifact{{File: "upload.go", Content: strings.Repeat("upload document\n", 100)}}, mapping.Options{TopK: 1})
			if err != nil || len(embedder.texts) != 4 || len(changes[0].Impacts) != 1 {
				t.Fatalf("result=%v err=%v texts=%d", changes, err, len(embedder.texts))
			}
			e5 := model != DefaultModel
			if strings.HasPrefix(embedder.texts[0], "query: ") != e5 {
				t.Fatal("incorrect query prefix")
			}
			for _, text := range embedder.texts[1:] {
				if strings.HasPrefix(text, "passage: ") != e5 {
					t.Fatal("incorrect passage prefix")
				}
			}
			if strings.Contains(changes[0].Impacts[0].Evidence[0].Excerpt, "passage:") {
				t.Fatal("embedding prefix leaked into evidence")
			}
			if e5 && (len(warnings) != 1 || warnings[0].Code != "SEMANTIC_MODEL_CALIBRATION") {
				t.Fatalf("missing calibration warning: %v", warnings)
			}
			if strategy.MethodMetadata()["input_profile"] != modelInputProfile(model) {
				t.Fatal("profile missing from reproducibility metadata")
			}
		})
	}
}
