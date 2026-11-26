package planning

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

var numberedSQLMigration = regexp.MustCompile(`^[0-9]+[^/]*\.(up|down)\.sql$`)
var numberedMigrationPrefix = regexp.MustCompile(`(?i)\b([0-9]{3,6})_[a-z0-9_]+\.(?:up|down)\.sql\b`)
var migrationStemPattern = regexp.MustCompile(`^[0-9]+_([a-z0-9_]+)\.(?:up|down)\.sql$`)
var sqlTableHeader = regexp.MustCompile(`(?i)\b(?:CREATE|ALTER)\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`)
var sqlCheckInConstraint = regexp.MustCompile(`(?i)\bCHECK\s*\(\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+IN\s*\(([^)]+)\)\s*\)`)
var sqlNonZeroCheck = regexp.MustCompile(`(?i)\bCHECK\s*\(\s*[a-zA-Z_][a-zA-Z0-9_]*\s*(?:<>|!=)\s*0\s*\)`)
var sqlQuantityColumn = regexp.MustCompile(`(?i)\b([a-z][a-z0-9_]*quantity[a-z0-9_]*)\b`)
var requestPayloadDeltaCue = regexp.MustCompile(`(?i)\b(?:line(?:\s+items?)?|deltas?|quantities)\s+(?:in|from)\s+(?:the\s+)?(?:adjustment\s+)?request(?:\s+payload)?\b`)
var sqlQuotedUpperToken = regexp.MustCompile(`'([A-Z][A-Z0-9_]{2,})'`)
var upperEnumToken = regexp.MustCompile(`\b([A-Z][A-Z0-9_]{2,})\b`)
var forwardMigrationMention = regexp.MustCompile(`(?i)\b(?:forward\s+migration|new\s+(?:\w+\s+){0,2}migration|(?:add|create|write|apply|introduce)\s+(?:a\s+|an\s+|the\s+)?(?:new\s+|forward\s+|database\s+|sql\s+|schema\s+)?migration|migration\s+to\s+(?:allow|add|include|update|expand|alter)|alter\s+table)\b`)
var migrationRollbackCue = regexp.MustCompile(`(?i)(?:\.down\.sql\b|\b(?:rollback|down\s+migration|revert(?:ing)?\s+(?:the\s+)?(?:migration|constraint)|downgrade)\b)`)
var rollbackRowStrategyCue = regexp.MustCompile(`(?i)\b(?:existing\s+(?:[A-Z0-9_]+\s+)?(?:rows?|records?|data)|rows?\s+(?:with|using|having|in)\s+['"` + "`" + `"]?[A-Z0-9_]+|remap(?:ping|ped)?|migrat(?:e|ing)\s+back|backfill|update\s+existing|delete\s+existing|clean\s*up|block(?:ing|ed)?\s+rollback|prevent\s+rollback|abort(?:ing)?\s+rollback|abort\s+if|fail\s+if|raise\s+exception|before\s+restor(?:ing|e)|prior\s+to\s+restor(?:ing|e))\b`)

var genericMigrationStems = map[string]bool{
	"init": true, "initial": true, "schema": true, "core": true, "base": true,
	"main": true, "operations": true, "common": true, "shared": true, "app": true,
}

// IsHistoricalMigration identifies an existing numbered migration in the inspected snapshot.
func IsHistoricalMigration(file string) bool {
	file = strings.ToLower(strings.ReplaceAll(file, "\\", "/"))
	return strings.Contains("/"+file, "/migrations/") && numberedSQLMigration.MatchString(path.Base(file))
}

// A numbered migration in the analyzed snapshot may already be applied.
// Describe new DDL in the storage plan while preserving migration history.
func validateMigrationEdit(p FilePlan) error {
	if (p.Action != domain.ActionModify && p.Action != domain.ActionRemove) || !IsHistoricalMigration(p.File) {
		return nil
	}
	return fmt.Errorf("%w: existing migration %q is schema evidence; do not edit or remove it. Use NO_CHANGE for this target and describe a new forward migration in the affected storage proposal if the source constraint requires one", ErrAgentPlanIncomplete, p.File)
}

type sqlCheckConstraint struct {
	File    string
	Table   string
	Column  string
	Allowed map[string]bool
}

