package planning

import (
	"fmt"
	"regexp"
	"strings"
)

var everyZeroDifference = regexp.MustCompile(`(?i)\b(?:all\s+(?:\w+[ -]){0,4}(?:deltas?|differences?)\s+(?:are\s+|equal\s+|=\s*)?(?:zero|0)|semua\s+selisih\s+nol)\b`)
var signedSumEdit = regexp.MustCompile(`(?i)\b(?:calculate|compute|check)\s+(?:whether\s+)?(?:the\s+)?(?:sum of (?:the )?(?:signed |all |line |per-line )*(?:deltas?|differences?)|(?:net |total )?(?:delta|difference) sum)\b`)
var zeroSumBranch = regexp.MustCompile(`(?i)\bif\s+(?:the\s+)?(?:calculated\s+)?(?:delta|difference) sum is (?:zero|0)\s*,?\s*(?:then\s+)?(?:set|close|skip|return)\b`)
var skippedStatusWrite = regexp.MustCompile(`(?i)\breturn\s+(?:success|a successful response|(?:http\s+)?2\d\d)\b[^.]{0,160}\bwithout\s+(?:writing|saving|persisting|updating)\s+(?:the\s+)?new\s+status\b`)
var returnAfterStateAssignment = regexp.MustCompile(`(?i)\bset\s+\w+\.(?:status|state)\s*(?:=|to)\s*[^\w\s]?\w+[^\w\s]?\s+and\s+return\s+immediately\b`)

// This checks explicit contradictions. Prose alone cannot prove a transaction
// is correct, so the reviewer still traces the write and the effects.
func validateRoutingPlan(change ChangeInput, plan FilePlan) error {
	if len(change.BranchChanges) == 0 || plan.Layer == "test" || plan.Layer == "documentation" {
		return nil
	}
	var ruleText []string
	for _, branch := range change.BranchChanges {
		ruleText = append(ruleText, branch.NewCondition, branch.NewLabel)
	}
	perItem := everyZeroDifference.MatchString(strings.Join(ruleText, "\n"))
	for _, edits := range [][]string{plan.ProposedChanges, plan.Steps} {
		for _, edit := range edits {
			if perItem && (signedSumEdit.MatchString(edit) || zeroSumBranch.MatchString(edit)) {
				return fmt.Errorf("%w: %q substitutes an aggregate for an all-items-zero branch; compare each item and preserve the normal path for opposite nonzero differences", ErrAgentPlanIncomplete, plan.File)
			}
			if skippedStatusWrite.MatchString(edit) || (perItem && returnAfterStateAssignment.MatchString(edit)) {
				return fmt.Errorf("%w: %q explicitly bypasses the changed status write; trace it through persistence, audit where applicable, and commit before returning success", ErrAgentPlanIncomplete, plan.File)
			}
		}
	}
	return nil
}
