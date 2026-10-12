// Package scenario provides deterministic experiment-input mutation tooling.
// It does not participate in impact inference and contains no case-study rules.
package scenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

const Version = "1"

type Spec struct {
	Version        string      `json:"version"`
	ScenarioID     string      `json:"scenario_id"`
	Description    string      `json:"description"`
	BaselineSHA256 string      `json:"baseline_sha256"`
	Operations     []Operation `json:"operations"`
}

type Operation struct {
	Kind      string `json:"kind"`
	ElementID string `json:"element_id"`
	Attribute string `json:"attribute,omitempty"`
	Child     string `json:"child,omitempty"`
	Value     string `json:"value,omitempty"`
	XML       string `json:"xml,omitempty"`
}

func DecodeSpec(data []byte) (Spec, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var spec Spec
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("decode mutation specification: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Spec{}, err
	}
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func (s Spec) Validate() error {
	if s.Version != Version {
		return fmt.Errorf("unsupported mutation specification version %q", s.Version)
	}
	if strings.TrimSpace(s.ScenarioID) == "" {
		return errors.New("scenario_id is required")
	}
	hash, err := hex.DecodeString(s.BaselineSHA256)
	if err != nil || len(hash) != sha256.Size {
		return errors.New("baseline_sha256 must be a 64-character hexadecimal SHA-256")
	}
	if len(s.Operations) == 0 {
		return errors.New("at least one mutation operation is required")
	}
	seen := map[string]bool{}
	removed := map[string]bool{}
	for i, operation := range s.Operations {
		operation.ElementID = strings.TrimSpace(operation.ElementID)
		if operation.ElementID == "" {
			return fmt.Errorf("operation %d: element_id is required", i)
		}
		switch operation.Kind {
		case "set_attribute":
			if strings.TrimSpace(operation.Attribute) == "" || operation.Child != "" {
				return fmt.Errorf("operation %d: set_attribute requires attribute and forbids child", i)
			}
		case "set_child_text":
			if strings.TrimSpace(operation.Child) == "" || operation.Attribute != "" {
				return fmt.Errorf("operation %d: set_child_text requires child and forbids attribute", i)
			}
		case "remove_element":
			if operation.Attribute != "" || operation.Child != "" || operation.Value != "" {
				return fmt.Errorf("operation %d: remove_element accepts only element_id", i)
			}
			removed[operation.ElementID] = true
		case "append_xml":
			if operation.Attribute != "" || operation.Child != "" || operation.Value != "" || strings.TrimSpace(operation.XML) == "" {
				return fmt.Errorf("operation %d: append_xml requires xml and forbids attribute, child, and value", i)
			}
			if err := validateFragment(operation.XML); err != nil {
				return fmt.Errorf("operation %d: append_xml: %w", i, err)
			}
		default:
			return fmt.Errorf("operation %d: unsupported kind %q", i, operation.Kind)
		}
		key := operation.Kind + "\x00" + operation.ElementID + "\x00" + operation.Attribute + "\x00" + operation.Child + "\x00" + operation.XML
		if seen[key] {
			return fmt.Errorf("operation %d duplicates an earlier operation", i)
		}
		seen[key] = true
	}
	for i, operation := range s.Operations {
		if operation.Kind != "remove_element" && removed[operation.ElementID] {
			return fmt.Errorf("operation %d mutates element %q that is also removed", i, operation.ElementID)
		}
	}
	return nil
}

func Mutate(baseline []byte, spec Spec) ([]byte, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(baseline)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), spec.BaselineSHA256) {
		return nil, fmt.Errorf("baseline SHA-256 mismatch: got %x", sum)
	}

	decoder := xml.NewDecoder(bytes.NewReader(baseline))
	decoder.Strict = true
	applied := make([]int, len(spec.Operations))
	var parentIDs []string
	type edit struct {
		start, end int
		value      []byte
	}
	var edits []edit

	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode baseline BPMN: %w", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			return nil, errors.New("XML directives are not allowed")
		case xml.StartElement:
			after := int(decoder.InputOffset())
			tagStart := before + bytes.IndexByte(baseline[before:after], '<')
			if tagStart < before {
				return nil, errors.New("cannot locate XML start tag")
			}
			id := attribute(value, "id")
			removeIndex := matchingRemove(spec.Operations, id)
			if removeIndex >= 0 {
				applied[removeIndex]++
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("remove element %q: %w", id, err)
				}
				edits = append(edits, edit{start: tagStart, end: int(decoder.InputOffset())})
				continue
			}
			mutatedTag := append([]byte(nil), baseline[tagStart:after]...)
			tagChanged := false
			for i, operation := range spec.Operations {
				if operation.Kind != "set_attribute" || operation.ElementID != id {
					continue
				}
				var found bool
				mutatedTag, found = replaceAttribute(mutatedTag, operation.Attribute, operation.Value)
				if !found {
					return nil, fmt.Errorf("element %q has no attribute %q", id, operation.Attribute)
				}
				tagChanged = true
				applied[i]++
			}
			if tagChanged {
				edits = append(edits, edit{start: tagStart, end: after, value: mutatedTag})
			}

			parentID := ""
			if len(parentIDs) > 0 {
				parentID = parentIDs[len(parentIDs)-1]
			}
			childIndex := matchingChild(spec.Operations, parentID, value.Name.Local)
			if childIndex >= 0 {
				contentStart := after
				var ignored string
				if err := decoder.DecodeElement(&ignored, &value); err != nil {
					return nil, fmt.Errorf("replace child text for %q: %w", parentID, err)
				}
				childEnd := int(decoder.InputOffset())
				closingRelative := bytes.LastIndexByte(baseline[contentStart:childEnd], '<')
				if closingRelative < 0 {
					return nil, fmt.Errorf("cannot locate closing child %q", value.Name.Local)
				}
				var escaped bytes.Buffer
				if err := xml.EscapeText(&escaped, []byte(spec.Operations[childIndex].Value)); err != nil {
					return nil, fmt.Errorf("escape replacement text: %w", err)
				}
				edits = append(edits, edit{start: contentStart, end: contentStart + closingRelative, value: escaped.Bytes()})
				applied[childIndex]++
				continue
			}
			parentIDs = append(parentIDs, id)
		case xml.EndElement:
			if len(parentIDs) == 0 {
				return nil, errors.New("malformed XML element stack")
			}
			parentIDs = parentIDs[:len(parentIDs)-1]
		}
	}
	for i, count := range applied {
		if spec.Operations[i].Kind == "append_xml" {
			continue
		}
		if count != 1 {
			return nil, fmt.Errorf("operation %d (%s on %q) applied %d times; expected exactly once", i, spec.Operations[i].Kind, spec.Operations[i].ElementID, count)
		}
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	generated := append([]byte(nil), baseline...)
	lastStart := len(generated)
	for _, current := range edits {
		if current.start < 0 || current.end < current.start || current.end > lastStart {
			return nil, errors.New("overlapping or invalid XML edits")
		}
		generated = append(append(append([]byte(nil), generated[:current.start]...), current.value...), generated[current.end:]...)
		lastStart = current.start
	}
	for i, operation := range spec.Operations {
		if operation.Kind != "append_xml" {
			continue
		}
		var err error
		generated, err = appendXML(generated, operation.ElementID, operation.XML)
		if err != nil {
			return nil, fmt.Errorf("operation %d: %w", i, err)
		}
	}
	if _, err := bpmn.Parse(generated, bpmn.Limits{}); err != nil {
		return nil, fmt.Errorf("generated BPMN is invalid: %w", err)
	}
	return append([]byte(nil), generated...), nil
}