func evidenceLineKey(file, location string) string {
	rest := strings.TrimPrefix(location, file+":")
	if idx := strings.IndexByte(rest, ':'); idx >= 0 {
		rest = rest[:idx]
	}
	if _, err := strconv.Atoi(rest); err == nil {
		return file + "|" + rest
	}
	return ""
}

func extractSQLCheckConstraints(bundle EvidenceBundle) []sqlCheckConstraint {
	return extractSQLCheckConstraintsWithRelated(bundle, nil)
}

func extractSQLCheckConstraintsWithRelated(bundle EvidenceBundle, related []ProposalContract) []sqlCheckConstraint {
	seen := map[string]bool{}
	lineTable := map[string]string{}
	for _, change := range bundle.Changes {
		for _, candidate := range change.Candidates {
			lowerFile := strings.ToLower(candidate.File)
			if !strings.HasSuffix(lowerFile, ".sql") || strings.HasSuffix(lowerFile, ".down.sql") {
				continue
			}
			for _, ev := range candidate.Evidence {
				if key := evidenceLineKey(candidate.File, ev.Location); key != "" {
					if m := sqlTableHeader.FindStringSubmatch(ev.Excerpt); len(m) == 2 {
						lineTable[key] = strings.ToLower(m[1])
					}
				}
			}
		}
	}
	var out []sqlCheckConstraint
	addExcerpt := func(file, location, excerpt string) {
		tableHeaders := sqlTableHeader.FindAllStringSubmatchIndex(excerpt, -1)
		for _, idx := range sqlCheckInConstraint.FindAllStringSubmatchIndex(excerpt, -1) {
			col := strings.ToLower(excerpt[idx[2]:idx[3]])
			vals := sqlQuotedUpperToken.FindAllStringSubmatch(excerpt[idx[4]:idx[5]], -1)
			if len(vals) < 2 {
				continue
			}
			table := ""
			for _, th := range tableHeaders {
				if th[0] <= idx[0] {
					table = strings.ToLower(excerpt[th[2]:th[3]])
				}
			}
			if table == "" && location != "" {
				if key := evidenceLineKey(file, location); key != "" {
					table = lineTable[key]
				}
			}
			allowed := map[string]bool{}
			keys := make([]string, 0, len(vals))
			for _, v := range vals {
				allowed[v[1]] = true
				keys = append(keys, v[1])
			}
			sort.Strings(keys)
			sig := file + "|" + table + "|" + col + "|" + strings.Join(keys, ",")
			if seen[sig] {
				continue
			}
			seen[sig] = true
			out = append(out, sqlCheckConstraint{File: file, Table: table, Column: col, Allowed: allowed})
		}
	}
	for _, change := range bundle.Changes {
		for _, candidate := range change.Candidates {
			lowerFile := strings.ToLower(candidate.File)
			if !strings.HasSuffix(lowerFile, ".sql") || strings.HasSuffix(lowerFile, ".down.sql") {
				continue
			}
			for _, ev := range candidate.Evidence {
				addExcerpt(candidate.File, ev.Location, ev.Excerpt)
			}
		}
	}
	for _, r := range related {
		lf := strings.ToLower(r.File)
		if !strings.HasSuffix(lf, ".sql") || strings.HasSuffix(lf, ".down.sql") {
			continue
		}
		addExcerpt(r.File, "", r.TargetBehavior+"\n"+strings.Join(r.Assumptions, "\n"))
	}
	return out
}

func introducedProcessUpperTokens(pc *domain.ProcessContext) map[string]bool {
	out := map[string]bool{}
	if pc == nil || pc.After == nil {
		return out
	}
	collect := func(g *domain.Graph) map[string]bool {
		tokens := map[string]bool{}
		if g == nil {
			return tokens
		}
		scan := func(s string) {
			for _, m := range upperEnumToken.FindAllStringSubmatch(s, -1) {
				if !nonStatusUpperTokens[m[1]] {
					tokens[m[1]] = true
				}
			}
		}
		for _, el := range g.Elements {
			if el != nil {
				scan(el.Name)
				scan(el.Documentation)
			}
		}
		for _, f := range g.Flows {
			if f != nil {
				scan(f.Name)
				scan(f.Condition)
			}
		}
		return tokens
	}
	afterTokens := collect(pc.After)
	beforeTokens := collect(pc.Before)
	for tok := range afterTokens {
		if pc.Before == nil || !beforeTokens[tok] {
			out[tok] = true
		}
	}
	return out
}

