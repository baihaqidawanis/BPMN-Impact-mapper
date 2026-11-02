package bpmn

import (
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// Finding separates local structural concerns from a research modelling policy.
// Neither category is a proof of execution correctness or source equivalence.
type Finding struct {
	Code      string `json:"code"`
	Category  string `json:"category"`
	ElementID string `json:"element_id,omitempty"`
	Scope     string `json:"scope"`
	Message   string `json:"message"`
}

type CheckOptions struct{ ResearchConvention bool }

// Check is read-only. Checks use actual sequence flows rather than direction
// attributes. A join has multiple incoming flows, regardless of its marker.
func Check(g *domain.Graph, options CheckOptions) []Finding {
	findings := []Finding{}
	if g == nil {
		return findings
	}
	for _, warning := range g.Warnings {
		findings = append(findings, Finding{Code: "BPMN_PARTIAL_SUPPORT", Category: "capability", Message: warning})
	}
	type scopeKey struct{ process, parent string }
	scopes := map[scopeKey][]*domain.Element{}
	for _, e := range g.Elements {
		if e.Type == "dataObjectReference" || e.Type == "dataStoreReference" {
			continue
		}
		key := scopeKey{e.ProcessID, e.ParentID}
		scopes[key] = append(scopes[key], e)
	}
	for key, elements := range scopes {
		scope := key.process
		if key.parent != "" {
			scope += "/" + key.parent
		}
		add := func(code, category, id, message string) {
			findings = append(findings, Finding{code, category, id, scope, message})
		}
		starts, ends := 0, 0
		for _, e := range elements {
			incoming, outgoing := g.Incoming[e.ID], g.Outgoing[e.ID]
			if e.Type == "startEvent" {
				starts++
				if len(incoming) > 0 {
					add("START_HAS_INCOMING", "structure", e.ID, "Start event has incoming sequence flow.")
				}
			}
			if e.Type == "endEvent" {
				ends++
				if len(outgoing) > 0 {
					add("END_HAS_OUTGOING", "structure", e.ID, "End event has outgoing sequence flow.")
				}
			}
			if len(incoming) == 0 && len(outgoing) == 0 && e.Type != "boundaryEvent" {
				add("ISOLATED_NODE", "structure", e.ID, "No sequence flow connects this node; review its scope.")
			}
			if !strings.HasSuffix(e.Type, "Gateway") {
				continue
			}
			if e.DefaultFlow != "" {
				f := g.Flows[e.DefaultFlow]
				if f == nil || f.SourceRef != e.ID {
					add("INVALID_GATEWAY_DEFAULT", "structure", e.ID, "Default flow is not an outgoing flow of this gateway.")
				}
			}
			if e.Type == "parallelGateway" {
				for _, id := range outgoing {
					if f := g.Flows[id]; f != nil && f.Condition != "" {
						add("AND_CONDITION_IGNORED", "structure", e.ID, "Parallel split does not select branches using conditions; review flow "+id+".")
					}
				}
				// Only a direct XOR->AND join is flagged. Do not pretend this local
				// heuristic can diagnose arbitrary loops or nested synchronization.
				if len(incoming) > 1 {
					for _, id := range incoming {
						f := g.Flows[id]
						if f == nil {
							continue
						}
						source := g.Elements[f.SourceRef]
						if source != nil && source.Type == "exclusiveGateway" && len(g.Outgoing[source.ID]) > 1 {
							add("REVIEW_XOR_TO_AND_JOIN", "review", e.ID, "An exclusive split feeds this AND join directly; check that every required incoming branch can arrive.")
							break
						}
					}
				}
			}
			if (e.Type == "exclusiveGateway" || e.Type == "inclusiveGateway") && len(outgoing) > 1 {
				for _, id := range outgoing {
					f := g.Flows[id]
					if f != nil && id != e.DefaultFlow && strings.TrimSpace(f.Condition) == "" && strings.TrimSpace(f.Name) == "" {
						add("BRANCH_RULE_UNSPECIFIED", "review", e.ID, "Branch "+id+" has neither a condition nor a descriptive label; review the decision rule.")
					}
				}
			}
			if options.ResearchConvention && len(incoming) > 1 {
				for _, id := range outgoing {
					f := g.Flows[id]
					if f == nil {
						continue
					}
					target := g.Elements[f.TargetRef]
					if target != nil && !activity(target.Type) {
						add("ACTIVITY_AFTER_JOIN", "convention", e.ID, "Research convention: flow "+id+" should lead to a meaningful activity. Do not invent an activity solely to satisfy this rule.")
					}
				}
			}
		}
		if options.ResearchConvention {
			if starts != 1 {
				add("SINGLE_START", "convention", "", fmt.Sprintf("Research convention expects one start event in this scope; found %d. Multiple or implicit starts can be valid BPMN.", starts))
			}
			if ends != 1 {
				add("SINGLE_END", "convention", "", fmt.Sprintf("Research convention expects one end event in this scope; found %d. Multiple or implicit ends can be valid BPMN.", ends))
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		return a.Scope+"\x00"+a.ElementID+"\x00"+a.Code+"\x00"+a.Message < b.Scope+"\x00"+b.ElementID+"\x00"+b.Code+"\x00"+b.Message
	})
	return findings
}

func activity(kind string) bool {
	switch kind {
	case "task", "userTask", "serviceTask", "manualTask", "scriptTask", "businessRuleTask", "sendTask", "receiveTask", "callActivity", "subProcess":
		return true
	}
	return false
}
