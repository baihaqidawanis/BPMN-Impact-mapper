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

func (Hybrid) MethodMetadata() map[string]string {
	return map[string]string{"evidence_version": "structured-span-evidence-v1", "query_version": "named-task-short-context-anchor-v1"}
}

func (Hybrid) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, config Config) ([]domain.Change, []domain.Warning, error) {
	attached, warnings := Attach(changes, artifacts, config)
	return attached, warnings, nil
}

// DiversifyFiles selects the highest-ranked artifact per file for a file-level
// report. Keep it disabled inside fusion so artifact ranks remain comparable.
type BM25Only struct{ DiversifyFiles bool }

func (BM25Only) Method() domain.AnalysisMethod { return domain.MethodBM25 }

func (BM25Only) MethodMetadata() map[string]string {
	return map[string]string{"query_version": "element-and-direct-branch-conditions-v2"}
}

func (strategy BM25Only) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, config Config) ([]domain.Change, []domain.Warning, error) {
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
		withoutContext := *change
		withoutContext.Context = nil
		// Branch conditions define the changed gateway itself. Keep them even
		// in the lexical baseline; process and neighbour expansion stay off.
		if change.Context != nil {
			withoutContext.Context = &domain.ChangeContext{BranchTerms: change.Context.BranchTerms}
		}
		query, _ := queryFor(withoutContext)
		ranked := removeShadowedPackageDocuments(model.Search(query))
		if len(ranked) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "BM25 found no lexical candidate")
			continue
		}
		maximum := ranked[0].Score
		seenFiles := map[string]bool{}
		for _, candidate := range ranked {
			if len(change.Impacts) >= config.TopK {
				break
			}
			normalized := round(candidate.Score / maximum)
			artifact := candidate.Artifact
			if strategy.DiversifyFiles && seenFiles[artifact.File] {
				continue
			}
			seenFiles[artifact.File] = true
			change.Impacts = append(change.Impacts, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence:       []domain.Evidence{evidenceAtQuery("BM25", artifact, query)},
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
		scored = propagateHTTPRoutes(scored, artifacts, query, config.CandidateThreshold)
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
	pathTokens := textutil.Tokenize(artifact.File)
	symbolTokens := textutil.Tokenize(artifact.Symbol.Name + " " + artifact.Symbol.Receiver)
	scope := insertTaskScopeTokens(change)
	trace := 0.0
	evidence := []domain.Evidence{}
	for _, id := range stableIDs {
		if containsStableID(artifact.Content, id) {
			trace = 1
			evidence = append(evidence, evidenceAtStableID(artifact, id))
			break
		}
	}
	symbol := math.Max(overlapInflections(symbolTokens, query), overlapInflections(symbolTokens, scope))
	if symbol > 0 {
		evidence = append(evidence, evidenceAt("SYMBOL", artifact, artifact.Symbol.Name))
	}
	pathScore := math.Max(overlapInflections(pathTokens, query), overlapInflections(pathTokens, scope))
	if pathScore > 0 {
		evidence = append(evidence, domain.Evidence{Type: "PATH", Location: artifact.File, Excerpt: bounded(artifact.File, 240)})
	}
	lexical := overlap(artifact.Tokens, query)
	if lexical > 0 {
		evidence = append(evidence, evidenceAtQuery("LABEL_TOKEN", artifact, query))
	}
	structured, structuredEvidence := structuredSignal(artifact, structuralQueryFor(change, query))
	if structured > 0 {
		evidence = append(evidence, structuredEvidence)
	}
	bm25Score := 0.0
	if maxBM25 > 0 {
		bm25Score = rawBM25 / maxBM25
	}
	if bm25Score > 0 {
		evidence = append(evidence, evidenceAtQuery("BM25", artifact, query))
	}
	total := 0.45*trace + 0.15*symbol + 0.10*pathScore + 0.10*lexical + 0.10*structured + 0.10*bm25Score
	total = round(total)
	if total < config.CandidateThreshold || len(evidence) == 0 {
		return domain.Impact{}, false
	}
	// Neighbour IDs explain the context that already contributed query terms.
	// They are not direct traces of the changed element, cannot earn its 45%
	// contribution, and must never promote an action or seed propagation.
	evidence = append(evidence, contextTraceEvidence(change, artifact, stableIDs)...)
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
		ScoreBreakdown: domain.ScoreBreakdown{EvidenceScore: &total, TraceID: trace, Symbol: round(symbol), Path: round(pathScore), Lexical: round(lexical), Structured: round(structured), BM25: round(bm25Score), Total: total},
		Confidence:     total,
	}, true
}

