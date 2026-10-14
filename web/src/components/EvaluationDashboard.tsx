"use client";

import { useState } from "react";
import type { Report, ReviewerAnnotation, ReviewerCase } from "@/lib/types";
import { downloadEvaluation, evaluationRequest, readAnnotation } from "@/lib/evaluation";
import type { EvaluationBundle, EvaluationResult, Reconciliation } from "@/lib/evaluation";
import styles from "./EvaluationDashboard.module.css";

export function EvaluationDashboard({ reports }: { reports: Report[] }) {
  const [a, setA] = useState<ReviewerAnnotation | null>(null);
  const [b, setB] = useState<ReviewerAnnotation | null>(null);
  const [reconciliation, setReconciliation] = useState<Reconciliation | null>(null);
  const [decisions, setDecisions] = useState<Record<number, string>>({});
  const [custom, setCustom] = useState<Record<number, string>>({});
  const [confirmed, setConfirmed] = useState(false);
  const [result, setResult] = useState<EvaluationBundle | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function load(file: File | undefined, side: "a" | "b") {
    setError(""); setReconciliation(null); setResult(null); setDecisions({}); setConfirmed(false);
    if (side === "a") setA(null); else setB(null);
    if (!file) return;
    setBusy(true);
    try { const value = await readAnnotation(file); if (side === "a") setA(value); else setB(value); }
    catch (e) { setError(e instanceof Error ? e.message : "Invalid annotation."); }
    finally { setBusy(false); }
  }

  async function reconcile() {
    setBusy(true); setError(""); setResult(null); setDecisions({}); setCustom({});
    try { setReconciliation(await evaluationRequest<Reconciliation>("reviews/reconcile", { reviewer_a: a, reviewer_b: b })); }
    catch (e) { setReconciliation(null); setError(e instanceof Error ? e.message : "Reconciliation failed."); }
    finally { setBusy(false); }
  }

  async function evaluate() {
    if (!reconciliation) return;
    setBusy(true); setError(""); setResult(null);
    try {
      const resolutions: ReviewerCase[] = reconciliation.disagreements.map((item, i) => {
        const choice = decisions[i];
        if (!choice) throw new Error("Resolve every disagreement first.");
        let expected: ReviewerCase["expected"];
        if (choice === "custom") expected = JSON.parse(custom[i] ?? "");
        else if (choice === "empty") expected = [];
        else expected = (choice === "a" ? item.reviewer_a : item.reviewer_b) ?? [];
        if (!Array.isArray(expected)) throw new Error("Custom resolution must be a JSON array of expected impacts.");
        return { change_type: item.change_type, element_id: item.element_id, expected };
      });
      setResult(await evaluationRequest<EvaluationBundle>("evaluations", {
        reviewer_a: a, reviewer_b: b, resolutions,
        analysis_ids: reports.map(r => r.analysis_id), input_fingerprint: reports[0].input_fingerprint, snapshot_confirmed: confirmed,
      }));
    } catch (e) { setError(e instanceof Error ? e.message : "Evaluation failed."); }
    finally { setBusy(false); }
  }

  const ready = reconciliation && reconciliation.disagreements.every((_, i) => Boolean(decisions[i]));
  return <details className={styles.panel} data-testid="evaluation-dashboard">
    <summary>Evaluation dashboard <span>Reconcile reviewers and measure mapping results</span></summary>
    <div className={styles.body}>
      <p>Evaluate the {reports.length} stored report(s) currently selected. Upload two independently prepared annotations for this repository snapshot and BPMN scenario. Files are processed by the local evaluation API and are not saved on the server.</p>
      <div className={styles.uploads}>
        <label>Reviewer A JSON<input type="file" accept=".json,application/json" disabled={busy} onChange={e => void load(e.target.files?.[0], "a")} /><small>{a?.reviewer_id}</small></label>
        <label>Reviewer B JSON<input type="file" accept=".json,application/json" disabled={busy} onChange={e => void load(e.target.files?.[0], "b")} /><small>{b?.reviewer_id}</small></label>
      </div>
      <button type="button" disabled={busy || !a || !b} onClick={() => void reconcile()}>Reconcile reviewers</button>
      {reconciliation && <>
        <p role="status">{reconciliation.agreed_ground_truth.cases.length} agreed cases · {reconciliation.disagreements.length} disagreements</p>
        <button type="button" onClick={() => downloadEvaluation(reconciliation, "reconciliation.json")}>Download reconciliation JSON</button>
        {reconciliation.disagreements.map((item, i) => <fieldset key={item.change_type + item.element_id}>
          <legend>{item.change_type} / {item.element_id}</legend>
          <p>{item.reason}</p>
          <div className={styles.uploads}>
            <div><b>Reviewer A</b><pre>{JSON.stringify(item.reviewer_a, null, 2)}</pre></div>
            <div><b>Reviewer B</b><pre>{JSON.stringify(item.reviewer_b, null, 2)}</pre></div>
          </div>
          <label>Resolution {i + 1}<select aria-label={`Resolution ${i + 1}`} disabled={busy} value={decisions[i] ?? ""} onChange={e => { setDecisions({ ...decisions, [i]: e.target.value }); setResult(null); }}>
            <option value="">Choose a decision</option>
            {a?.cases.some(c => c.change_type === item.change_type && c.element_id === item.element_id) && <option value="a">Use reviewer A</option>}
            {b?.cases.some(c => c.change_type === item.change_type && c.element_id === item.element_id) && <option value="b">Use reviewer B</option>}
            <option value="empty">Explicitly no expected impact</option>
            <option value="custom">Custom expected impacts JSON</option>
          </select></label>
          {decisions[i] === "custom" && <label>Expected impacts JSON<textarea disabled={busy} rows={5} placeholder={'[{"file":"path/file.go","required_action":"MODIFY"}]'} value={custom[i] ?? ""} onChange={e => { setCustom({ ...custom, [i]: e.target.value }); setResult(null); }} /></label>}
        </fieldset>)}
        <label className={styles.confirm}><input type="checkbox" disabled={busy} checked={confirmed} onChange={e => { setConfirmed(e.target.checked); setResult(null); }} /> I confirm these reviews apply to the selected snapshot and BPMN scenario.</label>
        <button type="button" disabled={busy || !ready || !confirmed} onClick={() => void evaluate()}>{busy ? "Processing…" : "Evaluate stored reports"}</button>
      </>}
      {error && <p role="alert">{error}</p>}
      {result && <EvaluationTable bundle={result} />}
    </div>
  </details>;
}

