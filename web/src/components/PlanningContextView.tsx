"use client";

import { useEffect, useState } from "react";
import { planningContextURL } from "@/lib/api";
import { createPlan, getPlanningContext, listPlans, planDownloadURL } from "@/lib/planning";
import type { ChangePlan, PlanningContext, PlanSummary } from "@/lib/planning";
import styles from "./PlanningContextView.module.css";
import { QwenReview } from "./QwenReview";

export function PlanningContextView({ analysisID, repositoryPath = "", onRepositoryPath }: { analysisID: string; repositoryPath?: string; onRepositoryPath?: (path: string) => void }) {
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
    <QwenReview context={context} repositoryPath={repositoryPath} onRepositoryPath={onRepositoryPath} />
    <details className={styles.contextDetails}><summary>Source context and audit exports</summary>
    <header className={styles.header}>
			<div><span>Mapping evidence</span><h3>{context.repository.name}</h3><p>The original mapping evidence. Full repository review scans eligible source separately; this package is not the entire repository.</p></div>
      <a href={planningContextURL(context.analysis_id)}>Download JSON</a>
    </header>
    <div className={styles.metrics}>
      <Metric value={context.changes.length} label="Changes" />
      <Metric value={candidates} label="Candidates" />
      <Metric value={evidence} label="Evidence items" />
      <Metric value={formatBytes(bytes)} label="Payload" />
    </div>
		<div className={styles.notice}><b>Review is opt-in.</b><span>Review starts only when you choose Run. Full mode sends eligible source in chunks; focused mode uses selected evidence and source searches. Original mapping rankings stay unchanged.</span></div>
    <details className={styles.offline}><summary>Offline audit snapshots</summary><section className={styles.planBoundary}>
      <div><span>Offline record</span><h4>{latestPlan ? "Context snapshot saved" : "Save context without a model"}</h4><p>This stores an immutable offline record. It does not call Qwen or generate a plan. Use Qwen review above for the lab model.</p></div>
      <div className={styles.planActions}>
        <button type="button" onClick={preparePlan} disabled={planning}>{planning ? "Preparing…" : "Create planning record"}</button>
        {latestPlan && <><a href={planDownloadURL(latestPlan.plan_id, "json")}>Plan JSON</a><a href={planDownloadURL(latestPlan.plan_id, "md")}>Plan Markdown</a></>}
      </div>
      {planError && <p className={styles.planError} role="alert">{planError}</p>}
      <footer><span>{plans.length} immutable record{plans.length === 1 ? "" : "s"}</span>{latestPlan && <><code>{latestPlan.plan_id}</code><b>{latestPlan.status}</b></>}</footer>
      {plans.length > 0 && <div className={styles.savedPlans}>{plans.map(plan => <div key={plan.plan_id}><code>{plan.plan_id}</code><span>{plan.provider.provider} / {plan.provider.model}</span><b>{plan.status}</b><a href={planDownloadURL(plan.plan_id, "json")}>JSON</a><a href={planDownloadURL(plan.plan_id, "md")}>Markdown</a></div>)}</div>}
    </section></details>
    <div className={styles.changes}>
      {!!context.warnings?.length && <p className={styles.reason}>Analysis warnings: {context.warnings.map(warning => `${warning.code}: ${warning.message}`).join("; ")}</p>}
      {context.changes.map((change, changeIndex) => <article className={styles.change} key={`${change.change_type}-${change.element_id}`}>
        <header><span>{String(changeIndex + 1).padStart(2, "0")}</span><div><small>{change.change_type.replaceAll("_", " ")}</small><h4>{change.element_name || change.element_id}</h4><code>{change.element_id}</code></div><b>{change.candidates.length} possible impacts</b></header>
        {change.reason && <p className={styles.reason}>{change.reason}</p>}
        {(change.before || change.after) && <details className={styles.candidate}><summary>BPMN requirement before / after</summary><div className={styles.evidence}><p>Before: {change.before ? `${change.before.name || change.before.id} · ${change.before.type} · ${change.before.actor || "no actor"} ${change.before.state || ""}` : "Element did not exist"}</p><p>After: {change.after ? `${change.after.name || change.after.id} · ${change.after.type} · ${change.after.actor || "no actor"} ${change.after.state || ""}` : "Element removed"}</p>{change.branch_changes?.map(branch => <div key={branch.flow_id}><b>{branch.flow_id}</b><p>Before: {branch.old_condition || branch.old_label || "No condition recorded"}</p><p>After: {branch.new_condition || branch.new_label || "No condition recorded"}</p></div>)}</div></details>}
        {(change.context_truncated || change.context_incomplete) && <p className={styles.reason}>BPMN requirement context is incomplete or truncated. Recheck the original BPMN before accepting a plan.</p>}
        {change.candidates.length === 0 ? <p className={styles.empty}>No evidence-backed file or function is available for planning.</p> : change.candidates.map((candidate, index) => <details className={styles.candidate} key={`${candidate.file}-${candidate.symbol ?? ""}`}>
          <summary><span>{String(index + 1).padStart(2, "0")}</span><div><b>{candidate.file}</b><small>{candidate.symbol || "whole file"} · {candidate.module} / {candidate.layer}{candidate.origin === "source-caller" ? " · connected source file" : ""}</small></div><em>{candidate.suggested_action}</em><strong>{typeof candidate.retrieval_score === "number" ? candidate.retrieval_score.toFixed(3) : "Not ranked"}</strong></summary>
          <div className={styles.evidence}>{candidate.evidence.map(item => <div key={item.id}><header><b>{item.id}</b><span>{item.type}</span><code>{item.location}</code></header><p>{item.excerpt}</p></div>)}</div>
        </details>)}
      </article>)}
    </div>
    <footer>Context <code>{context.context_id}</code><span>Method <b>{context.method}</b> · Revision <code>{context.repository.revision || "unspecified"}</code></span></footer>
    </details>
  </div>;
}

function Metric({ value, label }: { value: number | string; label: string }) {
  return <div><b>{value}</b><span>{label}</span></div>;
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  return `${(value / 1024).toFixed(1)} KiB`;
}
