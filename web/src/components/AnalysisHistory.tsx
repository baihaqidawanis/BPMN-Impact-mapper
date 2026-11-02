"use client";

import { useEffect, useState } from "react";
import { getAnalysis, listAnalyses } from "@/lib/api";
import type { Report, ReportSummary } from "@/lib/types";
import styles from "./AnalysisHistory.module.css";

export function AnalysisHistory({ refreshKey, onOpen }: { refreshKey: string; onOpen: (report: Report) => void }) {
  const [items, setItems] = useState<ReportSummary[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [loading,setLoading]=useState(true);
  const [refresh,setRefresh]=useState(0);
  const [query,setQuery]=useState("");
  const [method,setMethod]=useState("");
  useEffect(() => {
    let active = true;setLoading(true);
    void listAnalyses().then(values => { if (active) { setItems(values); setError(""); } }).catch(reason => { if (active) setError(reason instanceof Error ? reason.message : "History unavailable."); }).finally(()=>{if(active)setLoading(false);});
    return () => { active = false; };
  }, [refreshKey,refresh]);

  async function open(id: string) {
    setBusy(id); setError("");
    try { onOpen(await getAnalysis(id)); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Analysis unavailable."); }
    finally { setBusy(""); }
  }
  const visible=items.filter(item=>(!method||item.method===method)&&`${item.repository.name} ${item.repository.revision} ${item.method} ${item.analysis_id}`.toLowerCase().includes(query.trim().toLowerCase()));
  return <section className={styles.panel} aria-label="Saved analysis history" data-testid="analysis-history">
    <header><div><h3>Saved analyses</h3><p>Reports stay unchanged. Open one to inspect its results or continue Planning.</p></div><button type="button" disabled={loading||!!busy} onClick={()=>setRefresh(n=>n+1)}>{loading?"Loading...":"Refresh history"}</button></header>
    <div className={styles.filters}>
      <label>Find reports<input type="search" value={query} onChange={e=>setQuery(e.target.value)} placeholder="Repository, revision or analysis ID" /></label>
      <div><label htmlFor="history-method-filter">Mapping method</label><select id="history-method-filter" value={method} onChange={e=>setMethod(e.target.value)}><option value="">All methods</option>{[...new Set(items.map(item=>item.method))].sort().map(value=><option key={value} value={value}>{({bm25:"BM25",semantic:"Semantic",fusion:"BM25 + Semantic",hybrid:"Context-aware Hybrid",configuration:"Configuration (legacy)"})[value]||value}</option>)}</select></div>
      <button type="button" disabled={!query&&!method} onClick={()=>{setQuery("");setMethod("");}}>Clear filters</button>
    </div>
    <p role="status" className={styles.count}>{loading?"Loading saved reports...":`Showing ${visible.length} of ${items.length} saved reports.`}</p>
    {error && <p role="alert" className={styles.error}>{error}</p>}
    {!loading&&!error&&!items.length&&<p className={styles.empty}>No saved analysis yet. Run an analysis from the Analysis page to create one.</p>}
    {!loading&&!!items.length&&!visible.length&&<p className={styles.empty}>No matching reports. Change the search or clear filters.</p>}
    <div className={styles.body}>{visible.map(item => <article key={item.analysis_id}>
      <div><b>{item.repository.name || "Unnamed repository"}</b><small>{item.repository.revision||"Local snapshot"} / {item.method}</small><code>{item.analysis_id}</code></div>
      <dl><div><dt>Changes</dt><dd>{item.changes}</dd></div><div><dt>Candidates</dt><dd>{item.candidates}</dd></div><div><dt>Unresolved</dt><dd>{item.unresolved}</dd></div></dl>
      <button type="button" disabled={Boolean(busy)} onClick={() => void open(item.analysis_id)}>{busy === item.analysis_id ? "Opening..." : "Open report"}</button>
    </article>)}</div>
  </section>;
}
