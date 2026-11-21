package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Indices are local to one immutable model turn. They are never persisted as
// source identities; the host restores exact candidate/evidence tuples.
type wireTarget struct {
	change    planning.ChangeInput
	candidate planning.Candidate
}

func turnTargets(t planning.AgentTurn) []wireTarget {
	var targets []wireTarget
	for _, change := range t.Bundle.Changes {
		for _, c := range change.Candidates {
			targets = append(targets, wireTarget{change, c})
		}
	}
	return targets
}

func annotateAgentInput(input []byte, t planning.AgentTurn) ([]byte, error) {
	var wire map[string]any
	if err := json.Unmarshal(input, &wire); err != nil {
		return nil, err
	}
	bundle, ok := wire["bundle"].(map[string]any)
	if !ok {
		return input, nil
	}
	if t.Role == planning.Reviewer && len(t.Draft) > 0 {
		roles := map[string]string{}
		for i, proposal := range t.Draft {
			file := strings.ToLower(proposal.File)
			role := "implementation/configuration"
			switch {
			case path.Ext(file) == ".md" || path.Ext(file) == ".rst" || path.Ext(file) == ".adoc":
				role = "documentation: propose edits to this document's rules/text"
			case strings.HasSuffix(file, "_test.go") || strings.Contains(path.Base(file), ".test.") || strings.Contains(path.Base(file), ".spec."):
				role = "test: propose assertions/fixtures; new tests need not exist yet"
			}
			roles[strconv.Itoa(i)] = role
		}
		wire["review_frame"] = map[string]any{"source_time": "BEFORE", "draft_time": "PROPOSED_AFTER, not implemented", "current_behavior": "untrusted claims to verify against source", "file_responsibilities": roles}
	}
	indices := map[string]int{}
	for i, target := range turnTargets(t) {
		indices[string(target.change.ChangeType)+"\x00"+target.change.ElementID+"\x00"+target.candidate.File+"\x00"+target.candidate.Symbol] = i
	}
	if changes, ok := bundle["changes"].([]any); ok {
		for _, value := range changes {
			change := value.(map[string]any)
			if candidates, ok := change["candidates"].([]any); ok {
				for _, value := range candidates {
					c := value.(map[string]any)
					symbol, _ := c["symbol"].(string)
					key := fmt.Sprint(change["change_type"]) + "\x00" + fmt.Sprint(change["element_id"]) + "\x00" + fmt.Sprint(c["file"]) + "\x00" + symbol
					index, ok := indices[key]
					if !ok {
						return nil, errors.New("wire candidate has no original target")
					}
					c["target_index"] = index
					if evidence, ok := c["evidence"].([]any); ok {
						for i, value := range evidence {
							value.(map[string]any)["evidence_index"] = i
						}
					}
				}
			}
		}
	}
	// Provide original, unaliased facts for each change. Graph sharing is a wire
	// compression detail, not a statement that before is empty or after invalid.
	pc := t.Bundle.ProcessContext
	if pc != nil && pc.Before != nil && pc.After != nil {
		var facts []map[string]any
		lookup := func(g *domain.Graph, id string) any {
			if e := g.Elements[id]; e != nil {
				return e
			}
			if f := g.Flows[id]; f != nil {
				return f
			}
			return nil
		}
		for i, c := range t.Bundle.Changes {
			before, after := lookup(pc.Before, c.ElementID), lookup(pc.After, c.ElementID)
			fact := map[string]any{"change_index": i, "change_type": c.ChangeType, "element_id": c.ElementID, "before_present": before != nil, "after_present": after != nil, "before": before, "after": after}
			if name := processNameForChange(pc.After, c.ElementID); name != "" {
				fact["process_after"] = name
			}
			if name := processNameForChange(pc.Before, c.ElementID); name != "" {
				fact["process_before"] = name
			}
			facts = append(facts, fact)
		}
		wire["change_facts"] = facts
	}
	if t.Role == planning.Planner {
		slots := []map[string]any{}
		for i, target := range turnTargets(t) {
			for _, d := range t.Decisions {
				if d.ChangeType == target.change.ChangeType && d.ElementID == target.change.ElementID && d.File == target.candidate.File && d.Symbol == target.candidate.Symbol && d.Verdict == planning.VerdictRelevant {
					var cited []int
					for j, e := range target.candidate.Evidence {
						for _, id := range d.EvidenceIDs {
							if id == e.ID {
								cited = append(cited, j)
								break
							}
						}
					}
					slot := map[string]any{"key": strconv.Itoa(i), "file": d.File, "symbol": d.Symbol, "required": true, "analyst_evidence_indices": cited}
					// Always forward source references. A lengthy model rationale
					// should not displace that source or be cut mid-claim.
					if len(d.Rationale) <= 480 {
						slot["analyst_hypothesis"] = d.Rationale
					} else if end := strings.LastIndex(d.Rationale[:480], ". "); end >= 0 {
						slot["analyst_hypothesis"] = d.Rationale[:end+1]
					}
					slots = append(slots, slot)
				}
			}
		}
		wire["planning_targets"] = slots
	}
	if t.Role == planning.Reviewer {
		slots := []map[string]any{}
		for i, p := range t.Draft {
			slot := map[string]any{"key": strconv.Itoa(i), "file": p.File, "symbol": p.Symbol, "change_type": p.ChangeType, "element_id": p.ElementID, "basis": "proposal_only"}
			for j, target := range turnTargets(t) {
				if target.change.ChangeType == p.ChangeType && target.change.ElementID == p.ElementID && target.candidate.File == p.File && target.candidate.Symbol == p.Symbol {
					slot["candidate_target_index"], slot["source_evidence_count"], slot["basis"] = j, len(target.candidate.Evidence), "source_and_proposal"
					break
				}
			}
			slots = append(slots, slot)
		}
		wire["review_targets"] = slots
	}
	return json.Marshal(wire)
}

