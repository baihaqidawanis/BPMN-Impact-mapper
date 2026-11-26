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
var clauseBranchSplit = regexp.MustCompile(`(?i)(?:[.\n]+|\b(?:otherwise|whereas)\b)`)

var nonStatusUpperTokens = map[string]bool{
	"HTTP": true, "JSON": true, "SQL": true, "API": true, "BPMN": true,
	"UUID": true, "CRUD": true, "NULL": true, "TRUE": true, "FALSE": true,
	"XML": true, "URL": true, "URI": true, "DTO": true, "DDL": true,
	"ERR": true, "EOF": true, "NIL": true, "OK": true,
}

var prematureSubmitStockMutation = regexp.MustCompile(`(?i)\b(?:(?:execute|perform|trigger|run)\s+(?:the\s+)?(?:\w+\s+){0,4}(?:stock|inventory|ledger|balance)\s+(?:posting|update(?:\s+query)?|mutation)|(?:with|including|plus|and)\s+(?:standard\s+|normal\s+|existing\s+)?(?:stock|inventory|ledger|balance)\s+(?:posting|mutation|movements?)|(?:stock|inventory|ledger|balance)\s+(?:posting|mutation)\s+(?:is\s+)?(?:queued|scheduled|triggered|initiated)|(?:queue|schedule|initiate)s?\s+(?:the\s+)?(?:stock|inventory|ledger|balance)\s+(?:posting|mutation)|posts?\s+(?:stock|ledger|balance)\b|insert(?:s|ing)?\s+into\s+` + "`" + `?[\w]*(?:balance|movement|ledger)[\w]*` + "`" + `?` + `|[\w]*(?:balance|movement|ledger)[\w]*\s+rows?\s+(?:are\s+)?created|audit(?:s|\s+records?)?\s+['"` + "`" + `"]?(?:APPROVED|POSTED)['"` + "`" + `"]?)\b`)
var postDeferralCue = regexp.MustCompile(`(?i)\b(?:(?:is\s+|are\s+|remains?\s+)?deferred|remains?\s+(?:a\s+)?separate|does\s+not\s+(?:happen|occur|run|execute)|is\s+(?:skipped|omitted|not\s+(?:executed|performed|triggered))|only\s+(?:after|on)\s+approval|until\s+(?:after\s+)?approval|pending\s+approval|separate\s+post-?approval|without\s+(?:posting|mutating))\b`)
var repurposedApprovalTest = regexp.MustCompile(`(?i)\b(?:update|modify|change)\s+(?:the\s+)?(?:\w+\s+){0,3}` + "`" + `?Test\w*Approval\w*` + "`" + `?[^.;\n]{0,200}\b(?:zero[- ]delta|bypass|rather\s+than\s+['"` + "`" + `"]?[A-Z][A-Z0-9_]{2,})\b`)
var preservedApprovalTestCue = regexp.MustCompile(`(?i)\b(?:preserv(?:e|es|ing)\s+(?:its\s+|the\s+)?(?:original|existing)|keep(?:s|ing)?\s+(?:its\s+|the\s+)?(?:original|existing)|retain(?:s|ing)?\s+(?:its\s+|the\s+)?(?:original\s+|existing\s+)?(?:assertion|check|behavior|coverage)|unchanged|separate\s+(?:test|subtest|function)|add\s+(?:a\s+|new\s+)?(?:separate\s+)?test|does\s+not\s+(?:conflict|break|regress|interfere)|without\s+(?:conflicting|breaking|interfering|altering))\b`)
var opposingNonZeroTestCue = regexp.MustCompile(`(?i)\b(?:oppos(?:ing|ite)|cancel(?:l?ing)?\s+(?:out|sum|deltas?)|\+\s*\d+[^.\n]{0,30}-\s*\d+|\+delta[^.\n]{0,20}-delta|sum(?:s|ming)?\s+to\s+(?:zero|0))\b`)
var opposingNonZeroSetupCue = regexp.MustCompile(`(?i)\b(?:\+\s*\d+[^.\n]{0,30}-\s*\d+|\+delta[^.\n]{0,20}-delta|individual\s+lines?\s+(?:differ|are\s+non-?zero)|oppos(?:ing|ite)\s+(?:non-?zero|deltas?|differences?))\b`)
var allZeroLabelCue = regexp.MustCompile(`(?i)\b(?:all[- ]zero|every\s+line[^.;\n]{0,30}(?:zero|0|equals?\s+the\s+snapshot))\b`)

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
	zeroBypassOK, expectedStatuses, _, normalStatuses := zeroBypassTargetContract(pc)
	bypassLabel := joinTokensOr(expectedStatuses, "the bypass status")
	normalLabel := joinTokensOr(normalStatuses, "the normal approval status")
	for _, edits := range [][]string{{plan.Summary, plan.TargetBehavior}, plan.ProposedChanges, plan.Steps, plan.AcceptanceChecks} {
		for _, edit := range edits {
			if strings.TrimSpace(edit) == "" {
				continue
			}
			mentionsBypass := mentionsAnyToken(edit, expectedStatuses)
			mentionsNormal := mentionsAnyToken(edit, normalStatuses)
			isOpposingTestSetup := opposingNonZeroSetupCue.MatchString(edit) && !mentionsBypass
			if perItem && (matchUnnegated(signedSumEdit, edit) || matchUnnegated(zeroSumBranch, edit) || (!isOpposingTestSetup && matchUnnegated(deltasSumToZero, edit)) || matchUnnegated(explicitAggregateBranch, edit)) {
				return fmt.Errorf("%w: %q substitutes an aggregate for an all-items-zero branch (offending text: %q); rewrite this item to compare each line individually and preserve the normal path for opposite nonzero differences", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if nonZeroContinues && (matchUnnegatedBridge(nonZeroRejectionClause, edit) || matchUnnegatedBridge(rejectNonZeroDirect, edit)) {
				return fmt.Errorf("%w: %q rejects the valid non-zero difference branch (offending text: %q); in the after process, non-zero differences must continue along the normal path while precondition failures remain on their own gateway", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160))
			}
			if deferredApproval && (nonZeroBranchCue.MatchString(edit) || mentionsNormal) && matchUnnegatedActive(prematureSubmitStockMutation, edit) {
				return fmt.Errorf("%w: %q attributes downstream posting or approval mutation to the non-zero %s submit branch (offending text: %q); during submit, non-zero differences only transition to %s for approval while posting remains a separate post-approval operation", ErrAgentPlanIncomplete, plan.File, normalLabel, compactSnippet(edit, 160), normalLabel)
			}
			if zeroBypassOK && opposingNonZeroSetupCue.MatchString(edit) && (mentionsBypass || allZeroLabelCue.MatchString(edit)) && !mentionsNormal {
				return fmt.Errorf("%w: %q routes opposing non-zero lines (+2 and -2) to the all-zero bypass outcome (offending text: %q); use truly equal lines (e.g., counted 5 vs snapshot 5 on every line) for the all-zero %s case, and keep opposing non-zero lines (+2 and -2) on the normal %s path", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160), bypassLabel, normalLabel)
			}
			if zeroBypassOK && (plan.Layer == "test" || strings.HasSuffix(strings.ToLower(plan.File), "_test.go")) && repurposedApprovalTest.MatchString(edit) && !preservedApprovalTestCue.MatchString(edit) {
				return fmt.Errorf("%w: %q repurposes an existing Approval test into a zero-difference bypass test (offending text: %q); preserve existing approval tests reaching %s and add a separate test function for zero-difference bypass and opposite-nonzero (+delta/-delta) cases", ErrAgentPlanIncomplete, plan.File, compactSnippet(edit, 160), normalLabel)
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
					return fmt.Errorf("%w: %q test plan for an all-items-zero branch must keep existing approval test assertions intact and add separate test cases for: (1) all lines truly equal (counted == snapshot, 0 delta) -> %s, (2) opposing non-zero lines (+2 on one line and -2 on another) -> %s (normal approval path, not %s), and (3) stale version / invalid precondition state -> 409 Conflict", ErrAgentPlanIncomplete, plan.File, bypassLabel, normalLabel, bypassLabel)
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
// and dynamically extracts status tokens and element IDs for both the bypass
// and normal continuation targets.
func zeroBypassTargetContract(pc *domain.ProcessContext) (bool, map[string]bool, []string, map[string]bool) {
	if pc == nil || pc.After == nil {
		return false, nil, nil, nil
	}
	introduced := introducedProcessUpperTokens(pc)
	validBypass := false
	expected := map[string]bool{}
	normal := map[string]bool{}
	seenTarget := map[string]bool{}
	var bypassTargets []string
	for _, f := range pc.After.Flows {
		if f == nil {
			continue
		}
		src := pc.After.Elements[f.SourceRef]
		if src == nil || !strings.HasSuffix(src.Type, "Gateway") {
			continue
		}
		flowText := f.Condition + " " + f.Name
		target := pc.After.Elements[f.TargetRef]
		if target == nil || failureTargetCue.MatchString(target.Name) || failureTargetCue.MatchString(target.ID) || strings.Contains(strings.ToLower(target.Type), "error") {
			continue
		}
		if everyZeroDifference.MatchString(flowText) {
			validBypass = true
			if id := strings.ToLower(strings.TrimSpace(target.ID)); id != "" && !seenTarget[id] {
				seenTarget[id] = true
				bypassTargets = append(bypassTargets, id)
			}
			for _, m := range upperEnumToken.FindAllStringSubmatch(target.Name+" "+target.Documentation, -1) {
				if introduced[m[1]] {
					expected[m[1]] = true
				}
			}
		} else if nonZeroBranchCue.MatchString(flowText) {
			for _, m := range upperEnumToken.FindAllStringSubmatch(target.Name+" "+target.Documentation, -1) {
				if !nonStatusUpperTokens[m[1]] {
					normal[m[1]] = true
				}
			}
		}
	}
	sort.Strings(bypassTargets)
	return validBypass, expected, bypassTargets, normal
}

func joinTokensOr(values map[string]bool, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return strings.Join(sortedMapKeys(values), "/")
}

func mentionsAnyToken(text string, tokens map[string]bool) bool {
	if len(tokens) == 0 {
		return false
	}
	upper := strings.ToUpper(text)
	for tok := range tokens {
		if strings.Contains(upper, tok) {
			return true
		}
	}
	return false
}

func mentionsAnyTarget(text string, targets []string) bool {
	if len(targets) == 0 {
		return false
	}
	lower := strings.ToLower(text)
	for _, id := range targets {
		if strings.Contains(lower, id) {
			return true
		}
	}
	return false
}

func sortedMapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var deferredQueueCue = regexp.MustCompile(`(?i)\b(?:antre(?:an)?\s+approval|approval\s+queue|pending\s+approval)\b`)
var separateStockCue = regexp.MustCompile(`(?i)\b(?:tanpa\s+(?:posting|mutasi)\s+\w+|(?:posting|mutasi)\s+\w+\s+tetap\s+aksi\s+terpisah|without\s+(?:posting|mutating)\s+\w+|posting\s+(?:is\s+)?deferred\s+until\s+(?:after\s+)?approval)\b`)

func matchUnnegatedActive(re *regexp.Regexp, text string) bool {
	for _, loc := range re.FindAllStringIndex(text, -1) {
		start, end := loc[0], loc[1]
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
		suffixEnd := end + 52
		if suffixEnd > len(text) {
			suffixEnd = len(text)
		}
		suffix := text[end:suffixEnd]
		if idx := strings.IndexAny(suffix, ".;\n"); idx >= 0 {
			suffix = suffix[:idx]
		}
		if postDeferralCue.MatchString(suffix) {
			continue
		}
		return true
	}
	return false
}

func hasDeferredApprovalWithoutStockMutation(pc *domain.ProcessContext) bool {
	if pc == nil || pc.After == nil {
		return false
	}
	hasQueue, hasSeparateEffect := false, false
	for _, el := range pc.After.Elements {
		if el == nil {
			continue
		}
		text := el.Name + " " + el.Documentation
		if deferredQueueCue.MatchString(text) {
			hasQueue = true
		}
		if separateStockCue.MatchString(text) {
			hasSeparateEffect = true
		}
	}
	return hasQueue && hasSeparateEffect
}
