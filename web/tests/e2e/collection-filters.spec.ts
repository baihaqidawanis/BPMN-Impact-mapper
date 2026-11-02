import {test,expect} from "@playwright/test";

test("library and history filters compose, reset, and recover after refresh failure",async({page})=>{
  const entries=[
    {id:"a".repeat(64),repository:"ERP",process:"Purchase",role:"Staff",level:2,version:"v1",findings:[]},
    {id:"b".repeat(64),repository:"ERP",process:"Purchase detail",role:"Supervisor",level:3,version:"v2",findings:[]},
    {id:"c".repeat(64),repository:"CRM",process:"Contact",role:"Staff",level:3,version:"v1",findings:[]},
  ];
  let fail=false;
  await page.route("**/api/v1/bpmn-library",route=>route.fulfill({json:{entries}}));
  await page.route("**/api/v1/analyses",route=>route.fulfill(fail?{status:503,json:{}}:{json:{analyses:[
    {analysis_id:"a".repeat(32),repository:{name:"ERP",revision:"baseline-1"},method:"bm25",changes:1,candidates:5,unresolved:0,warnings:0},
    {analysis_id:"b".repeat(32),repository:{name:"CRM",revision:"baseline-2"},method:"hybrid",changes:2,candidates:10,unresolved:1,warnings:0},
  ]}}));
  await page.goto("/?view=library");
  const library=page.getByRole("region",{name:"BPMN library"});
  await expect(library.locator("article")).toHaveCount(3);
  await library.getByLabel("Repository",{exact:true}).selectOption("ERP");
  await library.getByLabel("Level",{exact:true}).selectOption("3");
  await expect(library.locator("article")).toHaveCount(1);
  await expect(library.locator("article")).toContainText("Supervisor");
  await library.getByLabel("Role",{exact:true}).selectOption("Staff");
  await expect(library.getByText("No matching diagrams.")).toBeVisible();
  await library.getByRole("button",{name:"Clear filters",exact:true}).click();
  await expect(library.locator("article")).toHaveCount(3);
  await library.getByLabel("Find diagrams").fill("v2");
  await expect(library.locator("article")).toHaveCount(1);
  await page.getByRole("navigation",{name:"Workspace navigation"}).getByRole("button",{name:"History",exact:true}).click();
  const history=page.getByTestId("analysis-history");
  await expect(history.locator("article")).toHaveCount(2);
  await history.getByLabel("Mapping method",{exact:true}).selectOption("hybrid");
  await history.getByLabel("Find reports").fill("baseline-2");
  await expect(history.locator("article")).toHaveCount(1);
  await expect(history.locator("article")).toContainText("CRM");
  await history.getByLabel("Find reports").fill("missing-report");
  await expect(history.getByText("No matching reports. Change the search or clear filters.")).toBeVisible();
  await history.getByRole("button",{name:"Clear filters",exact:true}).click();
  await expect(history.locator("article")).toHaveCount(2);
  fail=true;await history.getByRole("button",{name:"Refresh history"}).click();
  await expect(history.getByRole("alert")).toContainText("could not be loaded");
  fail=false;await history.getByRole("button",{name:"Refresh history"}).click();
  await expect(history.getByRole("alert")).toHaveCount(0);
  await page.setViewportSize({width:390,height:844});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  await page.screenshot({path:test.info().outputPath("history-mobile.png"),fullPage:true});
});