// migrationPrerequisiteIssues verifies that when inspected SQL evidence contains
// a CHECK(col IN (...)) constraint and a non-test implementation proposal writes
// a newly introduced BPMN value excluded by that CHECK, at least one executable
// proposal specifies a forward migration or constraint update.
func migrationPrerequisiteIssues(bundle EvidenceBundle, plans []FilePlan) []string {
	return migrationPrerequisiteIssuesWithRelated(bundle, plans, nil)
}

func migrationPrerequisiteIssuesWithRelated(bundle EvidenceBundle, plans []FilePlan, related []ProposalContract) []string {
	constraints := extractSQLCheckConstraintsWithRelated(bundle, related)
	introduced := introducedProcessUpperTokens(bundle.ProcessContext)
	maxMigNum, maxMigWidth, existingMigFiles := maxExistingMigrationNumber(bundle, related)
	nonZeroTables, lineTables, lineCols := extractLedgerAndLineTablesWithRelated(bundle, related)
	var issues []string
	reported := map[string]bool{}
	for _, p := range plans {
		if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "test" || p.Layer == "documentation" {
			continue
		}
		if maxMigNum > 0 {
			for _, group := range [][]string{p.ProposedChanges, p.Steps} {
				for _, step := range group {
					if !forwardMigrationMention.MatchString(step) {
						continue
					}
					for _, m := range numberedMigrationPrefix.FindAllStringSubmatch(step, -1) {
						if existingMigFiles[strings.ToLower(m[0])] {
							continue
						}
						if n, err := strconv.Atoi(m[1]); err == nil && n <= maxMigNum {
							key := fmt.Sprintf("mignum|%s|%d", p.File, n)
							if !reported[key] {
								reported[key] = true
								issues = append(issues, fmt.Sprintf("Schema prerequisite invalid: %s proposes migration number %0*d which collides with existing migrations (highest observed: %0*d); use the next unused migration number (%0*d_...) instead.", p.File, maxMigWidth, n, maxMigWidth, maxMigNum, maxMigWidth, maxMigNum+1))
							}
						}
					}
				}
			}
		}
		if len(lineTables) > 0 && hasEveryZeroDifference(bundle.ProcessContext, ChangeInput{}) {
			lowerFile := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
			submod := ""
			if sm := internalSubmodulePath.FindStringSubmatch(lowerFile); len(sm) == 2 {
				submod = sm[1]
			}
			prefixes := collectProcessWordPrefixes(bundle.ProcessContext)
			matchingLine := ""
			for _, lt := range lineTables {
				if submod == "" || identifierMatchesScope(lt, submod, prefixes) {
					matchingLine = lt
					break
				}
			}
			if matchingLine != "" {
				colHint := ""
				if cols := lineCols[matchingLine]; len(cols) >= 2 {
					colHint = fmt.Sprintf(" (%s)", strings.Join(cols, " vs "))
				}
				for _, ledger := range nonZeroTables {
					ledgerPattern := strings.ReplaceAll(regexp.QuoteMeta(ledger), "_", `[_ ]`)
					queryLedger := regexp.MustCompile(`(?i)\b(?:query(?:ing)?|queries|select(?:s|ing)?|read(?:s|ing)?|fetch(?:es|ing)?|retriev(?:e|es|ing)|load(?:s|ing)?|inspect(?:s|ing)?|join(?:s|ing)?|from)\b[^.;\n]{0,50}\b` + ledgerPattern + `\b`)
					for _, group := range [][]string{{p.CurrentBehavior, p.DesignRationale}, p.ProposedChanges, p.Steps} {
						for _, step := range group {
							if !negationCue.MatchString(step) && queryLedger.MatchString(step) {
								key := "ledger|" + p.File + "|" + ledger
								if !reported[key] {
									reported[key] = true
									issues = append(issues, fmt.Sprintf("Schema source invalid: %s queries non-zero-only ledger table %q (CHECK <> 0) to evaluate per-line zero deltas before posting; query each line's quantities from %q%s instead.", p.File, ledger, matchingLine, colHint))
								}
							}
						}
					}
				}
				if p.Layer == "adapter" || strings.Contains(lowerFile, "/postgres/") {
					joinedSteps := strings.Join(append([]string{}, append(p.ProposedChanges, p.Steps...)...), "\n")
					if requestPayloadDeltaCue.MatchString(joinedSteps) && !strings.Contains(strings.ToLower(joinedSteps), matchingLine) {
						key := "payload|" + p.File + "|" + matchingLine
						if !reported[key] {
							reported[key] = true
							issues = append(issues, fmt.Sprintf("Schema source invalid: %s computes line deltas from the request payload instead of querying %q%s; query %q inside the transaction because the store transition method receives scalar identifiers rather than line items.", p.File, matchingLine, colHint, matchingLine))
						}
					}
				}
			}
		}
	}
	if len(constraints) == 0 || len(introduced) == 0 {
		sort.Strings(issues)
		return issues
	}
	for _, c := range constraints {
		for _, p := range plans {
			if !constraintAppliesToPlan(bundle.ProcessContext, c, p) {
				continue
			}
			for tok := range introduced {
				if c.Allowed[tok] || !writesEnumValue(p, c.Column, tok) {
					continue
				}
				if mutatesHistoricalMigrationInPlan(p, c.File) {
					key := "mut|" + c.File + "|" + c.Column + "|" + tok + "|" + p.File
					if !reported[key] {
						reported[key] = true
						issues = append(issues, fmt.Sprintf("Schema prerequisite invalid: %s proposes editing historical migration %s directly; keep %s unchanged and add a step in %s creating a new forward migration (ALTER TABLE %s ... CHECK (%s IN (...))) to allow %q.", p.File, c.File, c.File, p.File, fallbackTable(c.Table), c.Column, tok))
					}
					continue
				}
				if hasForwardMigrationFor(plans, c, tok) {
					if c.Table != "" && !forwardMigrationNamesTable(plans, c, tok) {
						key := "tbl|" + c.File + "|" + c.Table + "|" + tok + "|" + p.File
						if !reported[key] {
							reported[key] = true
							issues = append(issues, fmt.Sprintf("Schema prerequisite invalid: forward migration in %s must target the exact SQL table %q (column %s) from %s to allow %q; do not abbreviate or guess the table name.", p.File, c.Table, c.Column, c.File, tok))
						}
					}
					if !hasForwardMigrationRollback(plans) {
						key := "rb|" + c.File + "|" + c.Column + "|" + tok + "|" + p.File
						if !reported[key] {
							reported[key] = true
							issues = append(issues, fmt.Sprintf("Schema prerequisite incomplete: forward migration in %s adds %q to %s.%s without a rollback (.down.sql) strategy; add a matching .down.sql rollback plan explaining how existing %q rows are handled before restoring the previous CHECK constraint.", p.File, tok, fallbackTable(c.Table), c.Column, tok))
						}
					}
					continue
				}
				key := c.File + "|" + c.Column + "|" + tok + "|" + p.File
				if reported[key] {
					continue
				}
				reported[key] = true
				tableHint := ""
				if c.Table != "" {
					tableHint = fmt.Sprintf(" on table %q", c.Table)
				}
				issues = append(issues, fmt.Sprintf("Schema prerequisite missing: %s restricts %s%s via CHECK(...) without %q, yet %s proposes using %q without specifying a new forward migration to update the constraint; keep %s as NO_CHANGE and add a step in %s creating a new forward migration updating the %s CHECK constraint%s to allow %q plus a matching .down.sql rollback strategy for existing %q rows (do not edit %s).", c.File, c.Column, tableHint, tok, p.File, tok, c.File, p.File, c.Column, tableHint, tok, tok, c.File))
			}
		}
	}
	sort.Strings(issues)
	return issues
}

