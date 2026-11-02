"use client";

import {useEffect,useState} from "react";
import {getAnalysis,listAnalyses} from "@/lib/api";
import {listDiagrams,type DiagramEntry} from "@/lib/bpmn-library";
import type {Report,ReportSummary} from "@/lib/types";
import styles from "./WorkspaceOverview.module.css";

export function WorkspaceOverview({active,current,onStart,onContinue,onOpen,onLibrary,onHistory}:{active:boolean;current:Report|null;onStart:()=>void;onContinue:()=>void;onOpen:(r:Report)=>void;onLibrary:()=>void;onHistory:()=>void}){
  const [reports,setReports]=useState<ReportSummary[]|null>(null);
  const [diagrams,setDiagrams]=useState<DiagramEntry[]|null>(null);
  const [notice,setNotice]=useState("");const [error,setError]=useState("");const [busy,setBusy]=useState("");
  useEffect(()=>{
    if(!active)return;let mounted=true;
    Promise.allSettled([listAnalyses(),listDiagrams()]).then(([a,b])=>{
      if(!mounted)return;
      setReports(a.status==="fulfilled"?a.value:null);setDiagrams(b.status==="fulfilled"?b.value:null);
      setNotice(a.status==="rejected"||b.status==="rejected"?"Some workspace data could not be loaded. Check that the backend is running.":"");
    });return()=>{mounted=false;};
  },[active,current?.analysis_id]);
  async function open(id:string){setBusy(id);setError("");try{onOpen(await getAnalysis(id));}catch{setError("This report could not be opened. Check the backend and retry.");}finally{setBusy("");}}
  return <section className={styles.overview} aria-label="Workspace overview">
    <div className={styles.intro}><div><h1>Your analysis workspace</h1><p>Compare process versions, inspect the affected files, and prepare a change plan.</p></div><button type="button" onClick={onStart}>New analysis <span aria-hidden="true">↗</span></button></div>
    {notice&&<p role="status" className={styles.notice}>{notice}</p>}{error&&<p role="alert">{error}</p>}
    <div className={styles.summary}><div><b>{reports?.length??"—"}</b><span>Saved analyses</span></div><div><b>{diagrams?.length??"—"}</b><span>BPMN diagrams</span></div><div><b>{current?current.repository.name:"None selected"}</b><span>Current analysis</span></div></div>
    {current&&<div className={styles.resume}><div><small>Continue your work</small><h3>{current.repository.name}</h3><p>{current.changes.length} process changes / {current.method}</p></div><button type="button" onClick={onContinue}>Open current analysis</button></div>}
    <section className={styles.reports}><div className={styles.heading}><div><h3>Saved reports</h3><p>Open a completed analysis without running it again.</p></div><button type="button" onClick={onHistory}>View history</button></div>
      {reports===null?<p className={styles.empty}>Saved reports are loading or unavailable.</p>:!reports.length?<div className={styles.empty}><h4>No analyses yet</h4><p>Start with a local repository and two versions of a BPMN process.</p><button type="button" onClick={onStart}>Configure an analysis</button></div>:<div className={styles.table}><div className={styles.tableHead}><span>Repository / method</span><span>Changes</span><span>Matches</span><span/></div>{reports.slice(0,5).map(r=><div key={r.analysis_id} className={styles.reportRow}><div><b>{r.repository.name||"Unnamed repository"}</b><small>{r.repository.revision||"Local snapshot"} / {r.method}</small></div><span>{r.changes}</span><span>{r.candidates}</span><button type="button" disabled={!!busy} onClick={()=>open(r.analysis_id)}>{busy===r.analysis_id?"Opening...":"Open report"}</button></div>)}</div>}
    </section>
    <section className={styles.library}><div><h3>Keep your process diagrams together</h3><p>Save versions in BPMN Library and link activities to their detail processes. Choose a saved diagram when starting an analysis.</p></div><button type="button" onClick={onLibrary}>Browse BPMN Library</button></section>
    <p className={styles.footnote}>Mapping finds possible impacts. Review the evidence before deciding which files need changes. AI review runs only when you request it.</p>
  </section>;
}
