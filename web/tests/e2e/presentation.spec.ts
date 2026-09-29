import { expect, test } from "@playwright/test";
import path from "node:path";

test("synthetic demo completes and exposes evidence downloads", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "BPMN Impact Mapper" })).toBeVisible();

  await page.getByRole("button", { name: "Load synthetic demo" }).click();
  await expect(page.getByText("Synthetic demo ready")).toBeVisible();
  await expect(page.getByLabel(/Repository path/)).toHaveValue("testdata\\repository");

  await page.getByRole("button", { name: "Run impact analysis" }).click();
  await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("heading", { name: "Synthetic Request Workflow" })).toBeVisible();
  await expect(page.getByText("BPMN changes")).toBeVisible();
  await expect(page.getByText("MODIFY TASK LABEL").first()).toBeVisible();

	const firstCandidate = page.getByTestId("impact-details").first();
  await firstCandidate.locator("summary").click();
  await expect(firstCandidate.getByRole("heading", { name: "Evidence" })).toBeVisible();
  await expect(firstCandidate.getByRole("heading", { name: "Score breakdown" })).toBeVisible();

  const jsonLink = page.getByRole("link", { name: "JSON" });
  const markdownLink = page.getByRole("link", { name: "Markdown" });
  await expect(jsonLink).toHaveAttribute("href", /report\.json$/);
  await expect(markdownLink).toHaveAttribute("href", /report\.md$/);
  const downloadPromise = page.waitForEvent("download");
  await markdownLink.click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^impact-report-[a-f0-9]{32}\.md$/);
});

test("manual BPMN upload workflow remains available", async ({ page }) => {
	await page.goto("/");
	await page.getByLabel(/Repository path/).fill("testdata\\repository");
	await page.getByLabel("Repository name").fill("Manual fixture run");
	await page.locator('input[name="old_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/old.bpmn"));
	await page.locator('input[name="new_bpmn"]').setInputFiles(path.resolve(process.cwd(), "../testdata/bpmn/new.bpmn"));
	await page.getByLabel("Mapping method").selectOption("bm25");
	await page.getByRole("button", { name: "Run impact analysis" }).click();
	await expect(page.getByText("Analysis complete")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByRole("heading", { name: "Manual fixture run" })).toBeVisible();
	await expect(page.getByTestId("report-view").getByText("bm25", { exact: false }).first()).toBeVisible();
});
