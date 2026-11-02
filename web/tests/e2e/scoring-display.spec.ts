import { expect, test } from "@playwright/test";
import type { Impact, Report } from "../../src/lib/types";

test("hybrid separates neighbouring IDs, unused checks and combined ranks", async ({ page }) => {
  const candidate: Impact = {
    file: "application/request.go", symbol: "SubmitRequest", module: "orders", layer: "application", required_action: "VERIFY",
    confidence: 0.9357,
    score_breakdown: { trace_id: 0, symbol: 0, path: 0, lexical: 0, structured: 0, bm25: 0.6, semantic: 0.36, rrf: 0.9357, total: 0.9357 },
    evidence: [
      { type: "BM25", location: "application/request.go:9", excerpt: "request" },
      { type: "CONTEXT_TRACE_ID", location: "application/request.go:12", excerpt: 'trace := "STEP-BEFORE"' },
      { type: "SYMBOL", location: "application/request.go:9", excerpt: "func SubmitRequest()" },
      { type: "RRF_FUSION", location: "application/request.go", excerpt: "left_method=hybrid; left_rank=1; right_method=semantic; right_rank=10; k=60" },
    ],
  };
  const report: Report = {
    analysis_id: "scoring-display-test", input_fingerprint: "fixture", method: "hybrid", repository: { name: "Scoring fixture", revision: "fixture" }, warnings: [],
    changes: [{ change_type: "ADD_TASK", old_element: null, new_element: { id: "STEP-NEW", name: "Validate request", type: "userTask" }, unresolved_impacts: [],
      impacts: [candidate, { ...candidate, file: "service/validate.ts", symbol: "validate", confidence: 0.5,
        evidence: [{ type: "RRF_FUSION", location: "service/validate.ts", excerpt: "left_method=hybrid; left_rank=absent; right_method=semantic; right_rank=1; k=60" }],
        score_breakdown: { trace_id: 0, symbol: 0, path: 0, lexical: 0, structured: 0, bm25: 0, semantic: 0.5, rrf: 0.5, total: 0.5 } }],
    }],
  };
  await page.route("**/backend/api/v1/capabilities/semantic/check", route => route.fulfill({ json: { available: true, provider: "fixture", model: "fixture", message: "Ready" } }));
  await page.route("**/backend/api/v1/analyses", route => route.fulfill({ status: route.request().method() === "POST" ? 201 : 200, json: route.request().method() === "POST" ? report : { analyses: [] } }));
  await page.goto("/?view=analysis");
  await page.getByRole("button", { name: "Load synthetic demo" }).click();
  await expect(page.getByText("Synthetic demo ready")).toBeVisible();
  await page.getByRole("button", { name: "Run impact analysis" }).click();
  const first = page.getByTestId("impact-details").first();
  await first.locator(":scope > summary").click();
  await expect(first.getByText("rank agreement", { exact: true })).toBeVisible();
  await expect(first.getByText("Neighbouring BPMN ID found", { exact: true })).toBeVisible();
  await expect(first.getByText(/BPMN-context BM25: rank 1. Semantic search: rank 10/)).toBeVisible();
  await first.getByText("Show search signals", { exact: true }).click();
  await expect(first.getByText("Weights used by code-evidence search", { exact: true })).toHaveCount(0);
  await expect(first.getByText(/they do not subtract from either score/)).toBeVisible();
  const second = page.getByTestId("impact-details").nth(1);
  await second.locator(":scope > summary").click();
  await expect(second.getByText("No direct ID evidence recorded", { exact: true })).toBeVisible();
  await second.getByText("Show search signals", { exact: true }).click();
  await expect(second.getByText("Weights used by code-evidence search", { exact: true })).toHaveCount(0);
  await page.screenshot({ path: "test-results/scoring-display.png", fullPage: true });
});
