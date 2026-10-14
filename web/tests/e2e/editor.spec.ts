import { test, expect } from "@playwright/test";
import path from "node:path";
import { readFile } from "node:fs/promises";

test("canvas draft preserves baseline, exports XML and submits edited BPMN", async ({ page }) => {
  page.setDefaultTimeout(15_000);
  await page.goto("/");
  await page.locator('input[name="repo_path"]').fill(" testdata/repository ");
  await page.locator('input[name="old_bpmn"]').setInputFiles(path.resolve(__dirname, "../../../docs/demo-bpmn/simple-erp-p2p-add-user-task.bpmn"));
  await page.getByRole("button", { name: "Edit baseline in canvas" }).click();
  const dialog = page.getByRole("region", { name: "Edit new BPMN" });
  await expect(dialog.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
  await dialog.locator('[data-element-id="PUR-PR-002A"]').dblclick();
  const label = dialog.locator('.djs-direct-editing-content');
  await label.fill("Review supplier eligibility");
  await label.press("Control+Enter");
  await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name:"Mapping", exact:true }).click();
  await expect(page.locator('input[name="repo_path"]')).toHaveValue(" testdata/repository ");
  await expect(page.getByText("Canvas has unapplied edits.", { exact:false })).toBeVisible();
  await page.getByRole("button", { name:"BPMN Canvas", exact:true }).click();
  await expect(dialog.locator('[data-element-id="PUR-PR-002A"]')).toContainText("Review supplier eligibility");
  await dialog.screenshot({ path: test.info().outputPath("canvas.png") });
  await dialog.getByRole("button", { name: "Undo", exact: true }).click();
  await dialog.getByRole("button", { name: "Redo", exact: true }).click();
  const download = page.waitForEvent("download");
  await dialog.getByRole("button", { name: "Download BPMN" }).click();
  expect((await download).suggestedFilename()).toBe("edited-process.bpmn");
  await dialog.getByRole("button", { name: "Use edited BPMN" }).click();
  await expect(dialog).not.toBeVisible();
  await page.locator('select[name="method"]').selectOption("bm25");
  await page.getByRole("button", { name: "Run impact analysis" }).click();
  await expect(page.getByTestId("report-view")).toBeVisible();
  await expect(page.getByTestId("report-view")).toContainText("MODIFY TASK LABEL");
  await expect(page.getByTestId("report-view")).toContainText("Review supplier eligibility");
  await page.getByRole("button", { name: "Edit baseline in canvas" }).click();
  await expect(dialog.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
  await expect(dialog.locator('[data-element-id="PUR-PR-002A"]')).toContainText("Review supplier eligibility");
  page.once("dialog", d => d.accept());
  await dialog.getByRole("button", { name: "Reset to baseline" }).click();
  await expect(dialog.locator('[data-element-id="PUR-PR-002A"]')).toContainText(/Validate purchase\s*request details/);
});

test("workspace navigation supports canvas-first upload and mobile layout", async ({ page }) => {
  await page.goto("/");
  const nav = page.getByRole("navigation", { name:"Workspace navigation" });
  await nav.getByRole("button", { name:"BPMN Canvas", exact:true }).click();
  await expect(page.getByText("Bring your baseline BPMN.")).toBeVisible();
  await page.getByLabel("Canvas baseline BPMN").setInputFiles(path.resolve(__dirname, "../../../docs/demo-bpmn/simple-erp-p2p-add-user-task.bpmn"));
  await expect(page.getByRole("button", { name:"Use edited BPMN" })).toBeEnabled();
  await nav.getByRole("button", { name:"Guide", exact:true }).click();
  await expect(page.getByText("One baseline, two ways to compare")).toBeVisible();
  await nav.getByRole("button", { name:"BPMN Canvas", exact:true }).click();
  await page.getByRole("button", { name:"Use edited BPMN" }).click();
  await expect(nav.getByRole("button", { name:"Mapping", exact:true })).toHaveAttribute("aria-current", "page");
  await expect(page.getByText("Loaded: simple-erp-p2p-add-user-task.bpmn")).toBeVisible();
  await page.screenshot({ path:test.info().outputPath("workspace-desktop.png"), fullPage:true });
  await page.setViewportSize({ width:390, height:844 });
  await expect(nav).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path:test.info().outputPath("workspace-mobile.png"), fullPage:true });
});

test("canvas can add a task and gateway and export them with diagram coordinates", async ({ page }) => {
  page.setDefaultTimeout(15_000);
  await page.goto("/");
  await page.locator('input[name="old_bpmn"]').setInputFiles({ name:"baseline.bpmn", mimeType:"application/xml", buffer:Buffer.from(`<?xml version="1.0" encoding="UTF-8"?>
  <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" id="Definitions_editor" targetNamespace="urn:editor-test">
    <bpmn:process id="Process_editor" isExecutable="false"><bpmn:task id="Task_original" name="Check request" /></bpmn:process>
    <bpmndi:BPMNDiagram id="Diagram_editor"><bpmndi:BPMNPlane id="Plane_editor" bpmnElement="Process_editor"><bpmndi:BPMNShape id="Shape_original" bpmnElement="Task_original"><dc:Bounds x="200" y="150" width="100" height="80" /></bpmndi:BPMNShape></bpmndi:BPMNPlane></bpmndi:BPMNDiagram>
  </bpmn:definitions>`) });
  await page.getByRole("button", { name:"Edit baseline in canvas" }).click();
  const dialog = page.getByRole("region", { name:"Edit new BPMN" });
  await expect(dialog.getByRole("button", { name:"Use edited BPMN" })).toBeEnabled();
  const canvas = dialog.getByTestId("bpmn-editor-canvas");
  await dialog.locator('[data-action="create.task"]').click();
  await canvas.click({ position:{ x:500, y:180 } });
  await dialog.locator('[data-action="create.exclusive-gateway"]').click();
  await canvas.click({ position:{ x:700, y:180 } });
  const download = page.waitForEvent("download");
  await dialog.getByRole("button", { name:"Download BPMN" }).click();
  const xml = await readFile((await (await download).path())!, "utf8");
  expect(xml.match(/<bpmn:task\b/g)).toHaveLength(2);
  expect(xml).toContain("bpmn:exclusiveGateway");
  expect(xml.match(/<bpmndi:BPMNShape\b/g)).toHaveLength(3);
  expect(xml).toContain('id="Task_original"');
});
