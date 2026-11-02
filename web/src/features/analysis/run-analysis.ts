import type {Report} from "../../lib/types";

// Application use case: callers inject HTTP and persistence adapters. No React,
// window/localStorage, endpoint URLs or case-study knowledge belongs here.
export type AnalysisPort=(input:FormData)=>Promise<Report>;
export interface AnalysisRun {report:Report; comparison:Report[]}

export async function runAnalysis(input:FormData,analyze:AnalysisPort,onCreated?:(report:Report,repository:string)=>void):Promise<AnalysisRun>{
  const compare=input.get("compare_methods")==="on";
  const repository=String(input.get("repo_path")??"");
  function request(){const copy=new FormData();for(const [key,value]of input.entries())copy.append(key,value);copy.delete("compare_methods");return copy;}
  if(!compare){const report=await analyze(request());onCreated?.(report,repository);return{report,comparison:[]};}
  const comparison:Report[]=[];
  for(const method of ["bm25","semantic","fusion","hybrid"] as const){
    const data=request();data.set("method",method);data.delete("mapping_config_path");
    const report=await analyze(data);onCreated?.(report,repository);comparison.push(report);
  }
  if(new Set(comparison.map(report=>report.input_fingerprint)).size!==1)throw new Error("Comparison reports did not share the same input fingerprint.");
  return{report:comparison.find(report=>report.method==="hybrid")??comparison[0],comparison};
}
