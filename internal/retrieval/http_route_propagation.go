package retrieval

import (
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// Route propagation is deliberately forward-only. The frontend must already
// have lexical evidence on the exact call line, or that call line must share
// two meaningful query words. Merely sharing a large file with a relevant
// candidate does not activate every endpoint in that file.
// Linked candidates remain VERIFY. The exact method+path handler link carries
// the source relevance score; one additional static service call decays by
// 0.5. These are ranking heuristics, not probabilities of a required edit.
func propagateHTTPRoutes(scored []artifactImpact, artifacts []domain.Artifact, query []string, threshold float64) []artifactImpact {
	byID := make(map[string]domain.Artifact, len(artifacts))
	for _, artifact := range artifacts {
		byID[artifact.ID] = artifact
	}
	index := make(map[string]int, len(scored))
	for i, candidate := range scored {
		index[impactKey(candidate.impact)] = i
	}
	for _, client := range artifacts {
		seedIndex, ok := index[impactKeyForArtifact(client)]
		if !ok {
			continue
		}
		seed := scored[seedIndex].impact
		for _, edge := range client.Relations {
			if edge.Kind != domain.RelationHTTPRoute || (!hasRouteLineEvidence(seed, edge.Location) && !routeMatchesQuery(edge.Excerpt, query)) {
				continue
			}
			handler, ok := byID[edge.TargetID]
			if !ok {
				continue
			}
			var linked bool
			scored, linked = routeCandidate(scored, index, handler, seed.Confidence, threshold,
				[]domain.Evidence{
					{Type: "HTTP_CLIENT_CALL", Location: edge.Location, Excerpt: edge.Excerpt},
					{Type: "HTTP_ROUTE_HANDLER", Location: edge.TargetLocation, Excerpt: edge.TargetExcerpt},
				})
			if !linked {
				continue
			}
			// One further static Go call from the resolved handler can identify a
			// service. Do not recurse through arbitrary call chains.
			for _, call := range handler.Relations {
				if _, _, supported := relationEvidence(call.Kind); !supported {
					continue
				}
				service, ok := byID[call.TargetID]
				if !ok {
					continue
				}
				scored, _ = routeCandidate(scored, index, service, round(seed.Confidence*0.5), threshold,
					[]domain.Evidence{
						{Type: "HTTP_ROUTE_HANDLER", Location: edge.TargetLocation, Excerpt: edge.TargetExcerpt},
						{Type: "HANDLER_CALLS", Location: call.Location, Excerpt: call.Excerpt},
					})
			}
		}
	}
	return scored
}

func hasRouteLineEvidence(impact domain.Impact, location string) bool {
	for _, evidence := range impact.Evidence {
		if evidence.Location == location && (evidence.Type == "BM25" || evidence.Type == "LABEL_TOKEN") {
			return true
		}
	}
	return false
}

func routeMatchesQuery(excerpt string, query []string) bool {
	words := make([]string, 0)
	for _, word := range textutil.Tokenize(excerpt) {
		if len(word) >= 4 {
			words = append(words, word)
		}
	}
	words = textutil.UniqueSorted(words)
	matches := 0
	used := make(map[string]bool)
	for _, word := range textutil.UniqueSorted(query) {
		if len(word) < 4 {
			continue
		}
		for _, candidate := range words {
			if used[candidate] {
				continue
			}
			if candidate == word || inflectsTo(word, candidate) || inflectsTo(candidate, word) {
				used[candidate] = true
				matches++
				break
			}
		}
	}
	return matches >= 2
}

func routeCandidate(scored []artifactImpact, index map[string]int, artifact domain.Artifact, score, threshold float64, evidence []domain.Evidence) ([]artifactImpact, bool) {
	key := impactKeyForArtifact(artifact)
	if i, ok := index[key]; ok {
		scored[i].impact.Evidence = appendUniqueEvidence(scored[i].impact.Evidence, evidence)
		return scored, true
	}
	if score < threshold {
		return scored, false
	}
	impact := domain.Impact{File: artifact.File, Module: artifact.Module, Layer: artifact.Layer, Symbol: displaySymbol(artifact.Symbol),
		RequiredAction: domain.ActionVerify, Confidence: score,
		ScoreBreakdown: domain.ScoreBreakdown{Dependency: score, Total: score}, Evidence: evidence}
	index[key] = len(scored)
	return append(scored, artifactImpact{impact: impact, symbolKind: artifact.Symbol.Kind}), true
}

func appendUniqueEvidence(existing, added []domain.Evidence) []domain.Evidence {
	for _, item := range added {
		found := false
		for _, prior := range existing {
			if prior == item {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, item)
		}
	}
	sort.Slice(existing, func(i, j int) bool {
		if existing[i].Type != existing[j].Type {
			return existing[i].Type < existing[j].Type
		}
		if existing[i].Location != existing[j].Location {
			return existing[i].Location < existing[j].Location
		}
		return existing[i].Excerpt < existing[j].Excerpt
	})
	return existing
}

func impactKey(impact domain.Impact) string { return impact.File + "\x00" + impact.Symbol }
func impactKeyForArtifact(artifact domain.Artifact) string {
	return artifact.File + "\x00" + displaySymbol(artifact.Symbol)
}
