package bpmnlib

import (
	"errors"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

type Activity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Link struct {
	ParentID  string `json:"parent_id"`
	ElementID string `json:"element_id"`
	TargetID  string `json:"target_id"`
}

type LinkStore interface {
	Links(string) ([]Link, error)
	SaveLink(Link) error
	DeleteLink(string, string) error
}

func Activities(entry Entry) ([]Activity, error) {
	g, err := bpmn.Parse([]byte(entry.XML), bpmn.Limits{})
	if err != nil {
		return nil, err
	}
	result := []Activity{}
	for _, e := range g.Elements {
		if e.Type == "task" || strings.HasSuffix(e.Type, "Task") || e.Type == "callActivity" || e.Type == "subProcess" {
			result = append(result, Activity{e.ID, e.Name, e.Type})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// ValidateLink checks explicit decomposition; names and calledElement do not
// silently create links. Strictly increasing levels prevent cycles.
func ValidateLink(parent Entry, element string, target *Entry) error {
	activities, err := Activities(parent)
	if err != nil {
		return err
	}
	found := false
	for _, a := range activities {
		if a.ID == element {
			found = true
			break
		}
	}
	if !found {
		return errors.New("select an activity in the parent diagram")
	}
	if target != nil && (target.Repository != parent.Repository || target.Level <= parent.Level || target.ID == parent.ID) {
		return errors.New("detail diagram must belong to the same repository label and a deeper process level")
	}
	return nil
}
