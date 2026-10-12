package index

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

type callFile struct {
	file           ingest.File
	set            *token.FileSet
	tree           *ast.File
	dir            string
	pkg            string
	imports        map[string]string
	importPackages map[string]string
	allImports     map[string]string
}

type callSignature struct {
	params    string
	results   string
	variadic  bool
	supported bool
}

type indexedType struct {
	interfaceType bool
	unsupported   bool
	fields        map[string]string
	required      map[string]callSignature
}

type methodTarget struct {
	id        string
	signature callSignature
}

type callModel struct {
	packages map[string]string
	types    map[string]*indexedType
	methods  map[string]map[string][]methodTarget
	free     map[string][]string
}

// attachCalls resolves only relations supported by repository-local syntax:
// free functions, statically typed receiver methods, and unique structural
// interface implementations. Unsupported or ambiguous dispatch abstains.
func attachCalls(inventory ingest.Inventory, artifacts []domain.Artifact) {
	model := newCallModel(artifacts)
	files := parseCallFiles(inventory, model.packages)
	byID := map[string]int{}
	for i := range artifacts {
		byID[artifacts[i].ID] = i
	}
	model.collectTypesAndMethods(files, byID)

	for _, file := range files {
		lines := strings.Split(string(file.file.Content), "\n")
		for _, declaration := range file.tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			id := functionArtifactID(file, function)
			artifactIndex, ok := byID[id]
			if !ok {
				continue
			}
			environment := model.functionEnvironment(file, function)
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if _, nested := node.(*ast.FuncLit); nested {
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				targetID, relationKind := model.resolveCall(file, call, environment)
				if targetID == "" || targetID == id {
					return true
				}
				line := file.set.Position(call.Pos()).Line
				if line < 1 || line > len(lines) {
					return true
				}
				excerpt := []rune(strings.TrimSpace(lines[line-1]))
				if len(excerpt) > 240 {
					excerpt = excerpt[:240]
				}
				artifacts[artifactIndex].Relations = append(artifacts[artifactIndex].Relations, domain.SourceRelation{
					Kind: relationKind, TargetID: targetID,
					Location: fmt.Sprintf("%s:%d", file.file.RelativePath, line), Excerpt: string(excerpt),
				})
				return true
			})
		}
	}
}

func newCallModel(artifacts []domain.Artifact) *callModel {
	model := &callModel{
		packages: map[string]string{}, types: map[string]*indexedType{},
		methods: map[string]map[string][]methodTarget{}, free: map[string][]string{},
	}
	for _, artifact := range artifacts {
		if artifact.Symbol.Kind == "package" {
			model.packages[artifact.File] = artifact.Symbol.Name
		}
	}
	for _, artifact := range artifacts {
		if artifact.Symbol.Kind == "function" {
			key := packageKey(path.Dir(artifact.File), model.packages[artifact.File]) + "|" + artifact.Symbol.Name
			model.free[key] = append(model.free[key], artifact.ID)
		}
	}
	return model
}

func parseCallFiles(inventory ingest.Inventory, packages map[string]string) []callFile {
	modules := map[string]string{}
	for _, file := range inventory.Files {
		if path.Base(file.RelativePath) != "go.mod" {
			continue
		}
		for _, line := range strings.Split(string(file.Content), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "module" {
				modules[strings.Trim(fields[1], "\"")] = path.Dir(file.RelativePath)
			}
		}
	}

	result := []callFile{}
	for _, file := range inventory.Files {
		if path.Ext(file.RelativePath) != ".go" {
			continue
		}
		set := token.NewFileSet()
		tree, err := parser.ParseFile(set, file.RelativePath, file.Content, 0)
		if err != nil {
			continue
		}
		parsed := callFile{
			file: file, set: set, tree: tree, dir: path.Dir(file.RelativePath), pkg: tree.Name.Name,
			imports: map[string]string{}, importPackages: map[string]string{}, allImports: map[string]string{},
		}
		for _, specification := range tree.Imports {
			importPath, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				continue
			}
			directory := localImportDirectory(importPath, modules)
			alias := path.Base(importPath)
			if directory != "" {
				alias = uniquePackage(packages, directory)
			}
			if specification.Name != nil {
				alias = specification.Name.Name
			}
			if alias != "" && alias != "." && alias != "_" {
				parsed.allImports[alias] = importPath
			}
			if directory != "" && alias != "" && alias != "." && alias != "_" {
				parsed.imports[alias] = directory
				parsed.importPackages[alias] = uniquePackage(packages, directory)
			}
		}
		result = append(result, parsed)
	}
	return result
}

