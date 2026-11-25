package planning

import (
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// combinedPlanConflictIssues detects deterministic cross-proposal contradictions
// and missing schema prerequisites across the assembled focused plan. It never
// drops proposals; callers attach these issues so the exported draft remains
// honestly marked NEEDS_REVIEW.
func combinedPlanConflictIssues(bundle EvidenceBundle, plans []FilePlan) []string {
	if len(plans) == 0 {
		return nil
	}
	var issues []string
	issues = append(issues, migrationPrerequisiteIssues(bundle, plans)...)

	changes := make(map[string]ChangeInput, len(bundle.Changes))
	for _, c := range bundle.Changes {
		changes[planKey(c.ChangeType, c.ElementID, "", "")] = c
	}
	for _, p := range plans {
		if err := validateRoutingPlanWithContext(bundle.ProcessContext, changes[planKey(p.ChangeType, p.ElementID, "", "")], p); err != nil {
			issues = append(issues, err.Error())
		}
	}

	if hasEveryZeroDifference(bundle.ProcessContext, ChangeInput{}) {
		byFile := map[string][]FilePlan{}
		var files []string
		for _, p := range plans {
			if p.Action == domain.ActionNoAction || p.Layer == "documentation" {
				continue
			}
			if _, ok := byFile[p.File]; !ok {
				files = append(files, p.File)
			}
			byFile[p.File] = append(byFile[p.File], p)
		}
		sort.Strings(files)
		for _, file := range files {
			group := byFile[file]
			if len(group) < 2 {
				continue
			}
			outcomeToElements := map[string][]string{}
			for _, p := range group {
				for _, out := range zeroDeltaOutcomes(p) {
					outcomeToElements[out] = append(outcomeToElements[out], p.ElementID)
				}
			}
			if len(outcomeToElements) > 1 {
				keys := make([]string, 0, len(outcomeToElements))
				for k := range outcomeToElements {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				parts := make([]string, 0, len(keys))
				for _, k := range keys {
					elems := cleanStrings(outcomeToElements[k])
					parts = append(parts, fmt.Sprintf("%s (%s)", k, strings.Join(elems, ", ")))
				}
				issues = append(issues, fmt.Sprintf("Cross-proposal conflict on %s: different BPMN change proposals assign conflicting zero-difference outcomes: %s.", file, strings.Join(parts, " vs ")))
			}
		}
	}
	return cleanStrings(issues)
}