// Scope terms must occur in the changed task label and at least one adjacent
// BPMN label. This gives a new task's known process context a fair chance in
// path/function matching without manufacturing a direct task-ID trace.
func insertTaskScopeTokens(change domain.Change) []string {
	if change.ChangeType != domain.AddTask || change.NewElement == nil || change.Context == nil || change.Context.Pattern != domain.PatternInsertTask {
		return nil
	}
	neighbors := append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...)
	neighborSet := map[string]bool{}
	for _, neighbor := range neighbors {
		for _, token := range textutil.Tokenize(neighbor.Name) {
			neighborSet[token] = true
		}
	}
	shared := []string{}
	for _, token := range textutil.UniqueSorted(textutil.Tokenize(change.NewElement.Name)) {
		if len([]rune(token)) >= 4 && (neighborSet[token] || containsInflection(neighborSet, token)) {
			shared = append(shared, token)
		}
	}
	if len(shared) < 2 {
		return nil
	}
	return shared
}

// Structural clues are about the changed element, not all adjacent BPMN
// labels. Neighbour terms remain available to lexical/BM25 retrieval but
// cannot make an unrelated table or API route look like direct task evidence.
func structuralQueryFor(change domain.Change, fallback []string) []string {
	parts := []string{}
	for _, element := range queryElements(change) {
		if element == nil {
			continue
		}
		parts = append(parts, element.Name, element.State)
		if (change.ChangeType == domain.AddTask || change.ChangeType == domain.RemoveTask) && strings.TrimSpace(element.Name) != "" {
			parts = append(parts, textutil.ContextAnchor(element.Documentation, retrievalNeighborNames(change.Context)))
		} else {
			parts = append(parts, element.Documentation)
		}
		parts = append(parts, element.DataRequirements...)
	}
	tokens := textutil.UniqueSorted(textutil.Tokenize(strings.Join(parts, " ")))
	if len(tokens) == 0 {
		return fallback
	}
	return tokens
}

func contextTraceEvidence(change domain.Change, artifact domain.Artifact, directIDs []string) []domain.Evidence {
	if change.Context == nil {
		return nil
	}
	seen := make(map[string]bool, len(directIDs))
	for _, id := range directIDs {
		seen[strings.ToLower(id)] = true
	}
	var result []domain.Evidence
	neighbors := append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...)
	for _, neighbor := range neighbors {
		key := strings.ToLower(neighbor.ID)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if containsStableID(artifact.Content, neighbor.ID) {
			item := evidenceAtStableID(artifact, neighbor.ID)
			item.Type = "CONTEXT_TRACE_ID"
			result = append(result, item)
		}
	}
	return result
}

func queryFor(change domain.Change) ([]string, []string) {
	var raw []string
	var ids []string
	for _, element := range queryElements(change) {
		if element == nil {
			continue
		}
		ids = append(ids, element.ID)
		// Generated BPMN IDs, flow IDs, and change-type words are structural
		// metadata. Splitting them into lexical terms makes unrelated source
		// artifacts look relevant (for example every task mentioning "flow").
		// Direct IDs are matched independently as exact trace evidence.
		raw = append(raw, element.Name, element.State)
		if (change.ChangeType == domain.AddTask || change.ChangeType == domain.RemoveTask) && strings.TrimSpace(element.Name) != "" {
			raw = append(raw, textutil.ContextAnchor(element.Documentation, retrievalNeighborNames(change.Context)))
		} else {
			raw = append(raw, element.Documentation)
		}
		if change.ChangeType == domain.ModifyActorRole {
			raw = append(raw, element.Actor)
		}
		if strings.HasSuffix(element.Type, "Gateway") {
			raw = append(raw, element.GatewayDirection)
		}
		raw = append(raw, element.DataRequirements...)
	}
	if change.Context != nil {
		raw = append(raw, change.Context.ProcessName)
		raw = append(raw, change.Context.BranchTerms...)
		for _, element := range append(append([]domain.ElementContext{}, change.Context.Predecessors...), change.Context.Successors...) {
			raw = append(raw, element.Name)
		}
	}
	if len(raw) == 0 || strings.TrimSpace(strings.Join(raw, "")) == "" {
		// Unlabelled BPMN elements still need a lexical fallback. Exact-ID
		// evidence remains the decisive signal for these cases.
		raw = append(raw, ids...)
	}
	return textutil.UniqueSorted(textutil.Tokenize(strings.Join(raw, " "))), textutil.UniqueSorted(ids)
}

