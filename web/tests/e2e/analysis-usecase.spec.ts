import {expect,test} from "@playwright/test";
import {runAnalysis} from "../../src/features/analysis/run-analysis";
import type {Report} from "../../src/lib/types";

function saved(method:Report["method"],fingerprint="same"):Report{
  return {analysis_id:method,input_fingerprint:fingerprint,method,repository:{name:"Example",revision:"snapshot"},changes:[],warnings:[]};
}

test("single analysis preserves caller input and invokes the injected adapter once",async()=>{
  const input=new FormData();input.set("repo_path","repo");input.set("method","bm25");
  let calls=0;let remembered="";
  const result=await runAnalysis(input,async data=>{calls++;data.set("repo_path","mutated request");return saved("bm25");},(_,repo)=>{remembered=repo;});
  expect(calls).toBe(1);expect(result.comparison).toEqual([]);expect(input.get("repo_path")).toBe("repo");expect(remembered).toBe("repo");
});
test("comparison isolates inputs, uses one fingerprint and selects hybrid",async()=>{
  const input=new FormData();input.set("compare_methods","on");input.set("mapping_config_path","private.json");input.set("method","hybrid");
  const calls:string[]=[];
  const result=await runAnalysis(input,async data=>{expect(data.has("compare_methods")).toBe(false);expect(data.has("mapping_config_path")).toBe(false);const method=data.get("method") as Report["method"];calls.push(method);data.delete("method");return saved(method);});
  expect(calls).toEqual(["bm25","semantic","fusion","hybrid"]);expect(result.report.method).toBe("hybrid");expect(result.comparison).toHaveLength(4);expect(input.get("compare_methods")).toBe("on");expect(input.has("mapping_config_path")).toBe(true);
});
test("comparison rejects mixed snapshots and preserves created report notifications",async()=>{
  const input=new FormData();input.set("compare_methods","on");let remembered=0;
  await expect(runAnalysis(input,async data=>{const method=data.get("method") as Report["method"];return saved(method,method);},()=>{remembered++;})).rejects.toThrow("same input fingerprint");
  expect(remembered).toBe(4);
});
test("adapter failure propagates and stops comparison",async()=>{
  const input=new FormData();input.set("compare_methods","on");let calls=0;
  await expect(runAnalysis(input,async()=>{calls++;throw new Error("Offline");})).rejects.toThrow("Offline");expect(calls).toBe(1);
});
