package retrieval

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

type Config = mapping.Options

func DefaultConfig() Config {
	return Config{TopK: 10, CandidateThreshold: 0.20, HighThreshold: 0.70, TieDelta: 0.01}
}

type Hybrid struct{}

type artifactImpact struct {
	impact     domain.Impact
	symbolKind string
}

func (Hybrid) Method() domain.AnalysisMethod { return domain.MethodHybrid }

func (Hybrid) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, config Config) ([]domain.Change, []domain.Warning, error) {
	attached, warnings := Attach(changes, artifacts, config)
	return attached, warnings, nil
}

type BM25Only struct{}

func (BM25Only) Method() domain.AnalysisMethod { return domain.MethodBM25 }

func (BM25Only) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, config Config) ([]domain.Change, []domain.Warning, error) {
	defaults := DefaultConfig()
	if config.TopK <= 0 {
		config.TopK = defaults.TopK
	}
	model := NewBM25(artifacts)
	for changeIndex := range changes {
		change := &changes[changeIndex]
		change.Impacts = []domain.Impact{}
		if change.ChangeType == domain.UnresolvedMatch {
			if len(change.UnresolvedImpact) == 0 {
				change.UnresolvedImpact = []string{"graph element matching is ambiguous"}
			}
			continue
		}
		query, _ := queryFor(*change)
		ranked := removeShadowedPackageDocuments(model.Search(query))
		if len(ranked) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "BM25 found no lexical candidate")
			continue
		}
		maximum := ranked[0].Score
		for _, candidate := range ranked {
			if len(change.Impacts) >= config.TopK {
				break
			}
			normalized := round(candidate.Score / maximum)
			artifact := candidate.Artifact
			change.Impacts = append(change.Impacts, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence:       []domain.Evidence{evidenceAt("BM25", artifact, firstPresent(query, artifact.Content))},
				ScoreBreakdown: domain.ScoreBreakdown{BM25: normalized, Total: normalized}, Confidence: normalized,
			})
		}
	}
	return changes, []domain.Warning{}, nil
}

func Attach(changes []domain.Change, artifacts []domain.Artifact, config Config) ([]domain.Change, []domain.Warning) {
	defaults := DefaultConfig()
	if config.TopK <= 0 {
		config.TopK = defaults.TopK
	}
	if config.CandidateThreshold <= 0 {
		config.CandidateThreshold = defaults.CandidateThreshold
	}
	if config.HighThreshold <= 0 {
		config.HighThreshold = defaults.HighThreshold
	}
	if config.TieDelta <= 0 {
		config.TieDelta = defaults.TieDelta
	}
	model := NewBM25(artifacts)
	warnings := []domain.Warning{}
	for changeIndex := range changes {
		change := &changes[changeIndex]
		if change.ChangeType == domain.UnresolvedMatch {
			change.Impacts = []domain.Impact{}
			if len(change.UnresolvedImpact) == 0 {
				change.UnresolvedImpact = []string{"graph element matching is ambiguous"}
			}
			continue
		}
		query, stableIDs := queryFor(*change)
		ranked := removeShadowedPackageDocuments(model.Search(query))
		bm25 := map[string]float64{}
		maxBM25 := 0.0
		for _, result := range ranked {
			bm25[result.Artifact.ID] = result.Score
			if result.Score > maxBM25 {
				maxBM25 = result.Score
			}
		}
		scored := make([]artifactImpact, 0)
		for _, artifact := range artifacts {
			impact, ok := scoreArtifact(*change, artifact, query, stableIDs, bm25[artifact.ID], maxBM25, config)
			if ok {
				scored = append(scored, artifactImpact{impact: impact, symbolKind: artifact.Symbol.Kind})
			}
		}
		scored = propagate(scored, artifacts, config.CandidateThreshold)
		impacts := removeShadowedPackageImpacts(scored)
		sort.SliceStable(impacts, func(i, j int) bool {
			if math.Abs(impacts[i].Confidence-impacts[j].Confidence) > 1e-12 {
				return impacts[i].Confidence > impacts[j].Confidence
			}
			if impacts[i].File != impacts[j].File {
				return impacts[i].File < impacts[j].File
			}
			return impacts[i].Symbol < impacts[j].Symbol
		})
		if len(impacts) > config.TopK {
			impacts = impacts[:config.TopK]
		}
		change.Impacts = impacts
		if len(impacts) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "no repository candidate met the evidence threshold")
		} else if len(impacts) > 1 && math.Abs(impacts[0].Confidence-impacts[1].Confidence) <= config.TieDelta {
			warnings = append(warnings, domain.Warning{Code: "CANDIDATE_TIE", Message: fmt.Sprintf("%s has top candidates within %.2f", changeIdentifier(*change), config.TieDelta)})
		}
	}
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Code != warnings[j].Code {
			return warnings[i].Code < warnings[j].Code
		}
		return warnings[i].Message < warnings[j].Message
	})
	return changes, warnings
}

