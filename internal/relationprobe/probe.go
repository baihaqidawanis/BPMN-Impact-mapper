// Package relationprobe inspects source links next to an existing retrieval
// result. It never changes canonical impacts, scores, or required actions.
package relationprobe

import (
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type Step struct {
	Kind     domain.RelationKind `json:"kind"`
	Location string              `json:"location"`
	Excerpt  string              `json:"excerpt"`
}

type Finding struct {
	ChangeType   domain.ChangeType `json:"change_type"`
	ElementID    string            `json:"element_id"`
	SeedRank     int               `json:"seed_rank"`
	SeedFile     string            `json:"seed_file"`
	SeedSymbol   string            `json:"seed_symbol,omitempty"`
	TargetFile   string            `json:"target_file"`
	TargetSymbol string            `json:"target_symbol,omitempty"`
	ExistingRank int               `json:"existing_rank,omitempty"`
	Steps        []Step            `json:"steps"`
}

// Probe follows only a frontend call line cited by lexical evidence to its
// uniquely indexed HTTP handler, then at most one static method/interface call.
// The result is an ablation diagnostic, not a prediction or a score boost.
func Probe(report domain.Report, artifacts []domain.Artifact, maxSeedRank int) []Finding {
	if maxSeedRank <= 0 || maxSeedRank > 20 {
		maxSeedRank = 5
	}
	byID := make(map[string]domain.Artifact, len(artifacts))
	byFileSymbol := make(map[string][]domain.Artifact)
	for _, artifact := range artifacts {
		byID[artifact.ID] = artifact
		key := artifact.File + "\x00" + symbol(artifact.Symbol)
		byFileSymbol[key] = append(byFileSymbol[key], artifact)
	}
	findings := []Finding{}
	for _, change := range report.Changes {
		if change.PartOf != "" || change.ChangeType == domain.UnresolvedMatch {
			continue
		}
		rankByFile := map[string]int{}
		for rank, impact := range change.Impacts {
			if rankByFile[impact.File] == 0 {
				rankByFile[impact.File] = rank + 1
			}
		}
		for rank, seed := range change.Impacts {
			if rank >= maxSeedRank {
				break
			}
			for _, client := range byFileSymbol[seed.File+"\x00"+seed.Symbol] {
				for _, route := range client.Relations {
					if route.Kind != domain.RelationHTTPRoute || !citesCallLine(seed.Evidence, route.Location) {
						continue
					}
					handler, ok := byID[route.TargetID]
					if !ok || handler.File == seed.File {
						continue
					}
					first := Step{Kind: route.Kind, Location: route.Location, Excerpt: route.Excerpt}
					findings = append(findings, finding(change, rank+1, seed, handler, rankByFile[handler.File], []Step{first, {Kind: route.Kind, Location: route.TargetLocation, Excerpt: route.TargetExcerpt}}))
					for _, call := range handler.Relations {
						if call.Kind != domain.RelationCallsMethod && call.Kind != domain.RelationInterfaceDispatch {
							continue
						}
						target, ok := byID[call.TargetID]
						if !ok || target.File == handler.File || target.File == seed.File {
							continue
						}
						findings = append(findings, finding(change, rank+1, seed, target, rankByFile[target.File], []Step{first, {Kind: route.Kind, Location: route.TargetLocation, Excerpt: route.TargetExcerpt}, {Kind: call.Kind, Location: call.Location, Excerpt: call.Excerpt}}))
					}
				}
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.ChangeType != b.ChangeType {
			return a.ChangeType < b.ChangeType
		}
		if a.ElementID != b.ElementID {
			return a.ElementID < b.ElementID
		}
		if a.SeedRank != b.SeedRank {
			return a.SeedRank < b.SeedRank
		}
		if len(a.Steps) != len(b.Steps) {
			return len(a.Steps) < len(b.Steps)
		}
		if a.TargetFile != b.TargetFile {
			return a.TargetFile < b.TargetFile
		}
		return a.TargetSymbol < b.TargetSymbol
	})
	return findings
}

func finding(change domain.Change, rank int, seed domain.Impact, target domain.Artifact, existingRank int, steps []Step) Finding {
	id := ""
	if change.NewElement != nil {
		id = change.NewElement.ID
	} else if change.OldElement != nil {
		id = change.OldElement.ID
	}
	return Finding{ChangeType: change.ChangeType, ElementID: id, SeedRank: rank, SeedFile: seed.File, SeedSymbol: seed.Symbol, TargetFile: target.File, TargetSymbol: symbol(target.Symbol), ExistingRank: existingRank, Steps: steps}
}

func citesCallLine(evidence []domain.Evidence, location string) bool {
	for _, item := range evidence {
		if item.Location == location && (item.Type == "BM25" || item.Type == "LABEL_TOKEN") {
			return true
		}
	}
	return false
}

func symbol(value domain.Symbol) string {
	if value.Receiver != "" && value.Name != "" {
		return value.Receiver + "." + value.Name
	}
	return strings.TrimSpace(value.Name)
}
