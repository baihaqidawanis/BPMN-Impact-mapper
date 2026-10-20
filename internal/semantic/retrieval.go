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

// A short passage leaves room for file/function metadata within the model's
// limited token window. This is a character bound, not a tokenizer guarantee.
const maxPassageCharacters = 600

type passage struct {
	text     string
	excerpt  string
	location string
}

type Strategy struct{ Embedder Embedder }

func (Strategy) Method() domain.AnalysisMethod { return domain.MethodSemantic }

func (strategy Strategy) MethodMetadata() map[string]string {
	if strategy.Embedder == nil {
		return nil
	}
	identity := strategy.Embedder.Identity()
	metadata := map[string]string{"provider": identity.Provider, "model": identity.Model, "adapter_sha256": identity.AdapterSHA256}
	if identity.PythonVersion != "" {
		metadata["python_version"] = identity.PythonVersion
		metadata["sentence_transformers_version"] = identity.SentenceTransformersVersion
		metadata["model_artifact_sha256"] = identity.ModelArtifactSHA256
		metadata["embedding_dimension"] = fmt.Sprint(identity.EmbeddingDimension)
	}
	return metadata
}

func (strategy Strategy) Attach(ctx context.Context, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	if strategy.Embedder == nil {
		return nil, nil, errors.New("semantic embedder is not configured")
	}
	if len(artifacts)+len(changes) > maxEmbeddingTexts {
		return nil, nil, fmt.Errorf("%w: at most %d combined changes and artifacts", ErrInputLimit, maxEmbeddingTexts)
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
	passages := make([][]passage, len(artifacts))
	passageIndexes := make([][]int, len(artifacts))
	// Every artifact gets a first passage. A second, tail passage is added
	// where budget permits, without making corpus size depend on a query.
	remaining := maxEmbeddingTexts - len(changes) - len(artifacts)
	for index, artifact := range artifacts {
		passages[index] = artifactPassages(artifact)
		passageIndexes[index] = append(passageIndexes[index], len(texts))
		texts = append(texts, passages[index][0].text)
	}
	for index := range artifacts {
		if remaining == 0 {
			break
		}
		if len(passages[index]) > 1 {
			passageIndexes[index] = append(passageIndexes[index], len(texts))
			texts = append(texts, passages[index][1].text)
			remaining--
		}
	}
	omittedTailPassages := 0
	for index := range artifacts {
		if len(passages[index]) > len(passageIndexes[index]) {
			omittedTailPassages++
		}
	}
	vectors, err := strategy.Embedder.Embed(ctx, texts)
	if err != nil {
		return nil, nil, err
	}
	warnings := []domain.Warning{}
	if omittedTailPassages > 0 {
		warnings = append(warnings, domain.Warning{Code: "SEMANTIC_PASSAGE_LIMIT", Message: fmt.Sprintf("%d long source artifacts were represented only by their first passage because of the embedding input limit", omittedTailPassages)})
	}
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
			score, bestPassage := 0.0, 0
			for passageIndex, vectorIndex := range passageIndexes[artifactIndex] {
				candidate := math.Max(0, cosine(vectors[changeIndex], vectors[vectorIndex]))
				if candidate > score {
					score, bestPassage = candidate, passageIndex
				}
			}
			score = round(score)
			if score < threshold {
				continue
			}
			best := passages[artifactIndex][bestPassage]
			ranked = append(ranked, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence: []domain.Evidence{{
					Type: "SEMANTIC_SIMILARITY", Location: best.location,
					Excerpt: best.excerpt,
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
			"state", element.State, "gateway direction", element.GatewayDirection,
			"default flow", element.DefaultFlow, "documentation", element.Documentation)
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

func artifactPassages(artifact domain.Artifact) []passage {
	prefix := strings.Join(nonEmpty([]string{
		"repository artifact", artifact.File, "module", artifact.Module, "layer", artifact.Layer,
		"language", artifact.Language, "symbol", displaySymbol(artifact.Symbol), "kind", artifact.Symbol.Kind,
	}), " ")
	runes := []rune(artifact.Content)
	starts := []int{0}
	if len(runes) > maxPassageCharacters {
		starts = append(starts, len(runes)-maxPassageCharacters)
	}
	result := make([]passage, 0, len(starts))
	for _, start := range starts {
		end := start + maxPassageCharacters
		if end > len(runes) {
			end = len(runes)
		}
		content := strings.TrimSpace(string(runes[start:end]))
		excerptRunes := []rune(content)
		excerptStart := 0
		if start > 0 && len(excerptRunes) > 240 {
			excerptStart = len(excerptRunes) - 240
		}
		line := artifact.Symbol.StartLine
		if line == 0 {
			line = 1
		}
		line += strings.Count(string(runes[:start]), "\n") + strings.Count(string(excerptRunes[:excerptStart]), "\n")
		location := fmt.Sprintf("%s:%d", artifact.File, line)
		result = append(result, passage{
			text: prefix + " content " + content, excerpt: bounded(string(excerptRunes[excerptStart:]), 240), location: location,
		})
	}
	return result
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
