package wiring

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/relationprobe"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
)

// NewContextStudyAnalyzer exposes controlled offline experiments, not web methods.
// All variants retain production file-level fusion and conservative symbol scope.
func NewContextStudyAnalyzer(embedder semantic.Embedder, mode string) (app.Analyzer, error) {
	if mode == "production" {
		return NewAnalyzerWithSemanticEmbedder(embedder), nil
	}
	switch mode {
	case "full", "no-context", "no-start-end", "both-top10", "multi-hop4", "condition-focus", "generic-windows", "code-windows", "source-callers":
	default:
		return app.Analyzer{}, fmt.Errorf("unknown context-study mode %q", mode)
	}
	analyzer := app.Analyzer{Resolver: contextStudyResolver{strategyResolver{embedder}, mode}}
	analyzer.ConditionDeltaQueries = mode == "condition-focus"
	if mode == "source-callers" {
		registry := index.SourceContextRegistry()
		analyzer.Indexers = &registry
		analyzer.IncludeSourceCallers = true
		analyzer.SourceCallers = relationprobe.CallerFiles
	}
	if mode == "generic-windows" || mode == "code-windows" {
		registry := index.DefaultRegistry()
		registry.Adapters = append(registry.Adapters, index.TextWindowAdapter{CodeOnly: mode == "code-windows"})
		analyzer.Indexers = &registry
	}
	if mode == "multi-hop4" {
		analyzer.BaselineContextHops = 4
	}
	return analyzer, nil
}

type contextStudyResolver struct {
	base strategyResolver
	mode string
}

func (resolver contextStudyResolver) Resolve(method domain.AnalysisMethod, path string) (mapping.Strategy, []byte, error) {
	if method != domain.MethodHybrid {
		return nil, nil, fmt.Errorf("context study only accepts hybrid")
	}
	strategy, identity, err := resolver.base.Resolve(method, path)
	if err != nil {
		return nil, nil, err
	}
	if resolver.mode == "full" {
		return strategy, identity, nil
	}
	combined := strategy.(fusion.Strategy)
	if resolver.mode == "no-start-end" {
		combined.Left = retrieval.ContextBM25{OmitStartEndLabels: true}
	}
	if resolver.mode == "multi-hop4" {
		combined.Left = retrieval.ContextBM25{UseBaselineAnchors: true}
		right := combined.Right.(semantic.Strategy)
		right.UseBaselineAnchors = true
		combined.Right = right
	}
	wrapped := contextStudyStrategy{inner: combined, mode: resolver.mode}
	config, err := json.Marshal(struct {
		Base  json.RawMessage `json:"base"`
		Study string          `json:"context_study_v1"`
	}{identity, resolver.mode})
	return wrapped, config, err
}

type contextStudyStrategy struct {
	inner mapping.Strategy
	mode  string
}

func (contextStudyStrategy) Method() domain.AnalysisMethod { return domain.MethodHybrid }
func (strategy contextStudyStrategy) MethodMetadata() map[string]string {
	metadata := strategy.inner.(mapping.MetadataProvider).MethodMetadata()
	metadata["context_study"] = strategy.mode
	return metadata
}
func (strategy contextStudyStrategy) Attach(ctx context.Context, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	input := append([]domain.Change(nil), changes...)
	if strategy.mode == "no-context" {
		for index := range input {
			input[index].Context = nil
		}
	}
	if strategy.mode == "condition-focus" {
		for index := range input {
			if input[index].ChangeType != domain.ModifyGatewayCondition || input[index].Context == nil || len(input[index].Context.ConditionDelta) == 0 {
				continue
			}
			focused := *input[index].Context
			focused.BranchTerms = append([]string(nil), focused.ConditionDelta...)
			focused.Predecessors, focused.Successors = nil, nil
			focused.ProcessName = ""
			input[index].Context = &focused
		}
	}
	result, warnings, err := strategy.inner.Attach(ctx, input, artifacts, options)
	if err != nil {
		return nil, nil, err
	}
	for index := range result {
		result[index].Context = changes[index].Context
		if strategy.mode != "both-top10" {
			continue
		}
		retained := make([]domain.Impact, 0, len(result[index].Impacts))
		for _, impact := range result[index].Impacts {
			if hasBothRanksWithin(impact, 10) {
				retained = append(retained, impact)
			}
		}
		result[index].Impacts = retained
		if len(retained) == 0 {
			result[index].UnresolvedImpact = append(result[index].UnresolvedImpact, "experimental both-top10 filter retained no candidate")
		}
	}
	return result, warnings, nil
}
func hasBothRanksWithin(impact domain.Impact, limit int) bool {
	for _, evidence := range impact.Evidence {
		if evidence.Type != "RRF_FUSION" {
			continue
		}
		fields := map[string]string{}
		for _, part := range strings.Split(evidence.Excerpt, ";") {
			pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(pair) == 2 {
				fields[pair[0]] = pair[1]
			}
		}
		left, errLeft := strconv.Atoi(fields["left_rank"])
		right, errRight := strconv.Atoi(fields["right_rank"])
		return errLeft == nil && errRight == nil && left > 0 && right > 0 && left <= limit && right <= limit
	}
	return false
}
