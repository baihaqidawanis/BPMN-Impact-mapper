package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type testCallAudit struct {
	file  string
	sites []agenttools.TestCallSite
}

func addLockedTestCalls(ctx context.Context, snapshot *agenttools.Snapshot, bundle planning.EvidenceBundle, selector string) (planning.EvidenceBundle, testCallAudit, error) {
	parts := strings.Split(selector, ":")
	if len(parts) < 4 || len(selector) > 512 {
		return bundle, testCallAudit{}, fmt.Errorf("invalid --test-calls; use file:package.Constructor:Method:position=literal")
	}
	query := agenttools.TestCallQuery{File: parts[0], Constructor: parts[1], Method: parts[2], Literals: map[int]string{}}
	for _, filter := range parts[3:] {
		raw, literal, ok := strings.Cut(filter, "=")
		position, err := strconv.Atoi(raw)
		if !ok || err != nil || position < 1 {
			return bundle, testCallAudit{}, fmt.Errorf("invalid test call argument filter %q", filter)
		}
		if _, exists := query.Literals[position]; exists {
			return bundle, testCallAudit{}, fmt.Errorf("duplicate test call argument position %d", position)
		}
		query.Literals[position] = literal
	}
	var target *planning.Candidate
	for i := range bundle.Changes {
		for j := range bundle.Changes[i].Candidates {
			c := &bundle.Changes[i].Candidates[j]
			if c.File != query.File {
				continue
			}
			if target != nil || c.Symbol != "" {
				return bundle, testCallAudit{}, fmt.Errorf("test calls require one locked whole-file target for %q", query.File)
			}
			target = c
		}
	}
	if target == nil {
		return bundle, testCallAudit{}, fmt.Errorf("test call file %q is outside locked targets", query.File)
	}
	sites, err := snapshot.TestCallSites(ctx, query)
	if err != nil {
		return bundle, testCallAudit{}, err
	}
	for _, site := range sites {
		target.Evidence = append(target.Evidence, site.Evidence)
	}
	bundle.ContextID, err = planning.ComputeContextID(bundle)
	return bundle, testCallAudit{file: query.File, sites: sites}, err
}

func bindTestCallEvidence(plan *planning.FilePlan, sites []agenttools.TestCallSite) error {
	edits := strings.Join(plan.ProposedChanges, "\n") + "\n" + strings.Join(plan.Steps, "\n")
	cited := make(map[string]bool, len(plan.EvidenceIDs))
	for _, id := range plan.EvidenceIDs {
		cited[id] = true
	}
	var missing []string
	for _, site := range sites {
		if !hasSourceLineReference(edits, site.Reference) {
			missing = append(missing, site.Reference)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: address each test call by exact file:line in proposed changes or steps: %s",
			planning.ErrAgentModelOutput, strings.Join(missing, ", "))
	}
	for _, site := range sites {
		if !cited[site.Evidence.ID] {
			plan.EvidenceIDs = append(plan.EvidenceIDs, site.Evidence.ID)
			cited[site.Evidence.ID] = true
		}
	}
	return nil
}

func hasSourceLineReference(text, ref string) bool {
	for {
		before, after, found := strings.Cut(text, ref)
		if !found {
			return false
		}
		prefixIsFilename := len(before) > 0 && strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-", rune(before[len(before)-1]))
		if !prefixIsFilename && (len(after) == 0 || after[0] < '0' || after[0] > '9') {
			return true
		}
		text = after
	}
}

// Operator-selected windows and verified call sites are the primary evidence
// in locked revisions. Keep two inspected anchors for local context. The
// original bundle still validates all persisted citations and target bindings.
func focusLockedPlannerEvidence(bundle planning.EvidenceBundle, selected map[string][]string) planning.EvidenceBundle {
	if len(selected) == 0 {
		return bundle
	}
	out := bundle
	out.Changes = append([]planning.ChangeInput(nil), bundle.Changes...)
	for i, change := range out.Changes {
		out.Changes[i].Candidates = append([]planning.Candidate(nil), change.Candidates...)
		for j, candidate := range out.Changes[i].Candidates {
			picks := selected[candidate.File]
			if len(picks) == 0 {
				continue
			}
			byID := make(map[string]planning.EvidenceItem, len(candidate.Evidence))
			for _, item := range candidate.Evidence {
				byID[item.ID] = item
			}
			var evidence []planning.EvidenceItem
			for _, id := range picks {
				if item, found := byID[id]; found {
					evidence = append(evidence, item)
					delete(byID, id)
				}
			}
			if len(evidence) == 0 {
				continue
			}
			maxItems := len(evidence) + 2
			for _, item := range candidate.Evidence {
				if len(evidence) == maxItems {
					break
				}
				switch item.Type {
				case "AST_SOURCE_SCOPE", "AST_DECLARATION_FACT", "SOURCE_DECLARATION", "SOURCE_TOOL":
					if _, remaining := byID[item.ID]; remaining {
						evidence = append(evidence, item)
						delete(byID, item.ID)
					}
				}
			}
			out.Changes[i].Candidates[j].Evidence = evidence
		}
	}
	return out
}