func fallbackTable(table string) string {
	if table != "" {
		return table
	}
	return "<table>"
}

func maxExistingMigrationNumber(bundle EvidenceBundle, related []ProposalContract) (int, int, map[string]bool) {
	maxNum, width := 0, 6
	existing := map[string]bool{}
	check := func(file string) {
		base := strings.ToLower(path.Base(strings.ReplaceAll(file, "\\", "/")))
		if m := numberedMigrationPrefix.FindStringSubmatch(base); len(m) == 2 {
			existing[base] = true
			if n, err := strconv.Atoi(m[1]); err == nil && n > maxNum {
				maxNum = n
				width = len(m[1])
			}
		}
	}
	for _, change := range bundle.Changes {
		for _, c := range change.Candidates {
			check(c.File)
		}
	}
	for _, r := range related {
		check(r.File)
	}
	return maxNum, width, existing
}

func evidenceLineNumber(file, location string) (int, bool) {
	rest := strings.TrimPrefix(location, file+":")
	if idx := strings.IndexByte(rest, ':'); idx >= 0 {
		rest = rest[:idx]
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

func extractLedgerAndLineTables(bundle EvidenceBundle) (nonZeroTables []string, lineTables []string, lineCols map[string][]string) {
	return extractLedgerAndLineTablesWithRelated(bundle, nil)
}

func extractLedgerAndLineTablesWithRelated(bundle EvidenceBundle, related []ProposalContract) (nonZeroTables []string, lineTables []string, lineCols map[string][]string) {
	seenNZ, seenLine := map[string]bool{}, map[string]bool{}
	lineCols = map[string][]string{}
	type headerAtLine struct {
		line  int
		table string
	}
	addQtyCols := func(ownerTbl, excerpt string) {
		if !strings.HasSuffix(ownerTbl, "_lines") {
			return
		}
		for _, qm := range sqlQuantityColumn.FindAllStringSubmatch(excerpt, -1) {
			col := strings.ToLower(qm[1])
			dup := false
			for _, old := range lineCols[ownerTbl] {
				if old == col {
					dup = true
					break
				}
			}
			if !dup {
				lineCols[ownerTbl] = append(lineCols[ownerTbl], col)
			}
		}
	}
	for _, change := range bundle.Changes {
		for _, c := range change.Candidates {
			lf := strings.ToLower(c.File)
			if !strings.HasSuffix(lf, ".sql") || strings.HasSuffix(lf, ".down.sql") {
				continue
			}
			var headers []headerAtLine
			for _, ev := range c.Evidence {
				ln, hasLn := evidenceLineNumber(c.File, ev.Location)
				for _, m := range sqlTableHeader.FindAllStringSubmatch(ev.Excerpt, -1) {
					tbl := strings.ToLower(m[1])
					if hasLn {
						headers = append(headers, headerAtLine{line: ln, table: tbl})
					}
					if strings.HasSuffix(tbl, "_lines") && !seenLine[tbl] {
						seenLine[tbl] = true
						lineTables = append(lineTables, tbl)
					}
				}
			}
			nearestTable := func(ln int) string {
				best, bestLine := "", -1
				for _, h := range headers {
					if h.line <= ln && ln-h.line <= 6 && h.line >= bestLine {
						best, bestLine = h.table, h.line
					}
				}
				return best
			}
			for _, ev := range c.Evidence {
				ln, hasLn := evidenceLineNumber(c.File, ev.Location)
				ownerTbl := ""
				if m := sqlTableHeader.FindStringSubmatch(ev.Excerpt); len(m) == 2 {
					ownerTbl = strings.ToLower(m[1])
				} else if hasLn {
					ownerTbl = nearestTable(ln)
				}
				addQtyCols(ownerTbl, ev.Excerpt)
				if sqlNonZeroCheck.MatchString(ev.Excerpt) && ownerTbl != "" && !seenNZ[ownerTbl] {
					seenNZ[ownerTbl] = true
					nonZeroTables = append(nonZeroTables, ownerTbl)
				}
			}
		}
	}
	for _, r := range related {
		lf := strings.ToLower(r.File)
		if !strings.HasSuffix(lf, ".sql") || strings.HasSuffix(lf, ".down.sql") {
			continue
		}
		for _, text := range append([]string{r.TargetBehavior}, r.Assumptions...) {
			ownerTbl := ""
			if m := sqlTableHeader.FindStringSubmatch(text); len(m) == 2 {
				ownerTbl = strings.ToLower(m[1])
				if strings.HasSuffix(ownerTbl, "_lines") && !seenLine[ownerTbl] {
					seenLine[ownerTbl] = true
					lineTables = append(lineTables, ownerTbl)
				}
			}
			addQtyCols(ownerTbl, text)
			if sqlNonZeroCheck.MatchString(text) && ownerTbl != "" && !seenNZ[ownerTbl] {
				seenNZ[ownerTbl] = true
				nonZeroTables = append(nonZeroTables, ownerTbl)
			}
		}
	}
	sort.Strings(nonZeroTables)
	sort.Strings(lineTables)
	return nonZeroTables, lineTables, lineCols
}

func schemaFactContractsForPlan(pc *domain.ProcessContext, p FilePlan, facts []SchemaFact) []ProposalContract {
	if len(facts) == 0 {
		return nil
	}
	submod := ""
	lf := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
	if m := internalSubmodulePath.FindStringSubmatch(lf); len(m) == 2 {
		submod = m[1]
	}
	prefixes := collectProcessWordPrefixes(pc)
	byFile := map[string][]string{}
	var order []string
	for _, f := range facts {
		m := sqlTableHeader.FindStringSubmatch(f.Excerpt)
		if len(m) != 2 {
			continue
		}
		tbl := strings.ToLower(m[1])
		keep := sqlNonZeroCheck.MatchString(f.Excerpt) || submod == "" || identifierMatchesScope(tbl, submod, prefixes)
		if !keep {
			continue
		}
		if _, ok := byFile[f.File]; !ok {
			order = append(order, f.File)
		}
		byFile[f.File] = append(byFile[f.File], f.Excerpt)
	}
	var out []ProposalContract
	for _, file := range order {
		out = append(out, ProposalContract{
			File:           file,
			Action:         domain.ActionNoAction,
			TargetBehavior: "Read-only historical schema facts.",
			Assumptions:    byFile[file],
		})
	}
	return out
}

func rollbackRowStrategyIssuesWithRelated(bundle EvidenceBundle, plans []FilePlan, related []ProposalContract) []string {
	constraints := extractSQLCheckConstraintsWithRelated(bundle, related)
	introduced := introducedProcessUpperTokens(bundle.ProcessContext)
	if len(constraints) == 0 || len(introduced) == 0 {
		return nil
	}
	hasRowStrategy := false
	for _, p := range plans {
		for _, group := range [][]string{p.ProposedChanges, p.Steps, p.Assumptions} {
			for _, item := range group {
				if rollbackRowStrategyCue.MatchString(item) {
					hasRowStrategy = true
				}
			}
		}
	}
	if hasRowStrategy {
		return nil
	}
	var issues []string
	seen := map[string]bool{}
	for _, c := range constraints {
		for _, p := range plans {
			if !constraintAppliesToPlan(bundle.ProcessContext, c, p) {
				continue
			}
			for tok := range introduced {
				if c.Allowed[tok] || !writesEnumValue(p, c.Column, tok) || !hasForwardMigrationFor(plans, c, tok) {
					continue
				}
				key := c.Table + "|" + c.Column + "|" + tok + "|" + p.File
				if seen[key] {
					continue
				}
				seen[key] = true
				issues = append(issues, fmt.Sprintf("Rollback strategy incomplete: forward migration in %s adds %q to %s.%s without explaining how existing %q rows are handled on rollback; update the rollback (.down.sql) step to specify how existing %q rows are handled (e.g., remapped, deleted, or blocking rollback if any exist) before restoring the prior CHECK constraint.", p.File, tok, fallbackTable(c.Table), c.Column, tok, tok))
			}
		}
	}
	sort.Strings(issues)
	return issues
}

func hasForwardMigrationRollback(plans []FilePlan) bool {
	for _, p := range plans {
		if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "test" || p.Layer == "documentation" {
			continue
		}
		for _, group := range [][]string{p.ProposedChanges, p.Steps, p.Assumptions} {
			for _, item := range group {
				if migrationRollbackCue.MatchString(item) {
					return true
				}
			}
		}
	}
	return false
}

func forwardMigrationNamesTable(plans []FilePlan, constraint sqlCheckConstraint, value string) bool {
	tbl := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(constraint.Table) + `\b`)
	column := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(constraint.Column) + `\b`)
	token := regexp.MustCompile(`\b` + regexp.QuoteMeta(value) + `\b`)
	for _, p := range plans {
		if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "test" || p.Layer == "documentation" {
			continue
		}
		for _, group := range [][]string{p.ProposedChanges, p.Steps} {
			for _, step := range group {
				if !mutatesHistoricalMigrationProse(step, constraint.File) && forwardMigrationMention.MatchString(step) && column.MatchString(step) && token.MatchString(step) && tbl.MatchString(step) {
					return true
				}
			}
		}
	}
	return false
}

