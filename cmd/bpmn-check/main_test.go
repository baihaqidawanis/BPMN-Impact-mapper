package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditReportsRelativeFilesAndParseFailures(t *testing.T) {
	dir := t.TempDir()
	valid := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><task id="T"/></process></definitions>`
	for name, body := range map[string]string{"a.bpmn": valid, "b.BPMN": "broken", "ignore.txt": "ignored"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	results, err := audit(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].File != "a.bpmn" || results[0].ParseFailed || !results[1].ParseFailed {
		t.Fatal(results)
	}
	if len(results[0].Findings) != 3 {
		t.Fatal(results[0])
	}
}
