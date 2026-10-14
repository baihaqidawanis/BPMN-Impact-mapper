import { expect, test } from "@playwright/test";
import path from "node:path";
import { readFile } from "node:fs/promises";
import type { Report } from "../../src/lib/types";

test("evaluation reconciles conflicts and evaluates immutable comparison reports", async ({ page }) => {
	test.setTimeout(420_000);
  await page.goto("/");
  await page.getByRole("button", { name: "Load synthetic demo" }).click();
	await expect(page.getByText("Synthetic demo ready")).toBeVisible();
  await page.getByLabel("Compare mapping methods").check();
  const reports: Report[] = [];
  page.on("response", async response => {
    if (response.url().endsWith("/api/v1/analyses") && response.status() === 201) reports.push(await response.json());
  });
  await page.getByRole("button", { name: "Run impact analysis" }).click();
  await expect(page.getByTestId("method-comparison")).toBeVisible({ timeout: 120_000 });
  await expect.poll(() => reports.length).toBe(4);
  const prediction = reports.find(r => r.method === "hybrid")!;
  const cases = prediction.changes.map(c => ({
    change_type: c.change_type, element_id: (c.new_element ?? c.old_element)!.id,
    expected: [] as { file: string; required_action: string }[],
  }));
  const other = structuredClone(cases);
  other[0].expected = [{ file: "missing/expected.go", required_action: "MODIFY" }];
  await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "Research", exact: true }).click();
  const dashboard = page.getByTestId("evaluation-dashboard");
  await dashboard.locator("summary").click();
  const upload = (reviewer: string, values: typeof cases) => ({
    name: reviewer + ".json", mimeType: "application/json",
    buffer: Buffer.from(JSON.stringify({ version: "1", reviewer_id: reviewer, cases: values })),
  });
  await dashboard.getByLabel("Reviewer A JSON").setInputFiles(upload("a", cases));
  await expect(dashboard.getByRole("button", { name: "Reconcile reviewers" })).toBeDisabled();
  await dashboard.getByLabel("Reviewer B JSON").setInputFiles(upload("b", other));
  await dashboard.getByRole("button", { name: "Reconcile reviewers" }).click();
  await expect(dashboard.getByRole("status")).toContainText("1 disagreements");
  await dashboard.getByRole("checkbox").check();
  await expect(dashboard.getByRole("button", { name: "Evaluate stored reports" })).toBeDisabled();
  await dashboard.getByLabel("Resolution 1", { exact: true }).selectOption("b");
  await dashboard.getByRole("button", { name: "Evaluate stored reports" }).click();
  const results = dashboard.getByTestId("evaluation-results");
  await expect(results).toBeVisible();
  await expect(results.getByRole("columnheader", { name: "bm25" })).toBeVisible();
  await expect(results.getByRole("columnheader", { name: "semantic" })).toBeVisible();
  await expect(results.getByRole("columnheader", { name: "fusion" })).toBeVisible();
  await expect(results.getByRole("columnheader", { name: "hybrid" })).toBeVisible();
  const diagnostics = results.getByTestId("evaluation-diagnostics");
  await expect(diagnostics).toBeVisible();
  const hybridDiagnostics = diagnostics.locator("details").filter({ hasText: /^hybrid/i });
  await hybridDiagnostics.locator("summary").click();
  await expect(hybridDiagnostics.getByText("missing/expected.go").first()).toBeVisible();
  await expect(hybridDiagnostics.getByRole("columnheader", { name: "RR" })).toBeVisible();
  await results.screenshot({ path: test.info().outputPath("evaluation-results.png") });
  const downloaded = page.waitForEvent("download");
  await results.getByRole("button", { name: "Download evaluation JSON" }).click();
  const download = await downloaded;
  const bundle = JSON.parse(await readFile((await download.path())!, "utf8"));
  expect(bundle.results).toHaveLength(4);
  expect(bundle.ground_truth.cases).toHaveLength(cases.length);
  expect(bundle.resolutions[0].expected[0].file).toBe("missing/expected.go");
  expect(bundle.results.every((r: { metrics: { mrr: number } }) => r.metrics.mrr === 0)).toBe(true);
  await dashboard.getByLabel("Resolution 1", { exact: true }).selectOption("a");
  await expect(dashboard.getByTestId("evaluation-results")).toHaveCount(0);
});

