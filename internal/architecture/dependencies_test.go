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
		"bpmnlib":     {"bpmn": true},
		"domain":      {},
		"mapping":     {"domain": true},
		"planning":    {"domain": true},
		"agentjobs":   {"domain": true, "planning": true, "planstore": true},
		"planstore":   {"planning": true},
		"evaluation":  {"domain": true},
		"app":         {"bpmn": true, "diff": true, "domain": true, "index": true, "ingest": true, "mapping": true, "report": true},
		"benchmark":   {"app": true, "domain": true, "evaluation": true, "mapping": true},
		"review":      {"evaluation": true},
		"reviewpack":  {"domain": true, "evaluation": true},
		"scenario":    {"bpmn": true},
		"reportstore": {"domain": true},
		"llm":         {"domain": true, "planning": true},
		"agenttools":  {"domain": true, "index": true, "ingest": true, "planning": true, "retrieval": true, "textutil": true},
		"filestore":   {"bpmn": true, "bpmnlib": true, "domain": true, "planning": true, "planstore": true, "report": true, "reportstore": true},
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
				if packageName == "domain" || packageName == "mapping" || packageName == "planning" || packageName == "agentjobs" {
					for _, external := range []string{"os", "net", "database/sql", "syscall", "unsafe"} {
						if path == external || strings.HasPrefix(path, external+"/") {
							t.Errorf("core package %s imports I/O dependency %s in %s", packageName, path, file)
						}
					}
				}
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