func constraintAppliesToPlan(pc *domain.ProcessContext, c sqlCheckConstraint, p FilePlan) bool {
	lowerFile := strings.ToLower(strings.ReplaceAll(p.File, "\\", "/"))
	if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "test" || p.Layer == "documentation" || p.Layer == "domain" || p.Layer == "frontend" || strings.HasSuffix(lowerFile, ".sql") || strings.Contains("/"+lowerFile, "/domain/") {
		return false
	}
	submod := ""
	if m := internalSubmodulePath.FindStringSubmatch(lowerFile); len(m) == 2 {
		submod = m[1]
	}
	if submod == "" {
		return true
	}
	prefixes := collectProcessWordPrefixes(pc)
	if c.Table != "" {
		return identifierMatchesScope(c.Table, submod, prefixes)
	}
	if m := migrationStemPattern.FindStringSubmatch(strings.ToLower(path.Base(c.File))); len(m) == 2 && !genericMigrationStems[m[1]] {
		return identifierMatchesScope(m[1], submod, prefixes)
	}
	return true
}

func identifierMatchesScope(ident, submod string, prefixes map[string]bool) bool {
	for _, part := range strings.Split(strings.ToLower(ident), "_") {
		if len(part) < 4 {
			continue
		}
		if strings.Contains(submod, part[:4]) || (len(submod) >= 4 && strings.Contains(part, submod[:4])) || matchesProcessPrefix(part, prefixes) {
			return true
		}
	}
	return false
}

