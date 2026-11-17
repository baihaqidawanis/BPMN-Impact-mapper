import type { Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

// Test fixtures use the same upload path as an ordinary user; no demo UI is required.
export async function loadAnalysisFixture(page: Page, scenario = "comprehensive") {
  const prefix = scenario === "comprehensive" ? "" : `${scenario}-`;
  const folder = path.resolve(__dirname, "../../../public/demo");
  await page.locator('input[name="repo_path"]').fill("testdata\\repository");
  await page.getByLabel("Repository name", { exact: true }).fill("Synthetic Request Workflow");
  await page.locator('input[name="revision"]').fill("fixture-v1");
  await page.locator('input[name="old_bpmn"]').setInputFiles({ name: "synthetic-old.bpmn", mimeType: "application/xml", buffer: fs.readFileSync(path.join(folder, prefix + "old.bpmn")) });
  await page.locator('input[name="new_bpmn"]').setInputFiles({ name: "synthetic-new.bpmn", mimeType: "application/xml", buffer: fs.readFileSync(path.join(folder, prefix + "new.bpmn")) });
}
