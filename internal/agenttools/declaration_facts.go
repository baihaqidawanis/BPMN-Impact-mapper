package agenttools

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Declaration facts describe syntax in the frozen file, never inferred callers
// or runtime values. Parameter positions include every name in a grouped field.
func (s *Snapshot) withDeclarationFacts(result planning.ToolResult) planning.ToolResult {
	for i, c := range result.Candidates {
		if path.Ext(c.File) != ".go" || c.Symbol == "" || len(c.Evidence) >= 8 {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), c.File, s.files[c.File], parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range tree.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := fn.Name.Name
			receiver := ""
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				typ := fn.Recv.List[0].Type
				if ptr, ok := typ.(*ast.StarExpr); ok {
					typ = ptr.X
				}
				if id, ok := typ.(*ast.Ident); ok {
					receiver = id.Name
					name = receiver + "." + name
				}
			}
			if name != c.Symbol {
				continue
			}
			names := []string{}
			for _, field := range fn.Type.Params.List {
				if len(field.Names) == 0 {
					names = append(names, "(unnamed)")
				}
				for _, id := range field.Names {
					names = append(names, id.Name)
				}
			}
			parameters := strings.Join(names, ", ")
			if parameters == "" {
				parameters = "none"
			}
			text := "Go AST " + name + " parameters in order: " + parameters + "."
			if receiver != "" {
				for _, d := range tree.Decls {
					group, ok := d.(*ast.GenDecl)
					if !ok {
						continue
					}
					for _, spec := range group.Specs {
						typ, ok := spec.(*ast.TypeSpec)
						if !ok || typ.Name.Name != receiver {
							continue
						}
						fields, ok := typ.Type.(*ast.StructType)
						if !ok {
							continue
						}
						members := []string{}
						embedded := false
						for _, f := range fields.Fields.List {
							if len(f.Names) == 0 {
								embedded = true
							}
							for _, id := range f.Names {
								members = append(members, id.Name)
							}
						}
						addition := " Receiver declared fields: " + strings.Join(members, ", ") + "."
						if embedded {
							addition += " Embedded fields may promote other members."
						}
						if len([]rune(text+addition)) <= 240 {
							text += addition
						}
					}
				}
			}
			if len([]rune(text)) <= 240 {
				e := planning.EvidenceItem{Type: "AST_DECLARATION_FACT", Location: c.File, Excerpt: text}
				e.ID = digest(e)[:24]
				result.Candidates[i].Evidence = append([]planning.EvidenceItem{e}, c.Evidence...)
			}
			break
		}
	}
	return result
}
