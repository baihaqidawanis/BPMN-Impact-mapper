package planning

// MergeAgentEvidence treats a source window as an ordered unit: retain its beginning, not merely the last
// lines appended. Search snippets must not evict an inspected action boundary.
func MergeAgentEvidence(existing, incoming []EvidenceItem, sourceRead bool) []EvidenceItem {
	out := make([]EvidenceItem, 0, 8)
	seen := map[string]bool{}
	add := func(items []EvidenceItem) {
		for _, e := range items {
			if !seen[e.ID] && len(out) < 8 {
				out = append(out, e)
				seen[e.ID] = true
			}
		}
	}
	if !sourceRead {
		add(existing)
		add(incoming)
		return out
	}
	anchors := []EvidenceItem{}
	for _, e := range existing {
		if (e.Type == "SOURCE_DECLARATION" || e.Type == "SYMBOL") && len(anchors) < 2 {
			anchors = append(anchors, e)
		}
	}
	for _, e := range existing {
		if e.Type != "SOURCE_TOOL" && e.Type != "SOURCE_DECLARATION" && e.Type != "SYMBOL" && len(anchors) < 2 {
			anchors = append(anchors, e)
		}
	}
	add(anchors)
	add(incoming)
	add(existing)
	return out
}