func hostTransportInstruction(role planning.AgentRole) string {
	common := " change_facts are host-derived before/after facts with original IDs. New elements normally exist only in after; removed elements only in before. Shared graph entries belong to BOTH versions, not missing before data. supporting_flow_ids describe wiring for their owning requirement. Use the graph to locate the affected branch. Indices are scoped to THIS request only. Source-tool requests use exact approved paths. "
	switch role {
	case planning.Analyst:
		return common + "Return verification.decisions as an object keyed by each exact target_index string. Every candidate key is required exactly once; each value contains verdict, rationale and evidence_indices from that candidate. Do not emit target_index inside a keyed decision or file/function/change/evidence-ID identity fields; the host binds them."
	case planning.Planner:
		return common + "planning_targets lists the Analyst-validated RELEVANT targets. Return plan.plans as an object keyed by the exact target_index string, one assessment per required key. Set disposition CHANGE for intended edits or NO_CHANGE for an evidence-backed already adequate file; NO_CHANGE has empty steps and proposed_changes. Use that candidate's evidence_indices. Do not emit target_index inside a keyed proposal or file/function/change/module/layer/action/evidence-ID identity fields. For CHANGE, cite inspected source that shows the behavior gap and specify the edit and acceptance check. The host assigns the action."
	case planning.Reviewer:
		return common + "review_targets maps each required draft key to its matching source candidate. Draft indices and candidate_target_index are different namespaces. Key review.checks by the draft key and cite evidence_indices from its matching candidate. The host restores all identities."
	default:
		return common
	}
}