test("synthetic demo completes and exposes evidence downloads", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "BPMN Impact Mapper" })).toBeVisible();

  await page.getByRole("button", { name: "Load synthetic demo" }).click();
  await expect(page.getByText("Synthetic demo ready")).toBeVisible();
  await expect(page.getByLabel(/Repository path/)).toHaveValue("testdata\\repository");

  await page.getByRole("button", { name: "Run impact analysis" }).click();
  await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 120_000 });
  await expect(page.getByRole("heading", { name: "Synthetic Request Workflow" })).toBeVisible();
  await expect(page.getByTestId("report-view").getByText("BPMN changes", { exact: true })).toBeVisible();
  await page.getByText(/^Source analysis coverage/).click();
  await expect(page.getByTestId("report-view").getByText(/web\/request-form.ts/)).toContainText("TEXT_ONLY");
  await expect(page.getByText("MODIFY TASK LABEL").first()).toBeVisible();

	const firstCandidate = page.getByTestId("impact-details").first();
  await firstCandidate.locator(":scope > summary").click();
  await expect(firstCandidate.getByRole("heading", { name: "Evidence" })).toBeVisible();
  await expect(firstCandidate.getByRole("heading", { name: "Score breakdown" })).toBeVisible();
	if (process.env.UI_SCREENSHOT) {
		await page.screenshot({ path: path.resolve(process.cwd(), "../.data/ui-report.png"), fullPage: true });
	}
	const navigation = page.getByRole("navigation", { name: "Workspace navigation" });
	await navigation.getByRole("button", { name: "Planning", exact: true }).click();
	const planning = page.getByTestId("planning-context-view");
	await expect(planning).toBeVisible();
	await expect(planning.getByText("Pre-LLM audit boundary")).toBeVisible();
	await expect(planning.getByText("Evidence items")).toBeVisible();
	await expect(planning.getByRole("link", { name: "Download JSON" })).toHaveAttribute("href", /planning-context\.json$/);
	await planning.getByRole("button", { name: "Create planning record" }).click();
	await expect(planning.getByRole("heading", { name: "Planning record prepared" })).toBeVisible();
	await expect(planning.getByText("DISABLED", { exact: true }).first()).toBeVisible();
	await expect(planning.getByText("1 immutable record", { exact: true })).toBeVisible();
	const planDownloadPromise = page.waitForEvent("download");
	await planning.getByRole("link", { name: "Plan JSON" }).click();
	const planDownload = await planDownloadPromise;
	expect(planDownload.suggestedFilename()).toMatch(/^change-plan-[a-f0-9]{32}\.json$/);
	await navigation.getByRole("button", { name: "Mapping", exact: true }).click();

  const jsonLink = page.getByRole("link", { name: "JSON" });
  const markdownLink = page.getByRole("link", { name: "Markdown" });
  const planningContextLink = page.getByRole("link", { name: "Planning context" });
  await expect(jsonLink).toHaveAttribute("href", /report\.json$/);
  await expect(markdownLink).toHaveAttribute("href", /report\.md$/);
  await expect(planningContextLink).toHaveAttribute("href", /planning-context\.json$/);
  const downloadPromise = page.waitForEvent("download");
  await markdownLink.click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^impact-report-[a-f0-9]{32}\.md$/);

	await page.reload();
	await page.getByRole("navigation", { name:"Workspace navigation" }).getByRole("button", { name:"Saved analyses", exact:true }).click();
	const history = page.getByTestId("analysis-history");
	await history.locator("summary").click();
	const saved = history.locator("article").filter({ hasText: "Synthetic Request Workflow" }).first();
	await expect(saved).toBeVisible();
	await saved.getByRole("button", { name: "Open report" }).click();
	await expect(page.getByRole("heading", { name: "Synthetic Request Workflow" })).toBeVisible();
});

test("manual BPMN upload workflow remains available", async ({ page }) => {
	await page.goto("/");
	await page.getByLabel(/Repository path/).fill("testdata\\repository");
	await page.getByLabel("Repository name").fill("Manual fixture run");
	await page.locator('input[name="old_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/old.bpmn"));
	await page.locator('input[name="new_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/new.bpmn"));
	await page.locator('select[name="method"]').selectOption("bm25");
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByRole("heading", { name: "Manual fixture run" })).toBeVisible();
	await expect(page.getByTestId("report-view").getByText("bm25", { exact: false }).first()).toBeVisible();
});

test("inserted task exposes composite topology context", async ({ page }) => {
	await page.goto("/");
	await page.getByLabel(/Repository path/).fill("testdata\\repository");
	await page.getByLabel("Repository name").fill("Composite insertion fixture");
	await page.locator('input[name="old_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/insert-task-old.bpmn"));
	await page.locator('input[name="new_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/insert-task-new.bpmn"));
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByText("INSERT TASK", { exact: true })).toBeVisible();
	await expect(page.getByText("from Prepare request to Register request", { exact: true })).toBeVisible();
});

test("scenario presets expose move and unresolved controls", async ({ page }) => {
	await page.goto("/");
	await page.getByLabel("Demo scenario").selectOption("move-task");
	await page.getByRole("button", { name: "Load synthetic demo" }).click();
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("MOVE TASK", { exact: true }).first()).toBeVisible({ timeout: 30_000 });

	await page.reload();
	await page.getByLabel("Demo scenario").selectOption("unresolved");
	await page.getByRole("button", { name: "Load synthetic demo" }).click();
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("UNRESOLVED MATCH", { exact: true }).first()).toBeVisible({ timeout: 30_000 });
});

