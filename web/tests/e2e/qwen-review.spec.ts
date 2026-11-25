import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(__dirname, "../../..");

test("review sends the selected BPMN change and resumes with its saved scope", async ({ page, request }) => {
  const response = await request.post("/backend/api/v1/analyses", { multipart: {
    repo_path: path.join(root, "testdata/repository"),
    method: "bm25",
    old_bpmn: { name: "old.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/old.bpmn")) },
    new_bpmn: { name: "new.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/new.bpmn")) },
  } });
  expect(response.ok()).toBe(true);
  const report = await response.json();
  expect(report.changes.length).toBeGreaterThan(1);
  const index = 1;
  const sessionID = "7".repeat(32);
  const provider = { provider: "fixture", model: "qwen-fixture", prompt_version: "v1" };
  const checkpoint = {
    session_id: sessionID, analysis_id: report.analysis_id, mode: "ultra-light",
    change_index: index, status: "PAUSED", role: "analyst", turns: 1, tools: 0, revisions: 0, provider,
  };
  await page.route("**/api/v1/capabilities/agents/check", route => route.fulfill({
    json: { enabled: true, available: true, provider },
  }));
  const starts: { change_index: number; session_id?: string; mode: string }[] = [];
  await page.route(`**/api/v1/analyses/${report.analysis_id}/agent-sessions`, route => {
    starts.push(route.request().postDataJSON());
    return route.fulfill({ json: checkpoint });
  });
  await page.route(`**/api/v1/agent-sessions/${sessionID}`, route => route.fulfill({ json: checkpoint }));

  await page.goto(`/?analysis=${report.analysis_id}&view=analysis`);
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  const panel = page.getByTestId("qwen-review");
  await panel.getByLabel("Repository path for AI review").fill("C:\\fixture-repo");
  await panel.getByLabel("BPMN changes to review").selectOption(String(index));
  await panel.getByRole("radio", { name: /Ultra Light/ }).check();
  await panel.getByRole("button", { name: "Run AI review", exact: true }).click();
  await expect.poll(() => starts.length).toBe(1);
  expect(starts[0]).toMatchObject({ change_index: index, mode: "ultra-light" });

  await page.evaluate(id => localStorage.removeItem(`bpmn-agent-session:v2:${id}`), report.analysis_id);
  await page.goto(`/?analysis=${report.analysis_id}&agentSession=${sessionID}`);
  await expect(panel.getByRole("heading", { name: /paused/i })).toBeVisible();
  if (await panel.getByRole("button", { name: "Change path" }).isVisible()) {
    await panel.getByRole("button", { name: "Change path" }).click();
  }
  await panel.getByLabel("Repository path for AI review").fill("C:\\fixture-repo");
  await panel.getByRole("button", { name: "Resume review" }).click();
  await expect.poll(() => starts.length).toBe(2);
  expect(starts[1]).toMatchObject({ change_index: index, mode: "ultra-light", session_id: sessionID });
});

