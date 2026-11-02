import type { AnalysisMethod, Impact } from "./types";

function fusionFields(impact: Impact): Record<string, string> | null {
  const evidence = impact.evidence.find(item => item.type === "RRF_FUSION");
  return evidence ? Object.fromEntries(evidence.excerpt.split(";").map(part => part.trim().split("="))) : null;
}

export function hasFusion(impact: Impact): boolean {
  return !!impact.score_breakdown.rrf || fusionFields(impact) !== null;
}

export function fusionCalculation(impact: Impact): string | null {
  const fields = fusionFields(impact);
  if (!fields) return null;
  const k = Number(fields.k);
  if (!Number.isInteger(k) || k < 1) return null;
  const term = (value: string | undefined): string | null => value === "absent" ? "0" : /^[1-9]\d*$/.test(value ?? "") ? `1/(${k}+${value})` : null;
  const left = term(fields.left_rank), right = term(fields.right_rank);
  if (left === null || right === null) return null;
  return `RRF = (${left} + ${right}) / (2/(${k}+1)). Each search contributes equally. A missing candidate contributes 0; rank 1 in both searches gives 1. The displayed result is rounded. This is ranking agreement, not accuracy.`;
}

// A semantic-only or propagated candidate has no direct evidence score to explain.
export function usesCodeEvidence(impact: Impact, method: AnalysisMethod): boolean {
  return method === "hybrid" && impact.score_breakdown.evidence_score !== undefined;
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
  if (method !== "hybrid" || !impact.evidence.some(item => item.type === "BM25")) {
    return { status: "No direct ID evidence recorded", detail: "This candidate has no recorded direct BPMN ID match; that alone does not rule out an impact." };
  }
  if (changeType === "ADD_TASK" || changeType === "ADD_GATEWAY") {
    return { status: "New BPMN ID not yet available in source", detail: "Expected for a new task or gateway. The ID is an optional trace clue, not a penalty in the BM25 or semantic ranking." };
  }
  return { status: "Changed BPMN ID not found here", detail: "No direct ID match was found in this candidate's indexed text. Software does not need to store BPMN IDs; this alone does not rule out an impact." };
}

// Explain recorded ranks, never infer agreement from the fused score alone.
export function rankingExplanation(impact: Impact): string {
  const fields = fusionFields(impact);
  if (!fields) return "The score orders possible impacts. It does not measure the probability that code must change.";
  const method = (name: string) => ({ hybrid: impact.score_breakdown.evidence_score === undefined ? "BPMN-context BM25" : "Code-evidence search", bm25: "BM25", semantic: "Semantic search" }[name] ?? "Search");
  const describe = (side: string) => {
    const rank = fields[`${side}_rank`];
    const name = method(fields[`${side}_method`]);
    if (rank === "absent") return `${name}: absent from the ranked list.`;
    if (/^[1-9]\d*$/.test(rank ?? "")) return `${name}: rank ${rank}.`;
    return `${name}: rank unavailable.`;
  };
  return `${describe("left")} ${describe("right")} The combined score uses these positions, not the raw search scores. A high score is possible even without a direct BPMN ID match. It is not a probability that code must change.`;
}
