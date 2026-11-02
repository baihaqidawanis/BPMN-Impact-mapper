// bpmn-check inspects XML without executing or rewriting the inspected project.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

type fileResult struct {
	File        string         `json:"file"`
	SHA256      string         `json:"sha256"`
	ParseFailed bool           `json:"parse_failed,omitempty"`
	ParseError  string         `json:"parse_error,omitempty"`
	Findings    []bpmn.Finding `json:"findings"`
}

func main() {
	path := flag.String("path", "", "BPMN file or directory (read only)")
	research := flag.Bool("research-convention", false, "Check one start/end per scope and activity after join")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "--path is required")
		os.Exit(2)
	}
	results, err := audit(*path, *research)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, result := range results {
		if result.ParseFailed {
			os.Exit(1)
		}
	}
}

func audit(path string, research bool) ([]fileResult, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlink input is not supported")
	}
	base := root
	if !info.IsDir() {
		base = filepath.Dir(root)
	}
	results := []fileResult{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".bpmn") {
			return nil
		}
		name, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		result := fileResult{File: filepath.ToSlash(name), Findings: []bpmn.Finding{}}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, bpmn.DefaultMaxBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		graph, parseErr := bpmn.Parse(data, bpmn.Limits{})
		result.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		if parseErr != nil {
			result.ParseFailed = true
			result.ParseError = parseErr.Error()
		} else {
			result.Findings = bpmn.Check(graph, bpmn.CheckOptions{ResearchConvention: research})
		}
		results = append(results, result)
		return nil
	})
	if err == nil && len(results) == 0 {
		err = fmt.Errorf("no BPMN files found")
	}
	return results, err
}
