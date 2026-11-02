import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(__dirname, "../../..");
test("Qwen review shows scoped progress, plan and restores without another model start", async ({ page, request }) => {
  const saved = await request.post("/backend/api/v1/analyses", { multipart: {
    repo_path: path.join(root, "testdata/repository"), repository_name: "Qwen browser fixture", method: "bm25", top_k: "3",
    old_bpmn: { name: "old.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/old.bpmn")) },
    new_bpmn: { name: "new.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/new.bpmn")) },
  } });
  expect(saved.ok()).toBe(true); const report = await saved.json();
  const identity = { provider: "scripted-browser", model: "qwen-fixture", prompt_version: "v1" };
  const sessionID = "1".repeat(32), planID = "2".repeat(32);
  const plan = { version: "1", plan_id: planID, analysis_id: report.analysis_id, context_id: "3".repeat(32), input_fingerprint: report.input_fingerprint, provider: identity, status: "NEEDS_REVIEW", review_issues: ["Confirm the rejection contract."], plans: [{ file: "application/submit.go", module: "application", layer: "application", action: "VERIFY", summary: "Propose validation before submission.", steps: ["Check invalid input before mutating status."], proposed_changes: ["Reject invalid amounts before changing status."], acceptance_checks: ["Zero amount leaves the original state unchanged."], evidence_ids: ["E1"] }] };
  let starts = 0, polls = 0;
  await page.route("**/api/v1/capabilities/agents/check", route => route.fulfill({ json: { enabled: true, available: true, provider: identity, message: "Model metadata ready." } }));
  await page.route(`**/api/v1/analyses/${report.analysis_id}/agent-sessions`, async route => {
    starts++; const body = route.request().postDataJSON(); expect(body.change_index).toBe(0); expect(body.mode).toBe("full"); expect(body.repository_path).toBe("C:\\fixture-repo");
    await route.fulfill({ status: 202, json: { session_id: sessionID, analysis_id: report.analysis_id, provider: identity, status: "STARTING", role: "analyst", turns: 0, tools: 0, revisions: 0 } });
  });
  await page.route(`**/api/v1/agent-sessions/${sessionID}`, route => {
    polls++; return route.fulfill({ json: { session_id: sessionID, analysis_id: report.analysis_id, provider: identity, mode: "full", full_review: { phase: polls === 1 ? "scanning" : "finished", files_total: 2, files_reviewed: polls === 1 ? 1 : 2, chunks_total: 3, chunks_reviewed: polls === 1 ? 1 : 3, batches_total: 1, batches_reviewed: polls === 1 ? 0 : 1, relevant_files: 1, weak_files: 0 }, status: polls === 1 ? "RUNNING" : "NEEDS_REVIEW", role: polls === 1 ? "analyst" : "reviewer", turns: polls === 1 ? 1 : 3, tools: 0, revisions: 0, plan: polls > 1 ? plan : undefined } });
  });
  await page.goto(`/?analysis=${report.analysis_id}`);
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  const panel = page.getByTestId("qwen-review");
  await expect(panel).toContainText("Connected");
  await panel.getByLabel("Repository path for Qwen review").fill("C:\\fixture-repo");
  await panel.getByRole("button", { name: "Run Qwen review", exact: true }).click();
  await expect(panel).toContainText("NEEDS REVIEW", { timeout: 20_000 });
  await expect(panel).toContainText("Confirm the rejection contract.");
  await expect(panel).toContainText("2/2 eligible files");
  await expect(panel.getByRole("progressbar", {name:"Reviewed source chunks"})).toHaveAttribute("value","3");
  await expect(panel.getByRole("link", { name: "Download plan Markdown" })).toHaveAttribute("href", `/backend/api/v1/plans/${planID}/change-plan.md`);
  await expect(panel).toContainText("Reject invalid amounts before changing status.");
  await expect(panel).toContainText("Zero amount leaves the original state unchanged.");
  expect(starts).toBe(1);
  await page.reload();
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  await expect(panel).toContainText("NEEDS REVIEW"); expect(starts).toBe(1);
  await panel.screenshot({ path: test.info().outputPath("qwen-review.png") });
});