func mutatesHistoricalMigrationInPlan(p FilePlan, histFile string) bool {
	for _, group := range [][]string{p.ProposedChanges, p.Steps} {
		for _, step := range group {
			if mutatesHistoricalMigrationProse(step, histFile) {
				return true
			}
		}
	}
	return false
}

func mutatesHistoricalMigrationProse(step, histFile string) bool {
	base := path.Base(histFile)
	re := regexp.MustCompile(`(?i)\b(?:update|modify|edit|change|alter|rewrite|append\s+to)\s+[` + "`" + `'"]?(?:` + regexp.QuoteMeta(histFile) + `|` + regexp.QuoteMeta(base) + `)\b`)
	return re.MatchString(step)
}

// SQL excerpts can show several unrelated enums in one migration. Require
// the proposal to assign the new value to the actual constrained column.
func writesEnumValue(p FilePlan, column, value string) bool {
	assignment := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(column) + `\s*(?:=|to|as)\s*[^\w\s]?` + regexp.QuoteMeta(value) + `\b`)
	if assignment.MatchString(p.TargetBehavior) {
		return true
	}
	for _, group := range [][]string{p.ProposedChanges, p.Steps} {
		for _, step := range group {
			if !forwardMigrationMention.MatchString(step) && assignment.MatchString(step) {
				return true
			}
		}
	}
	return false
}

