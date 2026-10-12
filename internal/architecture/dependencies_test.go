package architecture_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInnerPackagesDoNotDependOnOuterAdapters(t *testing.T) {
	allowed := map[string]map[string]bool{
		"domain":      {},
		"mapping":     {"domain": true},
		"evaluation":  {"domain": true},
		"app":         {"bpmn": true, "diff": true, "domain": true, "index": true, "ingest": true, "mapping": true, "report": true},
		"benchmark":   {"app": true, "domain": true, "evaluation": true, "mapping": true},
		"review":      {"evaluation": true},
		"reviewpack":  {"domain": true, "evaluation": true},
		"scenario":    {"bpmn": true},
		"reportstore": {"domain": true},
	}
	for packageName, allowedInternal := range allowed {
		files, err := filepath.Glob(filepath.Join("..", packageName, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, spec := range parsed.Imports {
				path, _ := strconv.Unquote(spec.Path.Value)
				marker := "/internal/"
				position := strings.Index(path, marker)
				if position < 0 {
					continue
				}
				dependency := strings.Split(strings.TrimPrefix(path[position:], marker), "/")[0]
				if !allowedInternal[dependency] {
					t.Errorf("inner package %s imports outer/internal package %s in %s", packageName, dependency, file)
				}
			}
		}
	}
}
