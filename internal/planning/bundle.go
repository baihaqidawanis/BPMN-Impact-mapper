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
	return buildEvidenceBundle(report, limits, false)
}

func buildEvidenceBundle(report domain.Report, limits Limits, agent bool) EvidenceBundle {
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
	if limits.RelatedPerChange == 0 {
		limits.RelatedPerChange = defaults.RelatedPerChange
	}
	if limits.RelatedPerChange < 0 {
		limits.RelatedPerChange = 0
	}
	if limits.RelatedPerChange > 3 {
		limits.RelatedPerChange = 3
	}
	bundle := EvidenceBundle{
		Version: BundleVersion, AnalysisID: report.AnalysisID, ProcessContext: report.ProcessContext,
		InputFingerprint: report.InputFingerprint, Method: report.Method,
		Repository: report.Repository, Changes: make([]ChangeInput, 0, len(report.Changes)),
	}
	for _, w := range report.Warnings {
		if len(bundle.Warnings) >= 16 {
			bundle.WarningsTruncated = true
			break
		}
		w.Code = boundedContext(w.Code, 64, &bundle.WarningsTruncated)
		w.Message = boundedContext(w.Message, 512, &bundle.WarningsTruncated)
		bundle.Warnings = append(bundle.Warnings, w)
	}
	for changeIndex, change := range report.Changes {
		element := change.NewElement
		if element == nil {
			element = change.OldElement
		}
		input := ChangeInput{ChangeType: change.ChangeType, Reason: change.Reason, Candidates: []Candidate{}}
		input.Before = elementInput(change.OldElement, &input.ContextTruncated)
		input.After = elementInput(change.NewElement, &input.ContextTruncated)
		input.Reason = boundedContext(input.Reason, 512, &input.ContextTruncated)
		if change.Context != nil {
			for _, v := range change.Context.BranchChanges {
				if len(input.BranchChanges) >= 8 {
					input.ContextTruncated = true
					break
				}
				v.FlowID = boundedContext(v.FlowID, 256, &input.ContextTruncated)
				v.OldCondition = boundedContext(v.OldCondition, 512, &input.ContextTruncated)
				v.NewCondition = boundedContext(v.NewCondition, 512, &input.ContextTruncated)
				v.OldLabel = boundedContext(v.OldLabel, 240, &input.ContextTruncated)
				v.NewLabel = boundedContext(v.NewLabel, 240, &input.ContextTruncated)
				v.OldTarget = boundedContext(v.OldTarget, 256, &input.ContextTruncated)
				v.NewTarget = boundedContext(v.NewTarget, 256, &input.ContextTruncated)
				input.BranchChanges = append(input.BranchChanges, v)
			}
		}
		if change.ChangeType == domain.ModifyGatewayCondition && len(input.BranchChanges) == 0 {
			input.ContextIncomplete = true
		}
		if element != nil {
			input.ElementID, input.ElementName = element.ID, boundedContext(element.Name, 240, &input.ContextTruncated)
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
			score := impact.Confidence
			candidate := Candidate{
				File: impact.File, Symbol: impact.Symbol, Module: impact.Module, Layer: impact.Layer,
				SuggestedAction: impact.RequiredAction, RetrievalScore: &score, Evidence: []EvidenceItem{},
			}
			seenExcerpt := map[[2]string]bool{}
			// A metadata-only candidate remains available as a search lead.
			hasExcerpt := false
			if agent {
				for _, item := range impact.Evidence {
					if item.Type != "PATH" && item.Type != "RRF_FUSION" && item.Excerpt != "" {
						hasExcerpt = true
						break
					}
				}
			}
			for evidenceIndex, item := range impact.Evidence {
				if len(candidate.Evidence) == limits.EvidencePerCandidate {
					break
				}
				if agent {
					key := [2]string{item.Location, item.Excerpt}
					if seenExcerpt[key] || (hasExcerpt && (item.Type == "PATH" || item.Type == "RRF_FUSION" || item.Excerpt == "")) {
						continue
					}
					seenExcerpt[key] = true
				}
				candidate.Evidence = append(candidate.Evidence, EvidenceItem{
					ID:   fmt.Sprintf("CH%02d-C%02d-E%02d", changeIndex+1, candidateIndex+1, evidenceIndex+1),
					Type: item.Type, Location: item.Location, Excerpt: boundedRunes(item.Excerpt, limits.ExcerptRunes),
				})
			}
			input.Candidates = append(input.Candidates, candidate)
		}
		seen := map[string]bool{}
		for _, c := range input.Candidates {
			seen[c.File] = true
		}
		added := 0
		for _, related := range change.RelatedFiles {
			if added >= limits.RelatedPerChange {
				break
			}
			if seen[related.File] || len(related.Evidence) == 0 {
				continue
			}
			seen[related.File] = true
			added++
			candidate := Candidate{File: related.File, Module: related.Module, Layer: related.Layer, SuggestedAction: domain.ActionVerify, Origin: "source-caller", Evidence: []EvidenceItem{}}
			for i, e := range related.Evidence {
				if i >= limits.EvidencePerCandidate {
					break
				}
				candidate.Evidence = append(candidate.Evidence, EvidenceItem{ID: fmt.Sprintf("CH%02d-R%02d-E%02d", changeIndex+1, added, i+1), Type: e.Type, Location: e.Location, Excerpt: boundedRunes(e.Excerpt, limits.ExcerptRunes)})
			}
			input.Candidates = append(input.Candidates, candidate)
		}
		bundle.Changes = append(bundle.Changes, input)
	}
	bundle.ContextID, _ = ComputeContextID(bundle)
	return bundle
}

// BuildAgentBundle preserves the report's evidence IDs while assigning an
// explicitly subordinate flow to its owning requirement. Gateway grouping
// carries the branch details and source leads into the owner.
func BuildAgentBundle(report domain.Report, limits Limits) EvidenceBundle {
	b := buildEvidenceBundle(report, limits, true)
	owners := map[string][]int{}
	for i, c := range report.Changes {
		if c.PartOf == "" && b.Changes[i].ElementID != "" {
			id := b.Changes[i].ElementID
			owners[id] = append(owners[id], i)
		}
	}
	grouped := make(map[int]bool)
	for i, c := range report.Changes {
		parent := owners[c.PartOf]
		if c.ChangeType != domain.ModifySequenceFlow || len(parent) != 1 || b.Changes[i].ElementID == "" || len(c.Impacts) != 0 || len(c.RelatedFiles) != 0 {
			continue
		}
		owner := &b.Changes[parent[0]]
		owner.SupportingFlowIDs = append(owner.SupportingFlowIDs, b.Changes[i].ElementID)
		grouped[i] = true
	}
	if limits.GroupGatewayChanges && report.ProcessContext != nil && report.ProcessContext.Before != nil && report.ProcessContext.After != nil {
		groupJoinWiring(report, &b, grouped)
		groupGatewayLabels(report, &b, grouped)
		if limits.GroupGatewayBranches {
			groupGatewayBranches(report, &b, grouped)
		}
	}
	if len(grouped) > 0 {
		units := make([]ChangeInput, 0, len(b.Changes)-len(grouped))
		for i, c := range b.Changes {
			if !grouped[i] {
				units = append(units, c)
			}
		}
		b.Changes = units
		b.ContextID, _ = ComputeContextID(b)
	}
	return b
}

