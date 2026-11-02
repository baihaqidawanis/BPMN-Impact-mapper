package diff

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"sort"
	"strings"
)

// ExpandBaselineContext is an opt-in query experiment, not element matching.
// It preserves immediate neighbours and never crosses process/subprocess scope.
func ExpandBaselineContext(old, next *domain.Graph, changes []domain.Change, hops int) {
	if hops < 1 || hops > 4 {
		return
	}
	for i := range changes {
		c := &changes[i]
		if c.NewElement == nil || (c.ChangeType != domain.AddTask && c.ChangeType != domain.AddGateway) {
			continue
		}
		id := c.NewElement.ID
		e := next.Elements[id]
		if e == nil {
			continue
		}
		clustered := false
		for _, ids := range [][]string{next.Incoming[id], next.Outgoing[id]} {
			for _, fid := range ids {
				f := next.Flows[fid]
				other := f.TargetRef
				if other == id {
					other = f.SourceRef
				}
				if old.Elements[other] == nil {
					clustered = true
				}
			}
		}
		if !clustered {
			continue
		}
		anchors := []domain.BaselineAnchor{}
		for _, incoming := range []bool{true, false} {
			anchors = append(anchors, baselineTasks(old, next, e, incoming, hops)...)
		}
		if len(anchors) == 0 {
			continue
		}
		context := domain.ChangeContext{}
		if c.Context != nil {
			context = *c.Context
		}
		context.BaselineAnchors = anchors
		c.Context = &context
	}
}

func baselineTasks(old, next *domain.Graph, origin *domain.Element, incoming bool, maxHops int) []domain.BaselineAnchor {
	type entry struct {
		id   string
		path []string
	}
	queue := []entry{{origin.ID, nil}}
	seen := map[string]bool{origin.ID: true}
	anchors := []domain.BaselineAnchor{}
	for len(queue) > 0 && len(seen) <= 128 && len(anchors) < 2 {
		current := queue[0]
		queue = queue[1:]
		if len(current.path) >= maxHops {
			continue
		}
		ids := next.Outgoing[current.id]
		direction := "successor"
		if incoming {
			ids = next.Incoming[current.id]
			direction = "predecessor"
		}
		ids = append([]string(nil), ids...)
		sort.Strings(ids)
		for _, fid := range ids {
			f := next.Flows[fid]
			id := f.TargetRef
			if incoming {
				id = f.SourceRef
			}
			if seen[id] {
				continue
			}
			if len(seen) >= 128 {
				break
			}
			seen[id] = true
			e := next.Elements[id]
			if e == nil || e.ProcessID != origin.ProcessID || e.ParentID != origin.ParentID {
				continue
			}
			if e.Type == "subProcess" || e.Type == "callActivity" {
				continue
			}
			path := append(append([]string(nil), current.path...), fid)
			prior := old.Elements[id]
			if prior != nil && prior.Type == e.Type && prior.ProcessID == e.ProcessID && prior.ParentID == e.ParentID && strings.TrimSpace(prior.Name) != "" && (e.Type == "task" || strings.HasSuffix(e.Type, "Task")) {
				anchors = append(anchors, domain.BaselineAnchor{ElementContext: domain.ElementContext{ID: id, Name: prior.Name, Type: prior.Type}, Direction: direction, Hops: len(path), FlowPath: path})
				if len(anchors) == 2 {
					break
				}
				continue
			}
			queue = append(queue, entry{id, path})
		}
	}
	return anchors
}
