package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func BuildEvidenceBundle(report domain.Report, limits Limits) EvidenceBundle {
	defaults := DefaultLimits()
	if limits.CandidatesPerChange <= 0 {
		limits.CandidatesPerChange = defaults.CandidatesPerChange
	}
	if limits.EvidencePerCandidate <= 0 {
		limits.EvidencePerCandidate = defaults.EvidencePerCandidate
	}
	if limits.ExcerptRunes <= 0 {
		limits.ExcerptRunes = defaults.ExcerptRunes
	}
	bundle := EvidenceBundle{
		Version: BundleVersion, AnalysisID: report.AnalysisID,
		InputFingerprint: report.InputFingerprint, Method: report.Method,
		Repository: report.Repository, Changes: make([]ChangeInput, 0, len(report.Changes)),
	}
	for changeIndex, change := range report.Changes {
		element := change.NewElement
		if element == nil {
			element = change.OldElement
		}
		input := ChangeInput{ChangeType: change.ChangeType, Reason: change.Reason, Candidates: []Candidate{}}
		if element != nil {
			input.ElementID, input.ElementName = element.ID, element.Name
		}
		impacts := append([]domain.Impact(nil), change.Impacts...)
		sort.SliceStable(impacts, func(i, j int) bool {
			if impacts[i].Confidence != impacts[j].Confidence {
				return impacts[i].Confidence > impacts[j].Confidence
			}
			if impacts[i].File != impacts[j].File {
				return impacts[i].File < impacts[j].File
			}
			return impacts[i].Symbol < impacts[j].Symbol
		})
		if len(impacts) > limits.CandidatesPerChange {
			impacts = impacts[:limits.CandidatesPerChange]
		}
		for candidateIndex, impact := range impacts {
			candidate := Candidate{
				File: impact.File, Symbol: impact.Symbol, Module: impact.Module, Layer: impact.Layer,
				SuggestedAction: impact.RequiredAction, RetrievalScore: impact.Confidence, Evidence: []EvidenceItem{},
			}
			evidence := append([]domain.Evidence(nil), impact.Evidence...)
			if len(evidence) > limits.EvidencePerCandidate {
				evidence = evidence[:limits.EvidencePerCandidate]
			}
			for evidenceIndex, item := range evidence {
				candidate.Evidence = append(candidate.Evidence, EvidenceItem{
					ID:   fmt.Sprintf("CH%02d-C%02d-E%02d", changeIndex+1, candidateIndex+1, evidenceIndex+1),
					Type: item.Type, Location: item.Location, Excerpt: boundedRunes(item.Excerpt, limits.ExcerptRunes),
				})
			}
			input.Candidates = append(input.Candidates, candidate)
		}
		bundle.Changes = append(bundle.Changes, input)
	}
	bundle.ContextID, _ = ComputeContextID(bundle)
	return bundle
}

// ComputeContextID hashes the canonical bundle content while excluding the ID
// field itself. The shortened SHA-256 value is an identity, not a security token.
func ComputeContextID(bundle EvidenceBundle) (string, error) {
	bundle.ContextID = ""
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])[:32], nil
}

func boundedRunes(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum-1]) + "…"
}
