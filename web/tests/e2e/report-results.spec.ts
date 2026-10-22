import { expect, test } from "@playwright/test";
import { groupResults, independentChanges, resultCategory } from "../../src/lib/report-results";
import { candidateKeys, comparisonChanges, overlapPercent, unresolvedComparisonChanges } from "../../src/lib/method-comparison";
import { methodDifferences } from "../../src/lib/method-differences";
import type { Impact, Report } from "../../src/lib/types";

const impact = (file: string, symbol = "CheckRequest"): Impact => ({
  file, symbol, module: "requests", layer: "application", required_action: "VERIFY", confidence: 0.8,
  score_breakdown: { trace_id: 0, symbol: 0.2, path: 0.2, lexical: 1, structured: 0, bm25: 0.5, total: 0.8 },
  evidence: [{ type: "BM25", location: file+":12", excerpt: "func CheckRequest() {}" }],
});

const report: Report = {
  analysis_id: "report-filter-fixture", input_fingerprint: "fixture", repository: { name: "Report display fixture", revision: "fixture-v1" }, method: "bm25", warnings: [],
  changes: [
    { change_type: "ADD_TASK", old_element: null, new_element: { id: "T-2", name: "Check request", type: "userTask" }, unresolved_impacts: [], impacts: [impact("docs/rules.md"), impact("src/request.go"), impact("tests/request_test.go")] },
    { change_type: "MODIFY_SEQUENCE_FLOW", old_element: null, new_element: { id: "F-2", name: "Route request", type: "sequenceFlow" }, unresolved_impacts: ["Review the new route"], impacts: [impact("src/request.go", "RouteRequest")] },
  ],
};

test("result grouping preserves change links, original ranks, and report bytes", () => {
  const before = JSON.stringify(report);
  const groups = groupResults(report, "code");
  expect(groups).toHaveLength(1);
  expect(groups[0].entries.map(entry => [entry.changeIndex, entry.rank])).toEqual([[0, 2], [1, 1]]);
  expect(JSON.stringify(report)).toBe(before);
  expect(resultCategory(impact("frontend\\__tests__\\form.spec.tsx"))).toBe("tests");
  expect(resultCategory(impact("docs/testing.md"))).toBe("docs");
  expect(resultCategory(impact("src/latest.go"))).toBe("code");
});

test("simple inserted task groups supporting flow edits without deleting raw JSON changes", () => {
  const grouped: Report = { ...report, changes: [
    report.changes[0],
    { ...report.changes[1], part_of: "T-2", impacts: [], unresolved_impacts: [] },
  ] };
  const before = JSON.stringify(grouped);
  expect(independentChanges(grouped).map(item => item.index)).toEqual([0]);
  expect(comparisonChanges(grouped)).toHaveLength(1);
  expect(unresolvedComparisonChanges(grouped)).toBe(0);
  expect(candidateKeys(grouped).size).toBe(3);
  expect(overlapPercent(candidateKeys(grouped), candidateKeys(grouped))).toBe(100);
  expect(methodDifferences([grouped, { ...grouped, method: "semantic" }])).toHaveLength(3);
  expect(grouped.changes).toHaveLength(2);
  expect(JSON.stringify(grouped)).toBe(before);
});

test("report filters and file view work without rerunning analysis", async ({ page }) => {
  let analysisPosts = 0;
  page.on("request", request => { if (request.method() === "POST" && request.url().endsWith("/analyses")) analysisPosts++; });
  await page.route("**/backend/api/v1/analyses", route => route.fulfill({ json: { analyses: [{ analysis_id: report.analysis_id, repository: report.repository, method: report.method, changes: 2, candidates: 4, unresolved: 1 }] } }));
  await page.route(`**/backend/api/v1/analyses/${report.analysis_id}`, route => route.fulfill({ json: report }));
  await page.goto("/");
  await page.getByRole("navigation", {name:"Workspace navigation"}).getByRole("button", { name: "Saved analyses", exact: true }).click();
  await page.getByTestId("analysis-history").locator("summary").click();
  await page.getByRole("button", { name: "Open report", exact: true }).click();
  const results = page.getByTestId("report-view");
  await expect(results.getByTestId("impact-details")).toHaveCount(4);
  await results.getByRole("button", { name: "Code", exact: true }).click();
  await expect(results.getByTestId("impact-details")).toHaveCount(2);
  await expect(results.getByRole("status")).toContainText("Showing 2 of 4 matches across 1 file");
  await results.getByRole("button", { name: "By file", exact: true }).click();
  await expect(results.getByRole("heading", {name:"src/request.go", exact:true})).toHaveCount(1);
  await expect(results.getByRole("region", {name:"Related change 1"})).toContainText("CheckRequest");
  await expect(results.getByRole("region", {name:"Related change 2"})).toContainText("RouteRequest");
  await expect(results.getByRole("region", {name:"Unresolved BPMN changes"})).toContainText("Review the new route");
  const first = results.getByTestId("impact-details").first();
  await first.locator(":scope > summary").click();
  await first.getByText("Show technical evidence", {exact: true}).click();
  await expect(first.getByText("func CheckRequest() {}", {exact:true})).toBeVisible();
  expect(await first.getByText("Show technical evidence", {exact:true}).evaluate(el => getComputedStyle(el).display)).not.toBe("grid");
  await page.screenshot({ path: "test-results/report-results-desktop.png", fullPage: true });
  await page.setViewportSize({width:390, height:844});
  await expect.poll(() => results.evaluate(el => el.scrollWidth <= el.clientWidth + 1)).toBe(true);
  await page.screenshot({ path: "test-results/report-results-mobile.png", fullPage: true });
  await results.getByRole("button", { name: "Docs", exact: true }).click();
  await expect(results.getByRole("heading", {name:"docs/rules.md", exact:true})).toBeVisible();
  await results.getByRole("button", { name: "Tests", exact: true }).click();
  await expect(results.getByRole("heading", {name:"tests/request_test.go", exact:true})).toBeVisible();
  await results.getByRole("button", { name: "All", exact: true }).click();
  await results.getByRole("button", { name: "By BPMN change", exact: true }).click();
  await expect(results.getByTestId("impact-details")).toHaveCount(4);
  await expect(results.getByText("Review the new route", {exact:true})).toBeVisible();
  expect(analysisPosts).toBe(0);
});
