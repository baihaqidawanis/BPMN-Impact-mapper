import { test, expect } from "@playwright/test";
import path from "node:path";

for (const scenario of [
  { name: "add-task baseline", file: "insert-task-old.bpmn", nodes: ["S", "A", "E"], flows: ["F1", "F2"] },
  { name: "add-task result / remove-task baseline", file: "insert-task-new.bpmn", nodes: ["S", "A", "B", "E"], flows: ["F1", "F2", "F3"] },
  { name: "gateway-branch baseline", file: "add-branch-old.bpmn", nodes: ["S", "A", "E"], flows: ["F1", "F2"] },
  { name: "gateway-branch result / removal baseline", file: "add-branch-new.bpmn", nodes: ["S", "G", "A", "B", "E"], flows: ["F1", "F2", "F3", "F4", "F5"] },
]) {
  test(`${scenario.name} renders every task and sequence flow in the BPMN canvas`, async ({ page }) => {
    await page.goto("/?view=analysis");
    await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "BPMN Canvas", exact: true }).click();
    await page.getByLabel("Canvas baseline BPMN").setInputFiles(path.resolve(__dirname, "../../public/demo", scenario.file));
    const editor = page.getByRole("region", { name: "Edit new BPMN" });
    const canvas = editor.getByTestId("bpmn-editor-canvas");
    await expect(editor.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
    await expect(editor.getByRole("status")).not.toContainText("import warnings");
    for (const id of scenario.nodes) {
      await expect(canvas.locator(`.djs-element[data-element-id="${id}"]`)).toBeVisible();
    }
    for (const id of scenario.flows) {
      const edge = canvas.locator(`.djs-connection[data-element-id="${id}"]`);
      await expect(edge).toHaveCount(1);
      expect(await edge.locator(".djs-visual path").first().evaluate(element => (element as SVGPathElement).getTotalLength())).toBeGreaterThan(0);
    }
    await editor.screenshot({ path: test.info().outputPath(`${scenario.file}.png`) });
  });
}

test("Simple-ERP-derived gateway and task scenario imports with visible diagram elements", async ({ page }) => {
  await page.goto("/?view=analysis");
  await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "BPMN Canvas", exact: true }).click();
  await page.getByLabel("Canvas baseline BPMN").setInputFiles(path.resolve(__dirname, "../../../docs/demo-bpmn/simple-erp-p2p-add-gateway-task.bpmn"));
  const editor = page.getByRole("region", { name: "Edit new BPMN" });
  const canvas = editor.getByTestId("bpmn-editor-canvas");
  await expect(editor.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
  await expect(editor.getByRole("status")).not.toContainText("import warnings");
  for (const id of ["GW-P2P-DETAIL-CHECK", "PUR-PR-002A", "PUR-PR-003"]) {
    await expect(canvas.locator(`.djs-element[data-element-id="${id}"]`)).toBeVisible();
  }
  for (const id of ["FLOW-P2P-003", "FLOW-P2P-003A", "FLOW-P2P-003B", "FLOW-P2P-003C"]) {
    const edge = canvas.locator(`.djs-connection[data-element-id="${id}"]`);
    await expect(edge).toHaveCount(1);
    expect(await edge.locator(".djs-visual path").first().evaluate(element => (element as SVGPathElement).getTotalLength())).toBeGreaterThan(0);
  }
  await editor.screenshot({ path: test.info().outputPath("simple-erp-p2p-gateway-task.png") });
});

test("inventory reason-task scenario has a visible inserted task and both connecting flows", async ({ page }) => {
  await page.goto("/?view=analysis");
  await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "BPMN Canvas", exact: true }).click();
  await page.getByLabel("Canvas baseline BPMN").setInputFiles(path.resolve(__dirname, "../../../experiments/simple-erp/staff-inventory/role-staff-l3-adj-submit-reason-task.bpmn"));
  const editor = page.getByRole("region", { name: "Edit new BPMN" });
  const canvas = editor.getByTestId("bpmn-editor-canvas");
  await expect(editor.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
  await expect(editor.getByRole("status")).not.toContainText("import warnings");
  await expect(canvas.locator('[data-element-id="role_staff_l3_adj_submit_reason"]')).toBeVisible();
  for (const id of ["role_staff_l3_adj_submit_Flow_0", "role_staff_l3_adj_submit_Flow_0A"]) {
    const edge = canvas.locator(`.djs-connection[data-element-id="${id}"]`);
    await expect(edge).toHaveCount(1);
    expect(await edge.locator(".djs-visual path").first().evaluate(element => (element as SVGPathElement).getTotalLength())).toBeGreaterThan(0);
  }
});
