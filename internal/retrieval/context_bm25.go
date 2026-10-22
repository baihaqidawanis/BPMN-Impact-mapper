package retrieval

import (
	"context"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// ContextBM25 is the lexical branch of the production hybrid. BPMN context
// forms the query; source clues explain results but never alter their rank.
type ContextBM25 struct{}

func (ContextBM25) Method() domain.AnalysisMethod { return domain.MethodHybrid }

func (ContextBM25) MethodMetadata() map[string]string {
	return map[string]string{"ranking_version": "context-bm25-v1", "evidence_version": "non-ranking-clues-v1"}
}

func (ContextBM25) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, options Config) ([]domain.Change, []domain.Warning, error) {
	if options.TopK <= 0 {
		options.TopK = DefaultConfig().TopK
	}
	model := NewBM25(artifacts)
	for index := range changes {
		change := &changes[index]
		change.Impacts = []domain.Impact{}
		if change.ChangeType == domain.UnresolvedMatch {
			if len(change.UnresolvedImpact) == 0 {
				change.UnresolvedImpact = []string{"graph element matching is ambiguous"}
			}
			continue
		}
		query, ids := queryFor(*change)
		ranked := removeShadowedPackageDocuments(model.Search(query))
		if len(ranked) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "context BM25 found no lexical candidate")
			continue
		}
		maximum := ranked[0].Score
		for _, candidate := range ranked {
			if len(change.Impacts) >= options.TopK {
				break
			}
			artifact := candidate.Artifact
			score := round(candidate.Score / maximum)
			evidence := contextBM25Evidence(*change, artifact, query, ids)
			change.Impacts = append(change.Impacts, domain.Impact{
				Module: artifact.Module, Layer: artifact.Layer, File: artifact.File,
				Symbol: displaySymbol(artifact.Symbol), RequiredAction: domain.ActionVerify,
				Evidence: evidence, ScoreBreakdown: domain.ScoreBreakdown{BM25: score, Total: score}, Confidence: score,
			})
		}
	}
	return changes, nil, nil
}

func contextBM25Evidence(change domain.Change, artifact domain.Artifact, query, ids []string) []domain.Evidence {
	evidence := []domain.Evidence{evidenceAtQuery("BM25", artifact, query)}
	for _, id := range ids {
		if containsStableID(artifact.Content, id) {
			evidence = append(evidence, evidenceAtStableID(artifact, id))
			break
		}
	}
	if overlapInflections(textutil.Tokenize(artifact.Symbol.Name+" "+artifact.Symbol.Receiver), query) > 0 {
		evidence = append(evidence, evidenceAt("SYMBOL", artifact, artifact.Symbol.Name))
	}
	if overlapInflections(textutil.Tokenize(artifact.File), query) > 0 {
		evidence = append(evidence, domain.Evidence{Type: "PATH", Location: artifact.File, Excerpt: bounded(artifact.File, 240)})
	}
	if overlap(artifact.Tokens, query) > 0 {
		evidence = append(evidence, evidenceAtQuery("LABEL_TOKEN", artifact, query))
	}
	if _, clue := structuredSignal(artifact, structuralQueryFor(change, query)); clue.Type != "" {
		evidence = append(evidence, clue)
	}
	evidence = append(evidence, contextTraceEvidence(change, artifact, ids)...)
	sort.SliceStable(evidence, func(i, j int) bool {
		if evidence[i].Type != evidence[j].Type {
			return evidence[i].Type < evidence[j].Type
		}
		if evidence[i].Location != evidence[j].Location {
			return evidence[i].Location < evidence[j].Location
		}
		return evidence[i].Excerpt < evidence[j].Excerpt
	})
	return evidence
}
