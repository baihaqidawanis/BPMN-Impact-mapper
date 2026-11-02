import { expect, test } from "@playwright/test";

test("partial BPMN semantics cannot appear as a complete zero-change result", async ({ page }) => {
  await page.route("**/backend/api/v1/analyses", async route => {
    if (route.request().method() !== "POST") return route.continue();
    await route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({
      analysis_id: "partial-bpmn-fixture", input_fingerprint: "fixture", method: "bm25",
      repository: { name: "Advanced BPMN fixture", revision: "fixture" },
      changes: [], warnings: [{ code: "BPMN_PARTIAL_SUPPORT", message: "changed: multi-instance execution characteristics are not compared. Diff may omit changes in these features." }],
      method_metadata: { bpmn_capability_policy: "partial-semantics-diagnostics-v1" }, source_analysis: [],
    }) });
  });
  await page.goto("/?view=analysis");
  await page.getByRole("button", { name: "Load synthetic demo" }).click();
  await expect(page.getByText("Synthetic demo ready")).toBeVisible();
  await page.getByRole("group", { name: "Analysis mode" }).getByRole("button", { name: "Research" }).click();
  await page.getByLabel("Experimental method", { exact: true }).selectOption("bm25");
  await page.getByRole("button", { name: "Run impact analysis" }).click();
  const report = page.getByTestId("report-view");
  await expect(report.getByText("Analysis finished with BPMN limitations")).toBeVisible();
  await expect(report.getByText("Detected BPMN changes", { exact: true })).toBeVisible();
  await expect(report.getByRole("status").filter({ hasText: "Zero detected changes" })).toContainText("Zero detected changes does not prove the process is unchanged");
  await expect(report.getByText("BPMN_PARTIAL_SUPPORT", { exact: true })).toBeVisible();
  await expect(report.getByText("Analysis complete", { exact: true })).toHaveCount(0);
  await report.screenshot({ path: test.info().outputPath("partial-bpmn.png") });
});
