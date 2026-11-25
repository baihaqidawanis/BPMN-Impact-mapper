package planning

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

var everyZeroDifference = regexp.MustCompile(`(?i)\b(?:all\s+(?:\w+[ -]){0,4}(?:deltas?|differences?)\s+(?:are\s+|equal\s+|=\s*)?(?:zero|0)|semua\s+(?:selisih(?:\s+stok)?|baris\s+berselisih)\s+nol)\b`)
var nonZeroBranchCue = regexp.MustCompile(`(?i)\b(?:non-?zero|nonzero|selisih\s+nonnol|berbeda|!=\s*0)\b`)
var failureTargetCue = regexp.MustCompile(`(?i)\b(?:tolak|reject|invalid|error|gagal|fail|cancel)\b`)

var signedSumEdit = regexp.MustCompile(`(?i)\b(?:(?:calculate|compute|check|derive|find|take|use|determine|verify|test)\s+(?:if\s+|whether\s+)?(?:the\s+)?(?:(?:signed|net|total)\s+(?:quantity\s+)?(?:deltas?|differences?)|sum\s+of\s+(?:all\s+|each\s+|every\s+|the\s+|signed\s+|line\s+|per-line\s+|quantity\s+|item\s+)*(?:deltas?|differences?)|(?:net\s+|total\s+)?(?:delta|difference)\s+sum)|sum\s+(?:up\s+)?(?:all\s+|each\s+|every\s+|the\s+|signed\s+|line\s+|per-line\s+|quantity\s+|item\s+)*(?:deltas?|differences?))\b`)
var zeroSumBranch = regexp.MustCompile(`(?i)\bif\s+(?:the\s+)?(?:calculated\s+)?(?:delta|difference)\s+sum\s+is\s+(?:zero|0)\s*,?\s*(?:then\s+)?(?:set|close|skip|return)\b`)
var deltasSumToZero = regexp.MustCompile(`(?i)\b(?:deltas?|differences?)\s+sum\s+to\s+(?:zero|0|non-?zero|nonzero)\b`)
var explicitAggregateBranch = regexp.MustCompile(`(?i)\b(?:if|when)\s+(?:the\s+)?(?:(?:calculated\s+)?(?:sum\s+of\s+(?:all\s+|each\s+|every\s+|the\s+|line\s+|per-line\s+|signed\s+|quantity\s+|item\s+)*(?:deltas?|differences?)|(?:net|total)\s+(?:quantity\s+)?(?:delta|difference)|(?:delta|difference)\s+sum))\s+(?:is|equals|==|!=)\s+(?:zero|0|non-?zero|nonzero)\s*[,;:]?\s*(?:then\s+)?(?:set|close|skip|return|reject|submit|transition|route|allow)\b`)

var skippedStatusWrite = regexp.MustCompile(`(?i)\breturn\s+(?:success|a successful response|(?:http\s+)?2\d\d)(?:\s+immediately)?\b[^.]{0,160}\bwithout\s+(?:writing|saving|persisting|updating|changing|modifying|setting)\s+(?:(?:the\s+)?new\s+status|(?:\w+\.)?(?:status|state))\b`)
var returnAfterStateAssignment = regexp.MustCompile(`(?i)\bset\s+\w+\.(?:status|state)\s*(?:=|to)\s*[^\w\s]?\w+[^\w\s]?\s+and\s+return\s+immediately\b`)

var negationCue = regexp.MustCompile(`(?i)\b(?:do\s+not|don't|never|not|no|instead\s+of|rather\s+than|without|even\s+if|even\s+when|despite|avoid|must\s+not|should\s+not|prohibit|forbid|cannot|skip(?:ping)?|defer(?:ring|red)?)\b`)

