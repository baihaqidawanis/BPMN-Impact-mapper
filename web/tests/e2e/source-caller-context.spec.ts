import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

test("unranked source caller and exact BPMN condition reach the planning context", async ({ page, request }) => {
  const root = path.resolve(__dirname, "../../../testdata/source-callers");
  const response = await request.post("/backend/api/v1/analyses", { multipart: {
    repo_path: root, repository_name: "Source connection fixture", method: "bm25", top_k: "3",
    old_bpmn: { name: "old.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "old.bpmn")) },
    new_bpmn: { name: "new.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "new.bpmn")) },
  } });
  expect(response.ok()).toBe(true);
  const report = await response.json();
  const changed = report.changes.find((c: { change_type: string }) => c.change_type === "MODIFY_GATEWAY_CONDITION");
  expect(changed.related_files[0].file).toBe("ui/panel.tsx");
  expect(changed.impacts.some((c: { file: string }) => c.file === "ui/panel.tsx")).toBe(false);
  const rawContext = await request.get(`/backend/api/v1/analyses/${report.analysis_id}/planning-context.json`);
  const payload = await rawContext.json();
  const input = payload.changes.find((c: { change_type: string }) => c.change_type === "MODIFY_GATEWAY_CONDITION");
  expect(input.branch_changes[0].old_condition).toBe("status == REJECTED");
  expect(input.branch_changes[0].new_condition).toContain("CANCELLED");
  const candidate = input.candidates.find((c: { file: string }) => c.file === "ui/panel.tsx");
  expect(candidate.origin).toBe("source-caller");
  expect(candidate.retrieval_score).toBeUndefined();
  expect(candidate.suggested_action).toBe("VERIFY");

  await page.goto(`/?analysis=${report.analysis_id}`);
  const view = page.getByTestId("report-view");
  await view.getByText("Related source connections (1)", { exact: true }).click();
  await expect(view.getByRole("region", { name: "Related file ui/panel.tsx" })).toContainText("Not ranked");
  await view.getByText("Show connection evidence", { exact: true }).click();
  await expect(view).toContainText("SOURCE_CALLER_EDGE");
  await expect(view).toContainText("HTTP_ROUTE_PATTERN");
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  const context = page.getByTestId("planning-context-view");
  await context.getByText("Source context and audit exports", {exact:true}).click();
  await context.getByText("BPMN requirement before / after", { exact: true }).first().click();
  await expect(context).toContainText("status == REJECTED || status == CANCELLED");
  const connected = context.locator("details").filter({ hasText: "connected source file" }).last();
  await expect(connected).toContainText("Not ranked");
  await page.screenshot({ path: test.info().outputPath("source-caller-planning.png"), fullPage: true });
});
