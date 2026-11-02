"use client";

import { useEffect, useRef, useState } from "react";
import { createAnalysis } from "@/lib/api";
import type { AnalysisMethod, Report } from "@/lib/types";
import styles from "./BatchAnalysis.module.css";

type Scenario = { id: string; name: string; old: File; next: File };
type Run = { scenario: string; topK: number; method: AnalysisMethod; status: "waiting" | "running" | "complete" | "failed" | "cancelled"; report?: Report; error?: string };
const methods = [["bm25", "BM25"], ["semantic", "Semantic"], ["fusion", "BM25 + Semantic"], ["hybrid", "BPMN-context Hybrid"]] as const;

export function BatchAnalysis({ busy, onBusy, onOpen }: { busy: boolean; onBusy: (busy: boolean) => void; onOpen: (reports: Report[]) => void }) {
  const [repo, setRepo] = useState("");
  const [revision, setRevision] = useState("");
  const [topK, setTopK] = useState(5);
  const [baseline, setBaseline] = useState<File | null>(null);
  const [scenarios, setScenarios] = useState<Scenario[]>([]);
  const [selected, setSelected] = useState<AnalysisMethod[]>(["bm25"]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [running, setRunning] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);

  async function loadDemo() {
    setLoading(true); setError("");
    try {
      const response = await fetch("/demo/scenarios.json");
      if (!response.ok) throw new Error("Scenario catalog could not be loaded.");
      const catalog = await response.json() as {id:string; name:string; old:string; new:string}[];
      const cases: Scenario[] = [];
      for (const item of catalog) {
        const old = await fetch(`/demo/${item.old}`), next = await fetch(`/demo/${item.new}`);
        if (!old.ok || !next.ok) throw new Error("A scenario BPMN file is unavailable.");
        cases.push({id:item.id, name:item.name, old:new File([await old.blob()],item.old), next:new File([await next.blob()],item.new)});
      }
      setScenarios(cases); setBaseline(null); setRepo("testdata/repository"); setRevision("fixture-v1"); setRuns([]);
    } catch (reason) {setError(reason instanceof Error ? reason.message : "Demo load failed.");}
    finally {setLoading(false);}
  }

  async function start() {
    if (running || busy || !repo.trim() || !scenarios.length || !selected.length) return;
    const abort = new AbortController(); controller.current = abort;
    const queue: Run[] = scenarios.flatMap(scenario => selected.map(method => ({scenario:scenario.id, topK, method, status:"waiting"})));
    const fingerprints = new Map<string,string>();
    const publish = () => setRuns(queue.map(run => ({...run})));
    setRunning(true); onBusy(true); setError(""); publish();
    try {
      for (const run of queue) {
        if (abort.signal.aborted) {run.status="cancelled"; publish(); continue;}
        run.status="running"; publish();
        const scenario = scenarios.find(s => s.id === run.scenario)!;
        const input = new FormData();
        input.set("repo_path",repo.trim()); input.set("revision",revision.trim()); input.set("top_k",String(topK)); input.set("method",run.method);
        input.set("old_bpmn",scenario.old); input.set("new_bpmn",scenario.next);
        try {
          const report = await createAnalysis(input,abort.signal);
          if (report.method !== run.method) throw new Error("The returned method does not match the requested method.");
          if (!report.input_fingerprint || (fingerprints.has(run.scenario) && fingerprints.get(run.scenario)!==report.input_fingerprint)) throw new Error("Input changed between methods. Freeze the repository before comparing results.");
          fingerprints.set(run.scenario,report.input_fingerprint);
          run.report=report; run.status="complete";
        } catch (reason) {
          run.status=abort.signal.aborted ? "cancelled" : "failed";
          run.error=abort.signal.aborted ? "Stopped. A completed server report may still appear in Saved analyses." : reason instanceof Error ? reason.message : "Analysis failed.";
        }
        publish();
      }
    } finally {setRunning(false); onBusy(false); controller.current=null;}
  }

  function download() {
    const data = {version:1, runs:runs.map(run => ({scenario:run.scenario, top_k:run.topK, method:run.method, status:run.status, analysis_id:run.report?.analysis_id, input_fingerprint:run.report?.input_fingerprint, error:run.error}))};
    const url=URL.createObjectURL(new Blob([JSON.stringify(data,null,2)],{type:"application/json"}));
    const anchor=document.createElement("a"); anchor.href=url; anchor.download="batch-runs.json"; anchor.click(); URL.revokeObjectURL(url);
  }

  return <section className={styles.root} aria-label="Batch analysis">
    <header><h3>Run multiple scenario experiments</h3><p>Each BPMN pair and selected method produces a separate saved report. A shared baseline with several changed files tests alternative scenarios; it does not merge them into one process. Accuracy is measured separately in Evaluation using expected files. This batch does not run Qwen review.</p></header>
    <fieldset disabled={running || busy || loading}>
      <div className={styles.fields}>
        <label>Repository path<input value={repo} onChange={e=>setRepo(e.target.value)} placeholder="Server-local repository path" /></label>
        <label>Revision<input value={revision} onChange={e=>setRevision(e.target.value)} placeholder="Commit or snapshot" /></label>
        <label>Top-K<input type="number" min={1} max={100} value={topK} onChange={e=>setTopK(Number(e.target.value))} /></label>
      </div>
      <div className={styles.fields}>
        <label>Batch baseline BPMN<input type="file" accept=".bpmn,.xml" onChange={e=>{setBaseline(e.target.files?.[0]??null);setScenarios([]);setRuns([]);}} /></label>
        <label>Changed BPMN files (up to 10)<input type="file" multiple accept=".bpmn,.xml" disabled={!baseline} onChange={e=>{
          const files=Array.from(e.target.files??[]); if(files.length>10){setError("Select at most 10 changed BPMN files.");setScenarios([]);return;}
          setError(""); setRuns([]); setScenarios(files.map((file,index)=>({id:`upload-${index+1}`,name:file.name,old:baseline!,next:file})));
        }} /></label>
      </div>
      <button type="button" onClick={()=>void loadDemo()}>{loading ? "Loading scenarios…" : "Load 8 synthetic scenarios"}</button>
      <div className={styles.methods}>{methods.map(([value,label])=><label key={value}><input type="checkbox" checked={selected.includes(value)} onChange={e=>setSelected(current=>e.target.checked?[...current,value]:current.filter(item=>item!==value))} />{label}</label>)}</div>
    </fieldset>
    {error && <p role="alert">{error}</p>}
    <div className={styles.actions}><button type="button" disabled={running||busy||loading||!repo.trim()||!scenarios.length||!selected.length||!Number.isInteger(topK)||topK<1||topK>100} onClick={()=>void start()}>Run batch ({scenarios.length*selected.length} analyses)</button>{running && <button type="button" onClick={()=>controller.current?.abort()}>Stop batch</button>}{runs.length>0 && <button type="button" onClick={download}>Download run summary</button>}</div>
    <p role="status">{runs.filter(run=>run.status==="complete").length} complete · {runs.filter(run=>run.status==="failed").length} failed · {runs.filter(run=>run.status==="cancelled").length} cancelled{running ? " · Running" : ""}</p>
    <p>Keep the repository unchanged during a run. Completed reports are saved; this queue stays only while the page is open.</p>
    <div className={styles.results}>{scenarios.map(scenario=><article key={scenario.id}><h4>{scenario.name}</h4><p>{scenario.old.name} → {scenario.next.name}</p>{runs.filter(run=>run.scenario===scenario.id).map(run=><div className={styles.run} key={run.method}><span>{methods.find(m=>m[0]===run.method)?.[1]}</span><b>{run.status}</b>{run.report && <small>{run.report.changes.length} changes · {run.report.changes.reduce((n,c)=>n+c.impacts.length,0)} matches</small>}{run.error && <p>{run.error}</p>}</div>)}{runs.some(run=>run.scenario===scenario.id&&run.report) && <button type="button" onClick={()=>onOpen(runs.filter(run=>run.scenario===scenario.id&&run.report).map(run=>run.report!))}>Open results for {scenario.name}</button>}</article>)}</div>
  </section>;
}