func retrievalNeighborNames(context *domain.ChangeContext) []string {
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

// overlapInflections extends exact token overlap only for common grammatical
// variants. It is intentionally limited to symbol and path signals: BM25 and
// lexical evidence retain their exact, auditable token semantics.
func overlapInflections(values, query []string) float64 {
	valueSet := make(map[string]bool, len(values))
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
	for queryToken := range querySet {
		if valueSet[queryToken] || containsInflection(valueSet, queryToken) {
			matches++
		}
	}
	return math.Min(1, float64(matches)/math.Min(5, float64(len(querySet))))
}

func containsInflection(values map[string]bool, query string) bool {
	if len(query) < 4 {
		return false
	}
	variants := []string{query + "s", query + "es", query + "ed", query + "ing"}
	if strings.HasSuffix(query, "e") && len(query) > 4 {
		stem := strings.TrimSuffix(query, "e")
		variants = append(variants, stem+"ed", stem+"ing")
	}
	if strings.HasSuffix(query, "y") && len(query) > 4 {
		variants = append(variants, strings.TrimSuffix(query, "y")+"ies")
	}
	for _, variant := range variants {
		if values[variant] {
			return true
		}
	}
	for value := range values {
		if len(value) >= 4 && inflectsTo(value, query) {
			return true
		}
	}
	return false
}

func inflectsTo(base, variant string) bool {
	if variant == base+"s" || variant == base+"es" || variant == base+"ed" || variant == base+"ing" {
		return true
	}
	if strings.HasSuffix(base, "e") && len(base) > 4 {
		stem := strings.TrimSuffix(base, "e")
		if variant == stem+"ed" || variant == stem+"ing" {
			return true
		}
	}
	return strings.HasSuffix(base, "y") && len(base) > 4 && variant == strings.TrimSuffix(base, "y")+"ies"
}

// containsStableID matches an ID at identifier boundaries. A hyphen is kept as
// a boundary so an operation ID such as TASK-001-create still carries the BPMN
// trace, while TASK-0010 and XTASK-001 do not collide with TASK-001.
func containsStableID(content, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	contentRunes := []rune(strings.ToLower(content))
	idRunes := []rune(strings.ToLower(id))
	if len(idRunes) > len(contentRunes) {
		return false
	}
	for start := 0; start+len(idRunes) <= len(contentRunes); start++ {
		if string(contentRunes[start:start+len(idRunes)]) != string(idRunes) {
			continue
		}
		beforeOK := start == 0 || !isIdentifierRune(contentRunes[start-1])
		after := start + len(idRunes)
		afterOK := after == len(contentRunes) || !isIdentifierRune(contentRunes[after])
		if beforeOK && afterOK {
			return true
		}
	}
	return false
}

func isIdentifierRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_'
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

func evidenceAtStableID(artifact domain.Artifact, id string) domain.Evidence {
	lines := strings.Split(artifact.Content, "\n")
	for index, line := range lines {
		if !containsStableID(line, id) {
			continue
		}
		lineNumber := index + 1
		if artifact.Symbol.StartLine > 0 {
			lineNumber += artifact.Symbol.StartLine - 1
		}
		return domain.Evidence{
			Type:     "TRACE_ID",
			Location: fmt.Sprintf("%s:%d", artifact.File, lineNumber),
			Excerpt:  bounded(strings.TrimSpace(line), 240),
		}
	}
	return domain.Evidence{Type: "TRACE_ID", Location: artifact.File, Excerpt: bounded(id, 240)}
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

// A lexical evidence excerpt must show the strongest local overlap, not the
// first occurrence of a common query word in a long documentation file.
// When BM25 matched only path/symbol tokens, cite the path instead of an
// unrelated first source line.
func evidenceAtQuery(kind string, artifact domain.Artifact, query []string) domain.Evidence {
	terms := make(map[string]bool, len(query))
	for _, term := range query {
		if len(term) > 1 {
			terms[term] = true
		}
	}
	bestLine, bestCount, bestExcerpt := 0, 0, ""
	for index, line := range strings.Split(artifact.Content, "\n") {
		count := 0
		seen := make(map[string]bool)
		for _, token := range textutil.Tokenize(line) {
			if terms[token] && !seen[token] {
				count++
				seen[token] = true
			}
		}
		if count > bestCount {
			bestLine, bestCount = index+1, count
			bestExcerpt = bounded(strings.TrimSpace(line), 240)
		}
	}
	if bestLine == 0 {
		return domain.Evidence{Type: kind, Location: artifact.File, Excerpt: bounded(artifact.File, 240)}
	}
	if artifact.Symbol.StartLine > 0 {
		bestLine += artifact.Symbol.StartLine - 1
	}
	return domain.Evidence{Type: kind, Location: fmt.Sprintf("%s:%d", artifact.File, bestLine), Excerpt: bestExcerpt}
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
