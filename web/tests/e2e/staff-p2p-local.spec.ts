import { expect, test } from "@playwright/test";
import path from "node:path";
import type { Report } from "../../src/lib/types";

// Local case-study smoke only. The source checkout is read-only and may be dirty;
// this test is not a research accuracy metric or a portable CI fixture.
test("Staff P2P upload produces a persistent canonical report", async ({ page, request }) => {
  const repoPath = process.env.BPMN_IMPACT_STAFF_REPO;
  test.skip(!repoPath, "set BPMN_IMPACT_STAFF_REPO to the local case-study checkout");
  test.setTimeout(720_000);

  const oldBpmn = path.join(repoPath!, "docs", "bpm-haqi", "roles", "staff", "p2p", "level-3", "role-staff-l3-pr-create.bpmn");
  const newBpmn = path.resolve(__dirname, "../../../experiments/simple-erp/staff-p2p/role-staff-l3-pr-create-add-document.bpmn");
  await page.goto("/?view=analysis");
  await page.locator('input[name="repo_path"]').fill(repoPath!);
  await page.getByLabel("Repository name").fill("Simple-ERP");
  await page.locator('input[name="revision"]').fill("c934497-dirty");
  await page.locator('input[name="old_bpmn"]').setInputFiles(oldBpmn);
  await page.locator('input[name="new_bpmn"]').setInputFiles(newBpmn);
  await page.locator('input[name="top_k"]').fill("5");

  const responsePromise = page.waitForResponse(response =>
    response.url().endsWith("/api/v1/analyses") && response.request().method() === "POST",
    { timeout: 660_000 },
  );
  await page.getByRole("button", { name: "Run impact analysis" }).click();
  const response = await responsePromise;
  expect(response.status(), await response.text()).toBe(201);
  const report = await response.json() as Report;
  expect(report.repository.name).toBe("Simple-ERP");
  expect(report.method).toBe("hybrid");
  expect(report.method_metadata?.right_query_version).toBe("named-task-process-context-v2");
  const insertion = report.changes.find(change => change.change_type === "ADD_TASK");
  expect(insertion?.new_element?.id).toBe("role_staff_l3_pr_create_attachment");
  expect(insertion?.context?.pattern).toBe("INSERT_TASK");
  expect(insertion?.impacts.length).toBeGreaterThan(0);
  expect(insertion?.impacts.every(impact => impact.evidence.length > 0)).toBe(true);

  const result = page.getByTestId("report-view");
  await expect(result.getByText("Analysis complete")).toBeVisible();
  await expect(result.getByText("Lampirkan dokumen pendukung PR", { exact: false }).first()).toBeVisible();
  await expect(result.getByText("FLOW DETAILS", { exact: true })).toBeVisible();

  const savedJSON = await request.get(`/backend/api/v1/analyses/${report.analysis_id}/report.json`);
  expect(savedJSON.status()).toBe(200);
  expect(await savedJSON.json()).toEqual(report);
  const savedMarkdown = await request.get(`/backend/api/v1/analyses/${report.analysis_id}/report.md`);
  expect(savedMarkdown.status()).toBe(200);
  expect(await savedMarkdown.text()).toContain("role_staff_l3_pr_create_attachment");

  await page.reload();
  await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "History", exact: true }).click();
  const history = page.getByTestId("analysis-history");
  await history.locator("summary").click();
  await history.locator("article").filter({ hasText: "c934497-dirty" }).first().getByRole("button", { name: "Open report" }).click();
  await expect(page.getByTestId("report-view").getByText("Analysis complete")).toBeVisible();
});

for (const scenario of [
  { file: "role-staff-l3-pr-create-gateway-condition.bpmn", label: "gateway condition", expected: ["MODIFY_GATEWAY_CONDITION", "MODIFY_SEQUENCE_FLOW"] },
  { file: "role-staff-l3-pr-create-sequence-label.bpmn", label: "branch label", expected: ["MODIFY_SEQUENCE_FLOW"] },
]) {
  test(`Staff P2P ${scenario.label} remains distinct in web report`, async ({ page }) => {
    const repoPath = process.env.BPMN_IMPACT_STAFF_REPO;
    test.skip(!repoPath, "set BPMN_IMPACT_STAFF_REPO to the local case-study checkout");
    const oldBpmn = path.join(repoPath!, "docs", "bpm-haqi", "roles", "staff", "p2p", "level-3", "role-staff-l3-pr-create.bpmn");
    const newBpmn = path.resolve(__dirname, "../../../experiments/simple-erp/staff-p2p", scenario.file);
    await page.goto("/?view=analysis");
    await page.locator('input[name="repo_path"]').fill(repoPath!);
    await page.getByLabel("Repository name").fill("Simple-ERP");
    await page.locator('input[name="old_bpmn"]').setInputFiles(oldBpmn);
    await page.locator('input[name="new_bpmn"]').setInputFiles(newBpmn);
    await page.getByRole("group", { name: "Analysis mode" }).getByRole("button", { name: "Research" }).click();
    await page.getByLabel("Experimental method", { exact: true }).selectOption("bm25");
    const responsePromise = page.waitForResponse(response =>
      response.url().endsWith("/api/v1/analyses") && response.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Run impact analysis" }).click();
    const response = await responsePromise;
    expect(response.status(), await response.text()).toBe(201);
    const report = await response.json() as Report;
    expect(report.changes.map(change => change.change_type).sort()).toEqual([...scenario.expected].sort());
    await expect(page.getByTestId("report-view").getByText("Analysis complete")).toBeVisible();
  });
}