function EvaluationTable({ bundle }: { bundle: EvaluationBundle }) {
  const rows: [string, (m: EvaluationResult["metrics"]) => number][] = [
    ["Recall@1", m => m.recall.at_1], ["Recall@3", m => m.recall.at_3], ["Recall@5", m => m.recall.at_5],
    ["Precision@1", m => m.precision.at_1], ["Precision@3", m => m.precision.at_3], ["Precision@5", m => m.precision.at_5],
    ["MRR", m => m.mrr], ["MAP", m => m.map], ["File precision", m => m.file_precision], ["File recall", m => m.file_recall], ["File F1", m => m.file_f1],
    ["Module accuracy*", m => m.module_accuracy], ["Layer accuracy*", m => m.layer_accuracy], ["Action accuracy", m => m.required_action_accuracy],
    ["Unresolved rate", m => m.unresolved_rate], ["False discovery proportion", m => m.false_positive_rate],
  ];
  return <section data-testid="evaluation-results">
    <h3>Evaluation results</h3>
    <p>All {bundle.ground_truth.cases.length} changes evaluated at file level. Values range from 0 to 1. No repository code was executed.</p>
    <div className={styles.tableWrap}><table><thead><tr><th>Metric</th>{bundle.results.map(r => <th key={r.analysis_id}>{r.method}</th>)}</tr></thead>
      <tbody>{rows.map(([name, value]) => <tr key={name}><th>{name}</th>{bundle.results.map(r => <td key={r.analysis_id}>{value(r.metrics).toFixed(4)}</td>)}</tr>)}</tbody>
    </table></div>
    <p>*Legacy module/layer metrics count unspecified expected metadata as matching. False discovery proportion is FP/(TP+FP); true-negative-based FPR is unavailable. Latency is not measured by this dashboard. These results describe the selected scenario only.</p>
    <section className={styles.diagnostics} data-testid="evaluation-diagnostics">
      <h4>Per-change diagnostics</h4>
      <p>Inspect which BPMN change was retrieved, missed, or left unresolved. This is the useful layer for improving a method; aggregate scores alone do not explain a failure.</p>
      {bundle.results.map(result => <details key={result.analysis_id}>
        <summary>{result.method} <span>{result.changes.length} changes</span></summary>
        <div className={styles.tableWrap}><table>
          <thead><tr><th>Change</th><th>Expected files</th><th>Ranked predictions</th><th>RR</th><th>AP</th><th>Status</th></tr></thead>
          <tbody>{result.changes.map(change => <tr key={`${change.change_type}:${change.element_id}`}>
            <th><span className={styles.changeType}>{change.change_type.replaceAll("_", " ")}</span><code>{change.element_id}</code></th>
            <td>{fileList(change.expected_files, "No expected file")}</td>
            <td>{fileList(change.predicted_files, "No candidate")}</td>
            <td>{change.reciprocal_rank.toFixed(4)}</td>
            <td>{change.average_precision.toFixed(4)}</td>
            <td><span className={change.unresolved ? styles.unresolved : styles.resolved}>{change.unresolved ? "Unresolved" : "Ranked"}</span></td>
          </tr>)}</tbody>
        </table></div>
      </details>)}
    </section>
    <div className={styles.downloads}>
      <button type="button" onClick={() => downloadEvaluation(bundle, "evaluation-bundle.json")}>Download evaluation JSON</button>
      <button type="button" onClick={() => downloadEvaluation(bundle.markdown, "evaluation-report.md")}>Download evaluation Markdown</button>
      <button type="button" onClick={() => downloadEvaluation(bundle.ground_truth, "ground-truth.json")}>Download final ground truth</button>
    </div>
  </section>;
}

function fileList(files: string[], empty: string) {
  if (files.length === 0) return <span className={styles.empty}>{empty}</span>;
  return <ol className={styles.fileList}>{files.map(file => <li key={file}><code>{file}</code></li>)}</ol>;
}
