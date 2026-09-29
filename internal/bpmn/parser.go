package bpmn

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

const (
	DefaultMaxBytes    = int64(8 << 20)
	DefaultMaxDepth    = 128
	DefaultMaxElements = 100000
)

type Limits struct {
	MaxBytes    int64
	MaxDepth    int
	MaxElements int
}

func Parse(data []byte, limits Limits) (*domain.Graph, error) {
	if limits.MaxBytes <= 0 {
		limits.MaxBytes = DefaultMaxBytes
	}
	if limits.MaxDepth <= 0 {
		limits.MaxDepth = DefaultMaxDepth
	}
	if limits.MaxElements <= 0 {
		limits.MaxElements = DefaultMaxElements
	}
	if int64(len(data)) > limits.MaxBytes {
		return nil, fmt.Errorf("BPMN exceeds byte limit %d", limits.MaxBytes)
	}
	decoder := xml.NewDecoder(io.LimitReader(bytes.NewReader(data), limits.MaxBytes+1))
	decoder.Strict = true
	graph := &domain.Graph{
		Elements: map[string]*domain.Element{}, Flows: map[string]*domain.Flow{},
		Incoming: map[string][]string{}, Outgoing: map[string][]string{}, Warnings: []string{},
	}
	lanes := map[string]string{}
	var currentElement *domain.Element
	var currentFlow *domain.Flow
	var currentLane string
	depth, elementCount, processCount := 0, 0, 0
	seenDefinitions := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode BPMN XML: %w", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			return nil, errors.New("XML directives are not allowed")
		case xml.StartElement:
			depth++
			elementCount++
			if depth > limits.MaxDepth {
				return nil, fmt.Errorf("BPMN XML depth exceeds %d", limits.MaxDepth)
			}
			if elementCount > limits.MaxElements {
				return nil, fmt.Errorf("BPMN XML element limit exceeds %d", limits.MaxElements)
			}
			local := value.Name.Local
			switch local {
			case "definitions":
				seenDefinitions = true
			case "process":
				processCount++
			case "lane":
				currentLane = attr(value, "name")
			case "flowNodeRef":
				ref, err := decodeText(decoder, value)
				if err != nil {
					return nil, err
				}
				if currentLane != "" && ref != "" {
					lanes[ref] = currentLane
				}
				depth--
			case "sequenceFlow":
				flow := &domain.Flow{ID: attr(value, "id"), Name: attr(value, "name"), SourceRef: attr(value, "sourceRef"), TargetRef: attr(value, "targetRef")}
				if flow.ID == "" || flow.SourceRef == "" || flow.TargetRef == "" {
					return nil, errors.New("sequenceFlow requires id, sourceRef, and targetRef")
				}
				if _, exists := graph.Flows[flow.ID]; exists {
					return nil, fmt.Errorf("duplicate BPMN ID %q", flow.ID)
				}
				graph.Flows[flow.ID] = flow
				currentFlow = flow
			case "conditionExpression":
				condition, err := decodeText(decoder, value)
				if err != nil {
					return nil, err
				}
				if currentFlow != nil {
					currentFlow.Condition = strings.TrimSpace(condition)
				}
				depth--
			case "documentation":
				documentation, err := decodeText(decoder, value)
				if err != nil {
					return nil, err
				}
				if currentElement != nil {
					currentElement.Documentation = strings.TrimSpace(documentation)
				}
				depth--
			case "sourceRef", "targetRef":
				if currentElement != nil {
					ref, err := decodeText(decoder, value)
					if err != nil {
						return nil, err
					}
					currentElement.DataRequirements = append(currentElement.DataRequirements, strings.TrimSpace(ref))
					depth--
				}
			case "dataState":
				if currentElement != nil {
					currentElement.State = strings.TrimSpace(attr(value, "name"))
				}
			default:
				if supportedElement(local) {
					element := &domain.Element{ID: attr(value, "id"), Name: strings.TrimSpace(attr(value, "name")), Type: local, DataRequirements: []string{}, Incoming: []string{}, Outgoing: []string{}}
					if element.ID == "" {
						return nil, fmt.Errorf("%s requires an id", local)
					}
					if _, exists := graph.Elements[element.ID]; exists {
						return nil, fmt.Errorf("duplicate BPMN ID %q", element.ID)
					}
					graph.Elements[element.ID] = element
					currentElement = element
				}
			}
		case xml.EndElement:
			if currentElement != nil && value.Name.Local == currentElement.Type {
				currentElement = nil
			}
			if value.Name.Local == "sequenceFlow" {
				currentFlow = nil
			}
			if value.Name.Local == "lane" {
				currentLane = ""
			}
			depth--
		}
	}
	if !seenDefinitions || processCount == 0 {
		return nil, errors.New("BPMN definitions with at least one process are required")
	}
	for id, actor := range lanes {
		if element := graph.Elements[id]; element != nil {
			element.Actor = actor
		}
	}
	for id, flow := range graph.Flows {
		if graph.Elements[flow.SourceRef] == nil || graph.Elements[flow.TargetRef] == nil {
			return nil, fmt.Errorf("flow %q references missing endpoint", id)
		}
		graph.Outgoing[flow.SourceRef] = append(graph.Outgoing[flow.SourceRef], id)
		graph.Incoming[flow.TargetRef] = append(graph.Incoming[flow.TargetRef], id)
	}
	for id, element := range graph.Elements {
		sort.Strings(graph.Incoming[id])
		sort.Strings(graph.Outgoing[id])
		element.Incoming = append([]string(nil), graph.Incoming[id]...)
		element.Outgoing = append([]string(nil), graph.Outgoing[id]...)
		element.DataRequirements = textutil.UniqueSorted(element.DataRequirements)
	}
	return graph, nil
}

func decodeText(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var text string
	if err := decoder.DecodeElement(&text, &start); err != nil {
		return "", fmt.Errorf("decode %s: %w", start.Name.Local, err)
	}
	return strings.TrimSpace(text), nil
}

func attr(start xml.StartElement, local string) string {
	for _, attribute := range start.Attr {
		if attribute.Name.Local == local {
			return attribute.Value
		}
	}
	return ""
}

func supportedElement(local string) bool {
	switch local {
	case "task", "userTask", "serviceTask", "manualTask", "scriptTask", "businessRuleTask", "sendTask", "receiveTask", "callActivity", "subProcess",
		"startEvent", "endEvent", "intermediateCatchEvent", "intermediateThrowEvent", "boundaryEvent",
		"exclusiveGateway", "parallelGateway", "inclusiveGateway", "complexGateway", "eventBasedGateway",
		"dataObjectReference", "dataStoreReference":
		return true
	default:
		return false
	}
}