func localImportDirectory(importPath string, modules map[string]string) string {
	bestModule, root := "", ""
	for module, candidateRoot := range modules {
		if (importPath == module || strings.HasPrefix(importPath, module+"/")) && len(module) > len(bestModule) {
			bestModule, root = module, candidateRoot
		}
	}
	if bestModule == "" {
		return ""
	}
	return path.Join(root, strings.TrimPrefix(strings.TrimPrefix(importPath, bestModule), "/"))
}

func (model *callModel) collectTypesAndMethods(files []callFile, artifacts map[string]int) {
	for _, file := range files {
		for _, declaration := range file.tree.Decls {
			switch value := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range value.Specs {
					typeSpecification, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					key := namedTypeKey(file, typeSpecification.Name.Name)
					indexed := &indexedType{fields: map[string]string{}, required: map[string]callSignature{}}
					switch expression := typeSpecification.Type.(type) {
					case *ast.StructType:
						for _, field := range expression.Fields.List {
							fieldType, ok := resolveNamedType(file, field.Type)
							if !ok || len(field.Names) == 0 {
								continue
							}
							for _, name := range field.Names {
								indexed.fields[name.Name] = fieldType
							}
						}
					case *ast.InterfaceType:
						indexed.interfaceType = true
						for _, field := range expression.Methods.List {
							functionType, ok := field.Type.(*ast.FuncType)
							if !ok || len(field.Names) != 1 {
								indexed.unsupported = true
								continue
							}
							signature := signatureOf(file, functionType)
							if !signature.supported {
								indexed.unsupported = true
								continue
							}
							indexed.required[field.Names[0].Name] = signature
						}
					}
					model.types[key] = indexed
				}
			case *ast.FuncDecl:
				if value.Recv == nil || len(value.Recv.List) == 0 {
					continue
				}
				receiver, ok := resolveNamedType(file, value.Recv.List[0].Type)
				if !ok {
					continue
				}
				id := functionArtifactID(file, value)
				if _, ok := artifacts[id]; !ok {
					continue
				}
				if model.methods[receiver] == nil {
					model.methods[receiver] = map[string][]methodTarget{}
				}
				model.methods[receiver][value.Name.Name] = append(model.methods[receiver][value.Name.Name], methodTarget{id: id, signature: signatureOf(file, value.Type)})
			}
		}
	}
}

func (model *callModel) functionEnvironment(file callFile, function *ast.FuncDecl) map[string]string {
	environment := map[string]string{}
	blocked := map[string]bool{}
	addFieldsToEnvironment(file, function.Type.Params, environment)
	if function.Recv != nil {
		addFieldsToEnvironment(file, function.Recv, environment)
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		switch value := node.(type) {
		case *ast.DeclStmt:
			declaration, ok := value.Decl.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				return true
			}
			for _, specification := range declaration.Specs {
				variable, ok := specification.(*ast.ValueSpec)
				if !ok || variable.Type == nil {
					continue
				}
				typeName, typeKnown := resolveNamedType(file, variable.Type)
				for _, name := range variable.Names {
					defineEnvironmentType(name.Name, typeName, typeKnown, environment, blocked)
				}
			}
		case *ast.AssignStmt:
			if value.Tok != token.DEFINE {
				return true
			}
			for index, left := range value.Lhs {
				identifier, ok := left.(*ast.Ident)
				if !ok || index >= len(value.Rhs) {
					continue
				}
				typeName, typeKnown := expressionDeclaredType(file, value.Rhs[index])
				defineEnvironmentType(identifier.Name, typeName, typeKnown, environment, blocked)
			}
		}
		return true
	})
	return environment
}

