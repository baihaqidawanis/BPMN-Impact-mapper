"use client";

import { useEffect, useState } from "react";
import { planningContextURL } from "@/lib/api";
import { createPlan, getPlanningContext, listPlans, planDownloadURL } from "@/lib/planning";
import type { ChangePlan, PlanningContext, PlanSummary } from "@/lib/planning";
import styles from "./PlanningContextView.module.css";

export function PlanningContextView({ analysisID }: { analysisID: string }) {
  const [context, setContext] = useState<PlanningContext | null>(null);
  const [bytes, setBytes] = useState(0);
  const [error, setError] = useState("");
  const [plans, setPlans] = useState<PlanSummary[]>([]);
  const [latestPlan, setLatestPlan] = useState<ChangePlan | null>(null);
  const [planning, setPlanning] = useState(false);
  const [planError, setPlanError] = useState("");

  useEffect(() => {
    let active = true;
    setContext(null); setBytes(0); setError("");
    setPlans([]); setLatestPlan(null); setPlanError("");
    void Promise.all([getPlanningContext(analysisID), listPlans(analysisID)]).then(([value, savedPlans]) => {
      if (active) { setContext(value.context); setBytes(value.bytes); setPlans(savedPlans); }
    }).catch(reason => {
      if (active) setError(reason instanceof Error ? reason.message : "Planning context could not be loaded.");
    });
    return () => { active = false; };
  }, [analysisID]);

  async function preparePlan() {
    setPlanning(true); setPlanError("");
    try {
      const value = await createPlan(analysisID);
      setLatestPlan(value);
      setPlans(await listPlans(analysisID));
    } catch (reason) {
      setPlanError(reason instanceof Error ? reason.message : "Planning could not be completed.");
    } finally {
      setPlanning(false);
    }
  }

  if (error) return <section className={styles.state} role="alert"><h3>Planning context unavailable</h3><p>{error}</p></section>;
  if (!context) return <section className={styles.state} aria-live="polite"><span className={styles.loader} /><h3>Preparing bounded evidence</h3><p>Reading the immutable analysis report. No model is being called.</p></section>;

  const candidates = context.changes.reduce((total, change) => total + change.candidates.length, 0);
  const evidence = context.changes.reduce((total, change) => total + change.candidates.reduce((sum, candidate) => sum + candidate.evidence.length, 0), 0);
  return <div className={styles.root} data-testid="planning-context-view">
    <header className={styles.header}>
      <div><span>Pre-LLM audit boundary</span><h3>{context.repository.name}</h3><p>Only these candidates and bounded excerpts may enter a future planning provider.</p></div>
      <a href={planningContextURL(context.analysis_id)}>Download JSON</a>
    </header>
    <div className={styles.metrics}>
      <Metric value={context.changes.length} label="Changes" />
      <Metric value={candidates} label="Candidates" />
      <Metric value={evidence} label="Evidence items" />
      <Metric value={formatBytes(bytes)} label="Payload" />
    </div>
    <div className={styles.notice}><b>No repository source is sent.</b><span>The future verifier cannot introduce files or symbols outside this context. All outputs are checked again by Go.</span></div>
    <section className={styles.planBoundary}>
      <div><span>Execution boundary</span><h4>{latestPlan ? "Planning record prepared" : "Provider-safe planning contract"}</h4><p>{latestPlan?.status === "DISABLED" ? "No LLM provider is configured. The immutable record proves the current payload and offline fallback without generating a fake plan." : "Create an immutable planning record from this exact context. Until a provider is configured, its status remains DISABLED and no network call occurs."}</p></div>
      <div className={styles.planActions}>
        <button type="button" onClick={preparePlan} disabled={planning}>{planning ? "Preparing…" : "Create planning record"}</button>
        {latestPlan && <><a href={planDownloadURL(latestPlan.plan_id, "json")}>Plan JSON</a><a href={planDownloadURL(latestPlan.plan_id, "md")}>Plan Markdown</a></>}
      </div>
      {planError && <p className={styles.planError} role="alert">{planError}</p>}
      <footer><span>{plans.length} immutable record{plans.length === 1 ? "" : "s"}</span>{latestPlan && <><code>{latestPlan.plan_id}</code><b>{latestPlan.status}</b></>}</footer>
      {plans.length > 0 && <div className={styles.savedPlans}>{plans.map(plan => <div key={plan.plan_id}><code>{plan.plan_id}</code><span>{plan.provider.provider} / {plan.provider.model}</span><b>{plan.status}</b><a href={planDownloadURL(plan.plan_id, "json")}>JSON</a><a href={planDownloadURL(plan.plan_id, "md")}>Markdown</a></div>)}</div>}
    </section>
    <div className={styles.changes}>
      {context.changes.map((change, changeIndex) => <article className={styles.change} key={`${change.change_type}-${change.element_id}`}>
        <header><span>{String(changeIndex + 1).padStart(2, "0")}</span><div><small>{change.change_type.replaceAll("_", " ")}</small><h4>{change.element_name || change.element_id}</h4><code>{change.element_id}</code></div><b>{change.candidates.length} candidates</b></header>
        {change.reason && <p className={styles.reason}>{change.reason}</p>}
        {change.candidates.length === 0 ? <p className={styles.empty}>No evidence-bearing candidate is available for planning.</p> : change.candidates.map((candidate, index) => <details className={styles.candidate} key={`${candidate.file}-${candidate.symbol ?? ""}`}>
          <summary><span>{String(index + 1).padStart(2, "0")}</span><div><b>{candidate.file}</b><small>{candidate.symbol || "file-level candidate"} · {candidate.module} / {candidate.layer}</small></div><em>{candidate.suggested_action}</em><strong>{candidate.retrieval_score.toFixed(3)}</strong></summary>
          <div className={styles.evidence}>{candidate.evidence.map(item => <div key={item.id}><header><b>{item.id}</b><span>{item.type}</span><code>{item.location}</code></header><p>{item.excerpt}</p></div>)}</div>
        </details>)}
      </article>)}
    </div>
    <footer>Context <code>{context.context_id}</code><span>Method <b>{context.method}</b> · Revision <code>{context.repository.revision || "unspecified"}</code></span></footer>
  </div>;
}

function Metric({ value, label }: { value: number | string; label: string }) {
  return <div><b>{value}</b><span>{label}</span></div>;
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  return `${(value / 1024).toFixed(1)} KiB`;
}
