import { expect, test } from "@playwright/test";
import { methodDifferences } from "../../src/lib/method-differences";
import type { Report } from "../../src/lib/types";

function report(method: Report["method"], file = "src/request.go"): Report {
  return { analysis_id: `fixture-${method}`, input_fingerprint: "same-input", repository: {name:"Batch fixture", revision:"v1"}, method, warnings: [], changes: [{change_type:"ADD_TASK",old_element:null,new_element:{id:"T",name:"Check request",type:"task"},unresolved_impacts:[],impacts:[{file,module:"request",layer:"application",required_action:"VERIFY",confidence:0.5,evidence:[],score_breakdown:{trace_id:0,symbol:0,path:0,lexical:0,structured:0,bm25:0.5,total:0.5}}]}] };
}

test("method difference keeps shared/exclusive files and refuses mixed inputs", () => {
  const left=report("bm25"), right=report("semantic","src/validate.go");
  right.changes[0].impacts.push(left.changes[0].impacts[0]);
  const rows=methodDifferences([left,right])!;
  expect(rows.find(r=>r.file==="src/request.go")?.methods).toEqual(["bm25","semantic"]);
  expect(rows.find(r=>r.file==="src/validate.go")?.methods).toEqual(["semantic"]);
  right.input_fingerprint="different";
  expect(methodDifferences([left,right])).toBeNull();
});

test("batch retains successful runs, records failures, exports and opens partial results", async ({page}) => {
  await page.route("**/backend/api/v1/analyses", async route=>{
    if(route.request().method()==="GET") return route.fulfill({json:{analyses:[]}});
    const method=route.request().postData()?.match(/name="method"\r?\n\r?\n([^\r\n]+)/)?.[1];
    if(method==="semantic") return route.fulfill({status:503,json:{error:{message:"Local model has insufficient memory."}}});
    return route.fulfill({status:201,json:report("bm25")});
  });
  await page.goto("/?view=analysis");
  await page.getByRole("navigation",{name:"Workspace navigation"}).getByRole("button",{name:"Research",exact:true}).click();
  const batch=page.getByRole("region",{name:"Batch analysis"});
  await batch.getByLabel("Repository path",{exact:true}).fill("testdata/repository");
  await batch.getByLabel("Batch baseline BPMN").setInputFiles({name:"old.bpmn",mimeType:"application/xml",buffer:Buffer.from("<definitions/>")});
  await batch.getByLabel("Changed BPMN files (up to 10)").setInputFiles([{name:"new-a.bpmn",mimeType:"application/xml",buffer:Buffer.from("<definitions/>")},{name:"new-b.bpmn",mimeType:"application/xml",buffer:Buffer.from("<definitions/>")}]);
  await batch.getByLabel("Semantic",{exact:true}).check();
  await batch.getByRole("button",{name:"Run batch (4 analyses)"}).click();
  await expect(batch.getByRole("status")).toContainText("2 complete · 2 failed");
  await expect(batch.getByText("Local model has insufficient memory.",{exact:true})).toHaveCount(2);
  const download=page.waitForEvent("download");
  await batch.getByRole("button",{name:"Download run summary"}).click();
  expect((await download).suggestedFilename()).toBe("batch-runs.json");
  await page.screenshot({path:"test-results/batch-desktop.png",fullPage:true});
  await batch.getByRole("button",{name:"Open results for new-a.bpmn"}).click();
  await expect(page.getByTestId("report-view")).toContainText("Batch fixture");
  await expect(page.getByTestId("method-comparison")).toHaveCount(0);
});

test("batch stop prevents queued requests and restores controls", async ({page}) => {
  let posts=0;
  let release: ()=>void = ()=>{};
  const barrier=new Promise<void>(resolve=>{release=resolve;});
  await page.route("**/backend/api/v1/analyses",async route=>{
    if(route.request().method()==="GET") return route.fulfill({json:{analyses:[]}});
    posts++; await barrier; await route.abort().catch(()=>{});
  });
  await page.goto("/?view=analysis");
  await page.getByRole("navigation",{name:"Workspace navigation"}).getByRole("button",{name:"Research",exact:true}).click();
  const batch=page.getByRole("region",{name:"Batch analysis"});
  await batch.getByRole("button",{name:"Load 8 synthetic scenarios"}).click();
  await batch.getByRole("button",{name:"Run batch (8 analyses)"}).click();
  await expect.poll(()=>posts).toBe(1);
  await batch.getByRole("button",{name:"Stop batch"}).click(); release();
  await expect(batch.getByRole("status")).toContainText("8 cancelled");
  await expect(batch.getByRole("button",{name:"Run batch (8 analyses)"})).toBeEnabled();
  expect(posts).toBe(1);
});