test("Qwen review shows scoped progress, plan and restores without another model start", async ({ page, request }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const saved = await request.post("/backend/api/v1/analyses", { multipart: {
    repo_path: path.join(root, "testdata/repository"), repository_name: "Qwen browser fixture", method: "bm25", top_k: "3",
    old_bpmn: { name: "old.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/old.bpmn")) },
    new_bpmn: { name: "new.bpmn", mimeType: "text/xml", buffer: fs.readFileSync(path.join(root, "testdata/bpmn/new.bpmn")) },
  } });
  expect(saved.ok()).toBe(true); const report = await saved.json();
  const identity = { provider: "scripted-browser", model: "qwen-fixture", prompt_version: "v1" };
  const sessionID = "1".repeat(32), planID = "2".repeat(32);
  const plan = { version: "1", plan_id: planID, analysis_id: report.analysis_id, context_id: "3".repeat(32), input_fingerprint: report.input_fingerprint, provider: identity, change_coverage: [{change_type:"ADD_TASK",element_id:"TaskNew",plans:1,status:"PLANNED"},{change_type:"ADD_GATEWAY",element_id:"GatewayNew",plans:0,status:"UNRESOLVED"}], status: "NEEDS_REVIEW", review_issues: ["Confirm the rejection contract."], plans: [{ file: "application/submit.go", module: "application", layer: "application", action: "VERIFY", current_behavior:"Submit accepts an invalid amount.",target_behavior:"Reject invalid amounts before state mutation.",design_rationale:"Reuse the existing submission entry point.",assumptions:["Confirm error response contract."], summary: "Propose validation before submission.", steps: ["Check invalid input before mutating status."], proposed_changes: ["Reject invalid amounts before changing status."], acceptance_checks: ["Zero amount leaves the original state unchanged."], evidence_ids: ["E1"] }] };
  let starts = 0, polls = 0;
  await page.route("**/api/v1/capabilities/agents/check", route => route.fulfill({ json: { enabled: true, available: true, provider: identity, message: "Model metadata ready." } }));
  await page.route(`**/api/v1/analyses/${report.analysis_id}/agent-sessions`, async route => {
    starts++; const body = route.request().postDataJSON(); expect(body.change_index).toBe(-1); expect(body.mode).toBe("full"); expect(body.repository_path).toBe("C:\\fixture-repo");
    await route.fulfill({ status: 202, json: { session_id: sessionID, analysis_id: report.analysis_id, provider: identity, status: "STARTING", role: "analyst", turns: 0, tools: 0, revisions: 0 } });
  });
  await page.route(`**/api/v1/agent-sessions/${sessionID}`, route => {
    polls++; return route.fulfill({ json: { session_id: sessionID, analysis_id: report.analysis_id, provider: identity, context: {repository_overview:{file:"README.md",summary:"README describes an inventory application; source verification is required.",truncated:true}}, source_activity: [{change_index:0,tool:"search_source",query:"validate reason",files:["application/submit.go"],truncated:true},{change_index:0,tool:"read_source",file:"application/submit.go",files:["application/submit.go"],next_line:42}], mode: "full", full_review: { consistency_review: polls===1 ? "pending" : "needs_review", phase: polls === 1 ? "scanning" : "finished", files_total: 2, files_reviewed: polls === 1 ? 1 : 2, chunks_total: 3, chunks_reviewed: polls === 1 ? 1 : 3, batches_total: 1, batches_reviewed: polls === 1 ? 0 : 1, relevant_files: 1, weak_files: 0 }, status: polls === 1 ? "RUNNING" : "NEEDS_REVIEW", role: polls === 1 ? "analyst" : "reviewer", turns: polls === 1 ? 1 : 3, tools: 0, revisions: 0, plan: polls > 1 ? plan : undefined } });
  });
  await page.goto(`/?analysis=${report.analysis_id}`);
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  const panel = page.getByTestId("qwen-review");
  await expect(panel).toContainText("Connected");
  await expect(panel.getByRole("radio", {name: /Light review/})).toBeChecked();
  await panel.getByRole("radio", {name: /Full repository review/}).check();
  if (await panel.getByRole("button", {name: "Change path"}).isVisible()) {
    await panel.getByRole("button", {name: "Change path"}).click();
  }
  await panel.getByLabel("Repository path for AI review").fill("C:\\fixture-repo");
  await panel.getByRole("button", { name: "Run AI review", exact: true }).click();
  const stages = panel.getByRole("list", { name: "AI agent progress" });
  await expect(stages.locator('[data-state="active"]')).toHaveCount(1);
  await expect(panel.getByRole("heading", { name: /needs review/i })).toBeVisible({ timeout: 20_000 });
  await expect(stages.locator('[data-state="active"]')).toHaveCount(0);
  await expect(stages.locator('[data-state="review"]')).toContainText("Needs review");
  await expect(panel.getByRole("list",{name:"AI agent progress"})).toBeVisible();
  await expect(panel.getByRole("list",{name:"AI agent progress"}).getByText("Reviewer",{exact:true})).toBeVisible();
  await expect(panel.getByRole("button",{name:"Start new AI review",exact:true})).toBeVisible();
  await expect(panel).toContainText("Confirm the rejection contract.");
  await expect(panel).toContainText("2/2 eligible files");
  await expect(panel.getByRole("progressbar", {name:"Reviewed source chunks"})).toHaveAttribute("value","3");
  await expect(panel.getByRole("link", { name: "Download plan Markdown" })).toHaveAttribute("href", `/backend/api/v1/plans/${planID}/change-plan.md`);
  await expect(panel).toContainText("Reject invalid amounts before changing status.");
  await expect(panel).toContainText("Submit accepts an invalid amount.");
  await expect(panel).toContainText("Reuse the existing submission entry point.");
  await expect(panel).toContainText("Zero amount leaves the original state unchanged.");
  await panel.getByText("Repository background from README",{exact:true}).click();
  await expect(panel).toContainText("README describes an inventory application");
  await expect(panel).toContainText("first 8 KiB");
  await expect(panel).toContainText("Final consistency review: needs review");
  await panel.getByText("BPMN change coverage",{exact:true}).click();
  await expect(panel).toContainText("GatewayNew");await expect(panel).toContainText("UNRESOLVED");
  await panel.getByText("Source access log (2 lookups)",{exact:true}).click();
  await expect(panel).toContainText("Query: validate reason");
  await expect(panel).toContainText("Returned source: application/submit.go");
  await expect(panel).toContainText("Next source window starts at line 42.");
  await panel.getByRole("radio", {name: /Focused review/}).check();
  await expect(panel.getByRole("link", {name: "Download plan Markdown"})).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  const files = panel.getByRole("region", { name: "Proposed file edits" });
  await expect(files).toBeVisible();
  const row = files.locator("summary").first();
  await row.focus(); await row.press("Enter");
  await expect(files.getByText("Submit accepts an invalid amount.")).toBeHidden();
  await row.press("Enter");
  await expect(files.getByText("Submit accepts an invalid amount.")).toBeVisible();
  expect(await panel.evaluate(el => el.scrollWidth <= el.clientWidth + 1)).toBe(true);
  expect(starts).toBe(1);
  await page.reload();
  await page.getByRole("group", { name: "Analysis workflow" }).getByRole("button", { name: "Planning", exact: true }).click();
  await expect(panel.getByRole("heading", { name: /needs review/i })).toBeVisible(); expect(starts).toBe(1);
  await stages.screenshot({ path: test.info().outputPath("agent-stages.png") });
});

