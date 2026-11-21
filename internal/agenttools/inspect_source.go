package agenttools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// inspectSource selects separated text spans across one approved file. It is
// contextual source inspection, not proof of dependencies or absent behavior.
func (s *Snapshot) inspectSource(ctx context.Context, q planning.ToolRequest) (planning.ToolResult, error) {
	out := planning.ToolResult{Candidates: []planning.Candidate{}}
	content, ok := s.files[q.File]
	if !safeFile(q.File) || !ok {
		return out, errors.New("unapproved source file")
	}
	terms := strings.FieldsFunc(strings.ToLower(q.Query), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var tokens []string
	seen := map[string]bool{}
	for _, v := range terms {
		if len(v) >= 3 && !seen[v] {
			tokens = append(tokens, v)
			seen[v] = true
		}
	}
	if len(tokens) == 0 {
		return out, errors.New("nonempty inspection query required")
	}
	// A literal dispatch case matching a query word provides a useful local
	// behavior window. This is a textual anchor, not a proven call graph.
	lines := strings.Split(content, "\n")
	// A single file may implement the same action on unrelated entities.
	// Score each declaration once and apply it only to its own lines.
	ownerScores := make([]int, len(lines))
	for _, a := range s.artifacts {
		sym := a.Symbol
		if a.File != q.File || sym.Name == "" || sym.StartLine < 1 {
			continue
		}
		name := strings.ToLower(sym.Name)
		score := 0
		for _, term := range tokens {
			if strings.Contains(name, term) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		for n := sym.StartLine - 1; n < min(sym.EndLine, len(lines)); n++ {
			ownerScores[n] = max(ownerScores[n], score)
		}
	}
	bestLine, bestScore := 0, -1
	for i, line := range lines {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		lower := strings.ToLower(strings.TrimSpace(line))
		for _, token := range tokens {
			if strings.HasPrefix(lower, "case ") && (strings.Contains(lower, `"`+token+`"`) || strings.Contains(lower, "'"+token+"'")) {
				score := ownerScores[i]
				if score > bestScore {
					bestScore, bestLine = score, i+1
				}
				break
			}
		}
	}
	if bestLine > 0 {
		result, err := s.readSource(ctx, planning.ToolRequest{Name: "read_source", File: q.File, StartLine: bestLine})
		result.Note = "Action-label text matched the query; inspect this local dispatch window. Other actions and callers are not proven by this excerpt."
		return result, err
	}

	type span struct {
		line, start, end, score int
		text                    string
	}
	var spans []span
	for i, line := range lines {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		runes := []rune(line)
		lower := []rune(strings.ToLower(line))
		for start := 0; start < len(runes); start += 200 {
			end := min(start+240, len(runes))
			text := string(lower[start:min(end, len(lower))])
			score := 0
			for _, token := range tokens {
				if strings.Contains(text, token) {
					score += min(3, strings.Count(text, token))
				}
			}
			if score > 0 {
				spans = append(spans, span{i + 1, start, end, score + ownerScores[i], string(runes[start:end])})
			}
			if end == len(runes) {
				break
			}
		}
	}
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].score != spans[j].score {
			return spans[i].score > spans[j].score
		}
		if spans[i].line != spans[j].line {
			return spans[i].line < spans[j].line
		}
		return spans[i].start < spans[j].start
	})
	// First distribute evidence across matching lines, then retain additional
	// spans of compressed lines. Avoid spending the entire budget on one line.
	chosen := []span{}
	used := map[int]bool{}
	selected := map[int]bool{}
	for i, v := range spans {
		if !used[v.line] && len(chosen) < 4 {
			chosen = append(chosen, v)
			used[v.line] = true
			selected[i] = true
		}
	}
	for len(chosen) < 6 {
		best, novelBest := -1, -1
		for i, v := range spans {
			if selected[i] {
				continue
			}
			novel := 0
			if used[v.line] {
				previous := ""
				for _, c := range chosen {
					if c.line == v.line {
						previous += strings.ToLower(c.text)
					}
				}
				for _, token := range tokens {
					if strings.Contains(strings.ToLower(v.text), token) && !strings.Contains(previous, token) {
						novel++
					}
				}
			}
			if novel > novelBest {
				best = i
				novelBest = novel
			}
		}
		if best < 0 {
			break
		}
		chosen = append(chosen, spans[best])
		selected[best] = true
		used[spans[best].line] = true
	}

	if len(chosen) == 0 {
		fallback, err := s.readSource(ctx, planning.ToolRequest{Name: "read_source", File: q.File, StartLine: 1})
		fallback.Note = "No inspection query terms matched; fallback shows file beginning only. Do not infer absence of behavior."
		return fallback, err
	}
	sort.Slice(chosen, func(i, j int) bool {
		if chosen[i].line != chosen[j].line {
			return chosen[i].line < chosen[j].line
		}
		return chosen[i].start < chosen[j].start
	})
	c := planning.Candidate{File: q.File, Origin: "agent-source-tool", SuggestedAction: domain.ActionVerify}
	for _, a := range s.artifacts {
		if a.File == q.File {
			c.Module = a.Module
			c.Layer = a.Layer
			break
		}
	}
	for _, v := range chosen {
		e := planning.EvidenceItem{Type: "SOURCE_TOOL", Location: fmt.Sprintf("%s:%d:%d", q.File, v.line, v.start+1), Excerpt: v.text}
		e.ID = digest(e)[:24]
		c.Evidence = append(c.Evidence, e)
	}
	out.Candidates = append(out.Candidates, c)
	out.Truncated = true
	return out, nil
}
