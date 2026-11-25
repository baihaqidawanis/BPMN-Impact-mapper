package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type editTarget struct {
	change       domain.ChangeType
	element      string
	file, symbol string
}

type lockedPlan struct {
	bundle    planning.EvidenceBundle
	decisions []planning.CandidateDecision
	draft     []planning.FilePlan
	actions   map[editTarget]domain.RequiredAction
}

// Lock the source session's actual edits for the selected BPMN element.
// Evaluation references are never read by this workflow.
func lockPlan(source planning.AgentSession, element string) (lockedPlan, error) {
	if _, err := source.ChangePlan(); err != nil {
		return lockedPlan{}, fmt.Errorf("source session must have a valid finalized plan: %w", err)
	}
	result := lockedPlan{bundle: source.Bundle, actions: map[editTarget]domain.RequiredAction{}}
	result.bundle.Changes = nil
	for _, p := range source.Draft {
		if p.ElementID != element {
			continue
		}
		switch p.Action {
		case domain.ActionAdd, domain.ActionModify, domain.ActionRemove:
			key := editTarget{p.ChangeType, p.ElementID, p.File, p.Symbol}
			result.actions[key] = p.Action
			result.draft = append(result.draft, p)
		}
	}
	if len(result.draft) == 0 {
		return lockedPlan{}, errors.New("selected BPMN element has no finalized edit targets")
	}
	for _, d := range source.Decisions {
		if _, ok := result.actions[editTarget{d.ChangeType, d.ElementID, d.File, d.Symbol}]; ok && d.Verdict == planning.VerdictRelevant {
			result.decisions = append(result.decisions, d)
		}
	}
	for _, change := range source.Bundle.Changes {
		if change.ElementID != element {
			continue
		}
		selected := change
		selected.Candidates = nil
		for _, c := range change.Candidates {
			if _, ok := result.actions[editTarget{change.ChangeType, change.ElementID, c.File, c.Symbol}]; ok {
				selected.Candidates = append(selected.Candidates, c)
			}
		}
		if len(selected.Candidates) > 0 {
			result.bundle.Changes = append(result.bundle.Changes, selected)
		}
	}
	if pc := result.bundle.ProcessContext; pc != nil {
		result.bundle.ProcessContext = &domain.ProcessContext{
			Before: localProcessGraph(pc.Before, result.bundle.Changes),
			After:  localProcessGraph(pc.After, result.bundle.Changes),
		}
	}
	var err error
	result.bundle.ContextID, err = planning.ComputeContextID(result.bundle)
	return result, err
}

// Send the changed nodes/flows, their immediate neighbors, and explicit
// supporting branches. The archived session keeps the complete process.
func localProcessGraph(graph *domain.Graph, changes []planning.ChangeInput) *domain.Graph {
	if graph == nil {
		return nil
	}
	anchors, flows := map[string]bool{}, map[string]bool{}
	for _, change := range changes {
		ids := []string{change.ElementID}
		for _, element := range []*planning.ElementInput{change.Before, change.After} {
			if element != nil && element.ID != "" {
				ids = append(ids, element.ID)
			}
		}
		for _, id := range ids {
			anchors[id], flows[id] = true, true
		}
		for _, id := range change.SupportingFlowIDs {
			flows[id] = true
		}
		for _, branch := range change.BranchChanges {
			flows[branch.FlowID] = true
		}
	}
	result := &domain.Graph{
		ProcessNames: graph.ProcessNames,
		Elements:     map[string]*domain.Element{},
		Flows:        map[string]*domain.Flow{},
		Incoming:     map[string][]string{},
		Outgoing:     map[string][]string{},
		Warnings:     graph.Warnings,
	}
	for id := range anchors {
		if e := graph.Elements[id]; e != nil {
			result.Elements[id] = e
		}
	}
	for id, f := range graph.Flows {
		if f == nil || !(flows[id] || anchors[f.SourceRef] || anchors[f.TargetRef]) {
			continue
		}
		result.Flows[id] = f
		result.Incoming[f.TargetRef] = append(result.Incoming[f.TargetRef], id)
		result.Outgoing[f.SourceRef] = append(result.Outgoing[f.SourceRef], id)
		for _, node := range []string{f.SourceRef, f.TargetRef} {
			if e := graph.Elements[node]; e != nil {
				result.Elements[node] = e
			}
		}
	}
	for _, index := range []map[string][]string{result.Incoming, result.Outgoing} {
		for _, ids := range index {
			sort.Strings(ids)
		}
	}
	return result
}

