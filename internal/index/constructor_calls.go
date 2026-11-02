package index

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

// GoWiringExtractor records observed direct constructor injection, not general
// runtime interface resolution. Other/unknown production bindings abstain.
type GoWiringExtractor struct{}

func (GoWiringExtractor) Name() string { return "go-direct-constructor-bindings-v1" }
func (GoWiringExtractor) Supports(f domain.FileAnalysis) bool {
	return f.Adapter == "go-syntax" && !f.Fallback
}
func (GoWiringExtractor) Capabilities() []domain.Capability {
	return []domain.Capability{domain.TypeRelations}
}

type constructorBinding struct {
	owner, field, iface string
	parameter           int
	location, excerpt   string
}
type concreteBinding struct{ kind, location, excerpt, constructorLocation, constructorExcerpt string }

func (GoWiringExtractor) Extract(inventory ingest.Inventory, artifacts []domain.Artifact) (map[string][]domain.SourceRelation, error) {
	byID := map[string]int{}
	for i, a := range artifacts {
		byID[a.ID] = i
	}
	model := newCallModel(artifacts)
	files := parseCallFiles(inventory, model.packages)
	model.collectTypesAndMethods(files, byID)
	fieldOrder := map[string][]string{}
	for _, file := range files {
		for _, d := range file.tree.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range g.Specs {
				t, ok := s.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := t.Type.(*ast.StructType)
				if !ok {
					continue
				}
				names := []string{}
				valid := true
				for _, f := range st.Fields.List {
					if len(f.Names) == 0 {
						valid = false
						break
					}
					for _, n := range f.Names {
						names = append(names, n.Name)
					}
				}
				if valid {
					fieldOrder[namedTypeKey(file, t.Name.Name)] = names
				}
			}
		}
	}
	constructors := map[string][]constructorBinding{}
	knownLiterals := map[*ast.CompositeLit]bool{}
	for _, file := range files {
		if isTestSource(file.file.RelativePath) {
			continue
		}
		for _, d := range file.tree.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || f.Recv != nil || f.Body == nil || len(f.Body.List) != 1 {
				continue
			}
			ret, ok := f.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				continue
			}
			literal, ok := ret.Results[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			owner, ok := resolveNamedType(file, literal.Type)
			if !ok || model.types[owner] == nil {
				continue
			}
			params := map[string]int{}
			position := 0
			if f.Type.Params != nil {
				for _, field := range f.Type.Params.List {
					if len(field.Names) == 0 {
						position++
					}
					for _, name := range field.Names {
						params[name.Name] = position
						position++
					}
				}
			}
			for positionInLiteral, item := range literal.Elts {
				fieldName := ""
				expression := item
				if pair, ok := item.(*ast.KeyValueExpr); ok {
					key, ok := pair.Key.(*ast.Ident)
					if !ok {
						continue
					}
					fieldName = key.Name
					expression = pair.Value
				} else {
					order := fieldOrder[owner]
					if len(order) != len(literal.Elts) {
						continue
					}
					fieldName = order[positionInLiteral]
				}
				value, ok := expression.(*ast.Ident)
				if !ok {
					continue
				}
				index, ok := params[value.Name]
				if !ok {
					continue
				}
				iface := model.types[owner].fields[fieldName]
				required := model.types[iface]
				if required == nil || !required.interfaceType || required.unsupported {
					continue
				}
				line := file.set.Position(item.Pos()).Line
				constructors[functionArtifactID(file, f)] = append(constructors[functionArtifactID(file, f)], constructorBinding{owner, fieldName, iface, index, fmt.Sprintf("%s:%d", file.file.RelativePath, line), sourceLine(file, line)})
				knownLiterals[literal] = true
			}
		}
	}
	bindings := map[string][]concreteBinding{}
	for _, file := range files {
		if isTestSource(file.file.RelativePath) {
			continue
		}
		for _, d := range file.tree.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				continue
			}
			env := model.functionEnvironment(file, f)
			ast.Inspect(f.Body, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if name, ok := call.Fun.(*ast.Ident); ok && name.Obj != nil && name.Obj.Kind != ast.Fun {
					return true
				}
				id, _ := model.resolveCall(file, call, env)
				for _, c := range constructors[id] {
					key := c.owner + "\x00" + c.field
					kind := ""
					if c.parameter < len(call.Args) {
						kind, _ = expressionDeclaredType(file, call.Args[c.parameter])
						if kind == "" {
							kind, _ = model.expressionType(call.Args[c.parameter], env)
						}
					}
					line := file.set.Position(call.Pos()).Line
					bindings[key] = append(bindings[key], concreteBinding{kind, fmt.Sprintf("%s:%d", file.file.RelativePath, line), sourceLine(file, line), c.location, c.excerpt})
				}
				return true
			})
		}
	}
	unsafeBindings := map[string]bool{}
	for _, file := range files {
		if isTestSource(file.file.RelativePath) {
			continue
		}
		ast.Inspect(file.tree, func(n ast.Node) bool {
			if literal, ok := n.(*ast.CompositeLit); ok && !knownLiterals[literal] {
				if owner, ok := resolveNamedType(file, literal.Type); ok {
					for key := range bindings {
						if strings.HasPrefix(key, owner+"\x00") {
							unsafeBindings[key] = true
						}
					}
				}
			}
			return true
		})
		for _, d := range file.tree.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				continue
			}
			env := model.functionEnvironment(file, f)
			ast.Inspect(f.Body, func(n ast.Node) bool {
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range assign.Lhs {
					field, ok := lhs.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					if owner, ok := model.expressionType(field.X, env); ok {
						unsafeBindings[owner+"\x00"+field.Sel.Name] = true
					}
				}
				return true
			})
		}
	}
	result := map[string][]domain.SourceRelation{}
	for _, file := range files {
		if isTestSource(file.file.RelativePath) {
			continue
		}
		for _, d := range file.tree.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				continue
			}
			caller := functionArtifactID(file, f)
			if _, ok := byID[caller]; !ok {
				continue
			}
			env := model.functionEnvironment(file, f)
			ast.Inspect(f.Body, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				field, ok := method.X.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				owner, ok := model.expressionType(field.X, env)
				if !ok || model.types[owner] == nil {
					return true
				}
				iface := model.types[owner].fields[field.Sel.Name]
				required := model.types[iface]
				if required == nil || !required.interfaceType || required.unsupported {
					return true
				}
				key := owner + "\x00" + field.Sel.Name
				if unsafeBindings[key] {
					return true
				}
				seen := bindings[key]
				if len(seen) == 0 {
					return true
				}
				concrete := seen[0]
				for _, b := range seen {
					if b.kind == "" || b.kind != concrete.kind {
						return true
					}
				}
				if concrete.kind == "" || !model.implements(concrete.kind, required) {
					return true
				}
				targets := model.methods[concrete.kind][method.Sel.Name]
				if len(targets) != 1 {
					return true
				}
				line := file.set.Position(call.Pos()).Line
				result[caller] = append(result[caller], domain.SourceRelation{Kind: domain.RelationConstructorCall, TargetID: targets[0].id, Location: fmt.Sprintf("%s:%d", file.file.RelativePath, line), Excerpt: sourceLine(file, line), BindingLocation: concrete.location, BindingExcerpt: concrete.excerpt, TargetLocation: concrete.constructorLocation, TargetExcerpt: concrete.constructorExcerpt})
				return true
			})
		}
	}
	for id := range result {
		sort.SliceStable(result[id], func(i, j int) bool {
			a, b := result[id][i], result[id][j]
			if a.TargetID != b.TargetID {
				return a.TargetID < b.TargetID
			}
			return a.Location < b.Location
		})
	}
	return result, nil
}

func sourceLine(file callFile, line int) string {
	lines := strings.Split(string(file.file.Content), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return routeExcerpt(lines[line-1])
}