var nonZeroRejectionClause = regexp.MustCompile(`(?i)\b(?:(?:if|when)\s+(?:any\s+)?(?:line(?:\s+item)?|row|item)?\s*(?:has\s+(?:a\s+)?)?(?:non-?zero|nonzero)\s+(?:deltas?|differences?)|with\s+(?:a\s+|an\s+|at\s+least\s+one\s+)?(?:non-?zero|nonzero)\s+(?:line\s+)?(?:deltas?|differences?))\b((?:[^.;\n]|;\s*(?:expect|assert|verify|should|then)\b){0,160})\b(?:reject(?:s|ed|ing)?|return(?:s|ing)?\s+(?:a\s+|an\s+|the\s+)?(?:specific\s+|non-?nil\s+)?(?:validation\s+)?error|fail(?:s|ing)?\s+(?:validation|submission)|indicate(?:s)?\s+(?:a\s+|an\s+|the\s+)?(?:validation\s+)?(?:failure|error|rejection)|expect\s+(?:a\s+|an\s+|the\s+)?(?:validation\s+)?(?:failure|error|rejection))\b`)
var rejectNonZeroDirect = regexp.MustCompile(`(?i)\breject(?:s|ed|ing)?\b([^.;\n]{0,70})\b(?:non-?zero|nonzero)\s+(?:deltas?|differences?)\b`)

var zeroDeltaClauseCue = regexp.MustCompile(`(?i)\b(?:all\s+(?:lines?|rows?|items?)\s+(?:have\s+)?(?:a\s+)?zero\s+(?:deltas?|differences?)|all\s+(?:line\s+)?(?:deltas?|differences?)\s+(?:are\s+|equal\s+)?(?:zero|0)|(?:zero[- ]delta|0[- ]delta)(?:\s+(?:per\s+line|submission|submissions|case|path|adjustment|lines?))?|every\s+(?:line|row|item)\s+has\s+zero\s+(?:delta|difference)|semua\s+selisih(?:\s+stok)?\s+nol)\b`)
var statusAssignmentCue = regexp.MustCompile(`(?i)\b(?:set(?:ting|s)?\s+(?:the\s+)?(?:adjustment\s+|record\s+)?status\s+(?:to|=)|status\s+(?:becomes|is\s+set\s+to|changes\s+to|remains|stays)|transition(?:s|ed|ing)?\s+(?:the\s+)?(?:adjustment\s+|record\s+)?(?:status\s+)?to|close(?:s|d|ing)?\s+(?:the\s+)?(?:adjustment\s+)?(?:as|with\s+status)|return(?:s|ing)?\s+(?:the\s+)?(?:status\s+)?|mark(?:s|ed|ing)?\s+(?:as|status))\s*['"` + "`" + `"]?([A-Za-z][A-Za-z0-9_]{2,})['"` + "`" + `"]?\b`)
var zeroDeltaRejectCue = regexp.MustCompile(`(?i)\b(?:is\s+rejected|reject(?:s|ed|ing)?|return(?:s|ing)?\s+(?:a\s+|an\s+|the\s+)?(?:non-?nil\s+|specific\s+)?(?:validation\s+)?error|route(?:s|d|ing)?\b[^.;\n]{0,40}\b(?:invalid|reject|error))\b`)
var clauseBranchSplit = regexp.MustCompile(`(?i)(?:[.;\n]+|\b(?:otherwise|whereas)\b)`)

var nonStatusUpperTokens = map[string]bool{
	"HTTP": true, "JSON": true, "SQL": true, "API": true, "BPMN": true,
	"UUID": true, "CRUD": true, "NULL": true, "TRUE": true, "FALSE": true,
	"XML": true, "URL": true, "URI": true, "DTO": true, "DDL": true,
	"ERR": true, "EOF": true, "NIL": true, "OK": true,
}

var prematureSubmitStockMutation = regexp.MustCompile(`(?i)\b(?:(?:execute|perform|trigger|run)\s+(?:the\s+)?stock\s+(?:posting|update(?:\s+query)?|mutation)|audit(?:s|\s+records?)?\s+['"` + "`" + `"]?APPROVED['"` + "`" + `"]?)\b`)
var repurposedApprovalTest = regexp.MustCompile(`(?i)\b(?:update|modify|change)\s+(?:the\s+)?(?:existing\s+)?(?:test\s+)?` + "`" + `?Test\w*Approval\w*` + "`" + `?[^.;\n]{0,180}\b(?:zero[- ]delta|bypass|rather\s+than\s+['"` + "`" + `"]?SUBMITTED)\b`)
var preservedApprovalTestCue = regexp.MustCompile(`(?i)\b(?:preserv(?:e|es|ing)\s+(?:its\s+|the\s+)?original|keep(?:s|ing)?\s+(?:its\s+|the\s+)?original|separate\s+test|non-?zero\s+(?:delta|difference))\b`)
var opposingNonZeroTestCue = regexp.MustCompile(`(?i)\b(?:oppos(?:ing|ite)|cancel(?:l?ing)?\s+(?:out|sum|deltas?)|\+\s*\d+[^.\n]{0,30}-\s*\d+|\+delta[^.\n]{0,20}-delta|sum(?:s|ming)?\s+to\s+(?:zero|0))\b`)

