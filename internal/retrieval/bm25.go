package retrieval

import (
	"math"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type BM25 struct {
	documents []domain.Artifact
	frequency map[string]int
	lengths   []int
	average   float64
	k1        float64
	b         float64
}

type RankedDocument struct {
	Artifact domain.Artifact
	Score    float64
}

func NewBM25(documents []domain.Artifact) *BM25 {
	copyDocuments := append([]domain.Artifact(nil), documents...)
	sort.Slice(copyDocuments, func(i, j int) bool { return copyDocuments[i].ID < copyDocuments[j].ID })
	model := &BM25{documents: copyDocuments, frequency: map[string]int{}, k1: 1.2, b: 0.75}
	for _, document := range copyDocuments {
		model.lengths = append(model.lengths, len(document.Tokens))
		model.average += float64(len(document.Tokens))
		seen := map[string]bool{}
		for _, token := range document.Tokens {
			if !seen[token] {
				model.frequency[token]++
				seen[token] = true
			}
		}
	}
	if len(copyDocuments) > 0 {
		model.average /= float64(len(copyDocuments))
	}
	return model
}

func (model *BM25) Search(query []string) []RankedDocument {
	if len(query) == 0 || len(model.documents) == 0 || model.average == 0 {
		return []RankedDocument{}
	}
	queryTerms := unique(query)
	ranked := make([]RankedDocument, 0, len(model.documents))
	for index, document := range model.documents {
		termFrequency := map[string]int{}
		for _, token := range document.Tokens {
			termFrequency[token]++
		}
		score := 0.0
		for _, term := range queryTerms {
			frequency := termFrequency[term]
			if frequency == 0 {
				continue
			}
			documentFrequency := model.frequency[term]
			idf := math.Log(1 + (float64(len(model.documents)-documentFrequency)+0.5)/(float64(documentFrequency)+0.5))
			numerator := float64(frequency) * (model.k1 + 1)
			denominator := float64(frequency) + model.k1*(1-model.b+model.b*float64(model.lengths[index])/model.average)
			score += idf * numerator / denominator
		}
		if score > 0 {
			ranked = append(ranked, RankedDocument{Artifact: document, Score: score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].Score-ranked[j].Score) > 1e-12 {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Artifact.ID < ranked[j].Artifact.ID
	})
	return ranked
}

func unique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
