package bpmn

import (
	"encoding/xml"
	"fmt"
)

// Parsing a tag is not a claim of token semantics, external-reference resolution
// or complete semantic differencing. These findings describe known omissions.
func capabilityWarning(start xml.StartElement, owner string) string {
	if start.Name.Space != "" && start.Name.Space != "http://www.omg.org/spec/BPMN/20100524/MODEL" {
		return ""
	}
	local := start.Name.Local
	reason := ""
	switch local {
	case "standardLoopCharacteristics", "multiInstanceLoopCharacteristics":
		reason = "loop/multi-instance execution characteristics are not compared"
	case "messageFlow":
		reason = "message-flow communication dependencies are not compared"
	case "eventBasedGateway", "complexGateway":
		reason = "gateway type/topology is read, but event race and complex activation rules are not modelled"
	case "callActivity":
		reason = "call activity is read, but calledElement references are not resolved or compared; referenced models are not loaded"
	case "subProcess":
		if attr(start, "triggeredByEvent") == "true" {
			reason = "event-subprocess triggering semantics are not compared"
		}
	case "boundaryEvent":
		reason = "boundary attachment is read, but interruption/cancellation semantics are not compared"
	default:
		if isEventDefinition(local) {
			reason = "event definition type is read, but trigger payload, timing and reference parameters are not compared"
		}
	}
	if reason == "" && attr(start, "isForCompensation") == "true" {
		reason = "compensation activity execution semantics are not compared"
	}
	if reason == "" {
		return ""
	}
	id := attr(start, "id")
	if id == "" {
		id = owner
	}
	if len(id) > 120 {
		id = id[:120] + "..."
	}
	return fmt.Sprintf("%s %q: %s. Diff may omit changes in these features.", local, id, reason)
}
