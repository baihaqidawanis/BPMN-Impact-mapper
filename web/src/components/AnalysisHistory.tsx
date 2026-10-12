"use client";

import { useEffect, useState } from "react";
import { getAnalysis, listAnalyses } from "@/lib/api";
import type { Report, ReportSummary } from "@/lib/types";
import styles from "./AnalysisHistory.module.css";

export function AnalysisHistory({ refreshKey, onOpen }: { refreshKey: string; onOpen: (report: Report) => void }) {
  const [items, setItems] = useState<ReportSummary[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");

  useEffect(() => {
    let active = true;
    void listAnalyses().then(values => { if (active) { setItems(values); setError(""); } }).catch(reason => { if (active) setError(reason instanceof Error ? reason.message : "History unavailable."); });
    return () => { active = false; };
  }, [refreshKey]);

  async function open(id: string) {
    setBusy(id); setError("");
    try { onOpen(await getAnalysis(id)); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Analysis unavailable."); }
    finally { setBusy(""); }
  }

  return <details className={styles.panel} data-testid="analysis-history">
    <summary>Saved analyses <span>{items.length} immutable report{items.length === 1 ? "" : "s"}</span></summary>
    <div className={styles.body}>
      {items.length === 0 && !error && <p>No saved analysis yet.</p>}
      {error && <p role="alert" className={styles.error}>{error}</p>}
      {items.map(item => <article key={item.analysis_id}>
        <div><b>{item.repository.name || "Unnamed repository"}</b><small>{item.repository.revision} · {item.method}</small></div>
        <dl><div><dt>Changes</dt><dd>{item.changes}</dd></div><div><dt>Candidates</dt><dd>{item.candidates}</dd></div><div><dt>Unresolved</dt><dd>{item.unresolved}</dd></div></dl>
        <button type="button" disabled={Boolean(busy)} onClick={() => void open(item.analysis_id)}>{busy === item.analysis_id ? "Opening…" : "Open report"}</button>
      </article>)}
    </div>
  </details>;
}
