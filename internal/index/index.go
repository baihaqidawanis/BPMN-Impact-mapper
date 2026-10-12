package index

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

type Adapter interface {
	Supports(file ingest.File) bool
	Index(file ingest.File) ([]domain.Artifact, error)
}

func Build(inventory ingest.Inventory) ([]domain.Artifact, []string) {
	result := DefaultRegistry().Build(inventory)
	return result.Artifacts, result.Warnings
}

type TextAdapter struct{}

func (TextAdapter) Supports(file ingest.File) bool { return true }

func (TextAdapter) Index(file ingest.File) ([]domain.Artifact, error) {
	content := string(file.Content)
	artifact := baseArtifact(file, content)
	artifact.ID = file.RelativePath
	return []domain.Artifact{artifact}, nil
}

type GoAdapter struct{}

func (GoAdapter) Supports(file ingest.File) bool {
	return strings.EqualFold(path.Ext(file.RelativePath), ".go")
}

func (GoAdapter) Index(file ingest.File) ([]domain.Artifact, error) {
	set := token.NewFileSet()
	parsed, err := parser.ParseFile(set, file.RelativePath, file.Content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	fileArtifact := baseArtifact(file, string(file.Content))
	fileArtifact.ID = file.RelativePath
	fileArtifact.Symbol = domain.Symbol{Name: parsed.Name.Name, Kind: "package", StartLine: 1, EndLine: 1}
	artifacts := []domain.Artifact{fileArtifact}
	for _, declaration := range parsed.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			symbol := domain.Symbol{Name: value.Name.Name, Kind: "function", StartLine: set.Position(value.Pos()).Line, EndLine: set.Position(value.End()).Line}
			if value.Recv != nil && len(value.Recv.List) > 0 {
				symbol.Kind = "method"
				symbol.Receiver = receiverName(value.Recv.List[0].Type)
			}
			artifacts = append(artifacts, symbolArtifact(file, symbol))
		case *ast.GenDecl:
			for _, specification := range value.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				symbol := domain.Symbol{Name: typeSpec.Name.Name, Kind: "type", StartLine: set.Position(typeSpec.Pos()).Line, EndLine: set.Position(typeSpec.End()).Line}
				artifacts = append(artifacts, symbolArtifact(file, symbol))
			}
		}
	}
	return artifacts, nil
}

func symbolArtifact(file ingest.File, symbol domain.Symbol) domain.Artifact {
	lines := strings.Split(string(file.Content), "\n")
	start, end := symbol.StartLine-1, symbol.EndLine
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	content := strings.Join(lines[start:end], "\n")
	artifact := baseArtifact(file, content)
	artifact.ID = fmt.Sprintf("%s#%s:%s:%d", file.RelativePath, symbol.Kind, symbol.Name, symbol.StartLine)
	artifact.Symbol = symbol
	artifact.Tokens = textutil.Tokenize(file.RelativePath + " " + symbol.Name + " " + symbol.Receiver + " " + content)
	return artifact
}

func baseArtifact(file ingest.File, content string) domain.Artifact {
	return domain.Artifact{
		File: file.RelativePath, Module: module(file.RelativePath), Layer: layer(file.RelativePath, content),
		Language: language(file.RelativePath), Content: content,
		Tokens: textutil.Tokenize(file.RelativePath + " " + content),
	}
}

func receiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	case *ast.IndexExpr:
		return receiverName(value.X)
	case *ast.IndexListExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func module(relative string) string {
	parts := strings.Split(filepathSlash(relative), "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "."
}

func layer(relative, content string) string {
	lower := strings.ToLower(filepathSlash(relative))
	patterns := []struct {
		needle string
		layer  string
	}{
		{"/domain/", "domain"}, {"/entity/", "domain"}, {"/entities/", "domain"},
		{"/application/", "application"}, {"/app/", "application"}, {"/usecase/", "application"}, {"/service/", "application"},
		{"/adapter/", "adapter"}, {"/infrastructure/", "adapter"},
		{"/handler/", "transport"}, {"/transport/", "transport"}, {"/controller/", "transport"}, {"/api/", "transport"},
		{"/repository/", "persistence"}, {"/postgres/", "persistence"}, {"/migration", "persistence"}, {"/schema", "persistence"},
		{"/frontend/", "frontend"}, {"/component", "frontend"}, {"/pages/", "frontend"},
		{"/test/", "test"}, {"/tests/", "test"}, {"_test.go", "test"}, {".spec.", "test"}, {".test.", "test"},
		{"/docs/", "documentation"}, {".md", "documentation"},
	}
	padded := "/" + lower
	for _, pattern := range patterns {
		if strings.Contains(padded, pattern.needle) {
			return pattern.layer
		}
	}
	if strings.Contains(content, "http.Handle") || strings.Contains(content, "ServeHTTP") {
		return "transport"
	}
	return "unknown"
}

func language(relative string) string {
	ext := strings.ToLower(path.Ext(relative))
	languages := map[string]string{".go": "Go", ".js": "JavaScript", ".jsx": "JavaScript", ".ts": "TypeScript", ".tsx": "TypeScript", ".java": "Java", ".py": "Python", ".cs": "C#", ".sql": "SQL", ".md": "Markdown", ".xml": "XML", ".bpmn": "BPMN", ".json": "JSON", ".yaml": "YAML", ".yml": "YAML"}
	if value := languages[ext]; value != "" {
		return value
	}
	return "Text"
}

func filepathSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }
