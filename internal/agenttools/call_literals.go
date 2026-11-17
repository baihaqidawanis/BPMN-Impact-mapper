package agenttools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// withCallLiterals adds one bounded syntactic observation from the approved
// snapshot near an inspected source span. Argument positions are 1-based, not
// inferred parameter names. Expressions, dispatch targets and runtime values
// are deliberately not evaluated or resolved here.
func (s *Snapshot) withCallLiterals(ctx context.Context, q planning.ToolRequest, result planning.ToolResult) planning.ToolResult {
	if ctx.Err() != nil || !strings.EqualFold(path.Ext(q.File), ".go") || len(result.Candidates) != 1 {
		return result
	}
	content, approved := s.files[q.File]
	if !approved {
		return result
	}
	c := &result.Candidates[0]
	if len(c.Evidence) >= 8 {
		return result // Never evict an existing source window to add a fact.
	}
	set := token.NewFileSet()
	tree, err := parser.ParseFile(set, q.File, content, parser.SkipObjectResolution)
	if err != nil {
		return result // Text fallback remains usable; partial ASTs are not proof.
	}
	lines := map[int]bool{}
	for _, e := range c.Evidence {
		var line int
		if strings.HasPrefix(e.Location, q.File+":") {
			fmt.Sscanf(strings.TrimPrefix(e.Location, q.File+":"), "%d", &line)
			lines[line] = line > 0
		}
	}
	type observation struct {
		evidence      planning.EvidenceItem
		score, offset int
	}
	var found []observation
	ast.Inspect(tree, func(node ast.Node) bool {
		if ctx.Err() != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		near := false
		for line := range lines {
			if line >= set.Position(call.Pos()).Line && line <= set.Position(call.End()).Line {
				near = true
				break
			}
		}
		if !near {
			return true
		}
		name := ""
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			name = fn.Name
		case *ast.SelectorExpr:
			if receiver, ok := fn.X.(*ast.Ident); ok {
				name = receiver.Name + "." + fn.Sel.Name
			}
		}
		if name == "" || len(name) > 80 {
			return true
		}
		text := "Go AST call " + name + "; string literals (1-based arguments): "
		parts := []string{}
		score := 0
		for i, arg := range call.Args {
			literal, ok := arg.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil || len([]rune(value)) > 48 {
				continue
			}
			part := fmt.Sprintf("%d=%q", i+1, value)
			if len([]rune(text+strings.Join(append(append([]string{}, parts...), part), ", "))) > 220 {
				continue
			}
			parts = append(parts, part)
			if value == "" {
				score++
			}
			for _, term := range strings.Fields(strings.ToLower(q.Query)) {
				if len(term) >= 3 && strings.Contains(strings.ToLower(value), term) {
					score += 3
				}
			}
		}
		if len(parts) == 0 {
			return true
		}
		text += strings.Join(parts, ", ") + ". Others not listed."
		pos := set.Position(call.Pos())
		e := planning.EvidenceItem{Type: "AST_CALL_LITERALS", Location: fmt.Sprintf("%s:%d:%d", q.File, pos.Line, pos.Column), Excerpt: text}
		e.ID = digest(e)[:24]
		found = append(found, observation{e, score, pos.Offset})
		return true
	})
	if ctx.Err() != nil || len(found) == 0 {
		return result
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].offset < found[j].offset
	})
	c.Evidence = append([]planning.EvidenceItem{found[0].evidence}, c.Evidence...)
	result.Note += " AST_CALL_LITERALS reports syntax only; argument positions do not establish parameter meaning, dependencies or runtime behavior."
	return result
}
