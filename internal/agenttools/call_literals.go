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
	"unicode"

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
		line, endLine int
		focusTest     bool
	}
	var found []observation
	terms := map[string]bool{}
	for _, term := range strings.FieldsFunc(strings.ToLower(q.Query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(term) >= 3 {
			terms[term] = true
		}
	}
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
		operandMatches := map[string]bool{}
		literalMatches := map[string]int{}
		for i, arg := range call.Args {
			literal, ok := arg.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				start, end := set.Position(arg.Pos()).Offset, set.Position(arg.End()).Offset
				if start >= 0 && end >= start && end <= len(content) {
					operand := strings.ToLower(content[start:end])
					for term := range terms {
						if strings.Contains(operand, term) {
							operandMatches[term] = true
						}
					}
				}
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
			normalized := strings.ToLower(value)
			for term := range terms {
				if normalized == term {
					literalMatches[term] = 8
				} else if strings.Contains(normalized, term) {
					literalMatches[term] = max(1, literalMatches[term])
				}
			}
		}
		if len(parts) == 0 {
			return true
		}
		// A matching operand helps distinguish the same action on different
		// entities in integration tests. It is a ranking signal only.
		score += 4 * len(operandMatches)
		exactAction := false
		for _, weight := range literalMatches {
			score += weight
			exactAction = exactAction || weight == 8
		}
		text += strings.Join(parts, ", ") + ". Others not listed."
		pos := set.Position(call.Pos())
		e := planning.EvidenceItem{Type: "AST_CALL_LITERALS", Location: fmt.Sprintf("%s:%d:%d", q.File, pos.Line, pos.Column), Excerpt: text}
		e.ID = digest(e)[:24]
		found = append(found, observation{
			evidence: e, score: score, offset: pos.Offset, line: pos.Line, endLine: set.Position(call.End()).Line,
			focusTest: exactAction && len(operandMatches) > 0 && strings.HasSuffix(strings.ToLower(q.File), "_test.go"),
		})
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
	best := found[0]
	if best.focusTest {
		// Keep the call and its immediate outcome in the same test. A mixed
		// file can exercise the same verb on unrelated entities.
		for _, decl := range tree.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") || best.line < set.Position(fn.Pos()).Line || best.endLine > set.Position(fn.End()).Line {
				continue
			}
			if window, err := s.readSource(ctx, planning.ToolRequest{File: q.File, StartLine: best.line}); err == nil && len(window.Candidates) == 1 {
				var scoped []planning.EvidenceItem
				end := set.Position(fn.End()).Line
				for _, item := range window.Candidates[0].Evidence {
					var line int
					fmt.Sscanf(strings.TrimPrefix(item.Location, q.File+":"), "%d", &line)
					if line >= best.line && line <= end {
						scoped = append(scoped, item)
					}
				}
				if len(scoped) > 0 {
					c.Evidence = scoped[:min(6, len(scoped))]
				}
			}
			break
		}
	}
	c.Evidence = append([]planning.EvidenceItem{best.evidence}, c.Evidence...)
	result.Note += " AST_CALL_LITERALS reports syntax only; argument positions do not establish parameter meaning, dependencies or runtime behavior."
	return result
}