// This checks explicit contradictions. Prose alone cannot prove a transaction
// is correct, so the reviewer still traces the write and the effects.
func validateRoutingPlan(change ChangeInput, plan FilePlan) error {
	return validateRoutingPlanWithContext(nil, change, plan)
}

func validateRoutingPlanWithContext(pc *domain.ProcessContext, change ChangeInput, plan FilePlan) error {
	if plan.Action == domain.ActionNoAction || plan.Layer == "documentation" {
		return nil
	}
	perItem := hasEveryZeroDifference(pc, change)
	if !perItem && len(change.BranchChanges) == 0 {
		return nil
	}
	nonZeroContinues := perItem && hasNonZeroContinuation(pc)
	deferredApproval := perItem && hasDeferredApprovalWithoutStockMutation(pc)
	zeroBypassOK, expectedStatuses := zeroBypassTargetContract(pc)
	for _, edits := range [][]string{plan.ProposedChanges, plan.Steps, plan.AcceptanceChecks} {
		for _, edit := range edits {
			if perItem && (matchUnnegated(signedSumEdit, edit) || matchUnnegated(zeroSumBranch, edit) || matchUnnegated(deltasSumToZero, edit) || matchUnnegated(explicitAggregateBranch, edit)) {
				return fmt.Errorf("%w: %q substitutes an aggregate for an all-items-zero branch (offending text: %q); rewrite this item to compare each line individually and preserve the normal path for opposite nonzero differences", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if nonZeroContinues && (matchUnnegatedBridge(nonZeroRejectionClause, edit) || matchUnnegatedBridge(rejectNonZeroDirect, edit)) {
				return fmt.Errorf("%w: %q rejects the valid non-zero difference branch (offending text: %q); in the after process, non-zero differences must continue along the normal path while precondition failures remain on their own gateway", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if deferredApproval && nonZeroBranchCue.MatchString(edit) && matchUnnegated(prematureSubmitStockMutation, edit) {
				return fmt.Errorf("%w: %q triggers approval or stock mutation inside the non-zero submit branch (offending text: %q); keep status as SUBMITTED for approval and preserve stock mutation in its separate post-approval operation", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if zeroBypassOK && (plan.Layer == "test" || strings.HasSuffix(strings.ToLower(plan.File), "_test.go")) && repurposedApprovalTest.MatchString(edit) && !preservedApprovalTestCue.MatchString(edit) {
				return fmt.Errorf("%w: %q repurposes an existing Approval test into a zero-difference bypass test (offending text: %q); preserve existing approval tests reaching SUBMITTED and add a separate test function for zero-difference bypass and opposite-nonzero (+delta/-delta) cases", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if plan.Layer != "test" && (skippedStatusWrite.MatchString(edit) || (perItem && returnAfterStateAssignment.MatchString(edit))) {
				return fmt.Errorf("%w: %q explicitly bypasses the changed status write (offending text: %q); trace it through persistence, audit where applicable, and commit before returning success", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
		}
	}
	if perItem {
		outcomes := zeroDeltaOutcomes(plan)
		if len(outcomes) > 1 {
			return fmt.Errorf("%w: %q contradicts itself on the zero-difference outcome (%s); align proposed changes, steps, and acceptance checks to one branch result", ErrAgentPlanIncomplete, plan.File, strings.Join(outcomes, " vs "))
		}
		if zeroBypassOK {
			for _, outcome := range outcomes {
				if outcome == "REJECT" {
					return fmt.Errorf("%w: %q rejects the zero-difference branch with an error; in the after process, all-zero differences follow a valid bypass path rather than a validation rejection", ErrAgentPlanIncomplete, plan.File)
				}
				if len(expectedStatuses) > 0 && !expectedStatuses[outcome] {
					want := sortedMapKeys(expectedStatuses)
					return fmt.Errorf("%w: %q assigns status %q to the zero-difference branch, but the after process routes that branch to %s", ErrAgentPlanIncomplete, plan.File, outcome, strings.Join(want, "/"))
				}
			}
			if nonZeroContinues && plan.Action == domain.ActionModify && (plan.Layer == "test" || strings.HasSuffix(strings.ToLower(plan.File), "_test.go")) {
				joined := strings.Join(append(append([]string{}, plan.ProposedChanges...), append(plan.Steps, plan.AcceptanceChecks...)...), "\n")
				if !opposingNonZeroTestCue.MatchString(joined) {
					return fmt.Errorf("%w: %q test plan for an all-items-zero branch must include an explicit opposing non-zero case (e.g., +2 and -2 summing to 0) verifying it still follows the normal non-zero path, alongside all-zero bypass and stale-version/status conflict checks", ErrAgentPlanIncomplete, plan.File)
				}
			}
		}
	}
	return nil
}

func hasEveryZeroDifference(pc *domain.ProcessContext, change ChangeInput) bool {
	var ruleText []string
	if change.After != nil {
		ruleText = append(ruleText, change.After.Name, change.After.Documentation)
	}
	for _, branch := range change.BranchChanges {
		ruleText = append(ruleText, branch.NewCondition, branch.NewLabel)
	}
	if everyZeroDifference.MatchString(strings.Join(ruleText, "\n")) {
		return true
	}
	if pc == nil || pc.After == nil {
		return false
	}
	for _, el := range pc.After.Elements {
		if el != nil && (everyZeroDifference.MatchString(el.Name) || everyZeroDifference.MatchString(el.Documentation)) {
			return true
		}
	}
	for _, flow := range pc.After.Flows {
		if flow != nil && (everyZeroDifference.MatchString(flow.Condition) || everyZeroDifference.MatchString(flow.Name)) {
			return true
		}
	}
	return false
}

func hasNonZeroContinuation(pc *domain.ProcessContext) bool {
	if pc == nil || pc.After == nil {
		return false
	}
	bySource := map[string][]*domain.Flow{}
	for _, f := range pc.After.Flows {
		if f != nil {
			bySource[f.SourceRef] = append(bySource[f.SourceRef], f)
		}
	}
	for sourceID, flows := range bySource {
		el := pc.After.Elements[sourceID]
		if el == nil || !strings.HasSuffix(el.Type, "Gateway") || len(flows) < 2 {
			continue
		}
		hasZero := false
		hasValidNonZero := false
		for _, f := range flows {
			text := f.Condition + " " + f.Name
			if everyZeroDifference.MatchString(text) {
				hasZero = true
				continue
			}
			if nonZeroBranchCue.MatchString(text) {
				target := pc.After.Elements[f.TargetRef]
				if target == nil || (!failureTargetCue.MatchString(target.Name) && !failureTargetCue.MatchString(target.ID) && !strings.Contains(strings.ToLower(target.Type), "error")) {
					hasValidNonZero = true
				}
			}
		}
		if hasZero && hasValidNonZero {
			return true
		}
	}
	return false
}

func matchUnnegated(re *regexp.Regexp, text string) bool {
	for _, loc := range re.FindAllStringIndex(text, -1) {
		start := loc[0]
		prefixStart := start - 36
		if prefixStart < 0 {
			prefixStart = 0
		}
		prefix := text[prefixStart:start]
		if idx := strings.LastIndexAny(prefix, ",.;:\n"); idx >= 0 {
			prefix = prefix[idx+1:]
		}
		if !negationCue.MatchString(prefix) {
			return true
		}
	}
	return false
}

func matchUnnegatedBridge(re *regexp.Regexp, text string) bool {
	for _, sub := range re.FindAllStringSubmatchIndex(text, -1) {
		start := sub[0]
		prefixStart := start - 36
		if prefixStart < 0 {
			prefixStart = 0
		}
		prefix := text[prefixStart:start]
		if idx := strings.LastIndexAny(prefix, ",.;:\n"); idx >= 0 {
			prefix = prefix[idx+1:]
		}
		if negationCue.MatchString(prefix) {
			continue
		}
		if len(sub) >= 4 && sub[2] >= 0 {
			bridge := text[sub[2]:sub[3]]
			if len(bridge) > 36 {
				bridge = bridge[len(bridge)-36:]
			}
			if idx := strings.LastIndexAny(bridge, ",.;:\n"); idx >= 0 {
				bridge = bridge[idx+1:]
			}
			if negationCue.MatchString(bridge) {
				continue
			}
		}
		return true
	}
	return false
}

func zeroDeltaOutcomes(plan FilePlan) []string {
	found := map[string]bool{}
	for _, list := range [][]string{plan.ProposedChanges, plan.Steps, plan.AcceptanceChecks} {
		for _, item := range list {
			for _, clause := range clauseBranchSplit.Split(item, -1) {
				if !zeroDeltaClauseCue.MatchString(clause) || nonZeroBranchCue.MatchString(clause) {
					continue
				}
				for _, m := range statusAssignmentCue.FindAllStringSubmatch(clause, -1) {
					token := m[1]
					// Only true UPPER_SNAKE_CASE status literals count; ignore lowercase prose like "return success".
					if token != strings.ToUpper(token) || nonStatusUpperTokens[token] {
						continue
					}
					found[token] = true
				}
				if matchUnnegated(zeroDeltaRejectCue, clause) {
					found["REJECT"] = true
				}
			}
		}
	}
	out := make([]string, 0, len(found))
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func compactSnippet(text string, maxRunes int) string {
	clean := strings.Join(strings.Fields(text), " ")
	runes := []rune(clean)
	if len(runes) <= maxRunes {
		return clean
	}
	return string(runes[:maxRunes-1]) + "…"
}

// zeroBypassTargetContract inspects the BPMN after graph to determine whether
// an all-items-zero gateway branch leads to a valid (non-failure) bypass task,
// and extracts any newly introduced uppercase status tokens on that target.
func zeroBypassTargetContract(pc *domain.ProcessContext) (bool, map[string]bool) {
	if pc == nil || pc.After == nil {
		return false, nil
	}
	introduced := introducedProcessUpperTokens(pc)
	validBypass := false
	expected := map[string]bool{}
	for _, f := range pc.After.Flows {
		if f == nil {
			continue
		}
		src := pc.After.Elements[f.SourceRef]
		if src == nil || !strings.HasSuffix(src.Type, "Gateway") {
			continue
		}
		if !everyZeroDifference.MatchString(f.Condition + " " + f.Name) {
			continue
		}
		target := pc.After.Elements[f.TargetRef]
		if target == nil || failureTargetCue.MatchString(target.Name) || failureTargetCue.MatchString(target.ID) || strings.Contains(strings.ToLower(target.Type), "error") {
			continue
		}
		validBypass = true
		for _, m := range upperEnumToken.FindAllStringSubmatch(target.Name+" "+target.Documentation, -1) {
			if introduced[m[1]] {
				expected[m[1]] = true
			}
		}
	}
	return validBypass, expected
}

func sortedMapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var deferredQueueCue = regexp.MustCompile(`(?i)\b(?:antre(?:an)?\s+approval|approval\s+queue)\b`)
var separateStockCue = regexp.MustCompile(`(?i)\b(?:tanpa\s+(?:posting|mutasi)\s+stok|posting\s+stok\s+tetap\s+aksi\s+terpisah|without\s+(?:posting|mutating)\s+stock)\b`)

func hasDeferredApprovalWithoutStockMutation(pc *domain.ProcessContext) bool {
	if pc == nil || pc.After == nil {
		return false
	}
	hasQueue, hasSeparateStock := false, false
	for _, el := range pc.After.Elements {
		if el == nil {
			continue
		}
		text := el.Name + " " + el.Documentation
		if deferredQueueCue.MatchString(text) && strings.Contains(text, "SUBMITTED") {
			hasQueue = true
		}
		if separateStockCue.MatchString(text) {
			hasSeparateStock = true
		}
	}
	return hasQueue && hasSeparateStock
}