func defineEnvironmentType(name, typeName string, typeKnown bool, environment map[string]string, blocked map[string]bool) {
	if blocked[name] || environment[name] != "" || !typeKnown {
		delete(environment, name)
		blocked[name] = true
		return
	}
	environment[name] = typeName
}

func addFieldsToEnvironment(file callFile, fields *ast.FieldList, environment map[string]string) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		typeName, ok := resolveNamedType(file, field.Type)
		if !ok {
			continue
		}
		for _, name := range field.Names {
			environment[name.Name] = typeName
		}
	}
}

func expressionDeclaredType(file callFile, expression ast.Expr) (string, bool) {
	switch value := expression.(type) {
	case *ast.CompositeLit:
		return resolveNamedType(file, value.Type)
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			return expressionDeclaredType(file, value.X)
		}
	case *ast.ParenExpr:
		return expressionDeclaredType(file, value.X)
	}
	return "", false
}

func (model *callModel) resolveCall(file callFile, call *ast.CallExpr, environment map[string]string) (string, domain.RelationKind) {
	switch callee := call.Fun.(type) {
	case *ast.Ident:
		if callee.Obj != nil && callee.Obj.Kind != ast.Fun {
			return "", ""
		}
		matches := model.free[packageKey(file.dir, file.pkg)+"|"+callee.Name]
		if len(matches) == 1 {
			return matches[0], domain.RelationCalls
		}
	case *ast.SelectorExpr:
		if identifier, ok := callee.X.(*ast.Ident); ok && identifier.Obj == nil {
			if directory := file.imports[identifier.Name]; directory != "" {
				pkg := uniquePackage(model.packages, directory)
				matches := model.free[packageKey(directory, pkg)+"|"+callee.Sel.Name]
				if ast.IsExported(callee.Sel.Name) && len(matches) == 1 {
					return matches[0], domain.RelationCalls
				}
				return "", ""
			}
		}
		receiverType, ok := model.expressionType(callee.X, environment)
		if !ok {
			return "", ""
		}
		indexed := model.types[receiverType]
		if indexed != nil && indexed.interfaceType {
			matches := model.interfaceMethodTargets(receiverType, callee.Sel.Name)
			if len(matches) == 1 {
				return matches[0], domain.RelationInterfaceDispatch
			}
			return "", ""
		}
		matches := model.methods[receiverType][callee.Sel.Name]
		if len(matches) == 1 {
			return matches[0].id, domain.RelationCallsMethod
		}
	}
	return "", ""
}

func (model *callModel) expressionType(expression ast.Expr, environment map[string]string) (string, bool) {
	switch value := expression.(type) {
	case *ast.Ident:
		result, ok := environment[value.Name]
		return result, ok
	case *ast.SelectorExpr:
		base, ok := model.expressionType(value.X, environment)
		if !ok || model.types[base] == nil {
			return "", false
		}
		result, ok := model.types[base].fields[value.Sel.Name]
		return result, ok
	case *ast.ParenExpr:
		return model.expressionType(value.X, environment)
	case *ast.StarExpr:
		return model.expressionType(value.X, environment)
	}
	return "", false
}

func (model *callModel) interfaceMethodTargets(interfaceType, methodName string) []string {
	required := model.types[interfaceType]
	if required == nil || !required.interfaceType || required.unsupported || len(required.required) == 0 {
		return nil
	}
	if _, ok := required.required[methodName]; !ok {
		return nil
	}
	result := []string{}
	for typeName, candidate := range model.types {
		if candidate.interfaceType || !model.implements(typeName, required) {
			continue
		}
		methods := model.methods[typeName][methodName]
		if len(methods) == 1 {
			result = append(result, methods[0].id)
		}
	}
	return result
}

func (model *callModel) implements(typeName string, required *indexedType) bool {
	for methodName, expected := range required.required {
		methods := model.methods[typeName][methodName]
		if len(methods) != 1 || methods[0].signature != expected {
			return false
		}
	}
	return true
}

