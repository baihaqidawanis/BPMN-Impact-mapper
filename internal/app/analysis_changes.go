package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

// attachPrimaryChanges maps each independent process change once. A plain
// sequence-flow split remains in canonical JSON as a supporting graph diff,
// but cannot create three duplicate candidate lists for one inserted task.
func attachPrimaryChanges(ctx context.Context, strategy mapping.Strategy, changes []domain.Change, artifacts []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	indices := make([]int, 0, len(changes))
	primary := make([]domain.Change, 0, len(changes))
	warnings := []domain.Warning{}
	for index, change := range changes {
		supportingFlow := change.PartOf != "" && change.ChangeType == domain.ModifySequenceFlow
		if change.PartOf != "" && !supportingFlow {
			continue
		}
		// A changed display label with an unchanged explicit condition is not
		// evidence that neighboring implementation must change. All retrieval
		// methods use the same direct-reference eligibility rule; an explicit
		// configuration mapping remains an independent method.
		if supportingFlow || (strategy.Method() != domain.MethodConfiguration && isExplicitFlowLabelOnly(change)) {
			eligible, references := flowLabelReferences(change, artifacts)
			if len(eligible) == 0 {
				if supportingFlow {
					continue
				}
				changes[index].UnresolvedImpact = append(changes[index].UnresolvedImpact,
					"sequence-flow display label changed but no direct source reference was found; code impact is unproven")
				continue
			}
			if supportingFlow {
				change.PartOf = ""
				change.Reason += "; direct source reference requires separate verification"
			}
			mapped, focusedWarnings, err := strategy.Attach(ctx, []domain.Change{change}, eligible, options)
			if err != nil {
				return nil, nil, err
			}
			if len(mapped) != 1 {
				return nil, nil, fmt.Errorf("mapping strategy returned %d changes for one directly referenced flow", len(mapped))
			}
			for impactIndex := range mapped[0].Impacts {
				impact := &mapped[0].Impacts[impactIndex]
				reference, ok := references[impact.File+"\x00"+impact.Symbol]
				if !ok {
					return nil, nil, fmt.Errorf("mapping strategy returned a flow-label candidate outside the eligible source set")
				}
				impact.Evidence = append(impact.Evidence, reference)
			}
			changes[index] = mapped[0]
			warnings = append(warnings, focusedWarnings...)
			continue
		}
		indices = append(indices, index)
		primary = append(primary, change)
	}
	if len(primary) > 0 {
		mapped, primaryWarnings, err := strategy.Attach(ctx, primary, artifacts, options)
		if err != nil {
			return nil, nil, err
		}
		if len(mapped) != len(indices) {
			return nil, nil, fmt.Errorf("mapping strategy returned %d changes for %d primary changes", len(mapped), len(indices))
		}
		for index, originalIndex := range indices {
			changes[originalIndex] = mapped[index]
		}
		warnings = append(warnings, primaryWarnings...)
	}
	sort.SliceStable(warnings, func(i, j int) bool {
		if warnings[i].Code != warnings[j].Code {
			return warnings[i].Code < warnings[j].Code
		}
		return warnings[i].Message < warnings[j].Message
	})
	return changes, warnings, nil
}

func isExplicitFlowLabelOnly(change domain.Change) bool {
	old, next := change.OldElement, change.NewElement
	return change.ChangeType == domain.ModifySequenceFlow && old != nil && next != nil &&
		old.Type == "sequenceFlow" && next.Type == "sequenceFlow" && old.ID == next.ID &&
		old.Name != next.Name && strings.TrimSpace(old.Documentation) != "" &&
		old.Documentation == next.Documentation && sameRefs(old.Incoming, next.Incoming) && sameRefs(old.Outgoing, next.Outgoing)
}

func sameRefs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// Direct references are a conservative eligibility check, not proof that a
// code change is required. Single-word labels are too broad; the stable flow
// ID remains eligible even when its label is short.
func flowLabelReferences(change domain.Change, artifacts []domain.Artifact) ([]domain.Artifact, map[string]domain.Evidence) {
	eligible := make([]domain.Artifact, 0)
	references := make(map[string]domain.Evidence)
	for _, artifact := range artifacts {
		kind, line, excerpt := "", 0, ""
		for _, element := range []*domain.Element{change.OldElement, change.NewElement} {
			if element != nil {
				kind, line, excerpt = referenceLine(artifact.Content, element.ID, true)
				if kind != "" {
					break
				}
			}
		}
		if kind == "" {
			for _, element := range []*domain.Element{change.OldElement, change.NewElement} {
				if element == nil {
					continue
				}
				label := element.Name
				if len(strings.Fields(label)) < 2 || len([]rune(strings.TrimSpace(label))) < 8 {
					continue
				}
				kind, line, excerpt = referenceLine(artifact.Content, label, false)
				if kind != "" {
					break
				}
			}
		}
		if kind == "" {
			continue
		}
		if artifact.Symbol.StartLine > 0 {
			line += artifact.Symbol.StartLine - 1
		}
		key := artifact.File + "\x00" + impactSymbol(artifact.Symbol)
		references[key] = domain.Evidence{Type: kind, Location: fmt.Sprintf("%s:%d", artifact.File, line), Excerpt: excerpt}
		// Fusion can conservatively return a whole-file candidate instead of
		// asserting a function. Its source reference is still valid evidence.
		fileKey := artifact.File + "\x00"
		if _, exists := references[fileKey]; !exists {
			references[fileKey] = references[key]
		}
		eligible = append(eligible, artifact)
	}
	return eligible, references
}

func referenceLine(content, value string, identifier bool) (string, int, string) {
	if strings.TrimSpace(value) == "" {
		return "", 0, ""
	}
	for index, line := range strings.Split(content, "\n") {
		match := delimitedTextIndex(line, value, identifier)
		if match < 0 {
			continue
		}
		kind := "FLOW_LABEL_TEXT"
		if identifier {
			kind = "FLOW_ID_REFERENCE"
		}
		runes := []rune(line)
		start := match - 80
		if start < 0 {
			start = 0
		}
		if start+240 > len(runes) {
			start = len(runes) - 240
			if start < 0 {
				start = 0
			}
		}
		end := start + 240
		if end > len(runes) {
			end = len(runes)
		}
		return kind, index + 1, strings.TrimSpace(string(runes[start:end]))
	}
	return "", 0, ""
}

func delimitedTextIndex(content, value string, identifier bool) int {
	haystack, needle := []rune(strings.ToLower(content)), []rune(strings.ToLower(strings.TrimSpace(value)))
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	for start := 0; start+len(needle) <= len(haystack); start++ {
		if !sameRunes(haystack[start:start+len(needle)], needle) {
			continue
		}
		end := start + len(needle)
		boundary := isIdentifierRune
		if identifier {
			boundary = isFlowIDRune
		}
		if (start == 0 || !boundary(haystack[start-1])) && (end == len(haystack) || !boundary(haystack[end])) {
			return start
		}
	}
	return -1
}

func sameRunes(left, right []rune) bool {
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func isIdentifierRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_'
}

func isFlowIDRune(value rune) bool { return isIdentifierRune(value) || value == '-' }

func impactSymbol(symbol domain.Symbol) string {
	if symbol.Name == "" {
		return ""
	}
	if symbol.Receiver != "" {
		return symbol.Receiver + "." + symbol.Name
	}
	return symbol.Name
}
