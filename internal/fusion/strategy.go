// Package fusion combines independently ranked impact candidates without
// pretending that heterogeneous retrieval scores share the same scale.
package fusion

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

const DefaultRRFK = 60

type Strategy struct {
	MethodName          domain.AnalysisMethod
	Left                mapping.Strategy
	Right               mapping.Strategy
	PreserveLeftActions bool
	K                   int
}

func (strategy Strategy) Method() domain.AnalysisMethod { return strategy.MethodName }

func (strategy Strategy) MethodMetadata() map[string]string {
	k := strategy.K
	if k <= 0 {
		k = DefaultRRFK
	}
	metadata := map[string]string{
		"fusion":        "reciprocal-rank-fusion-v1",
		"rrf_k":         fmt.Sprint(k),
		"score_meaning": "normalized rank consensus; not probability",
	}
	if strategy.Left != nil {
		metadata["left_method"] = string(strategy.Left.Method())
	}
	if strategy.Right != nil {
		metadata["right_method"] = string(strategy.Right.Method())
	}
	if provider, ok := strategy.Right.(mapping.MetadataProvider); ok {
		for key, value := range provider.MethodMetadata() {
			metadata["right_"+key] = value
		}
	}
	return metadata
}

func (strategy Strategy) Attach(ctx context.Context, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	if strategy.Left == nil || strategy.Right == nil {
		return nil, nil, fmt.Errorf("fusion requires two retrieval strategies")
	}
	left, leftWarnings, err := strategy.Left.Attach(ctx, cloneChanges(changes), artifacts, options)
	if err != nil {
		return nil, nil, fmt.Errorf("%s retrieval: %w", strategy.Left.Method(), err)
	}
	right, rightWarnings, err := strategy.Right.Attach(ctx, cloneChanges(changes), artifacts, options)
	if err != nil {
		return nil, nil, fmt.Errorf("%s retrieval: %w", strategy.Right.Method(), err)
	}
	if len(left) != len(changes) || len(right) != len(changes) {
		return nil, nil, fmt.Errorf("fusion strategy returned a mismatched change count")
	}

	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}
	k := strategy.K
	if k <= 0 {
		k = DefaultRRFK
	}
	result := cloneChanges(changes)
	for index := range result {
		result[index].Impacts = fuse(left[index].Impacts, right[index].Impacts, topK, k, strategy.PreserveLeftActions, strategy.Left.Method(), strategy.Right.Method())
		result[index].UnresolvedImpact = []string{}
		if len(result[index].Impacts) == 0 {
			result[index].UnresolvedImpact = uniqueStrings(append(append([]string{}, left[index].UnresolvedImpact...), right[index].UnresolvedImpact...))
			if len(result[index].UnresolvedImpact) == 0 {
				result[index].UnresolvedImpact = []string{"rank fusion found no candidate"}
			}
		}
	}
	return result, mergeWarnings(leftWarnings, rightWarnings), nil
}

type rankedImpact struct {
	impact    domain.Impact
	leftRank  int
	rightRank int
	rrf       float64
}

