package relationprobe

import (
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// CallerFiles finds possible HTTP clients via bounded reverse source edges.
// Function seeds require a unique best >=2-word name match inside a ranked file.
// No labels/ground truth, retrieval scores, required-action inference or execution.
func CallerFiles(change domain.Change, artifacts []domain.Artifact) []domain.RelatedFile {
	if change.PartOf != "" || change.ChangeType == domain.UnresolvedMatch || change.ChangeType == domain.ModifySequenceFlow {
		return nil
	}
	query := map[string]bool{}
	for _, e := range []*domain.Element{change.OldElement, change.NewElement} {
		if e != nil {
			for _, w := range textutil.Tokenize(e.Name) {
				if len(w) >= 3 {
					query[w] = true
				}
			}
		}
	}
	byID := map[string]domain.Artifact{}
	byFile := map[string][]domain.Artifact{}
	type incoming struct {
		caller domain.Artifact
		edge   domain.SourceRelation
	}
	reverse := map[string][]incoming{}
	for _, a := range artifacts {
		byID[a.ID] = a
		byFile[a.File] = append(byFile[a.File], a)
	}
	for _, a := range artifacts {
		if testFile(a.File) {
			continue
		}
		for _, e := range a.Relations {
			if _, ok := byID[e.TargetID]; !ok {
				continue
			}
			if e.Kind != domain.RelationCalls && e.Kind != domain.RelationCallsMethod && e.Kind != domain.RelationInterfaceDispatch && e.Kind != domain.RelationConstructorCall && e.Kind != domain.RelationHTTPRoute && e.Kind != domain.RelationHTTPPattern {
				continue
			}
			reverse[e.TargetID] = append(reverse[e.TargetID], incoming{a, e})
		}
	}
	for key := range reverse {
		sort.Slice(reverse[key], func(i, j int) bool {
			a, b := reverse[key][i], reverse[key][j]
			if a.caller.ID != b.caller.ID {
				return a.caller.ID < b.caller.ID
			}
			return a.edge.Location < b.edge.Location
		})
	}
	found := map[string]domain.RelatedFile{}
	seedFiles := map[string]bool{}
	for rank, impact := range change.Impacts {
		if rank >= 5 {
			break
		}
		if seedFiles[impact.File] || testFile(impact.File) {
			continue
		}
		seedFiles[impact.File] = true
		var seed domain.Artifact
		best, ties := 1, 0
		for _, a := range byFile[impact.File] {
			if a.Symbol.Kind != "function" && a.Symbol.Kind != "method" {
				continue
			}
			n := 0
			for _, word := range textutil.UniqueSorted(textutil.Tokenize(a.Symbol.Name)) {
				if query[word] {
					n++
				}
			}
			if n > best {
				seed = a
				best = n
				ties = 1
			} else if n == best {
				ties++
			}
		}
		if seed.ID == "" || ties != 1 {
			continue
		}
		type entry struct {
			id       string
			depth    int
			evidence []domain.Evidence
		}
		queue := []entry{{id: seed.ID, evidence: []domain.Evidence{{Type: "SOURCE_CALLER_SEED", Location: seed.File, Excerpt: symbol(seed.Symbol) + "; unique two-word-or-more name match; association only"}}}}
		visited := map[string]bool{seed.ID: true}
		for len(queue) > 0 && len(visited) <= 128 {
			current := queue[0]
			queue = queue[1:]
			if current.depth >= 3 {
				continue
			}
			for _, link := range reverse[current.id] {
				if len(visited) >= 128 {
					break
				}
				if visited[link.caller.ID] {
					continue
				}
				visited[link.caller.ID] = true
				proof := append(append([]domain.Evidence(nil), current.evidence...), domain.Evidence{Type: "SOURCE_CALLER_EDGE", Location: link.edge.Location, Excerpt: link.edge.Excerpt})
				if link.edge.BindingLocation != "" {
					proof = append(proof, domain.Evidence{Type: "OBSERVED_CONSTRUCTOR_BINDING", Location: link.edge.BindingLocation, Excerpt: link.edge.BindingExcerpt})
				}
				if link.edge.Kind == domain.RelationConstructorCall && link.edge.TargetLocation != "" {
					proof = append(proof, domain.Evidence{Type: "OBSERVED_CONSTRUCTOR_RETURN", Location: link.edge.TargetLocation, Excerpt: link.edge.TargetExcerpt})
				}
				if link.edge.Kind == domain.RelationHTTPRoute || link.edge.Kind == domain.RelationHTTPPattern {
					proof = append(proof, domain.Evidence{Type: string(link.edge.Kind), Location: link.edge.TargetLocation, Excerpt: link.edge.TargetExcerpt})
					if snippet := callerExcerpt(change, link.caller); snippet.Excerpt != "" {
						proof = append(proof, snippet)
					}
					for i := range proof {
						r := []rune(proof[i].Excerpt)
						if len(r) > 240 {
							proof[i].Excerpt = string(r[:240])
						}
					}
					if link.caller.File != seed.File {
						if _, exists := found[link.caller.File]; !exists {
							found[link.caller.File] = domain.RelatedFile{File: link.caller.File, Module: link.caller.Module, Layer: link.caller.Layer, SeedFile: seed.File, SeedFunction: symbol(seed.Symbol), Hops: current.depth + 1, Evidence: proof}
						}
					}
					continue
				}
				if len(visited) <= 128 {
					queue = append(queue, entry{link.caller.ID, current.depth + 1, proof})
				}
			}
		}
	}
	result := make([]domain.RelatedFile, 0, len(found))
	for _, v := range found {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Hops != result[j].Hops {
			return result[i].Hops < result[j].Hops
		}
		return result[i].File < result[j].File
	})
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}

func testFile(file string) bool {
	v := "/" + strings.ToLower(file)
	return strings.Contains(v, "/test/") || strings.Contains(v, "/tests/") || strings.Contains(v, "/e2e/") || strings.HasSuffix(v, "_test.go") || strings.Contains(v, ".test.") || strings.Contains(v, ".spec.")
}

func callerExcerpt(c domain.Change, a domain.Artifact) domain.Evidence {
	query := ""
	for _, e := range []*domain.Element{c.OldElement, c.NewElement} {
		if e != nil {
			query += " " + e.Name
		}
	}
	if c.Context != nil {
		query += " " + strings.Join(c.Context.BranchTerms, " ")
	}
	words := map[string]bool{}
	for _, w := range textutil.Tokenize(query) {
		if len(w) >= 3 {
			words[w] = true
		}
	}
	best, line, text := 0, 0, ""
	for i, v := range strings.Split(a.Content, "\n") {
		n := 0
		for _, w := range textutil.UniqueSorted(textutil.Tokenize(v)) {
			if words[w] {
				n++
			}
		}
		if n > best {
			best = n
			line = i + 1
			text = strings.TrimSpace(v)
		}
	}
	if len([]rune(text)) > 240 {
		text = string([]rune(text)[:240])
	}
	return domain.Evidence{Type: "RELATED_SOURCE_EXCERPT", Location: fmt.Sprintf("%s:%d", a.File, line), Excerpt: text}
}