func resolveNamedType(file callFile, expression ast.Expr) (string, bool) {
	switch value := expression.(type) {
	case *ast.Ident:
		return namedTypeKey(file, value.Name), true
	case *ast.StarExpr:
		return resolveNamedType(file, value.X)
	case *ast.ParenExpr:
		return resolveNamedType(file, value.X)
	case *ast.IndexExpr:
		return resolveNamedType(file, value.X)
	case *ast.IndexListExpr:
		return resolveNamedType(file, value.X)
	case *ast.SelectorExpr:
		identifier, ok := value.X.(*ast.Ident)
		if !ok || file.imports[identifier.Name] == "" {
			return "", false
		}
		directory := file.imports[identifier.Name]
		return packageKey(directory, file.importPackages[identifier.Name]) + "|" + value.Sel.Name, true
	}
	return "", false
}

func signatureOf(file callFile, function *ast.FuncType) callSignature {
	params, paramsOK := fieldShapes(file, function.Params)
	results, resultsOK := fieldShapes(file, function.Results)
	result := callSignature{params: strings.Join(params, ","), results: strings.Join(results, ","), supported: paramsOK && resultsOK}
	if function.Params != nil && len(function.Params.List) > 0 {
		_, result.variadic = function.Params.List[len(function.Params.List)-1].Type.(*ast.Ellipsis)
	}
	return result
}

func fieldShapes(file callFile, fields *ast.FieldList) ([]string, bool) {
	if fields == nil {
		return nil, true
	}
	result := []string{}
	for _, field := range fields.List {
		shape, ok := typeShape(file, field.Type)
		if !ok {
			return nil, false
		}
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, shape)
		}
	}
	return result, true
}

func typeShape(file callFile, expression ast.Expr) (string, bool) {
	switch value := expression.(type) {
	case *ast.Ident:
		if isBuiltinType(value.Name) {
			return "builtin:" + value.Name, true
		}
		return namedTypeKey(file, value.Name), true
	case *ast.SelectorExpr:
		identifier, ok := value.X.(*ast.Ident)
		if !ok || file.allImports[identifier.Name] == "" {
			return "", false
		}
		if directory := file.imports[identifier.Name]; directory != "" {
			return packageKey(directory, file.importPackages[identifier.Name]) + "|" + value.Sel.Name, true
		}
		return "import:" + file.allImports[identifier.Name] + "." + value.Sel.Name, true
	case *ast.StarExpr:
		shape, ok := typeShape(file, value.X)
		return "*" + shape, ok
	case *ast.Ellipsis:
		shape, ok := typeShape(file, value.Elt)
		return "..." + shape, ok
	case *ast.ArrayType:
		if value.Len != nil {
			return "", false
		}
		shape, ok := typeShape(file, value.Elt)
		return "[]" + shape, ok
	case *ast.MapType:
		key, keyOK := typeShape(file, value.Key)
		element, elementOK := typeShape(file, value.Value)
		return "map[" + key + "]" + element, keyOK && elementOK
	case *ast.ChanType:
		element, ok := typeShape(file, value.Value)
		return fmt.Sprintf("chan%d:%s", value.Dir, element), ok
	case *ast.ParenExpr:
		return typeShape(file, value.X)
	case *ast.IndexExpr, *ast.IndexListExpr, *ast.FuncType, *ast.InterfaceType:
		return "", false
	default:
		return "", false
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}

func functionArtifactID(file callFile, function *ast.FuncDecl) string {
	kind := "function"
	if function.Recv != nil {
		kind = "method"
	}
	return fmt.Sprintf("%s#%s:%s:%d", file.file.RelativePath, kind, function.Name.Name, file.set.Position(function.Pos()).Line)
}

func namedTypeKey(file callFile, name string) string {
	return packageKey(file.dir, file.pkg) + "|" + name
}

func packageKey(directory, pkg string) string {
	return directory + "|" + pkg
}

func uniquePackage(packages map[string]string, directory string) string {
	result := ""
	for file, pkg := range packages {
		if path.Dir(file) != directory || strings.HasSuffix(pkg, "_test") {
			continue
		}
		if result != "" && result != pkg {
			return ""
		}
		result = pkg
	}
	return result
}