test("historical null plan arrays show an empty result without crashing",async({page,request})=>{
 const response=await request.post("/backend/api/v1/analyses",{multipart:{repo_path:path.join(root,"testdata/repository"),method:"bm25",old_bpmn:{name:"old.bpmn",mimeType:"text/xml",buffer:fs.readFileSync(path.join(root,"testdata/bpmn/old.bpmn"))},new_bpmn:{name:"new.bpmn",mimeType:"text/xml",buffer:fs.readFileSync(path.join(root,"testdata/bpmn/new.bpmn"))}}});
 expect(response.ok()).toBe(true);const report=await response.json();const id="9".repeat(32);const errors:string[]=[];page.on("pageerror",e=>errors.push(e.message));
 await page.addInitScript(({analysis,id})=>localStorage.setItem(`bpmn-agent-session:v2:${analysis}`,JSON.stringify({id,index:-1,mode:"focused"})),{analysis:report.analysis_id,id});
 await page.route("**/api/v1/capabilities/agents/check",r=>r.fulfill({json:{enabled:true,available:false,message:"Offline fixture"}}));
 await page.route(`**/api/v1/agent-sessions/${id}`,r=>r.fulfill({json:{session_id:id,analysis_id:report.analysis_id,status:"NEEDS_REVIEW",role:"planner",turns:2,tools:0,revisions:0,plan:{plan_id:"8".repeat(32),status:"NEEDS_REVIEW",plans:null,review_issues:["No valid plan for the selected requirement."]}}}));
 await page.goto(`/?analysis=${report.analysis_id}`);await page.getByRole("group",{name:"Analysis workflow"}).getByRole("button",{name:"Planning",exact:true}).click();
 const panel=page.getByTestId("qwen-review");await expect(panel.getByRole("heading",{name:"No source edits were proposed"})).toBeVisible();await expect(panel).toContainText("No valid plan for the selected requirement.");expect(errors).toEqual([]);
 await page.reload();await page.getByRole("group",{name:"Analysis workflow"}).getByRole("button",{name:"Planning",exact:true}).click();await expect(panel.getByRole("heading",{name:"No source edits were proposed"})).toBeVisible();expect(errors).toEqual([]);
});
