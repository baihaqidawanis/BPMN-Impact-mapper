import type { AnalysisMethod, Impact } from "./types";

function fusionFields(impact: Impact): Record<string, string> | null {
  const evidence = impact.evidence.find(item => item.type === "RRF_FUSION");
  return evidence ? Object.fromEntries(evidence.excerpt.split(";").map(part => part.trim().split("="))) : null;
}

export function hasFusion(impact: Impact): boolean {
  return !!impact.score_breakdown.rrf || fusionFields(impact) !== null;
}

// A semantic-only or propagated candidate has no direct evidence score to explain.
export function usesCodeEvidence(impact: Impact, method: AnalysisMethod): boolean {
  if (method !== "hybrid" || impact.evidence.some(item => item.type === "PROPAGATION_SEED")) return false;
  const fields = fusionFields(impact);
  return !fields || (fields.left_method === "hybrid" && /^[1-9]\d*$/.test(fields.left_rank ?? ""));
}

export function traceExplanation(impact: Impact, method: AnalysisMethod, changeType: string): { status: string; detail: string } {
  if (impact.evidence.some(item => item.type === "TRACE_ID")) {
    return { status: "Direct BPMN ID found", detail: "This candidate contains the ID of the changed element. This links the candidate to the process; the required action still needs supporting evidence." };
  }
  if (impact.evidence.some(item => item.type === "CONTEXT_TRACE_ID")) {
    return { status: "Neighbouring BPMN ID found", detail: "An ID from the preceding or following BPMN element appears here. It explains the process connection, but is not a direct match for the changed element and adds no separate score." };
  }
  if (impact.evidence.some(item => item.type === "PROPAGATION_SEED")) {
    return { status: "Linked through another function", detail: "A source-code relation connects this candidate to a directly traced function. The changed BPMN ID is not directly evidenced here." };
  }
  if (!usesCodeEvidence(impact, method)) {
    return { status: "No direct ID evidence recorded", detail: "This result does not include a direct BPMN-ID check from the code-evidence ranking. A zero in the exported field is not proof that the ID is absent." };
  }
  if (changeType === "ADD_TASK" || changeType === "ADD_GATEWAY") {
    return { status: "New BPMN ID not found here", detail: "A newly added element often has no ID in the existing implementation yet. Other words, process context and semantic similarity can still identify relevant files." };
  }
  return { status: "Changed BPMN ID not found here", detail: "No direct ID match was found in this candidate's indexed text. Software does not need to store BPMN IDs; this alone does not rule out an impact." };
}

// Explain recorded ranks, never infer agreement from the fused score alone.
export function rankingExplanation(impact: Impact): string {
  const fields = fusionFields(impact);
  if (!fields) return "The score orders possible impacts. It does not measure the probability that code must change.";
  const method = (name: string) => ({ hybrid: "Code-evidence search", bm25: "BM25", semantic: "Semantic search" }[name] ?? "Search");
  const describe = (side: string) => {
    const rank = fields[`${side}_rank`];
    const name = method(fields[`${side}_method`]);
    if (rank === "absent") return `${name}: absent from the ranked list.`;
    if (/^[1-9]\d*$/.test(rank ?? "")) return `${name}: rank ${rank}.`;
    return `${name}: rank unavailable.`;
  };
  return `${describe("left")} ${describe("right")} The combined score uses these positions, not the sum of the clue scores. A high score is possible even without a direct BPMN ID match. It is not a probability that code must change.`;
}
