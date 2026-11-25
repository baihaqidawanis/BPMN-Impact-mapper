"use client";

import { useEffect, useState } from "react";
import { getAnalysis, listAnalyses } from "@/lib/api";
import { listDiagrams, type DiagramEntry } from "@/lib/bpmn-library";
import type { Report, ReportSummary } from "@/lib/types";
import styles from "./WorkspaceOverview.module.css";

export function WorkspaceOverview({
  active,
  current,
  onStart,
  onContinue,
  onOpen,
  onLibrary,
  onHistory,
}: {
  active: boolean;
  current: Report | null;
  onStart: () => void;
  onContinue: () => void;
  onOpen: (r: Report) => void;
  onLibrary: () => void;
  onHistory: () => void;
}) {
  const [reports, setReports] = useState<ReportSummary[] | null>(null);
  const [diagrams, setDiagrams] = useState<DiagramEntry[] | null>(null);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");

  useEffect(() => {
    if (!active) return;
    let mounted = true;
    Promise.allSettled([listAnalyses(), listDiagrams()]).then(([a, b]) => {
      if (!mounted) return;
      setReports(a.status === "fulfilled" ? a.value : null);
      setDiagrams(b.status === "fulfilled" ? b.value : null);
      setNotice(
        a.status === "rejected" || b.status === "rejected"
          ? "Some workspace data could not be loaded. Check that the backend is running."
          : ""
      );
    });
    return () => {
      mounted = false;
    };
  }, [active, current?.analysis_id]);

  async function open(id: string) {
    setBusy(id);
    setError("");
    try {
      onOpen(await getAnalysis(id));
    } catch {
      setError("This report could not be opened. Check the backend and retry.");
    } finally {
      setBusy("");
    }
  }

  return (
    <section className={styles.overview} aria-label="Workspace overview">
      <header className={styles.commandBar}>
        <div>
          <h1>Overview</h1>
          <p>Trace BPMN process changes to affected source code and draft change plans.</p>
        </div>
        <div className={styles.commandActions}>
          {current && (
            <button type="button" className={styles.secondaryBtn} onClick={onContinue}>
              Open current analysis
            </button>
          )}
          <button type="button" className={styles.primaryBtn} onClick={onStart}>
            New analysis
          </button>
        </div>
      </header>

      {notice && <p role="status" className={styles.notice}>{notice}</p>}
      {error && <p role="alert" className={styles.error}>{error}</p>}

      <div className={styles.kpiRow}>
        <button type="button" className={styles.kpiCard} onClick={onHistory}>
          <span>Saved analyses</span>
          <b>{reports ? reports.length : "—"}</b>
          <small>View history →</small>
        </button>
        <button type="button" className={styles.kpiCard} onClick={onLibrary}>
          <span>BPMN library</span>
          <b>{diagrams ? diagrams.length : "—"}</b>
          <small>Browse diagrams →</small>
        </button>
        <button type="button" className={styles.kpiCard} data-accent={!!current || undefined} onClick={current ? onContinue : onStart}>
          <span>Active session</span>
          <b>{current ? current.repository.name : "None"}</b>
          <small>{current ? `${current.changes.length} changes · ${current.method} →` : "Start an analysis →"}</small>
        </button>
      </div>

      <section className={styles.reports} aria-label="Saved reports">
        <div className={styles.panelHead}>
          <div className={styles.panelTitle}>
            <h2>Recent analyses</h2>
            {reports && <span>Showing {Math.min(5, reports.length)} of {reports.length}</span>}
          </div>
          <button type="button" className={styles.linkBtn} onClick={onHistory}>
            All history →
          </button>
        </div>
        {reports === null ? (
          <p className={styles.empty}>Loading saved reports…</p>
        ) : !reports.length ? (
          <div className={styles.empty}>
            <h4>No analyses yet</h4>
            <p>Select a local repository and two BPMN files to run your first impact analysis.</p>
            <button type="button" className={styles.primaryBtn} onClick={onStart}>
              Configure an analysis
            </button>
          </div>
        ) : (
          <div className={styles.table}>
            <div className={styles.tableHead}>
              <span>Repository &amp; method</span>
              <span>Changes</span>
              <span>Matches</span>
              <span />
            </div>
            {reports.slice(0, 5).map(r => (
              <div
                key={r.analysis_id}
                className={styles.reportRow}
                data-active={current?.analysis_id === r.analysis_id || undefined}
              >
                <div className={styles.repoCell}>
                  <b>{r.repository.name || "Unnamed repository"}</b>
                  <small>
                    <code>{r.repository.revision || "local"}</code> ·{" "}
                    <span className={styles.methodTag}>{r.method}</span>
                  </small>
                </div>
                <span className={styles.numCell}>{r.changes}</span>
                <span className={styles.numCell}>{r.candidates}</span>
                <button
                  type="button"
                  className={styles.openBtn}
                  disabled={!!busy}
                  onClick={() => open(r.analysis_id)}
                >
                  {busy === r.analysis_id ? "Opening…" : "Open →"}
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
    </section>
  );
}
