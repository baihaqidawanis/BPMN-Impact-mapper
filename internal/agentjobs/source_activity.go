package agentjobs

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"

// SourceActivity records tool I/O, not hidden reasoning or proof of comprehension.
type SourceActivity struct {
	ChangeIndex int      `json:"change_index"`
	Tool        string   `json:"tool"`
	Query       string   `json:"query,omitempty"`
	File        string   `json:"file,omitempty"`
	Function    string   `json:"function,omitempty"`
	Files       []string `json:"files"`
	Error       string   `json:"error,omitempty"`
	Truncated   bool     `json:"truncated,omitempty"`
	NextLine    int      `json:"next_line,omitempty"`
}

func childActivity(parent, child planning.AgentSession, fallbackIndex int) []SourceActivity {
	if len(child.Bundle.Changes) == 1 {
		target := child.Bundle.Changes[0]
		for i, c := range parent.Bundle.Changes {
			if c.ChangeType == target.ChangeType && c.ElementID == target.ElementID {
				fallbackIndex = i
				break
			}
		}
	}
	items := sourceActivity(child.Observations)
	for i := range items {
		items[i].ChangeIndex = fallbackIndex
	}
	return items
}

func sourceActivity(observations []planning.ToolObservation) []SourceActivity {
	items := make([]SourceActivity, 0, len(observations))
	for _, o := range observations {
		item := SourceActivity{ChangeIndex: o.Request.ChangeIndex, Tool: o.Request.Name, Query: o.Request.Query, File: o.Request.File, Function: o.Request.Symbol, Files: []string{}, Error: o.Result.Error, Truncated: o.Result.Truncated, NextLine: o.Result.NextLine}
		seen := map[string]bool{}
		for _, c := range o.Result.Candidates {
			if !seen[c.File] {
				item.Files = append(item.Files, c.File)
				seen[c.File] = true
			}
		}
		items = append(items, item)
	}
	return items
}
