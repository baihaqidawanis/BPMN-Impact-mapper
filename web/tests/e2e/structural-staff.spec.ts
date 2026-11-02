import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

for (const scenario of ["remove-task", "add-xor", "remove-gateway", "remove-gateway-control", "move-role", "add-and", "add-or", "staff-single-end"]) {
  test(`${scenario} imports complete before and after diagrams`, async ({ page }) => {
    for (const name of ["before.bpmn", "after.bpmn"]) {
      await page.goto("/?view=analysis");
      await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "BPMN Canvas", exact: true }).click();
      const folder = scenario === "staff-single-end"
        ? path.resolve(__dirname, "../../../experiments/simple-erp/staff-single-end")
        : path.resolve(__dirname, "../../../experiments/simple-erp/structural-staff", scenario);
      const file = path.join(folder, name);
      const xml = fs.readFileSync(file, "utf8");
      const elementIDs = [...xml.matchAll(/bpmnElement="([^"]+)"/g)].map(match => match[1]);
      await page.getByLabel("Canvas baseline BPMN").setInputFiles(file);
      const editor = page.getByRole("region", { name: "Edit new BPMN" });
      await expect(editor.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
      await expect(editor.getByRole("status")).not.toContainText("import warnings");
      const canvas = editor.getByTestId("bpmn-editor-canvas");
      for (const id of elementIDs.filter(id => !id.startsWith("Collab_"))) {
        await expect(canvas.locator(`.djs-element[data-element-id="${id}"]`)).toHaveCount(1);
      }
      for (const edge of await canvas.locator(".djs-connection .djs-visual path").all()) {
        expect(await edge.evaluate(element => (element as SVGPathElement).getTotalLength())).toBeGreaterThan(0);
      }
      if (name === "after.bpmn") await editor.screenshot({ path: test.info().outputPath(`${scenario}.png`) });
    }
  });
}
