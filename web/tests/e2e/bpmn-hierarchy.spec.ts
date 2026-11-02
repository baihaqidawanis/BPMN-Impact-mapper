import {expect,test} from "@playwright/test";
import fs from "node:fs/promises";
import path from "node:path";

test("explicit detail link persists and opens a read-only child with breadcrumb return",async({page,request})=>{
  test.setTimeout(60_000);
  const xml=await fs.readFile(path.resolve(__dirname,"../../public/demo/insert-task-old.bpmn"),"utf8");
  const repo=`Hierarchy ${Date.now()}`;
  async function add(process:string,level:number){
    const response=await request.post("/backend/api/v1/bpmn-library",{data:{repository:repo,process,role:"Staff",level,version:"v1",xml}});
    expect(response.status()).toBe(201);return await response.json();
  }
  const parent=await add("Parent level 2",2);const child=await add("Child level 3",3);
  await page.goto("/?view=analysis");
  const nav=page.getByRole("navigation",{name:"Workspace navigation"});
  await nav.getByRole("button",{name:"BPMN Library",exact:true}).click();
  await page.getByLabel("Find diagrams").fill(repo);
  await page.locator("article").filter({hasText:parent.process}).getByRole("button",{name:"View process and details"}).click();
  const details=page.getByRole("region",{name:"Process detail browser"});
  const canvas=details.getByTestId("library-diagram");
  await expect(canvas.locator('[data-element-id="A"]').first()).toBeVisible();
  const transform=await canvas.locator(".viewport").getAttribute("transform");
  await details.getByRole("button",{name:"Zoom in",exact:true}).click();
  expect(await canvas.locator(".viewport").getAttribute("transform")).not.toBe(transform);
  await details.getByRole("button",{name:"Fit diagram",exact:true}).click();
  await canvas.locator('[data-element-id="A"]').first().click();
  await expect(details.getByLabel("Selected activity")).toHaveValue("A");
  await expect(details.getByText("Detail has not been linked.")).toBeVisible();
  await details.getByLabel("Detail diagram",{exact:true}).selectOption(child.id);
  await details.getByRole("button",{name:"Save detail link"}).click();
  await expect(details.getByRole("status")).toContainText("saved");
  await details.getByRole("button",{name:"Open process detail",exact:true}).click();
  await expect(details.getByRole("heading",{name:child.process,exact:true})).toBeVisible();
  await expect(canvas.locator(".djs-palette")).toHaveCount(0);
  await details.getByRole("navigation",{name:"Process breadcrumb"}).getByRole("button",{name:parent.process,exact:true}).click();
  await expect(details.getByRole("heading",{name:parent.process,exact:true})).toBeVisible();
  await details.getByLabel("Selected activity").selectOption("A");
  await expect(details.getByRole("button",{name:"Open process detail",exact:true})).toBeVisible();
  await page.reload();
  await nav.getByRole("button",{name:"BPMN Library",exact:true}).click();
  await page.getByLabel("Find diagrams").fill(repo);
  await page.locator("article").filter({hasText:parent.process}).getByRole("button",{name:"View process and details"}).click();
  await details.getByLabel("Selected activity").selectOption("A");
  await expect(details.getByRole("button",{name:"Open process detail",exact:true})).toBeVisible();
  await page.setViewportSize({width:390,height:844});
  await expect.poll(async()=>{
    const box=await canvas.boundingBox();const task=await canvas.locator('[data-element-id="A"]').first().boundingBox();
    return !!box&&!!task&&task.x>=box.x-1&&task.x+task.width<=box.x+box.width+1;
  }).toBe(true);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  await page.screenshot({path:test.info().outputPath("process-detail-mobile.png"),fullPage:true});
  await details.getByLabel("Detail diagram",{exact:true}).selectOption("");
  await details.getByRole("button",{name:"Save detail link"}).click();
  await expect(details.getByRole("status")).toContainText("removed");
  await expect(details.getByRole("button",{name:"Open process detail",exact:true})).toHaveCount(0);
});
