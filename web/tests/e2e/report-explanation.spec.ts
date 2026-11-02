import { expect, test } from "@playwright/test";
import { fusionCalculation, rankingExplanation, traceExplanation, usesCodeEvidence } from "../../src/lib/report-explanation";
import type { Impact } from "../../src/lib/types";

function candidate(excerpt?: string): Impact {
  return {
    file: "service.go", module: "orders", layer: "application", required_action: "VERIFY",
    confidence: 0.9, score_breakdown: { trace_id: 0, symbol: 0, path: 0, lexical: 0, structured: 0, bm25: 0, total: 0.9 },
    evidence: excerpt ? [{ type: "RRF_FUSION", location: "service.go", excerpt }] : [],
  };
}

test("RRF calculation uses recorded ranks and rejects incomplete metadata", () => {
  expect(fusionCalculation(candidate("left_rank=3; right_rank=2; k=60")))
    .toContain("(1/(60+3) + 1/(60+2)) / (2/(60+1))");
  expect(fusionCalculation(candidate("left_rank=absent; right_rank=1; k=60")))
    .toContain("(0 + 1/(60+1))");
  expect(fusionCalculation(candidate())).toBeNull();
  expect(fusionCalculation(candidate("left_rank=0; right_rank=2; k=60"))).toBeNull();
  expect(fusionCalculation(candidate("left_rank=1; right_rank=2; k=unknown"))).toBeNull();
});

test("ranking explanation distinguishes two ranks from a missing search result", () => {
  expect(rankingExplanation(candidate("left_method=hybrid; left_rank=1; right_method=semantic; right_rank=10; k=60")))
    .toContain("BPMN-context BM25: rank 1. Semantic search: rank 10.");
  expect(rankingExplanation(candidate("left_method=hybrid; left_rank=absent; right_method=semantic; right_rank=1; k=60")))
    .toContain("BPMN-context BM25: absent from the ranked list. Semantic search: rank 1.");
  expect(rankingExplanation(candidate("left_method=bm25; left_rank=1; right_method=semantic; right_rank=absent; k=60")))
    .toContain("BM25: rank 1. Semantic search: absent from the ranked list.");
});

test("missing or malformed fusion metadata does not invent agreement", () => {
  expect(rankingExplanation(candidate())).toContain("does not measure the probability");
  expect(rankingExplanation(candidate("left_method=hybrid; right_method=semantic"))).toContain("rank unavailable");
});

test("high fusion score with zero trace is explained without claiming direct evidence", () => {
  const impact = candidate("left_method=hybrid; left_rank=1; right_method=semantic; right_rank=10; k=60");
  impact.evidence.push({ type: "BM25", location: "service.go:1", excerpt: "request" });
  expect(traceExplanation(impact, "hybrid", "ADD_TASK").status).toBe("New BPMN ID not yet available in source");
  expect(rankingExplanation(impact)).toContain("high score is possible even without a direct BPMN ID match");
  impact.evidence.push({ type: "CONTEXT_TRACE_ID", location: "service.go:4", excerpt: 'trace := "STEP-BEFORE"' });
  expect(traceExplanation(impact, "hybrid", "ADD_TASK").status).toBe("Neighbouring BPMN ID found");
  expect(traceExplanation(impact, "hybrid", "ADD_TASK").detail).toContain("adds no separate score");
  expect(impact.score_breakdown.trace_id).toBe(0);
});

test("unused trace fields are not reported as failed ID searches", () => {
  for (const method of ["semantic", "bm25", "fusion"] as const) {
    expect(usesCodeEvidence(candidate(), method)).toBe(false);
    expect(traceExplanation(candidate(), method, "ADD_TASK").detail).toContain("does not rule out an impact");
  }
  const semanticOnly = candidate("left_method=hybrid; left_rank=absent; right_method=semantic; right_rank=1; k=60");
  expect(usesCodeEvidence(semanticOnly, "hybrid")).toBe(false);
  expect(traceExplanation(semanticOnly, "hybrid", "ADD_TASK").status).toBe("No direct ID evidence recorded");
});

test("direct and propagated evidence remain distinct", () => {
  const impact = candidate();
  impact.evidence.push({ type: "TRACE_ID", location: "service.go:2", excerpt: "STEP-1" });
  expect(traceExplanation(impact, "hybrid", "MODIFY_TASK_LABEL").status).toBe("Direct BPMN ID found");
  impact.evidence = [{ type: "PROPAGATION_SEED", location: "other.go", excerpt: "AnotherFunction (exact trace seed)" }];
  expect(usesCodeEvidence(impact, "hybrid")).toBe(false);
  expect(traceExplanation(impact, "hybrid", "ADD_TASK").status).toBe("Linked through another function");
});
