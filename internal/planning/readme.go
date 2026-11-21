package planning

import (
	"context"
	"errors"
	"strings"
)

type readmeSource interface{ RootReadme() *ReadmeInput }

// One successful orientation per session. Its compact summary is background,
// never a citable implementation candidate or an instruction source.
func (e FullRepositoryEngine) orient(ctx context.Context, id string, bundle EvidenceBundle) (EvidenceBundle, error) {
	source, ok := e.Source.(readmeSource)
	if !ok {
		return bundle, nil
	}
	readme := source.RootReadme()
	if readme == nil {
		return bundle, nil
	}
	if len(readme.Content) > 8192 || !agentRelativeFile(readme.File) {
		return bundle, errors.New("invalid README context")
	}
	binding := agentDigest([]any{"readme-orientation-v1", bundle.ContextID, e.Source.Identity(), e.Model.Identity(), readme})
	childID := id + "-orientation"
	receipt, err := e.Store.Load(ctx, childID)
	if errors.Is(err, ErrAgentSessionMissing) {
		receipt = AgentSession{Version: "1", ID: childID, Binding: binding, Identity: e.Model.Identity(), Bundle: bundle, Role: Analyst, Status: "RUNNING"}
	} else if err != nil {
		return bundle, err
	}
	if receipt.Binding != binding {
		return bundle, errors.New("README/source/model binding changed; start a new session")
	}
	if receipt.Status != "COMPLETE" {
		if receipt.Turns >= 3 {
			return bundle, errors.New("README orientation attempts exhausted")
		}
		receipt.Turns++
		if err := e.Store.Save(ctx, receipt); err != nil {
			return bundle, err
		}
		// Orientation only describes the repository, not BPMN impacts. Do not
		// duplicate full process graphs or evidence in this one-off request.
		input := bundle
		input.Changes = nil
		input.ProcessContext = nil
		input.Warnings = nil

		input.ContextID, _ = ComputeContextID(input)
		callCtx, cancel := context.WithTimeout(ctx, AgentTurnTimeout)
		reply, err := e.Model.Turn(callCtx, AgentTurn{Role: Analyst, Bundle: input, Readme: readme, RemainingTools: 0})
		cancel()
		if err != nil {
			return bundle, err
		}
		if reply.Verification == nil || reply.Tool != nil || reply.Plan != nil || reply.Review != nil || reply.NeedEvidence != nil || len(reply.Verification.Decisions) != 0 {
			return bundle, ErrAgentModelOutput
		}
		summary := strings.TrimSpace(reply.Verification.RepositorySummary)
		if summary == "" || len([]rune(summary)) > 1600 {
			return bundle, ErrAgentModelOutput
		}
		receipt.Bundle.RepositoryOverview = &RepositoryOverview{File: readme.File, Summary: summary, Truncated: readme.Truncated}
		receipt.Bundle.ContextID, _ = ComputeContextID(receipt.Bundle)
		receipt.Status = "COMPLETE"
		if err := e.Store.Save(ctx, receipt); err != nil {
			return bundle, err
		}
	}
	if receipt.Bundle.RepositoryOverview == nil {
		return bundle, errors.New("README summary checkpoint missing")
	}
	bundle.RepositoryOverview = receipt.Bundle.RepositoryOverview
	bundle.ContextID, _ = ComputeContextID(bundle)
	return bundle, nil
}