// A supplemental window can add evidence to a locked whole-file target only.
func addLockedSourceWindows(ctx context.Context, tools planning.SourceTools, bundle planning.EvidenceBundle, windows []string, focusedEvidence map[string][]string) (planning.EvidenceBundle, error) {
	for _, window := range windows {
		separator := strings.LastIndexByte(window, ':')
		if separator < 1 {
			return bundle, fmt.Errorf("invalid source window %q, use relative/path:line", window)
		}
		file := window[:separator]
		start, end, exact := strings.Cut(window[separator+1:], "-")
		line, err := strconv.Atoi(start)
		if err != nil || line < 1 {
			return bundle, fmt.Errorf("invalid source line in %q", window)
		}
		var last int
		if exact {
			last, err = strconv.Atoi(end)
			if err != nil || last < line || last-line >= 8 {
				return bundle, fmt.Errorf("source range %q must cover one to eight lines", window)
			}
		}
		var targets []*planning.Candidate
		for i := range bundle.Changes {
			for j := range bundle.Changes[i].Candidates {
				c := &bundle.Changes[i].Candidates[j]
				if c.File == file && c.Symbol == "" {
					targets = append(targets, c)
				}
			}
		}
		if len(targets) == 0 {
			return bundle, fmt.Errorf("%q is outside the locked whole-file targets", file)
		}
		result, err := tools.Execute(ctx, planning.ToolRequest{Name: "read_source", File: file, StartLine: line})
		if err != nil {
			return bundle, err
		}
		if len(result.Candidates) != 1 || result.Candidates[0].File != file || len(result.Candidates[0].Evidence) == 0 {
			return bundle, fmt.Errorf("source window %q returned no usable evidence", window)
		}
		items := result.Candidates[0].Evidence
		if exact {
			items = nil
			for _, item := range result.Candidates[0].Evidence {
				var n int
				if item.Type == "SOURCE_TOOL" && strings.HasPrefix(item.Location, file+":") {
					fmt.Sscanf(strings.TrimPrefix(item.Location, file+":"), "%d", &n)
				}
				if n >= line && n <= last {
					items = append(items, item)
				}
			}
			if len(items) == 0 {
				return bundle, fmt.Errorf("source range %q contains no source evidence", window)
			}
		}
		for _, item := range items {
			focusedEvidence[file] = append(focusedEvidence[file], item.ID)
		}
		for _, target := range targets {
			seen := map[string]bool{}
			for _, item := range target.Evidence {
				seen[item.ID] = true
			}
			for _, item := range items {
				if !seen[item.ID] {
					target.Evidence = append(target.Evidence, item)
					seen[item.ID] = true
				}
			}
		}
	}
	var err error
	bundle.ContextID, err = planning.ComputeContextID(bundle)
	return bundle, err
}

func readRevisionNotes(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	const maxBytes = 4 << 10
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return "", errors.New("review notes must be a regular file within 4 KiB")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if len(data) > maxBytes {
		return "", errors.New("review notes exceed 4 KiB")
	}
	return strings.TrimSpace(string(data)), err
}

const revisionInstruction = "Name the exact function, branch and input. Trace the caller payload and keep existing server guards and response contracts. Never use a correlation ID as a business input or call a mutating reject operation for a failed validation. Preserve the purpose of each existing success fixture. Add separate rejection checks for persisted state and version. State source gaps as assumptions."

type lockedPlanner struct {
	planning.AgentModel
	actions         map[editTarget]domain.RequiredAction
	notes           string
	testFile        string
	callSites       []agenttools.TestCallSite
	focusedEvidence map[string][]string
}

func (m lockedPlanner) Identity() planning.ProviderIdentity {
	identity := m.AgentModel.Identity()
	evidence, _ := json.Marshal(m.focusedEvidence)
	policy := sha256.Sum256([]byte(revisionInstruction + "\x00" + m.notes + "\x00" + string(evidence)))
	identity.PromptVersion += fmt.Sprintf("-locked-planner-v7-%x", policy[:8])
	return identity
}

func (m lockedPlanner) Turn(ctx context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	if turn.Role != planning.Planner {
		return planning.AgentReply{}, errors.New("locked revision can only invoke Planner")
	}
	feedback := []string{revisionInstruction}
	if m.notes != "" {
		feedback = append(feedback, "Operator review findings for this revision. Check against supplied source where possible and do not cite this note as source: "+m.notes)
	}
	expected := make(map[editTarget]domain.RequiredAction, len(turn.Decisions))
	for _, d := range turn.Decisions {
		key := editTarget{d.ChangeType, d.ElementID, d.File, d.Symbol}
		action, ok := m.actions[key]
		if !ok {
			return planning.AgentReply{}, errors.New("Planner received a target outside the locked revision")
		}
		expected[key] = action
		feedback = append(feedback, fmt.Sprintf("Locked action for %s (%s, %s): %s.", d.File, d.ElementID, d.Symbol, action))
		if d.File == m.testFile && len(m.callSites) > 0 {
			refs := make([]string, 0, len(m.callSites))
			for _, site := range m.callSites {
				refs = append(refs, site.Reference)
			}
			feedback = append(feedback, "Test fixture coverage is required: address each call at "+strings.Join(refs, ", ")+
				" by exact file:line in proposed_changes or steps. The host binds the AST_TEST_CALL evidence for those references. Preserve each original test's purpose and add a separate rejection test.")
		}
	}
	// Old prose can repeat the very errors this revision is intended to fix.
	turn.Draft, turn.RelatedProposals = nil, nil
	turn.Decisions = append([]planning.CandidateDecision(nil), turn.Decisions...)
	for i := range turn.Decisions {
		turn.Decisions[i].Rationale = ""
	}
	turn.Feedback = append(feedback, turn.Feedback...)
	turn.Bundle = focusLockedPlannerEvidence(turn.Bundle, m.focusedEvidence)
	reply, err := m.AgentModel.Turn(ctx, turn)
	if err != nil {
		return reply, err
	}
	if reply.Plan == nil || len(reply.Plan.Plans) != len(turn.Decisions) {
		return planning.AgentReply{}, fmt.Errorf("%w: revise every supplied target with its existing edit action", planning.ErrAgentModelOutput)
	}
	seen := make(map[editTarget]bool, len(reply.Plan.Plans))
	for i := range reply.Plan.Plans {
		p := &reply.Plan.Plans[i]
		key := editTarget{p.ChangeType, p.ElementID, p.File, p.Symbol}
		if action, ok := expected[key]; !ok || action != p.Action || seen[key] {
			return planning.AgentReply{}, fmt.Errorf("%w: keep the prior edit action and identity for %s", planning.ErrAgentModelOutput, p.File)
		}
		seen[key] = true
		if p.File == m.testFile {
			if err := bindTestCallEvidence(p, m.callSites); err != nil {
				return planning.AgentReply{}, err
			}
		}
	}
	return reply, nil
}
