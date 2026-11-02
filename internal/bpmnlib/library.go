// Package bpmnlib owns local versioned BPMN library contracts and validation.
package bpmnlib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	xmlcodec "encoding/xml"
	"errors"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

var ErrNotFound = errors.New("diagram not found")

type Metadata struct {
	Repository string `json:"repository"`
	Process    string `json:"process"`
	Role       string `json:"role"`
	Level      int    `json:"level"`
	Version    string `json:"version"`
}

type Entry struct {
	ID string `json:"id"`
	Metadata
	Findings   []bpmn.Finding `json:"findings"`
	XML        string         `json:"xml,omitempty"`
	Activities []Activity     `json:"activities,omitempty"`
}

// Store implementations persist immutable entries and must not use metadata as paths.
type Store interface {
	Put(Entry) error
	Get(string) (Entry, error)
	List() ([]Entry, error)
}

func Prepare(meta Metadata, xml string) (Entry, error) {
	meta.Repository = strings.TrimSpace(meta.Repository)
	meta.Process = strings.TrimSpace(meta.Process)
	meta.Role = strings.TrimSpace(meta.Role)
	meta.Version = strings.TrimSpace(meta.Version)
	for _, field := range []string{meta.Repository, meta.Process, meta.Role, meta.Version} {
		if len(field) > 160 || strings.ContainsAny(field, "\x00\r\n") {
			return Entry{}, errors.New("metadata must be single-line text up to 160 bytes")
		}
	}
	if meta.Repository == "" || meta.Process == "" || meta.Version == "" || meta.Level < 0 || meta.Level > 3 {
		return Entry{}, errors.New("repository, process, version and level 0-3 are required")
	}
	graph, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		return Entry{}, err
	}
	findings := bpmn.Check(graph, bpmn.CheckOptions{ResearchConvention: true})
	decoder := xmlcodec.NewDecoder(strings.NewReader(xml))
	hasLayout := false
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if start, ok := token.(xmlcodec.StartElement); ok && start.Name.Space == "http://www.omg.org/spec/BPMN/20100524/DI" && start.Name.Local == "BPMNDiagram" {
			hasLayout = true
			break
		}
	}
	if !hasLayout {
		findings = append(findings, bpmn.Finding{Code: "NO_DIAGRAM_LAYOUT", Category: "capability", Message: "No BPMN diagram layout is included. Mapping can analyze the process, but Canvas needs shapes and coordinates exported by a BPMN editor."})
	}
	// Additional project convention: any direct gateway-to-end/gateway edge.
	for _, flow := range graph.Flows {
		source, target := graph.Elements[flow.SourceRef], graph.Elements[flow.TargetRef]
		if source != nil && target != nil && strings.HasSuffix(source.Type, "Gateway") && (strings.HasSuffix(target.Type, "Gateway") || target.Type == "endEvent") {
			findings = append(findings, bpmn.Finding{Code: "ACTIVITY_AFTER_GATEWAY", Category: "convention", ElementID: source.ID, Scope: source.ProcessID, Message: "Flow " + flow.ID + " directly connects a gateway to " + target.Type + ". Review the local modelling convention; do not invent an activity."})
		}
	}
	// Map iteration must not affect stored content or UI order.
	sortFindings(findings)
	encoded, _ := json.Marshal(struct {
		Metadata
		XML string
	}{meta, xml})
	sum := sha256.Sum256(encoded)
	return Entry{ID: hex.EncodeToString(sum[:]), Metadata: meta, XML: xml, Findings: findings}, nil
}
