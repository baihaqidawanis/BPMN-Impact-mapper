export interface VisualDiff {
  added: string[];
  removed: string[];
  modified: string[];
}

const visibleTypes = new Set([
  "task", "userTask", "serviceTask", "scriptTask", "manualTask", "businessRuleTask",
  "sendTask", "receiveTask", "subProcess", "callActivity", "exclusiveGateway",
  "parallelGateway", "inclusiveGateway", "eventBasedGateway", "complexGateway",
  "startEvent", "endEvent", "intermediateCatchEvent", "intermediateThrowEvent",
  "boundaryEvent", "sequenceFlow",
]);

function elements(xml: string): Map<string, string> {
  const document = new DOMParser().parseFromString(xml, "application/xml");
  if (document.querySelector("parsererror")) throw new Error("BPMN XML is invalid.");
  const lanes = new Map<string, string>();
  for (const lane of document.getElementsByTagNameNS("http://www.omg.org/spec/BPMN/20100524/MODEL", "lane")) {
    for (const ref of lane.getElementsByTagNameNS("http://www.omg.org/spec/BPMN/20100524/MODEL", "flowNodeRef")) {
      lanes.set(ref.textContent?.trim() ?? "", lane.getAttribute("id") ?? "");
    }
  }
  const result = new Map<string, string>();
  const nodes = document.getElementsByTagNameNS("http://www.omg.org/spec/BPMN/20100524/MODEL", "*");
  for (const node of nodes) {
    if (!visibleTypes.has(node.localName)) continue;
    const id = node.getAttribute("id");
    if (!id) continue;
    const condition = node.getElementsByTagNameNS("http://www.omg.org/spec/BPMN/20100524/MODEL", "conditionExpression")[0]?.textContent?.trim() ?? "";
    result.set(id, JSON.stringify([
      node.localName, node.getAttribute("name") ?? "", lanes.get(id) ?? "",
      node.getAttribute("sourceRef") ?? "", node.getAttribute("targetRef") ?? "", condition,
    ]));
  }
  return result;
}

export function compareBpmnVisuals(beforeXml: string, afterXml: string): VisualDiff {
  const before = elements(beforeXml);
  const after = elements(afterXml);
  const added: string[] = [];
  const removed: string[] = [];
  const modified: string[] = [];
  for (const [id, value] of after) {
    if (!before.has(id)) added.push(id);
    else if (before.get(id) !== value) modified.push(id);
  }
  for (const id of before.keys()) if (!after.has(id)) removed.push(id);
  added.sort(); removed.sort(); modified.sort();
  return { added, removed, modified };
}
