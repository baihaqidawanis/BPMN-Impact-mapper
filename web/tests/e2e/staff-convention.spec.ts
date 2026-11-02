import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

// Opt-in inspection of prepared/canonical documents, never source execution.
const root = process.env.STAFF_BPMN_REVIEW_DIR;
const files: string[] = [];
function collect(folder: string) {
  for (const entry of fs.readdirSync(folder, { withFileTypes: true })) {
    const file = path.join(folder, entry.name);
    if (entry.isSymbolicLink()) continue;
    if (entry.isDirectory()) collect(file);
    else if (entry.name.endsWith(".bpmn")) files.push(file);
  }
}
if (root) collect(root);
if (!root) test.skip("Staff convention review requires STAFF_BPMN_REVIEW_DIR", () => {});
for (const file of files.sort()) {
  test(`Staff convention: ${root ? path.relative(root, file) : path.basename(file)}`, async ({ page }) => {
    await page.goto("/?view=analysis");
    await page.getByRole("navigation", { name: "Workspace navigation" }).getByRole("button", { name: "BPMN Canvas", exact: true }).click();
    await page.getByLabel("Canvas baseline BPMN").setInputFiles(file);
    const editor = page.getByRole("region", { name: "Edit new BPMN" });
    await expect(editor.getByRole("button", { name: "Use edited BPMN" })).toBeEnabled();
    await expect(editor.getByRole("status")).not.toContainText("import warnings");
    const xml = fs.readFileSync(file, "utf8");
    const ids = [...xml.matchAll(/bpmnElement="([^"]+)"/g)].map(m => m[1]).filter(id => !id.startsWith("Collab_"));
    const canvas = editor.getByTestId("bpmn-editor-canvas");
    for (const id of ids) await expect(canvas.locator(`.djs-element[data-element-id="${id}"]`)).toHaveCount(1);
    for (const edge of await canvas.locator(".djs-connection .djs-visual path").all()) {
      expect(await edge.evaluate(e => (e as SVGPathElement).getTotalLength())).toBeGreaterThan(0);
    }
    if (path.basename(file) === "after.bpmn" || /l0\.bpmn|l2-pr\.bpmn|l3-adj-submit\.bpmn|l2-inv-adjustment\.bpmn|l3-master-list\.bpmn/.test(file)) {
      await editor.getByRole("button", { name: "Expand editor" }).click();
      await editor.getByRole("button", { name: "Fit diagram", exact: true }).click();
      await editor.screenshot({ path: test.info().outputPath(`${path.basename(file)}.png`) });
    }
  });
}