func fuse(left, right []domain.Impact, topK, k int, preserveLeftActions bool, leftMethod, rightMethod domain.AnalysisMethod) []domain.Impact {
	byKey := map[string]*rankedImpact{}
	leftKeys := map[string]bool{}
	for rank, impact := range left {
		key := impactKey(impact)
		leftKeys[key] = true
		entry := byKey[key]
		if entry == nil {
			entry = &rankedImpact{impact: impact}
			byKey[key] = entry
		}
		entry.leftRank = rank + 1
		entry.rrf += 1 / float64(k+rank+1)
	}
	for rank, impact := range right {
		key := impactKey(impact)
		entry := byKey[key]
		if entry == nil {
			entry = &rankedImpact{impact: impact}
			byKey[key] = entry
		} else {
			entry.impact = mergeImpact(entry.impact, impact)
		}
		entry.rightRank = rank + 1
		entry.rrf += 1 / float64(k+rank+1)
	}

	maximum := 2 / float64(k+1)
	ranked := make([]domain.Impact, 0, len(byKey))
	for _, entry := range byKey {
		score := round(entry.rrf / maximum)
		entry.impact.ScoreBreakdown.RRF = score
		entry.impact.ScoreBreakdown.Total = score
		entry.impact.Confidence = score
		if !preserveLeftActions || entry.leftRank == 0 {
			entry.impact.RequiredAction = domain.ActionVerify
		}
		entry.impact.Evidence = append(entry.impact.Evidence, domain.Evidence{
			Type: "RRF_FUSION", Location: impactLocation(entry.impact),
			Excerpt: fmt.Sprintf("left_method=%s; left_rank=%s; right_method=%s; right_rank=%s; k=%d", leftMethod, rankText(entry.leftRank), rightMethod, rankText(entry.rightRank), k),
		})
		entry.impact.Evidence = uniqueEvidence(entry.impact.Evidence)
		ranked = append(ranked, entry.impact)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].Confidence-ranked[j].Confidence) > 1e-12 {
			return ranked[i].Confidence > ranked[j].Confidence
		}
		if preserveLeftActions && leftKeys[impactKey(ranked[i])] != leftKeys[impactKey(ranked[j])] {
			return leftKeys[impactKey(ranked[i])]
		}
		if ranked[i].File != ranked[j].File {
			return ranked[i].File < ranked[j].File
		}
		return ranked[i].Symbol < ranked[j].Symbol
	})
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	return ranked
}

func cloneChanges(changes []domain.Change) []domain.Change {
	result := append([]domain.Change(nil), changes...)
	for index := range result {
		result[index].Impacts = append([]domain.Impact(nil), changes[index].Impacts...)
		result[index].UnresolvedImpact = append([]string(nil), changes[index].UnresolvedImpact...)
	}
	return result
}

func mergeImpact(left, right domain.Impact) domain.Impact {
	left.Evidence = append(left.Evidence, right.Evidence...)
	left.ScoreBreakdown.Semantic = math.Max(left.ScoreBreakdown.Semantic, right.ScoreBreakdown.Semantic)
	left.ScoreBreakdown.BM25 = math.Max(left.ScoreBreakdown.BM25, right.ScoreBreakdown.BM25)
	left.ScoreBreakdown.Dependency = math.Max(left.ScoreBreakdown.Dependency, right.ScoreBreakdown.Dependency)
	left.ScoreBreakdown.TraceID = math.Max(left.ScoreBreakdown.TraceID, right.ScoreBreakdown.TraceID)
	left.ScoreBreakdown.Symbol = math.Max(left.ScoreBreakdown.Symbol, right.ScoreBreakdown.Symbol)
	left.ScoreBreakdown.Path = math.Max(left.ScoreBreakdown.Path, right.ScoreBreakdown.Path)
	left.ScoreBreakdown.Lexical = math.Max(left.ScoreBreakdown.Lexical, right.ScoreBreakdown.Lexical)
	left.ScoreBreakdown.Structured = math.Max(left.ScoreBreakdown.Structured, right.ScoreBreakdown.Structured)
	return left
}

func uniqueEvidence(values []domain.Evidence) []domain.Evidence {
	seen := map[string]bool{}
	result := make([]domain.Evidence, 0, len(values))
	for _, value := range values {
		key := value.Type + "\x00" + value.Location + "\x00" + value.Excerpt
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		if result[i].Location != result[j].Location {
			return result[i].Location < result[j].Location
		}
		return result[i].Excerpt < result[j].Excerpt
	})
	return result
}

func mergeWarnings(groups ...[]domain.Warning) []domain.Warning {
	seen := map[string]bool{}
	result := []domain.Warning{}
	for _, group := range groups {
		for _, warning := range group {
			key := warning.Code + "\x00" + warning.Message
			if !seen[key] {
				seen[key] = true
				result = append(result, warning)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Code != result[j].Code {
			return result[i].Code < result[j].Code
		}
		return result[i].Message < result[j].Message
	})
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func impactKey(impact domain.Impact) string { return impact.File + "\x00" + impact.Symbol }

func impactLocation(impact domain.Impact) string {
	if impact.Symbol == "" {
		return impact.File
	}
	return impact.File + " / " + impact.Symbol
}

func rankText(rank int) string {
	if rank == 0 {
		return "absent"
	}
	return fmt.Sprint(rank)
}

func round(value float64) float64 { return math.Round(value*10000) / 10000 }
