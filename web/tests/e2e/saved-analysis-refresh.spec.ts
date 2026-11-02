import { expect, test, type APIRequestContext } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(__dirname, "../../..");
async function savedFixture(request: APIRequestContext, name: string) {
  const response = await request.post("/backend/api/v1/analyses", { multipart: {
    repo_path: path.join(root, "testdata/repository"), repository_name: name,
    revision: "refresh-fixture", method: "bm25", top_k: "5",
    old_bpmn: { name: "old.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/old.bpmn")) },
    new_bpmn: { name: "new.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/new.bpmn")) },
  } });
  expect(response.ok()).toBe(true);
  return response.json();
}

test("new analysis is saved and reopens after refresh without another analysis POST", async ({ page, request }) => {
  await page.goto("/?view=analysis");
  await page.getByRole("group", { name: "Analysis mode" }).getByRole("button", { name: "Research", exact: true }).click();
  await page.getByLabel("Experimental method", { exact: true }).selectOption("bm25");
  await page.locator('input[name="repo_path"]').fill(path.join(root, "testdata/repository"));
  await page.locator('input[name="repository_name"]').fill("Refresh integration fixture");
  await page.locator('input[name="old_bpmn"]').setInputFiles(path.join(root, "testdata/bpmn/old.bpmn"));
  await page.locator('input[name="new_bpmn"]').setInputFiles(path.join(root, "testdata/bpmn/new.bpmn"));
  const created = page.waitForResponse(r => r.request().method() === "POST" && r.url().endsWith("/api/v1/analyses"));
  await page.getByRole("button", { name: "Run impact analysis", exact: true }).click();
  const canonical = await (await created).json();
  await expect(page.getByTestId("report-view")).toContainText(canonical.analysis_id);
  await expect(page).toHaveURL(new RegExp(`analysis=${canonical.analysis_id}`));
  let posts = 0;
  page.on("request", r => { if (r.method() === "POST" && r.url().endsWith("/api/v1/analyses")) posts++; });
  await page.reload();
  await expect(page.getByTestId("report-view")).toContainText(canonical.analysis_id);
  await expect(page.getByRole("status").filter({ hasText: "Saved report reopened" })).toBeVisible();
  const stored = await request.get(`/backend/api/v1/analyses/${canonical.analysis_id}`);
  expect(await stored.json()).toEqual(canonical);
  expect(posts).toBe(0);
  await page.screenshot({ path: test.info().outputPath("restored-report.png"), fullPage: true });
});

test("missing saved report has a recovery message and does not start inference", async ({ page }) => {
  let posts = 0;
  page.on("request", r => { if (r.method() === "POST" && r.url().endsWith("/api/v1/analyses")) posts++; });
  await page.goto("/?analysis=ffffffffffffffffffffffffffffffff");
  await expect(page.getByRole("status").filter({ hasText: "could not be reopened" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Reopening saved report" })).toHaveCount(0);
  await expect(page.getByTestId("report-view")).toHaveCount(0);
  expect(posts).toBe(0);
});

test("opening another saved report cancels an older slow restoration", async ({ page, request }) => {
  const old = await savedFixture(request, "Slow restoration fixture");
  const next = await savedFixture(request, "Chosen history fixture");
  let release!: () => void;
  let started!: () => void;
  let finished!: () => void;
  const blocked = new Promise<void>(resolve => { release = resolve; });
  const requested = new Promise<void>(resolve => { started = resolve; });
  const completed = new Promise<void>(resolve => { finished = resolve; });
  await page.route(`**/api/v1/analyses/${old.analysis_id}`, async route => {
    started(); await blocked;
    try { await route.fulfill({ json: old }); } catch { /* Aborted restoration may no longer have a receiver. */ }
    finished();
  });
  try {
    await page.goto(`/?analysis=${old.analysis_id}`);
    await requested;
    await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "History", exact: true }).click();
    const history = page.getByTestId("analysis-history");
    await history.locator("article").filter({ hasText: "Chosen history fixture" }).getByRole("button", { name: "Open report" }).click();
    await expect(page.getByTestId("report-view")).toContainText(next.analysis_id);
    release(); await completed;
    await page.evaluate(() => new Promise(requestAnimationFrame));
    await expect(page).toHaveURL(new RegExp(`analysis=${next.analysis_id}`));
    await expect(page.getByTestId("report-view")).not.toContainText(old.analysis_id);
  } finally { release(); }
});
