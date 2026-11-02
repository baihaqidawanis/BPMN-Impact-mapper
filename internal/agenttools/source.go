// Package agenttools adapts approved ingestion/index snapshots to read-only agent
// tools. No live filesystem reads, repository execution or network after New.
package agenttools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type Snapshot struct {
	artifacts []domain.Artifact
	identity  string
	files     map[string]string
}

func digest(v any) string {
	data, _ := json.Marshal(v)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func New(ctx context.Context, repo string, options ingest.Options) (*Snapshot, error) {
	inv, err := ingest.Build(ctx, repo, options)
	if err != nil {
		return nil, err
	}
	registry := index.SourceContextRegistry()
	result := registry.Build(inv)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Inventory warnings can mean incomplete input. Fail closed for agent tools;
	// callers may choose a different approved scope rather than hide omissions.
	if len(inv.Warnings) > 0 {
		return nil, errors.New("source tool inventory has exclusions or warnings; approve a clean scope first")
	}
	identity := digest(struct {
		ToolVersion string
		Registry    string
		Artifacts   []domain.Artifact
		Warnings    []string
	}{"agent-source-tools-v6-full-chunks", registry.Identity(), result.Artifacts, result.Warnings})
	files := make(map[string]string, len(inv.Files))
	for _, file := range inv.Files {
		files[file.RelativePath] = string(file.Content)
	}
	return &Snapshot{artifacts: result.Artifacts, identity: identity, files: files}, nil
}

func (s *Snapshot) Identity() string { return s.identity }

// CheckBundle prevents old/invented seed files or named functions from entering
// a live session. It does not prove an older report's every excerpt is current;
// callers must use an analysis produced against this source revision.
func (s *Snapshot) CheckBundle(bundle planning.EvidenceBundle) error {
	for _, change := range bundle.Changes {
		for _, c := range change.Candidates {
			found := false
			for _, a := range s.artifacts {
				if a.File == c.File && (c.Symbol == "" || matchesSourceName(a, c.Symbol)) {
					found = true
					break
				}
			}
			if !found {
				return errors.New("planning seed file/function absent from approved source snapshot; rerun analysis")
			}
		}
	}
	return nil
}

func safeFile(file string) bool {
	if file == "" || strings.ContainsAny(file, "\\:\x00") || strings.HasPrefix(file, "/") {
		return false
	}
	for _, part := range strings.Split(file, "/") {
		if part == ".." || part == "." || part == "" {
			return false
		}
	}
	return true
}

func (s *Snapshot) Execute(ctx context.Context, q planning.ToolRequest) (planning.ToolResult, error) {
	out := planning.ToolResult{Candidates: []planning.Candidate{}}
	if len(q.Query) > 200 || len(q.File) > 500 || len(q.Symbol) > 200 {
		return out, errors.New("tool arguments too large")
	}
	if q.Offset < 0 || q.Offset > 10000 {
		return out, errors.New("search offset outside bounds")
	}
	if q.Name == "read_source" {
		return s.readSource(ctx, q)
	}
	if q.Name == "inspect_source" {
		return s.inspectSource(ctx, q)
	}
	if q.Name != "search_source" && (!safeFile(q.File) || q.Symbol == "") {
		return out, errors.New("exact relative file/function required")
	}
	var target string
	if q.Name == "find_callers" {
		for _, a := range s.artifacts {
			if a.File == q.File && matchesSourceName(a, q.Symbol) {
				if target != "" {
					return out, errors.New("ambiguous target")
				}
				target = a.ID
			}
		}
		if target == "" {
			return out, errors.New("unknown target")
		}
	}
	type hit struct {
		a                 domain.Artifact
		score             int
		location, excerpt string
	}
	var hits []hit
	tokens := strings.Fields(strings.ToLower(q.Query))
	if q.Name == "search_source" && (len(tokens) == 0 || len(tokens) > 12) {
		return out, errors.New("bounded nonempty search query required")
	}
	for _, a := range s.artifacts {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if !safeFile(a.File) {
			continue
		}
		h := hit{a: a, location: fmt.Sprintf("%s:%d", a.File, max(1, a.Symbol.StartLine)), excerpt: a.Content}
		switch q.Name {
		case "search_source":
			text := strings.ToLower(a.File + " " + a.Symbol.Name + " " + a.Content)
			for _, token := range tokens {
				if strings.Contains(text, token) {
					h.score++
				}
			}
			if h.score == 0 {
				continue
			}
			for i, line := range strings.Split(a.Content, "\n") {
				match := false
				for _, token := range tokens {
					if strings.Contains(strings.ToLower(line), token) {
						match = true
					}
				}
				if match {
					h.location = fmt.Sprintf("%s:%d", a.File, max(1, a.Symbol.StartLine)+i)
					h.excerpt = line
					break
				}
			}
		case "read_function":
			if a.File != q.File || !matchesSourceName(a, q.Symbol) || (a.Symbol.Kind != "function" && a.Symbol.Kind != "method") {
				continue
			}
		case "find_callers":
			found := false
			for _, r := range a.Relations {
				if r.TargetID == target {
					h.location = r.Location
					h.excerpt = r.Excerpt
					found = true
					break
				}
			}
			if !found {
				continue
			}
		default:
			return out, errors.New("unknown source tool")
		}
		hits = append(hits, h)
	}
	if q.Name == "read_function" && len(hits) != 1 {
		return out, errors.New("unknown or ambiguous function")
	}
	// A Go package artifact represents whole-file text, not another function.
	// Prefer named hits in that same file; retain file-level fallback otherwise.
	specific := map[string]bool{}
	for _, h := range hits {
		if h.a.Symbol.Kind == "function" || h.a.Symbol.Kind == "method" || h.a.Symbol.Kind == "type" {
			specific[h.a.File] = true
		}
	}
	filtered := hits[:0]
	for _, h := range hits {
		if h.a.Symbol.Kind == "package" && specific[h.a.File] {
			continue
		}
		if h.a.Symbol.Kind == "package" {
			h.a.Symbol.Name = ""
		} else if h.a.Symbol.Receiver != "" {
			h.a.Symbol.Name = h.a.Symbol.Receiver + "." + h.a.Symbol.Name
		}
		filtered = append(filtered, h)
	}
	hits = filtered
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		if hits[i].a.File != hits[j].a.File {
			return hits[i].a.File < hits[j].a.File
		}
		return hits[i].a.Symbol.Name < hits[j].a.Symbol.Name
	})
	if q.Name == "search_source" {
		// Different functions in the same file must not fill a repository
		// discovery page. A later offset can reach beyond the initial results.
		unique := hits[:0]
		seenFiles := map[string]bool{}
		for _, h := range hits {
			if !seenFiles[h.a.File] {
				unique = append(unique, h)
				seenFiles[h.a.File] = true
			}
		}
		hits = unique
		if q.Offset >= len(hits) {
			hits = nil
		} else {
			hits = hits[q.Offset:]
		}
	}
	if len(hits) > 3 {
		hits = hits[:3]
		out.Truncated = true
	}
	for _, h := range hits {
		runes := []rune(strings.TrimSpace(h.excerpt))
		if len(runes) > 240 {
			runes = runes[:240]
			out.Truncated = true
		}
		e := planning.EvidenceItem{Type: "SOURCE_TOOL", Location: h.location, Excerpt: string(runes)}
		e.ID = digest(e)[:24]
		out.Candidates = append(out.Candidates, planning.Candidate{File: h.a.File, Symbol: h.a.Symbol.Name, Module: h.a.Module, Layer: h.a.Layer, SuggestedAction: domain.ActionVerify, Origin: "agent-source-tool", Evidence: []planning.EvidenceItem{e}})
	}
	return out, nil
}

func matchesSourceName(a domain.Artifact, name string) bool {
	return a.Symbol.Name == name || (a.Symbol.Receiver != "" && a.Symbol.Receiver+"."+a.Symbol.Name == name)
}
