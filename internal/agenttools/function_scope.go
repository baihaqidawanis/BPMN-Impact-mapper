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
		facts := s.withDeclarationFacts(planning.ToolResult{Candidates: []planning.Candidate{{File: candidate.File, Symbol: soleOwner}}})
		if evidence := facts.Candidates[0].Evidence; len(evidence) > 0 && len(candidate.Evidence) > 0 {
			ordered := []planning.EvidenceItem{candidate.Evidence[0], evidence[0]}
			candidate.Evidence = append(ordered, candidate.Evidence[1:min(7, len(candidate.Evidence))]...)
		}
	}
	return result
}
