package agenttools

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// TestCallSites observes direct calls in Go tests on a local initialized by a
// specified constructor. Argument filters compare literal strings by position.
// Dynamic dispatch and other initializer forms are outside this query's scope.
type TestCallQuery struct {
	File, Constructor, Method string
	Literals                  map[int]string
}

type TestCallSite struct {
	Test, Reference string
	Evidence        planning.EvidenceItem
}

const maxTestCallSites = 6

func (s *Snapshot) TestCallSites(ctx context.Context, q TestCallQuery) ([]TestCallSite, error) {
	content, approved := s.files[q.File]
	pkg, constructor, qualified := strings.Cut(q.Constructor, ".")
	if !approved || !safeFile(q.File) || !strings.HasSuffix(q.File, "_test.go") ||
		!qualified || !token.IsIdentifier(pkg) || !token.IsIdentifier(constructor) ||
		!token.IsIdentifier(q.Method) || len(q.Literals) == 0 {
		return nil, errors.New("test call query requires an approved Go test file, qualified constructor, method and literal filters")
	}
	for position := range q.Literals {
		if position < 1 {
			return nil, errors.New("argument positions are 1-based")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	set := token.NewFileSet()
	// Parser object bindings track the local declaration even when a receiver
	// name is reused in a different test or shadowed in an inner scope.
	tree, err := parser.ParseFile(set, q.File, content, 0)
	if err != nil {
		return nil, fmt.Errorf("parse test calls in %s: %w", q.File, err)
	}
	var sites []TestCallSite
	for _, declaration := range tree.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if err != nil || ctx.Err() != nil {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != q.Method || localConstructor(sel.X) != q.Constructor || !matchesCallLiterals(call, q.Literals) {
				return true
			}
			if len(sites) == maxTestCallSites {
				err = fmt.Errorf("more than %d matching test calls in %s; narrow the literal filters", maxTestCallSites, q.File)
				return false
			}
			pos, end := set.Position(call.Pos()), set.Position(call.End())
			excerpt := fn.Name.Name + ": " + content[pos.Offset:end.Offset]
			if len([]rune(excerpt)) > 240 {
				err = fmt.Errorf("test call at %s:%d exceeds the evidence limit", q.File, pos.Line)
				return false
			}
			evidence := planning.EvidenceItem{
				Type: "AST_TEST_CALL", Location: fmt.Sprintf("%s:%d:%d", q.File, pos.Line, pos.Column), Excerpt: excerpt,
			}
			evidence.ID = digest(evidence)[:24]
			sites = append(sites, TestCallSite{
				Test: fn.Name.Name, Reference: fmt.Sprintf("%s:%d", path.Base(q.File), pos.Line), Evidence: evidence,
			})
			return true
		})
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(sites) == 0 {
		return nil, fmt.Errorf("no test calls match %s.%s in %s", q.Constructor, q.Method, q.File)
	}
	// A line reference must identify an unambiguous edit within the file.
	for i := 1; i < len(sites); i++ {
		if sites[i].Reference == sites[i-1].Reference {
			return nil, fmt.Errorf("multiple matching test calls share %s", sites[i].Reference)
		}
	}
	return sites, nil
}

func localConstructor(receiver ast.Expr) string {
	id, ok := receiver.(*ast.Ident)
	if !ok || id.Obj == nil {
		return ""
	}
	assignment, ok := id.Obj.Decl.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return ""
	}
	init, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return ""
	}
	qualified, ok := init.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := qualified.X.(*ast.Ident)
	if !ok || pkg.Obj != nil {
		return ""
	}
	return pkg.Name + "." + qualified.Sel.Name
}

func matchesCallLiterals(call *ast.CallExpr, literals map[int]string) bool {
	for position, want := range literals {
		if position > len(call.Args) {
			return false
		}
		arg, ok := call.Args[position-1].(*ast.BasicLit)
		if !ok || arg.Kind != token.STRING {
			return false
		}
		value, err := strconv.Unquote(arg.Value)
		if err != nil || value != want {
			return false
		}
	}
	return true
}