func validateFragment(fragment string) error {
	decoder := xml.NewDecoder(strings.NewReader(`<root xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` + fragment + `</root>`))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := token.(xml.Directive); ok {
			return errors.New("XML directives are not allowed")
		}
	}
}

func appendXML(input []byte, elementID, fragment string) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(input))
	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("append target %q applied 0 times; expected exactly once", elementID)
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || attribute(start, "id") != elementID {
			continue
		}
		after := int(decoder.InputOffset())
		tagStart := before + bytes.IndexByte(input[before:after], '<')
		if tagStart < before {
			return nil, fmt.Errorf("cannot locate append target %q", elementID)
		}
		if err := decoder.Skip(); err != nil {
			return nil, err
		}
		end := int(decoder.InputOffset())
		closing := bytes.LastIndexByte(input[tagStart:end], '<')
		if closing < 0 {
			return nil, fmt.Errorf("cannot locate closing tag for append target %q", elementID)
		}
		insert := tagStart + closing
		return append(append(append([]byte(nil), input[:insert]...), []byte(fragment)...), input[insert:]...), nil
	}
}

func replaceAttribute(tag []byte, name, value string) ([]byte, bool) {
	for index := 0; index < len(tag); {
		position := bytes.Index(tag[index:], []byte(name))
		if position < 0 {
			return tag, false
		}
		position += index
		beforeOK := position > 0 && isXMLSpace(tag[position-1])
		after := position + len(name)
		for after < len(tag) && isXMLSpace(tag[after]) {
			after++
		}
		if !beforeOK || after >= len(tag) || tag[after] != '=' {
			index = position + len(name)
			continue
		}
		after++
		for after < len(tag) && isXMLSpace(tag[after]) {
			after++
		}
		if after >= len(tag) || (tag[after] != '\'' && tag[after] != '"') {
			return tag, false
		}
		quote := tag[after]
		valueStart := after + 1
		valueEndRelative := bytes.IndexByte(tag[valueStart:], quote)
		if valueEndRelative < 0 {
			return tag, false
		}
		valueEnd := valueStart + valueEndRelative
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(value)); err != nil {
			return tag, false
		}
		result := append([]byte(nil), tag[:valueStart]...)
		result = append(result, escaped.Bytes()...)
		result = append(result, tag[valueEnd:]...)
		return result, true
	}
	return tag, false
}

func isXMLSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func matchingRemove(operations []Operation, id string) int {
	for i, operation := range operations {
		if operation.Kind == "remove_element" && operation.ElementID == id {
			return i
		}
	}
	return -1
}

func matchingChild(operations []Operation, parentID, child string) int {
	for i, operation := range operations {
		if operation.Kind == "set_child_text" && operation.ElementID == parentID && operation.Child == child {
			return i
		}
	}
	return -1
}

func attribute(start xml.StartElement, local string) string {
	for _, candidate := range start.Attr {
		if candidate.Name.Local == local {
			return candidate.Value
		}
	}
	return ""
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("mutation specification contains multiple JSON values")
		}
		return fmt.Errorf("decode mutation specification: %w", err)
	}
	return nil
}