test("same-input method comparison exposes retrieval differences", async ({ page }) => {
	test.setTimeout(300_000);
	await page.goto("/");
	await page.getByRole("button", { name: "Load synthetic demo" }).click();
	await expect(page.getByText("Synthetic demo ready")).toBeVisible();
	await page.getByLabel("Compare mapping methods").check();
	await page.getByRole("button", { name: "Run impact analysis" }).click();

	const comparison = page.getByTestId("method-comparison");
	await expect(comparison).toBeVisible({ timeout: 120_000 });
	await expect(comparison.getByText("Input fingerprint verified")).toBeVisible();
	await expect(comparison.getByRole("button", { name: /^BM25 \d/ })).toBeVisible();
	await expect(comparison.getByRole("button", { name: /^Semantic / })).toBeVisible();
	await expect(comparison.getByRole("button", { name: /^BM25 \+ Semantic / })).toBeVisible();
	await expect(comparison.getByRole("button", { name: /^Hybrid / })).toBeVisible();
	if (process.env.UI_SCREENSHOT) {
		await page.screenshot({ path: path.resolve(process.cwd(), "../.data/ui-comparison.png"), fullPage: true });
	}
	await comparison.getByRole("button", { name: /^BM25 \d/ }).click();
	await expect(page.getByTestId("report-view").getByText(/bm25/).first()).toBeVisible();
	await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "Research", exact: true }).click();

	const blindPack = page.getByTestId("blind-review-pack");
	await blindPack.locator("summary").click();
	const responsePromise = page.waitForResponse(response => response.url().endsWith("/api/v1/review-packs") && response.status() === 200);
	await blindPack.getByRole("button", { name: "Prepare blind review pack" }).click();
	const packResponse = await (await responsePromise).json();
	expect(JSON.stringify(packResponse)).not.toContain('"impacts"');
	expect(JSON.stringify(packResponse)).not.toContain('"confidence"');
	expect(packResponse.reviewer_a.cases[0].expected[0].file).toBe("REVIEW_REQUIRED");
	await expect(blindPack.getByText("Pack ready")).toBeVisible();
	if (process.env.UI_SCREENSHOT) {
		await page.screenshot({ path: path.resolve(process.cwd(), "../.data/ui-blind-review.png"), fullPage: true });
	}
	const reviewerDownload = page.waitForEvent("download");
	await blindPack.getByRole("button", { name: "Reviewer A template" }).click();
	const reviewerFile = await reviewerDownload;
	const reviewerTemplate = JSON.parse(await readFile((await reviewerFile.path())!, "utf8"));
	expect(reviewerTemplate.reviewer_id).toBe("reviewer-a");
	expect(reviewerTemplate.cases.every((item: { expected: { file: string; required_action: string }[] }) => item.expected[0].file === "REVIEW_REQUIRED" && item.expected[0].required_action === "")).toBe(true);
});

test("reviewer annotation exports selected and missing impacts locally", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("button", { name: "Load synthetic demo" }).click();
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 30_000 });
	await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "Research", exact: true }).click();

	const review = page.getByTestId("review-panel");
	await review.locator("summary").click();
	await review.getByLabel("Reviewer ID").fill("reviewer-a");
	await review.locator('input[type="checkbox"]').first().check();
	await review.getByRole("button", { name: "Add missing expected impact" }).first().click();
	await review.getByLabel("Missing expected file").first().fill("manual/expected.go");
	await review.getByLabel("Missing expected symbol").first().fill("ExpectedChange");

	const downloadPromise = page.waitForEvent("download");
	await review.getByRole("button", { name: "Export reviewer JSON" }).click();
	const download = await downloadPromise;
	const downloadPath = await download.path();
	expect(downloadPath).not.toBeNull();
	const annotation = JSON.parse(await readFile(downloadPath!, "utf8")) as { version: string; reviewer_id: string; cases: { expected: { file: string }[] }[] };
	expect(annotation.version).toBe("1");
	expect(annotation.reviewer_id).toBe("reviewer-a");
	expect(annotation.cases.some((item) => item.expected.some((impact) => impact.file === "manual/expected.go"))).toBe(true);
});