// A branch already described by a condition change shares that review unit.
// A flow moved between gateways or absent from either version stays separate.
func groupGatewayBranches(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	before, after := report.ProcessContext.Before, report.ProcessContext.After
	owners := make(map[string]int)
	for i, change := range bundle.Changes {
		if change.ChangeType != domain.ModifyGatewayCondition || change.ContextTruncated || change.ContextIncomplete {
			continue
		}
		for _, branch := range change.BranchChanges {
			old, next := before.Flows[branch.FlowID], after.Flows[branch.FlowID]
			if old != nil && next != nil && old.SourceRef == change.ElementID && next.SourceRef == change.ElementID {
				owners[branch.FlowID] = i
			}
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.ChangeType != domain.ModifySequenceFlow || change.NewElement == nil {
			continue
		}
		id := change.NewElement.ID
		parent, ok := owners[id]
		if !ok || (change.PartOf != "" && change.PartOf != bundle.Changes[parent].ElementID) {
			continue
		}
		unit := &bundle.Changes[parent]
		unit.SupportingFlowIDs = append(unit.SupportingFlowIDs, id)
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

// The condition review already has the before and after gateway labels.
func groupGatewayLabels(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	conditions := map[string]int{}
	for i, change := range report.Changes {
		if change.ChangeType == domain.ModifyGatewayCondition && change.NewElement != nil {
			conditions[change.NewElement.ID] = i
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.PartOf != "" || change.ChangeType != domain.ModifyTaskLabel || change.NewElement == nil {
			continue
		}
		owner, ok := conditions[change.NewElement.ID]
		if !ok {
			continue
		}
		unit := &bundle.Changes[owner]
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

// Group only unconditional sequence flows incident to a newly added XOR join.
// Conditional flows and distinct activity changes retain their own review.
func groupJoinWiring(report domain.Report, bundle *EvidenceBundle, grouped map[int]bool) {
	before, after := report.ProcessContext.Before, report.ProcessContext.After
	joins := map[string]int{}
	for i, change := range report.Changes {
		e := change.NewElement
		if change.ChangeType == domain.AddGateway && e != nil && e.Type == "exclusiveGateway" &&
			len(after.Incoming[e.ID]) >= 2 && len(after.Outgoing[e.ID]) == 1 {
			joins[e.ID] = i
		}
	}
	for i, change := range report.Changes {
		if grouped[i] || change.PartOf != "" || change.ChangeType != domain.ModifySequenceFlow || change.NewElement == nil {
			continue
		}
		id := change.NewElement.ID
		flow := after.Flows[id]
		if flow == nil || flow.Name != "" || flow.Condition != "" {
			continue
		}
		if old := before.Flows[id]; old != nil && (old.Name != "" || old.Condition != "") {
			continue
		}
		source, fromJoin := joins[flow.SourceRef]
		target, toJoin := joins[flow.TargetRef]
		if fromJoin == toJoin {
			continue
		}
		owner := source
		if toJoin {
			owner = target
		}
		unit := &bundle.Changes[owner]
		unit.SupportingFlowIDs = append(unit.SupportingFlowIDs, id)
		for _, lead := range bundle.Changes[i].Candidates {
			mergeAgentLead(unit, lead)
		}
		grouped[i] = true
	}
}

func mergeAgentLead(unit *ChangeInput, lead Candidate) {
	for i := range unit.Candidates {
		c := &unit.Candidates[i]
		if c.File != lead.File || c.Symbol != lead.Symbol {
			continue
		}
		if c.SuggestedAction != lead.SuggestedAction {
			c.SuggestedAction = domain.ActionVerify
		}
		for _, evidence := range lead.Evidence {
			duplicate := false
			for _, prior := range c.Evidence {
				if prior.Location == evidence.Location && prior.Excerpt == evidence.Excerpt {
					duplicate = true
					break
				}
			}
			if !duplicate {
				c.Evidence = append(c.Evidence, evidence)
			}
		}
		return
	}
	unit.Candidates = append(unit.Candidates, lead)
}

func boundedContext(value string, limit int, truncated *bool) string {
	if len([]rune(value)) > limit {
		*truncated = true
	}
	return boundedRunes(value, limit)
}
func elementInput(e *domain.Element, truncated *bool) *ElementInput {
	if e == nil {
		return nil
	}
	v := &ElementInput{ID: boundedContext(e.ID, 256, truncated), Name: boundedContext(e.Name, 240, truncated), Type: boundedContext(e.Type, 64, truncated), Actor: boundedContext(e.Actor, 240, truncated), State: boundedContext(e.State, 240, truncated), Documentation: boundedContext(e.Documentation, 512, truncated)}
	v.GatewayDirection = boundedContext(e.GatewayDirection, 64, truncated)
	v.DefaultFlow = boundedContext(e.DefaultFlow, 256, truncated)
	v.AttachedToRef = boundedContext(e.AttachedToRef, 256, truncated)
	for _, s := range e.EventDefinitions {
		if len(v.EventDefinitions) >= 8 {
			*truncated = true
			break
		}
		v.EventDefinitions = append(v.EventDefinitions, boundedContext(s, 240, truncated))
	}
	for _, s := range e.DataRequirements {
		if len(v.DataRequirements) >= 8 {
			*truncated = true
			break
		}
		v.DataRequirements = append(v.DataRequirements, boundedContext(s, 240, truncated))
	}
	return v
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