func indexedTargetSchema(schema map[string]any, action, field string, t planning.AgentTurn, plans bool) {
	if choices, ok := schema["oneOf"].([]any); ok {
		for _, c := range choices {
			indexedTargetSchema(c.(map[string]any), action, field, t, plans)
		}
		return
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	value, ok := props[action].(map[string]any)
	if !ok {
		return
	}
	if plans {
		properties := map[string]any{}
		required := []string{}
		for i, target := range turnTargets(t) {
			for _, decision := range t.Decisions {
				if decision.ChangeType != target.change.ChangeType || decision.ElementID != target.change.ElementID || decision.File != target.candidate.File || decision.Symbol != target.candidate.Symbol || decision.Verdict != planning.VerdictRelevant {
					continue
				}
				item := schemaType(reflect.TypeOf(planning.FilePlan{}))
				props := item["properties"].(map[string]any)
				identities := map[string]bool{"change_type": true, "element_id": true, "file": true, "symbol": true, "module": true, "layer": true, "evidence_ids": true, "action": true}
				for key := range identities {
					delete(props, key)
				}
				fields := []string{}
				for _, key := range item["required"].([]string) {
					if !identities[key] {
						fields = append(fields, key)
					}
				}
				props["evidence_indices"] = map[string]any{"type": "array", "minItems": 1, "maxItems": 8, "items": map[string]any{"type": "integer", "minimum": 0, "maximum": max(0, len(target.candidate.Evidence)-1)}}
				props["disposition"] = map[string]any{"type": "string", "enum": []string{"CHANGE", "NO_CHANGE"}}
				// The host validates conditional edit cardinality. NO_CHANGE must
				// describe an existing supported contract without inventing edits.
				props["steps"].(map[string]any)["minItems"] = 0
				props["proposed_changes"].(map[string]any)["minItems"] = 0
				item["required"] = append(fields, "evidence_indices", "disposition")
				key := strconv.Itoa(i)
				properties[key] = item
				if decision.Verdict == planning.VerdictRelevant {
					required = append(required, key)
				}
			}
		}
		value["properties"].(map[string]any)[field] = map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
		return
	}

	properties := map[string]any{}
	required := []string{}
	for i, target := range turnTargets(t) {
		item := schemaType(reflect.TypeOf(planning.CandidateDecision{}))
		props := item["properties"].(map[string]any)
		for _, key := range []string{"change_type", "element_id", "file", "symbol", "evidence_ids"} {
			delete(props, key)
		}
		props["evidence_indices"] = map[string]any{"type": "array", "minItems": 1, "maxItems": 8, "items": map[string]any{"type": "integer", "minimum": 0, "maximum": max(0, len(target.candidate.Evidence)-1)}}
		item["required"] = []string{"verdict", "rationale", "evidence_indices"}
		key := strconv.Itoa(i)
		properties[key] = item
		required = append(required, key)
	}
	value["properties"].(map[string]any)[field] = map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func restoreTargetReferences(content string, t planning.AgentTurn) (string, error) {
	var wire map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &wire); err != nil {
		return "", err
	}
	targets := turnTargets(t)
	for action, field := range map[string]string{"verification": "decisions", "plan": "plans"} {
		if wire[action] == nil {
			continue
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(wire[action], &body); err != nil {
			return "", err
		}
		var items []map[string]json.RawMessage
		if strings.HasPrefix(strings.TrimSpace(string(body[field])), "{") {
			var err error
			items, err = decodePlanSlots(body[field], len(targets))
			if err != nil {
				return "", err
			}
		} else if err := json.Unmarshal(body[field], &items); err != nil {
			return "", err
		}
		for _, item := range items {
			raw, ok := item["target_index"]
			if !ok {
				continue
			} // legacy JSON remains subject to exact engine validation
			var index int
			if string(raw) == "null" || json.Unmarshal(raw, &index) != nil || index < 0 || index >= len(targets) {
				return "", errors.New("unknown target index")
			}
			for _, key := range []string{"file", "symbol", "module", "layer", "change_type", "element_id", "evidence_ids"} {
				if _, ok := item[key]; ok {
					return "", errors.New("mixed indexed and named identities")
				}
			}
			target := targets[index]
			c := target.candidate
			var refs []json.RawMessage
			if json.Unmarshal(item["evidence_indices"], &refs) != nil || len(refs) == 0 || len(refs) > 8 {
				return "", errors.New("invalid evidence indices")
			}
			ids := []string{}
			for _, rawRef := range refs {
				var ref int
				if string(rawRef) == "null" || json.Unmarshal(rawRef, &ref) != nil {
					return "", errors.New("invalid evidence index")
				}
				if ref < 0 || ref >= len(c.Evidence) {
					return "", errors.New("evidence outside selected target")
				}
				ids = append(ids, c.Evidence[ref].ID)
			}
			fields := map[string]any{"change_type": target.change.ChangeType, "element_id": target.change.ElementID, "file": c.File, "symbol": c.Symbol, "evidence_ids": ids}
			if action == "plan" {
				disposition := "CHANGE"
				if raw, ok := item["disposition"]; ok {
					if json.Unmarshal(raw, &disposition) != nil || (disposition != "CHANGE" && disposition != "NO_CHANGE") {
						return "", errors.New("invalid plan disposition")
					}
				}
				expected := c.SuggestedAction
				if expected == "" {
					expected = domain.ActionVerify
				}
				if disposition == "NO_CHANGE" {
					expected = domain.ActionNoAction
				} else if _, explicit := item["disposition"]; explicit && c.SuggestedAction == domain.ActionVerify {
					data, err := json.Marshal(item)
					var proposal planning.FilePlan
					if err != nil || json.Unmarshal(data, &proposal) != nil {
						return "", errors.New("invalid indexed proposal")
					}
					proposal.EvidenceIDs = ids
					if planning.SupportsVerifiedEdit(c, proposal) {
						expected = domain.ActionModify
					}
				}
				// New indexed schemas omit action. Legacy explicit values are checked,
				// never normalized from BPMN taxonomy or arbitrary free text.
				if rawAction, ok := item["action"]; ok {
					var provided domain.RequiredAction
					if json.Unmarshal(rawAction, &provided) != nil || (provided != expected && provided != domain.ActionVerify && provided != domain.ActionUnresolved) {
						return "", errors.New("invalid explicit planned action")
					}
					if disposition == "NO_CHANGE" && provided != domain.ActionNoAction {
						return "", errors.New("no-change disposition contradicts explicit action")
					}
					expected = provided
				}
				fields["action"] = expected
				fields["module"] = c.Module
				fields["layer"] = c.Layer
			}
			for key, value := range fields {
				item[key], _ = json.Marshal(value)
			}
			delete(item, "target_index")
			delete(item, "evidence_indices")
			delete(item, "disposition")
		}
		body[field], _ = json.Marshal(items)
		wire[action], _ = json.Marshal(body)
	}
	b, err := json.Marshal(wire)
	return string(b), err
}

// The object grammar provides one slot per target rather than an array in which
// a model may repeat one index and omit another. Duplicate JSON keys are errors.
func decodePlanSlots(raw json.RawMessage, targetCount int) ([]map[string]json.RawMessage, error) {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	slots := map[int]map[string]json.RawMessage{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("invalid plan slot")
		}
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= targetCount || strconv.Itoa(index) != key {
			return nil, errors.New("unknown plan slot")
		}
		if _, ok := slots[index]; ok {
			return nil, errors.New("duplicate plan slot")
		}
		var item map[string]json.RawMessage
		if err := d.Decode(&item); err != nil {
			return nil, err
		}
		if item == nil {
			return nil, errors.New("null plan slot")
		}
		if _, ok := item["target_index"]; ok {
			return nil, errors.New("mixed slot and indexed target")
		}
		item["target_index"], _ = json.Marshal(index)
		slots[index] = item
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	indices := make([]int, 0, len(slots))
	for i := range slots {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	items := make([]map[string]json.RawMessage, 0, len(slots))
	for _, i := range indices {
		items = append(items, slots[i])
	}
	return items, nil
}
