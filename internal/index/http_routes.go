package index

import (
	"fmt"
	"go/ast"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
)

// HTTPRouteExtractor recognizes a narrow, auditable cross-language pattern:
// literal JS/TS api/fetch calls with an explicit method and a unique Go route
// registration whose handler can be resolved by the native Go indexer.
// It is not a JavaScript AST or a general HTTP call graph.
type HTTPRouteExtractor struct{}

func (HTTPRouteExtractor) Name() string { return "literal-http-route-links-v1" }
func (HTTPRouteExtractor) Capabilities() []domain.Capability {
	return []domain.Capability{domain.HTTPRouteLinks}
}
func (HTTPRouteExtractor) Supports(file domain.FileAnalysis) bool {
	if file.Fallback {
		return false
	}
	if file.Adapter == "go-syntax" {
		return !isTestSource(file.File)
	}
	if file.Adapter != "generic-text" {
		return false
	}
	switch path.Ext(file.File) {
	case ".js", ".jsx", ".ts", ".tsx":
		return !isTestSource(file.File)
	}
	return false
}

type httpRoute struct {
	targetID string
	location string
	excerpt  string
}

var clientHTTPCall = regexp.MustCompile(`\b(?:api(?:<[^>]+>)?|fetch)\s*\(\s*["'](/[^"'\s]*)["']\s*,\s*\{[^}\n]*\bmethod\s*:\s*["'](GET|POST|PUT|PATCH|DELETE)["']`)

func (HTTPRouteExtractor) Extract(inventory ingest.Inventory, artifacts []domain.Artifact) (map[string][]domain.SourceRelation, error) {
	result := map[string][]domain.SourceRelation{}
	byID := make(map[string]int, len(artifacts))
	for i, artifact := range artifacts {
		byID[artifact.ID] = i
	}
	model := newCallModel(artifacts)
	files := parseCallFiles(inventory, model.packages)
	model.collectTypesAndMethods(files, byID)
	routes := map[string][]httpRoute{}
	for _, file := range files {
		if isTestSource(file.file.RelativePath) {
			continue
		}
		lines := strings.Split(string(file.file.Content), "\n")
		for _, declaration := range file.tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if _, exists := byID[functionArtifactID(file, function)]; !exists {
				continue
			}
			environment := model.functionEnvironment(file, function)
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if _, nested := node.(*ast.FuncLit); nested {
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) != 2 {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				method := strings.ToUpper(selector.Sel.Name)
				if !httpMethod(method) {
					return true
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					return true
				}
				routePath, err := strconv.Unquote(literal.Value)
				if err != nil || !literalRoute(routePath) {
					return true
				}
				handlerID, _ := model.resolveCall(file, &ast.CallExpr{Fun: call.Args[1]}, environment)
				if handlerID == "" {
					return true
				}
				line := file.set.Position(call.Pos()).Line
				if line < 1 || line > len(lines) {
					return true
				}
				routes[method+" "+routePath] = append(routes[method+" "+routePath], httpRoute{
					targetID: handlerID, location: fmt.Sprintf("%s:%d", file.file.RelativePath, line), excerpt: routeExcerpt(lines[line-1]),
				})
				return true
			})
		}
	}
	for _, file := range inventory.Files {
		if _, ok := byID[file.RelativePath]; !ok || isTestSource(file.RelativePath) {
			continue
		}
		switch path.Ext(file.RelativePath) {
		case ".js", ".jsx", ".ts", ".tsx":
		default:
			continue
		}
		for index, line := range strings.Split(string(file.Content), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			for _, bounds := range clientHTTPCall.FindAllStringSubmatchIndex(line, -1) {
				if comment := strings.Index(line, "//"); comment >= 0 && comment < bounds[0] {
					continue
				}
				match := clientHTTPCall.FindStringSubmatch(line[bounds[0]:bounds[1]])
				if !literalRoute(match[1]) {
					continue
				}
				serverRoutes := routes[match[2]+" "+match[1]]
				if len(serverRoutes) != 1 {
					continue // Duplicate registrations cannot establish a unique handler.
				}
				route := serverRoutes[0]
				result[file.RelativePath] = append(result[file.RelativePath], domain.SourceRelation{
					Kind: domain.RelationHTTPRoute, TargetID: route.targetID,
					Location: fmt.Sprintf("%s:%d", file.RelativePath, index+1), Excerpt: routeExcerpt(line),
					TargetLocation: route.location, TargetExcerpt: route.excerpt,
				})
			}
		}
	}
	return result, nil
}

func literalRoute(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "{}:*?`$")
}

func httpMethod(value string) bool {
	switch value {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

func isTestSource(value string) bool {
	lower := "/" + strings.ToLower(value)
	return strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") || strings.Contains(lower, "/e2e/") ||
		strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") || strings.HasSuffix(lower, "_test.go")
}

func routeExcerpt(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 240 {
		runes = runes[:240]
	}
	return string(runes)
}
