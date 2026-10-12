package retrieval

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

// Snapshot seeds before expansion: propagated candidates can never seed another hop.
func propagate(scored []artifactImpact, artifacts []domain.Artifact, threshold float64) []artifactImpact {
	seeds := map[string]domain.Impact{}
	existing := map[string]bool{}
	for _, a := range artifacts {
		for _, s := range scored {
			if s.impact.File == a.File && s.impact.Symbol == displaySymbol(a.Symbol) {
				existing[a.ID] = true
				if s.impact.ScoreBreakdown.TraceID == 1 && (a.Symbol.Kind == "function" || a.Symbol.Kind == "method") {
					seeds[a.ID] = s.impact
				}
			}
		}
	}
	byID := map[string]domain.Artifact{}
	for _, a := range artifacts {
		byID[a.ID] = a
	}
	additions := map[string]domain.Impact{}
	for _, caller := range artifacts {
		for _, edge := range caller.Relations {
			forwardEvidence, reverseEvidence, supported := relationEvidence(edge.Kind)
			if !supported {
				continue
			}
			target, ok := byID[edge.TargetID]
			if !ok {
				continue
			}
			for _, direction := range []struct {
				seedID    string
				candidate domain.Artifact
				kind      string
			}{
				{target.ID, caller, forwardEvidence}, {caller.ID, target, reverseEvidence},
			} {
				seed, ok := seeds[direction.seedID]
				if !ok || existing[direction.candidate.ID] {
					continue
				}
				score := round(seed.Confidence * 0.5)
				if score < threshold {
					continue
				}
				previous := additions[direction.candidate.ID]
				if previous.Confidence >= score {
					continue
				}
				a := direction.candidate
				additions[a.ID] = domain.Impact{File: a.File, Module: a.Module, Layer: a.Layer, Symbol: displaySymbol(a.Symbol), RequiredAction: domain.ActionVerify,
					Confidence: score, ScoreBreakdown: domain.ScoreBreakdown{Dependency: score, Total: score},
					Evidence: []domain.Evidence{{Type: direction.kind, Location: edge.Location, Excerpt: edge.Excerpt}, {Type: "PROPAGATION_SEED", Location: seed.File, Excerpt: seed.Symbol + " (exact trace seed)"}},
				}
			}
		}
	}
	for _, a := range artifacts {
		if impact, ok := additions[a.ID]; ok {
			scored = append(scored, artifactImpact{impact: impact, symbolKind: a.Symbol.Kind})
		}
	}
	return scored
}

func relationEvidence(kind domain.RelationKind) (string, string, bool) {
	switch kind {
	case domain.RelationCalls:
		return "CALLS_SYMBOL", "CALLED_BY", true
	case domain.RelationCallsMethod:
		return "CALLS_METHOD", "CALLED_BY_METHOD", true
	case domain.RelationInterfaceDispatch:
		return "INTERFACE_DISPATCH", "IMPLEMENTS_INTERFACE", true
	default:
		return "", "", false
	}
}
