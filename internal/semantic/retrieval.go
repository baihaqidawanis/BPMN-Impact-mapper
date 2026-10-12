package semantic

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

const maxDocumentCharacters = 8000

type Strategy struct{ Embedder Embedder }

func (Strategy) Method() domain.AnalysisMethod { return domain.MethodSemantic }

func (strategy Strategy) MethodMetadata() map[string]string {
	if strategy.Embedder == nil {
		return nil
	}
	identity := strategy.Embedder.Identity()
	return map[string]string{"provider": identity.Provider, "model": identity.Model}
}

func (strategy Strategy) Attach(ctx context.Context, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	if strategy.Embedder == nil {
		return nil, nil, errors.New("semantic embedder is not configured")
	}
	if len(artifacts)+len(changes) > maxEmbeddingTexts {
		return nil, nil, fmt.Errorf("semantic retrieval supports at most %d combined changes and artifacts", maxEmbeddingTexts)
	}
	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}
	threshold := options.CandidateThreshold
	if threshold <= 0 {
		threshold = 0.25
	}
	tieDelta := options.TieDelta
	if tieDelta <= 0 {
		tieDelta = 0.01
	}

	texts := make([]string, 0, len(changes)+len(artifacts))
	for _, change := range changes {
		texts = append(texts, queryText(change))
	}
	for _, artifact := range artifacts {
		texts = append(texts, artifactText(artifact))
	}
	vectors, err := strategy.Embedder.Embed(ctx, texts)
	if err != nil {
		return nil, nil, err
	}
	warnings := []domain.Warning{}
	artifactVectors := vectors[len(changes):]
	for changeIndex := range changes {
		change := &changes[changeIndex]
		change.Impacts = []domain.Impact{}
		if change.ChangeType == domain.UnresolvedMatch {
			if len(change.UnresolvedImpact) == 0 {
				change.UnresolvedImpact = []string{"graph element matching is ambiguous"}
			}
			continue
		}
		ranked := make([]domain.Impact, 0, len(artifacts))
		for artifactIndex, artifact := range artifacts {
			score := round(math.Max(0, cosine(vectors[changeIndex], artifactVectors[artifactIndex])))
			if score < threshold {
				continue
			}
			ranked = append(ranked, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence: []domain.Evidence{{
					Type: "SEMANTIC_SIMILARITY", Location: evidenceLocation(artifact),
					Excerpt: bounded(strings.TrimSpace(artifact.Content), 240),
				}},
				ScoreBreakdown: domain.ScoreBreakdown{Semantic: score, Total: score},
				Confidence:     score,
			})
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			if math.Abs(ranked[i].Confidence-ranked[j].Confidence) > 1e-12 {
				return ranked[i].Confidence > ranked[j].Confidence
			}
			if ranked[i].File != ranked[j].File {
				return ranked[i].File < ranked[j].File
			}
			return ranked[i].Symbol < ranked[j].Symbol
		})
		ranked = removeShadowedPackages(ranked, artifacts)
		if len(ranked) > topK {
			ranked = ranked[:topK]
		}
		change.Impacts = ranked
		if len(ranked) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "semantic retrieval found no candidate above the cosine threshold")
		} else if len(ranked) > 1 && math.Abs(ranked[0].Confidence-ranked[1].Confidence) <= tieDelta {
			warnings = append(warnings, domain.Warning{Code: "CANDIDATE_TIE", Message: fmt.Sprintf("%s has semantic candidates within %.2f", changeIdentifier(*change), tieDelta)})
		}
	}
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].Message < warnings[j].Message })
	return changes, warnings, nil
}

func queryText(change domain.Change) string {
	parts := []string{"BPMN change", string(change.ChangeType), change.Reason}
	for _, element := range queryElements(change) {
		if element == nil {
			continue
		}
		parts = append(parts, "element", element.Name, "type", element.Type, "actor", element.Actor,
			"state", element.State, "documentation", element.Documentation)
		parts = append(parts, element.DataRequirements...)
	}
	if change.Context != nil {
		parts = append(parts, "pattern", string(change.Context.Pattern))
		for _, element := range append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...) {
			parts = append(parts, "neighbor", element.Name, element.Type)
		}
	}
	return strings.Join(nonEmpty(parts), " ")
}

func artifactText(artifact domain.Artifact) string {
	prefix := strings.Join(nonEmpty([]string{
		"repository artifact", artifact.File, "module", artifact.Module, "layer", artifact.Layer,
		"language", artifact.Language, "symbol", displaySymbol(artifact.Symbol), "kind", artifact.Symbol.Kind,
	}), " ")
	return bounded(prefix+" content "+artifact.Content, maxDocumentCharacters)
}

func queryElements(change domain.Change) []*domain.Element {
	switch change.ChangeType {
	case domain.AddTask, domain.AddGateway, domain.AddDataRequirement, domain.AddState:
		return []*domain.Element{change.NewElement}
	case domain.RemoveTask, domain.RemoveGateway, domain.RemoveDataRequirement:
		return []*domain.Element{change.OldElement}
	default:
		return []*domain.Element{change.OldElement, change.NewElement}
	}
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func cosine(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	dot, leftNorm, rightNorm := 0.0, 0.0, 0.0
	for index := range left {
		dot += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func displaySymbol(symbol domain.Symbol) string {
	if symbol.Name == "" {
		return ""
	}
	if symbol.Receiver != "" {
		return symbol.Receiver + "." + symbol.Name
	}
	return symbol.Name
}

func evidenceLocation(artifact domain.Artifact) string {
	if artifact.Symbol.StartLine > 0 {
		return fmt.Sprintf("%s:%d", artifact.File, artifact.Symbol.StartLine)
	}
	return artifact.File
}

func removeShadowedPackages(impacts []domain.Impact, artifacts []domain.Artifact) []domain.Impact {
	kindByKey := map[string]string{}
	for _, artifact := range artifacts {
		kindByKey[artifact.File+"|"+displaySymbol(artifact.Symbol)] = artifact.Symbol.Kind
	}
	hasSpecific := map[string]bool{}
	for _, impact := range impacts {
		kind := kindByKey[impact.File+"|"+impact.Symbol]
		if kind != "" && kind != "package" {
			hasSpecific[impact.File] = true
		}
	}
	result := make([]domain.Impact, 0, len(impacts))
	for _, impact := range impacts {
		if kindByKey[impact.File+"|"+impact.Symbol] == "package" && hasSpecific[impact.File] {
			continue
		}
		result = append(result, impact)
	}
	return result
}

func changeIdentifier(change domain.Change) string {
	if change.NewElement != nil {
		return string(change.ChangeType) + ":" + change.NewElement.ID
	}
	if change.OldElement != nil {
		return string(change.ChangeType) + ":" + change.OldElement.ID
	}
	return string(change.ChangeType)
}

func round(value float64) float64 { return math.Round(value*10000) / 10000 }
