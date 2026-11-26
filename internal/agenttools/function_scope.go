package agenttools

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Attach syntactic owners to source windows without changing target identity.
func (s *Snapshot) withFunctionScopes(q planning.ToolRequest, result planning.ToolResult) planning.ToolResult {
	if path.Ext(q.File) != ".go" || len(result.Candidates) != 1 {
		return result
	}
	content, ok := s.files[q.File]
	if !ok {
		return result
	}
	set := token.NewFileSet()
	tree, err := parser.ParseFile(set, q.File, content, parser.SkipObjectResolution)
	if err != nil {
		return result
	}
	lines := map[int]bool{}
	for _, e := range result.Candidates[0].Evidence {
		if e.Type != "SOURCE_TOOL" || !strings.HasPrefix(e.Location, q.File+":") {
			continue
		}
		var line int
		fmt.Sscanf(strings.TrimPrefix(e.Location, q.File+":"), "%d", &line)
		if line > 0 {
			lines[line] = true
		}
	}
	var scopes []string
	var soleOwner string
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start, end := set.Position(fn.Pos()).Line, set.Position(fn.End()).Line
		matched := false
		for line := range lines {
			if line >= start && line <= end {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver := fn.Recv.List[0].Type
			if ptr, ok := receiver.(*ast.StarExpr); ok {
				receiver = ptr.X
			}
			if id, ok := receiver.(*ast.Ident); ok {
				name = id.Name + "." + name
			}
		}
		if len(scopes) == 0 {
			soleOwner = name
		} else {
			soleOwner = ""
		}
		if len(name) <= 120 && len(scopes) < 4 {
			scopes = append(scopes, fmt.Sprintf("%s lines %d-%d", name, start, end))
		}
	}
	if len(scopes) > 0 {
		result.Note += " Go AST source-window owners: " + strings.Join(scopes, "; ") + ". These are syntactic boundaries, not inferred dependencies; other excerpts may belong to different functions."
		if len(result.Candidates[0].Evidence) < 8 {
			text := "Go AST owners of inspected source spans: " + strings.Join(scopes, "; ") + ". Not a dependency proof."
			if len([]rune(text)) <= 240 {
				e := planning.EvidenceItem{Type: "AST_SOURCE_SCOPE", Location: q.File, Excerpt: text}
				e.ID = digest(e)[:24]
				result.Candidates[0].Evidence = append([]planning.EvidenceItem{e}, result.Candidates[0].Evidence...)
			}
		}
	}
		if q.Name == "inspect_source" && soleOwner != "" {
		candidate := &result.Candidates[0]
		candidate.Evidence = enrichDispatchTransactionWindow(q.File, content, tree, set, soleOwner, candidate.Evidence)
		facts := s.withDeclarationFacts(planning.ToolResult{Candidates: []planning.Candidate{{File: candidate.File, Symbol: soleOwner}}})
		if evidence := facts.Candidates[0].Evidence; len(evidence) > 0 && len(candidate.Evidence) > 0 {
			ordered := []planning.EvidenceItem{candidate.Evidence[0], evidence[0]}
			candidate.Evidence = append(ordered, candidate.Evidence[1:min(7, len(candidate.Evidence))]...)
		}
	}
	return result
}

// enrichDispatchTransactionWindow trims trailing lines from a subsequent switch
// case when inspecting a single Go action branch and appends key enclosing
// transaction lines (lock/guard/trace before switch, persistence/audit/commit
// after switch) within the bounded 8-item evidence cap.
func enrichDispatchTransactionWindow(file, content string, tree *ast.File, set *token.FileSet, owner string, evidence []planning.EvidenceItem) []planning.EvidenceItem {
	fileLines := strings.Split(content, "\n")
	var fn *ast.FuncDecl
	for _, decl := range tree.Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || f.Body == nil {
			continue
		}
		name := f.Name.Name
		if f.Recv != nil && len(f.Recv.List) > 0 {
			recv := f.Recv.List[0].Type
			if ptr, ok := recv.(*ast.StarExpr); ok {
				recv = ptr.X
			}
			if id, ok := recv.(*ast.Ident); ok {
				name = id.Name + "." + name
			}
		}
		if name == owner {
			fn = f
			break
		}
	}
	if fn == nil {
		return evidence
	}
	firstCaseLine := 0
	for _, ev := range evidence {
		if ev.Type != "SOURCE_TOOL" || !strings.HasPrefix(ev.Location, file+":") {
			continue
		}
		trimmed := strings.TrimSpace(ev.Excerpt)
		if strings.HasPrefix(trimmed, "case ") {
			fmt.Sscanf(strings.TrimPrefix(ev.Location, file+":"), "%d", &firstCaseLine)
			break
		}
	}
	if firstCaseLine == 0 {
		return evidence
	}
	// Find the enclosing switch statement covering firstCaseLine.
	var swStart, swEnd int
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		sLine, eLine := set.Position(sw.Pos()).Line, set.Position(sw.End()).Line
		if firstCaseLine >= sLine && firstCaseLine <= eLine {
			swStart, swEnd = sLine, eLine
			return false
		}
		return true
	})
	if swStart == 0 {
		return evidence
	}
	// Keep AST headers and lines belonging only to the matched case branch.
	var kept []planning.EvidenceItem
	seenLine := map[int]bool{}
	hitNextCase := false
	for _, ev := range evidence {
		if ev.Type != "SOURCE_TOOL" || !strings.HasPrefix(ev.Location, file+":") {
			kept = append(kept, ev)
			continue
		}
		var ln int
		fmt.Sscanf(strings.TrimPrefix(ev.Location, file+":"), "%d", &ln)
		trimmed := strings.TrimSpace(ev.Excerpt)
		if ln > firstCaseLine && (strings.HasPrefix(trimmed, "case ") || strings.HasPrefix(trimmed, "default:")) {
			hitNextCase = true
		}
		if hitNextCase {
			continue
		}
		kept = append(kept, ev)
		seenLine[ln] = true
	}
	fnStart, fnEnd := set.Position(fn.Pos()).Line, set.Position(fn.End()).Line
	addLine := func(ln int) {
		if ln < 1 || ln > len(fileLines) || seenLine[ln] || len(kept) >= 7 {
			return
		}
		text := strings.TrimSpace(fileLines[ln-1])
		if text == "" {
			return
		}
		runes := []rune(text)
		if len(runes) > 240 {
			runes = runes[:240]
		}
		ev := planning.EvidenceItem{Type: "SOURCE_TOOL", Location: fmt.Sprintf("%s:%d", file, ln), Excerpt: string(runes)}
		ev.ID = digest(ev)[:24]
		kept = append(kept, ev)
		seenLine[ln] = true
	}
	// Add key pre-switch lock/version/trace line(s) from the same function.
	for ln := fnStart + 1; ln < swStart && len(kept) < 7; ln++ {
		low := strings.ToLower(fileLines[ln-1])
		if strings.Contains(low, "for update") || strings.Contains(low, "trace :=") {
			addLine(ln)
		}
	}
	// Add key post-switch persistence/audit/commit lines from the same function.
	for ln := swEnd + 1; ln <= fnEnd && len(kept) < 7; ln++ {
		low := strings.ToLower(fileLines[ln-1])
		if strings.Contains(low, "update ") || strings.Contains(low, "audit(") || strings.Contains(low, ".commit(") {
			addLine(ln)
		}
	}
	return kept
}