func removeShadowedPackageDocuments(ranked []RankedDocument) []RankedDocument {
	hasSpecific := make(map[string]bool)
	for _, candidate := range ranked {
		kind := candidate.Artifact.Symbol.Kind
		if kind != "" && kind != "package" {
			hasSpecific[candidate.Artifact.File] = true
		}
	}
	filtered := make([]RankedDocument, 0, len(ranked))
	for _, candidate := range ranked {
		if candidate.Artifact.Symbol.Kind == "package" && hasSpecific[candidate.Artifact.File] {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

// removeShadowedPackageImpacts keeps file-level Go evidence only when no
// accepted type/function/method candidate from the same file explains it more
// precisely. Multiple genuinely matching symbols remain separate candidates.
func removeShadowedPackageImpacts(scored []artifactImpact) []domain.Impact {
	hasSpecific := make(map[string]bool)
	for _, candidate := range scored {
		if candidate.symbolKind != "" && candidate.symbolKind != "package" {
			hasSpecific[candidate.impact.File] = true
		}
	}
	impacts := make([]domain.Impact, 0, len(scored))
	for _, candidate := range scored {
		if candidate.symbolKind == "package" && hasSpecific[candidate.impact.File] {
			continue
		}
		impacts = append(impacts, candidate.impact)
	}
	return impacts
}

func scoreArtifact(change domain.Change, artifact domain.Artifact, query, stableIDs []string, rawBM25, maxBM25 float64, config Config) (domain.Impact, bool) {
	contentLower := strings.ToLower(artifact.Content)
	pathTokens := textutil.Tokenize(artifact.File)
	symbolTokens := textutil.Tokenize(artifact.Symbol.Name + " " + artifact.Symbol.Receiver)
	trace := 0.0
	evidence := []domain.Evidence{}
	for _, id := range stableIDs {
		if id != "" && strings.Contains(contentLower, strings.ToLower(id)) {
			trace = 1
			evidence = append(evidence, evidenceAt("TRACE_ID", artifact, id))
			break
		}
	}
	symbol := overlap(symbolTokens, query)
	if symbol > 0 {
		evidence = append(evidence, evidenceAt("SYMBOL", artifact, artifact.Symbol.Name))
	}
	pathScore := overlap(pathTokens, query)
	if pathScore > 0 {
		evidence = append(evidence, domain.Evidence{Type: "PATH", Location: artifact.File, Excerpt: bounded(artifact.File, 240)})
	}
	lexical := overlap(artifact.Tokens, query)
	if lexical > 0 {
		evidence = append(evidence, evidenceAt("LABEL_TOKEN", artifact, firstPresent(query, artifact.Content)))
	}
	structured, structuredType := structuredSignal(artifact, query)
	if structured > 0 {
		evidence = append(evidence, evidenceAt(structuredType, artifact, firstPresent(query, artifact.Content)))
	}
	bm25Score := 0.0
	if maxBM25 > 0 {
		bm25Score = rawBM25 / maxBM25
	}
	if bm25Score > 0 {
		evidence = append(evidence, evidenceAt("BM25", artifact, firstPresent(query, artifact.Content)))
	}
	total := 0.45*trace + 0.15*symbol + 0.10*pathScore + 0.10*lexical + 0.10*structured + 0.10*bm25Score
	total = round(total)
	if total < config.CandidateThreshold || len(evidence) == 0 {
		return domain.Impact{}, false
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		if evidence[i].Type != evidence[j].Type {
			return evidence[i].Type < evidence[j].Type
		}
		if evidence[i].Location != evidence[j].Location {
			return evidence[i].Location < evidence[j].Location
		}
		return evidence[i].Excerpt < evidence[j].Excerpt
	})
	action := domain.ActionVerify
	if trace == 1 && total >= config.HighThreshold {
		action = strongAction(change.ChangeType)
	}
	return domain.Impact{
		Module: artifact.Module, Layer: artifact.Layer, File: artifact.File, Symbol: displaySymbol(artifact.Symbol),
		RequiredAction: action, Evidence: evidence,
		ScoreBreakdown: domain.ScoreBreakdown{TraceID: trace, Symbol: round(symbol), Path: round(pathScore), Lexical: round(lexical), Structured: round(structured), BM25: round(bm25Score), Total: total},
		Confidence:     total,
	}, true
}

func queryFor(change domain.Change) ([]string, []string) {
	var raw []string
	var ids []string
	for _, element := range queryElements(change) {
		if element == nil {
			continue
		}
		ids = append(ids, element.ID)
		raw = append(raw, element.ID, element.Name, element.Actor, element.State, element.Documentation)
		raw = append(raw, element.DataRequirements...)
		raw = append(raw, element.Incoming...)
		raw = append(raw, element.Outgoing...)
	}
	if change.Context != nil {
		raw = append(raw, string(change.Context.Pattern))
		for _, element := range append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...) {
			raw = append(raw, element.ID, element.Name, element.Type)
		}
		raw = append(raw, change.Context.RelatedFlowIDs...)
	}
	raw = append(raw, string(change.ChangeType))
	return textutil.UniqueSorted(textutil.Tokenize(strings.Join(raw, " "))), textutil.UniqueSorted(ids)
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

func overlap(values, query []string) float64 {
	valueSet := map[string]bool{}
	for _, value := range values {
		valueSet[value] = true
	}
	querySet := map[string]bool{}
	for _, value := range query {
		if len(value) > 1 {
			querySet[value] = true
		}
	}
	if len(querySet) == 0 {
		return 0
	}
	matches := 0
	for value := range querySet {
		if valueSet[value] {
			matches++
		}
	}
	return math.Min(1, float64(matches)/math.Min(5, float64(len(querySet))))
}

func structuredSignal(artifact domain.Artifact, query []string) (float64, string) {
	lower := strings.ToLower(artifact.Content)
	if firstPresent(query, artifact.Content) == "" {
		return 0, ""
	}
	switch {
	case artifact.Layer == "test" || strings.Contains(strings.ToLower(artifact.File), "test"):
		return 1, "TEST_NAME"
	case strings.Contains(lower, "/api/") || strings.Contains(lower, "handlefunc") || strings.Contains(lower, "router."):
		return 1, "API_ROUTE"
	case strings.Contains(lower, "insert into") || strings.Contains(lower, "create table") || strings.Contains(lower, "select "):
		return 1, "DATABASE_TABLE"
	case strings.Contains(lower, "audit") || strings.Contains(lower, "trace"):
		return 1, "AUDIT_ACTION"
	case containsUpperConstant(artifact.Content):
		return 0.7, "STATE_STATUS"
	default:
		return 0, ""
	}
}

func containsUpperConstant(content string) bool {
	for _, field := range strings.Fields(content) {
		trimmed := strings.Trim(field, "\"'`,;:()[]{}")
		if len(trimmed) < 3 {
			continue
		}
		letters, upper := 0, 0
		for _, r := range trimmed {
			if unicode.IsLetter(r) {
				letters++
				if unicode.IsUpper(r) {
					upper++
				}
			}
		}
		if letters >= 3 && letters == upper {
			return true
		}
	}
	return false
}

func evidenceAt(kind string, artifact domain.Artifact, needle string) domain.Evidence {
	lineNumber, excerpt := excerpt(artifact.Content, needle)
	location := artifact.File
	if artifact.Symbol.StartLine > 0 {
		lineNumber += artifact.Symbol.StartLine - 1
	}
	if lineNumber > 0 {
		location = fmt.Sprintf("%s:%d", artifact.File, lineNumber)
	}
	return domain.Evidence{Type: kind, Location: location, Excerpt: excerpt}
}

func excerpt(content, needle string) (int, string) {
	lines := strings.Split(content, "\n")
	needle = strings.ToLower(needle)
	for index, line := range lines {
		if needle == "" || strings.Contains(strings.ToLower(line), needle) {
			return index + 1, bounded(strings.TrimSpace(line), 240)
		}
	}
	if len(lines) > 0 {
		return 1, bounded(strings.TrimSpace(lines[0]), 240)
	}
	return 0, ""
}

func bounded(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}

func firstPresent(query []string, content string) string {
	lower := strings.ToLower(content)
	for _, token := range query {
		if len(token) > 1 && strings.Contains(lower, token) {
			return token
		}
	}
	return ""
}

func strongAction(change domain.ChangeType) domain.RequiredAction {
	switch change {
	case domain.AddTask, domain.AddGateway, domain.AddDataRequirement, domain.AddState:
		return domain.ActionAdd
	case domain.RemoveTask, domain.RemoveGateway, domain.RemoveDataRequirement:
		return domain.ActionRemove
	case domain.ModifyTaskLabel:
		// A wording change can be directly traceable without changing software
		// behavior. Exact ID evidence therefore warrants review, not an inferred
		// code modification.
		return domain.ActionVerify
	default:
		return domain.ActionModify
	}
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
