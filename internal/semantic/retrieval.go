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
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// Short, evenly spaced passages leave room for file/function metadata within
// the model's token window. These are character bounds, not tokenizer guarantees.
const (
	maxPassageCharacters  = 600
	maxEvidenceCharacters = 240
)

type passage struct {
	text      string
	content   string
	startLine int
}

type Strategy struct {
	Embedder       Embedder
	DiversifyFiles bool
}

func (Strategy) Method() domain.AnalysisMethod { return domain.MethodSemantic }

func (strategy Strategy) MethodMetadata() map[string]string {
	if strategy.Embedder == nil {
		return nil
	}
	identity := strategy.Embedder.Identity()
	metadata := map[string]string{"provider": identity.Provider, "model": identity.Model, "adapter_sha256": identity.AdapterSHA256, "query_version": "named-task-short-context-anchor-v1", "passage_version": "head-middle-tail-v1"}
	if identity.PythonVersion != "" {
		metadata["python_version"] = identity.PythonVersion
		metadata["sentence_transformers_version"] = identity.SentenceTransformersVersion
		metadata["model_artifact_sha256"] = identity.ModelArtifactSHA256
		metadata["embedding_dimension"] = fmt.Sprint(identity.EmbeddingDimension)
	}
	if strategy.DiversifyFiles {
		metadata["display_selection"] = "highest-ranked-candidate-per-file-before-top-k"
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
	// Every artifact gets a first passage. Tail, then middle passages are added
	// round-robin where budget permits, without query-dependent corpus selection.
	remaining := maxEmbeddingTexts - len(changes) - len(artifacts)
	for index, artifact := range artifacts {
		passages[index] = artifactPassages(artifact)
		passageIndexes[index] = append(passageIndexes[index], len(texts))
		texts = append(texts, passages[index][0].text)
	}
	for passageNumber := 1; passageNumber < 3 && remaining > 0; passageNumber++ {
		for index := range artifacts {
			if remaining == 0 {
				break
			}
			if len(passages[index]) > passageNumber {
				passageIndexes[index] = append(passageIndexes[index], len(texts))
				texts = append(texts, passages[index][passageNumber].text)
				remaining--
			}
		}
	}
	omittedPassages := 0
	for index := range artifacts {
		omittedPassages += len(passages[index]) - len(passageIndexes[index])
	}
	vectors, err := strategy.Embedder.Embed(ctx, texts)
	if err != nil {
		return nil, nil, err
	}
	warnings := []domain.Warning{}
	if omittedPassages > 0 {
		warnings = append(warnings, domain.Warning{Code: "SEMANTIC_PASSAGE_LIMIT", Message: fmt.Sprintf("%d additional source passages omitted because of the embedding input limit", omittedPassages)})
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
			location, excerpt := passageEvidence(best, queryText(*change), artifact.File)
			ranked = append(ranked, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence: []domain.Evidence{{
					Type: "SEMANTIC_SIMILARITY", Location: location,
					Excerpt: excerpt,
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
		if strategy.DiversifyFiles {
			selected := make([]domain.Impact, 0, len(ranked))
			seenFiles := make(map[string]bool, len(ranked))
			for _, impact := range ranked {
				if seenFiles[impact.File] {
					continue
				}
				seenFiles[impact.File] = true
				selected = append(selected, impact)
			}
			ranked = selected
		}
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
	if element := namedAddedOrRemovedTask(change); element != nil {
		parts := []string{element.Name, element.State}
		parts = append(parts, element.DataRequirements...)
		if anchor := textutil.ContextAnchor(element.Documentation, neighborNames(change.Context)); anchor != "" && !strings.Contains(strings.ToLower(element.Name), anchor) {
			parts = append(parts, anchor)
		}
		return strings.Join(nonEmpty(parts), " ")
	}
	parts := []string{}
	hasElementMeaning := false
	for _, element := range queryElements(change) {
		if element == nil {
			continue
		}
		if element.Name != "" {
			parts = append(parts, element.Name)
			hasElementMeaning = true
		} else {
			parts = append(parts, element.Type)
		}
		parts = append(parts, element.State, element.Documentation)
		if element.State != "" || element.Documentation != "" || len(element.DataRequirements) > 0 {
			hasElementMeaning = true
		}
		if change.ChangeType == domain.ModifyActorRole {
			parts = append(parts, element.Actor)
		}
		if strings.HasSuffix(element.Type, "Gateway") {
			parts = append(parts, element.GatewayDirection)
		}
		parts = append(parts, element.DataRequirements...)
	}
	// The named add/remove case returned above. Other changes retain graph
	// context; an unlabeled add/remove may need it as its only meaningful text.
	useContext := change.Context != nil
	if (change.ChangeType == domain.AddTask || change.ChangeType == domain.RemoveTask) && hasElementMeaning {
		useContext = false
	}
	if useContext {
		for _, element := range append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...) {
			parts = append(parts, element.Name)
		}
	}
	if len(nonEmpty(parts)) == 0 {
		parts = append(parts, change.Reason, string(change.ChangeType))
	}
	return strings.Join(nonEmpty(parts), " ")
}

func namedAddedOrRemovedTask(change domain.Change) *domain.Element {
	var element *domain.Element
	switch change.ChangeType {
	case domain.AddTask:
		element = change.NewElement
	case domain.RemoveTask:
		element = change.OldElement
	}
	if element == nil || strings.TrimSpace(element.Name) == "" {
		return nil
	}
	return element
}

func neighborNames(context *domain.ChangeContext) []string {
	if context == nil {
		return nil
	}
	names := make([]string, 0, len(context.Predecessors)+len(context.Successors))
	for _, element := range context.Predecessors {
		names = append(names, element.Name)
	}
	for _, element := range context.Successors {
		names = append(names, element.Name)
	}
	return names
}

func artifactPassages(artifact domain.Artifact) []passage {
	prefix := strings.Join(nonEmpty([]string{
		"repository artifact", artifact.File, "module", artifact.Module, "layer", artifact.Layer,
		"language", artifact.Language, "symbol", displaySymbol(artifact.Symbol), "kind", artifact.Symbol.Kind,
	}), " ")
	runes := []rune(artifact.Content)
	starts := []int{0}
	if len(runes) > maxPassageCharacters {
		tail := len(runes) - maxPassageCharacters
		starts = append(starts, tail)
		if len(runes) > 2*maxPassageCharacters {
			starts = append(starts, tail/2)
		}
	}
	result := make([]passage, 0, len(starts))
	for _, start := range starts {
		end := start + maxPassageCharacters
		if end > len(runes) {
			end = len(runes)
		}
		segment := string(runes[start:end])
		leading := len(segment) - len(strings.TrimLeft(segment, " \t\r\n"))
		content := strings.TrimSpace(segment)
		line := artifact.Symbol.StartLine
		if line == 0 {
			line = 1
		}
		line += strings.Count(string(runes[:start]), "\n") + strings.Count(segment[:leading], "\n")
		result = append(result, passage{
			text: prefix + " content " + content, content: content, startLine: line,
		})
	}
	return result
}

// The cosine score belongs to the entire bounded passage, not one line. Show
// the strongest lexical line when present; otherwise show its center rather
// than always citing an unrelated return/commit at the end of the passage.
func passageEvidence(best passage, query, file string) (string, string) {
	querySet := map[string]bool{}
	for _, token := range textutil.Tokenize(query) {
		querySet[token] = true
	}
	bestCount, bestOffset, offset := 0, 0, 0
	bestLine := ""
	for _, line := range strings.Split(best.content, "\n") {
		seen := map[string]bool{}
		count := 0
		for _, token := range textutil.Tokenize(line) {
			if querySet[token] && !seen[token] {
				seen[token] = true
				count++
			}
		}
		if count > bestCount {
			bestCount, bestOffset, bestLine = count, offset, line
		}
		offset += len([]rune(line)) + 1
	}
	if bestCount > 0 {
		lineNumber := best.startLine + strings.Count(string([]rune(best.content)[:bestOffset]), "\n")
		return fmt.Sprintf("%s:%d", file, lineNumber), excerptAroundQuery(bestLine, querySet)
	}
	runes := []rune(best.content)
	start := 0
	if len(runes) > maxEvidenceCharacters {
		start = (len(runes) - maxEvidenceCharacters) / 2
	}
	end := start + maxEvidenceCharacters
	if end > len(runes) {
		end = len(runes)
	}
	lineNumber := best.startLine + strings.Count(string(runes[:start]), "\n")
	return fmt.Sprintf("%s:%d", file, lineNumber), strings.TrimSpace(string(runes[start:end]))
}

func excerptAroundQuery(line string, querySet map[string]bool) string {
	runes := []rune(strings.TrimSpace(line))
	if len(runes) <= maxEvidenceCharacters {
		return string(runes)
	}
	lower := strings.ToLower(string(runes))
	match := -1
	for token := range querySet {
		if index := strings.Index(lower, token); index >= 0 && (match < 0 || index < match) {
			match = index
		}
	}
	start := 0
	if match >= 0 {
		start = len([]rune(lower[:match])) - 80
		if start < 0 {
			start = 0
		}
	}
	if start+maxEvidenceCharacters > len(runes) {
		start = len(runes) - maxEvidenceCharacters
	}
	return strings.TrimSpace(string(runes[start : start+maxEvidenceCharacters]))
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
