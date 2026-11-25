"use client";

import { useEffect, useState } from "react";
import { getAnalysis, listAnalyses } from "@/lib/api";
import type { Report, ReportSummary } from "@/lib/types";
import styles from "./AnalysisHistory.module.css";

export function AnalysisHistory({ refreshKey, onOpen }: { refreshKey: string; onOpen: (report: Report) => void }) {
  const [items, setItems] = useState<ReportSummary[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [loading, setLoading] = useState(true);
  const [refresh, setRefresh] = useState(0);
  const [query, setQuery] = useState("");
  const [method, setMethod] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    void listAnalyses()
      .then(values => {
        if (active) {
          setItems(values);
          setError("");
        }
      })
      .catch(reason => {
        if (active) setError(reason instanceof Error ? reason.message : "Saved analyses could not be loaded.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [refreshKey, refresh]);

  async function open(id: string) {
    setBusy(id);
    setError("");
    try {
      onOpen(await getAnalysis(id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Analysis unavailable.");
    } finally {
      setBusy("");
    }
  }

  const search = query.trim().toLowerCase();
  const visible = items.filter(
    item =>
      (!method || item.method === method) &&
      [item.change_label, item.repository.name, item.repository.revision, item.method, item.analysis_id]
        .join(" ")
        .toLowerCase()
        .includes(search)
  );

  return (
    <section className={styles.panel} aria-label="Saved analysis history" data-testid="analysis-history">
      <div className={styles.filters}>
        <label className={styles.searchLabel}>
          <span className={styles.srOnly}>Find reports</span>
          <input
            type="search"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search change name, repository, revision or ID…"
          />
        </label>
        <div className={styles.filterActions}>
          <label htmlFor="history-method-filter" className={styles.srOnly}>Mapping method</label>
          <select id="history-method-filter" aria-label="Mapping method" value={method} onChange={e => setMethod(e.target.value)}>
            <option value="">All methods</option>
            {[...new Set(items.map(item => item.method))].sort().map(value => (
              <option key={value} value={value}>
                {({ bm25: "BM25", semantic: "Semantic", fusion: "BM25 + Semantic", hybrid: "Context-aware Hybrid", configuration: "Configuration (legacy)" })[value] || value}
              </option>
            ))}
          </select>
          {(query || method) && (
            <button
              type="button"
              onClick={() => {
                setQuery("");
                setMethod("");
              }}
            >
              Clear filters
            </button>
          )}
          <button type="button" disabled={loading || !!busy} onClick={() => setRefresh(n => n + 1)}>
            Refresh history
          </button>
        </div>
      </div>

      <p role="status" className={styles.count}>
        {loading ? "Loading saved reports…" : `Showing ${visible.length} of ${items.length} saved reports.`}
      </p>

      {error && <p role="alert" className={styles.error}>{error}</p>}
      {!loading && !error && !items.length && <p className={styles.empty}>No saved reports yet.</p>}
      {!loading && !!items.length && !visible.length && (
        <p className={styles.empty}>No matching reports. Change the search or clear filters.</p>
      )}

      {visible.length > 0 && (
        <div className={styles.body}>
          <div className={styles.tableHead} aria-hidden="true">
            <span>Report</span>
            <span>Changes</span>
            <span>Candidates</span>
            <span>Unresolved</span>
            <span />
          </div>
          {visible.map(item => (
            <article key={item.analysis_id}>
              <div className={styles.reportCell}>
                <b>{item.change_label || item.repository.name || "Unnamed repository"}</b>
                <small>
                  {[item.change_label && item.repository.name, item.repository.revision === "unspecified" ? "Local snapshot" : item.repository.revision, item.method]
                    .filter(Boolean)
                    .join(" · ")}{" "}
                  · <code title={item.analysis_id}>{item.analysis_id.slice(0, 8)}</code>
                </small>
              </div>
              <span className={styles.numCell}>{item.changes}</span>
              <span className={styles.numCell}>{item.candidates}</span>
              <span className={styles.numCell} data-warn={item.unresolved > 0 || undefined}>
                {item.unresolved}
              </span>
              <button type="button" className={styles.openBtn} disabled={Boolean(busy)} onClick={() => void open(item.analysis_id)}>
                {busy === item.analysis_id ? "Opening…" : "Open report"}
              </button>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