func hasForwardMigrationFor(plans []FilePlan, constraint sqlCheckConstraint, value string) bool {
	column := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(constraint.Column) + `\b`)
	token := regexp.MustCompile(`\b` + regexp.QuoteMeta(value) + `\b`)
	for _, p := range plans {
		if (p.Action != domain.ActionModify && p.Action != domain.ActionAdd) || p.Layer == "test" || p.Layer == "documentation" {
			continue
		}
		newSQL := p.Action == domain.ActionAdd && strings.HasSuffix(strings.ToLower(p.File), ".up.sql") && !strings.EqualFold(p.File, constraint.File)
		for _, group := range [][]string{p.ProposedChanges, p.Steps} {
			for _, step := range group {
				if !mutatesHistoricalMigrationProse(step, constraint.File) &&
					(newSQL || forwardMigrationMention.MatchString(step)) && column.MatchString(step) && token.MatchString(step) {
					return true
				}
			}
		}
	}
	return false
}

func validateMigrationPrerequisites(bundle EvidenceBundle, plans []FilePlan) error {
	return validateMigrationPrerequisitesWithRelated(bundle, plans, nil)
}

func validateMigrationPrerequisitesWithRelated(bundle EvidenceBundle, plans []FilePlan, related []ProposalContract) error {
	issues := migrationPrerequisiteIssuesWithRelated(bundle, plans, related)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrAgentPlanIncomplete, issues[0])
}
