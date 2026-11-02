"use client";
import { useEffect, useState } from "react";
import { checkAgent, getAgent, startAgent, type AgentCapability, type AgentSession } from "@/lib/agents";
import { planDownloadURL, type PlanningContext } from "@/lib/planning";
import styles from "./QwenReview.module.css";

const active = (status?: string) => status === "STARTING" || status === "RUNNING";
export function QwenReview({ context, repositoryPath = "", onRepositoryPath }: { context: PlanningContext; repositoryPath?: string; onRepositoryPath?: (path: string) => void }) {
  const [capability, setCapability] = useState<AgentCapability | null>(null);
  const [checking, setChecking] = useState(false);
  const [repository, setRepository] = useState(repositoryPath);
  const [editingRepository, setEditingRepository] = useState(false);
  useEffect(() => { if (repositoryPath) { setRepository(repositoryPath); setEditingRepository(false); } }, [repositoryPath]);
  const [mode, setMode] = useState<"focused" | "full">("full");
  const [changeIndex, setChangeIndex] = useState(0);
  const [sessionID, setSessionID] = useState("");
  const [session, setSession] = useState<AgentSession | null>(null);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const [pollVersion, setPollVersion] = useState(0);
  const storageKey = `bpmn-agent-session:${context.analysis_id}`;

  useEffect(() => {
    let mounted = true;
    void checkAgent().then(value => { if (mounted) setCapability(value); }).catch(() => { if (mounted) setError("Model status unavailable. Check the backend connection."); });
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) ?? "null") as { id: string; index: number; mode?: "focused" | "full" } | null;
      if (saved && typeof saved.id === "string" && Number.isInteger(saved.index) && saved.index >= 0 && saved.index < context.changes.length) { setSessionID(saved.id); setChangeIndex(saved.index); if (saved.mode) setMode(saved.mode); }
    } catch { /* Optional browser storage. */ }
    return () => { mounted = false; };
  }, [storageKey, context.changes.length]);

  useEffect(() => {
    if (!sessionID) return;
    let mounted = true; let timer: ReturnType<typeof setTimeout> | undefined;
    const controller = new AbortController();
    async function poll() {
      try {
        const value = await getAgent(sessionID, controller.signal);
        if (!mounted) return;
        if (value.analysis_id !== context.analysis_id) throw new Error("This session belongs to another analysis.");
        setSession(value);
        if (value.mode) setMode(value.mode);
        if (active(value.status)) timer = setTimeout(poll, 1200);
      } catch (reason) { if (mounted) setError(reason instanceof Error ? reason.message : "Session status unavailable. Refresh status before starting another run."); }
    }
    void poll();
    return () => { mounted = false; controller.abort(); if (timer) clearTimeout(timer); };
  }, [sessionID, pollVersion, context.analysis_id]);

  async function check() {
    setChecking(true); setError("");
    try { setCapability(await checkAgent()); } catch { setError("Model check failed. Check backend, VPN and SSH tunnel."); } finally { setChecking(false); }
  }
  async function run(resume = false) {
    setStarting(true); setError("");
    try {
      const value = await startAgent(context.analysis_id, repository.trim(), changeIndex, resume ? sessionID : undefined, mode);
      onRepositoryPath?.(repository.trim()); setEditingRepository(false);
      setSession(value); setSessionID(value.session_id); setPollVersion(n => n + 1);
      try { localStorage.setItem(storageKey, JSON.stringify({ id: value.session_id, index: changeIndex, mode })); } catch { /* Optional restoration. */ }
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Could not start Qwen review."); } finally { setStarting(false); }
  }
  function choose(index: number) {
    setChangeIndex(index); setSessionID(""); setSession(null); setError("");
    try { localStorage.removeItem(storageKey); } catch { /* Optional storage. */ }
  }
  function downloadContext() {
    if (!session?.context) return;
    const url = URL.createObjectURL(new Blob([JSON.stringify(session.context, null, 2)], { type: "application/json" }));
    const link = document.createElement("a"); link.href = url; link.download = `agent-context-${sessionID}.json`; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  const busy = starting || active(session?.status), plan = session?.plan;
  return <section className={styles.panel} data-testid="qwen-review">
    <header><div><span>{context.method === "hybrid" ? "Hybrid + Multi-Agent Qwen" : "Mapping + Multi-Agent Qwen"}</span><h3>Review one process change</h3></div><button type="button" onClick={check} disabled={checking || busy}>{checking ? "Checking…" : "Check model"}</button></header>
    <p>The analyst checks possible impacts, the planner proposes steps, and the reviewer can request revisions. Mapping rankings remain unchanged.</p>
    <div className={styles.connection} role="status">{capability ? `${capability.provider?.model ?? "Qwen"} · ${capability.available ? "Connected" : "Unavailable"}. ${capability.message}` : "Checking model connection…"}</div>
    {capability?.enabled === false && <p>Restart the Go backend with <code>--agent-model qwen3.5:9b-q8_0</code> to enable this feature.</p>}
    <div className={styles.fields}>
      {repositoryPath && !editingRepository ? <div className={styles.repository}><span>Repository from mapping</span><code>{repositoryPath}</code><button type="button" disabled={busy} onClick={() => setEditingRepository(true)}>Change path</button></div> : <label>Repository path for Qwen review<input value={repository} onChange={e => setRepository(e.target.value)} disabled={busy} placeholder="Path on the backend machine" autoComplete="off" /><small>Older saved reports may need this once on this device.</small></label>}
      <label>BPMN change to review<select value={changeIndex} onChange={e => choose(Number(e.target.value))} disabled={busy}>{context.changes.map((change, i) => <option value={i} key={`${change.change_type}:${change.element_id}`}>{i + 1}. {change.change_type.replaceAll("_", " ")} · {change.element_name || change.element_id}</option>)}</select></label>
    </div>
    <label className={styles.mode}>Review scope<select aria-label="Review scope" value={mode} disabled={busy} onChange={e => { setMode(e.target.value as "focused" | "full"); choose(changeIndex); }}><option value="full">Full repository review</option><option value="focused">Focused review</option></select></label>
    {mode === "full" ? <p className={styles.scope}>Every ingestion-approved file is sent to Qwen in overlapping chunks for this selected BPMN change. Secrets, dependencies and excluded files remain excluded. Hybrid provides initial leads; the final list is not limited to top candidates. This can take a long time. Keep the backend running; interrupted sessions can be resumed.</p> : <p className={styles.scope}>Only this selected change is reviewed. The initial package contains up to 3 ranked candidates and 1 source connection, with short excerpts. Agents may look up additional approved source. Clicking Run sends this evidence to the lab model; repository code is never executed or edited.</p>}
    <div className={styles.actions}>
      <button type="button" className={styles.primary} disabled={busy || !repository.trim() || !capability?.available || !context.changes.length} onClick={() => void run()}>{busy ? "Review running…" : session ? "Start new Qwen review" : "Run Qwen review"}</button>
      {(session?.status === "PAUSED" || session?.status === "FAILED" && session.context) && <button type="button" disabled={starting || !repository.trim() || !capability?.available} onClick={() => void run(true)}>Resume review</button>}
      {sessionID && <button type="button" onClick={() => { setError(""); setPollVersion(n => n + 1); }}>Refresh status</button>}
    </div>
    {error && <p className={styles.error} role="alert">{error}</p>}
    {session && <div className={styles.result}>
      <div className={styles.status} aria-live="polite"><b>{session.status.replaceAll("_", " ")}</b><span>{session.role} · {session.turns ?? 0} turns · {session.tools ?? 0} tools · {session.revisions ?? 0} revisions</span></div>
      {session.message && <p>{session.message}</p>}
      {session.full_review && <div className={styles.coverage}><h4>Repository coverage</h4><progress aria-label="Reviewed source chunks" max={session.full_review.chunks_total || 1} value={session.full_review.chunks_reviewed} /><span>{session.full_review.files_reviewed}/{session.full_review.files_total} eligible files | {session.full_review.chunks_reviewed}/{session.full_review.chunks_total} chunks</span><small>{session.full_review.phase} | {session.full_review.batches_reviewed}/{session.full_review.batches_total} planning batches</small>{session.full_review.current_file && <small>{session.full_review.current_file}</small>}{!!session.full_review.failed_chunks && <small>Failed chunks: {session.full_review.failed_chunks}</small>}<small>Coverage measures processed source, not accuracy.</small></div>}
      <small>Session {sessionID}. Refreshing the page restores this session; it does not restart inference.</small>
      {!plan && !!session.issues?.length && <div className={styles.issues}><h4>Session issues</h4><ul>{session.issues.map((issue, i) => <li key={i}>{issue}</li>)}</ul></div>}
      {!plan && session.context && <div className={styles.actions}><button type="button" onClick={downloadContext}>Agent evidence JSON</button></div>}
      {!!session.events?.length && <details><summary>Agent activity</summary><ol>{session.events.map(event => <li key={event.sequence}>{event.role} — {event.kind.toLowerCase().replaceAll("_", " ")}</li>)}</ol></details>}
      {!!session.decisions?.length && <details><summary>File recommendations after review</summary>{session.decisions.map((decision, i) => <article key={`${decision.file}:${decision.symbol}:${i}`}><b>{decision.file}</b><small>{decision.symbol || "whole file"} · {decision.verdict}</small><p>{decision.rationale}</p><small>Evidence: {decision.evidence_ids.join(", ")}</small></article>)}</details>}
      {plan && <>
        <div className={styles.actions}><a href={planDownloadURL(plan.plan_id, "md")}>Download plan Markdown</a><a href={planDownloadURL(plan.plan_id, "json")}>Plan JSON</a><button type="button" onClick={downloadContext}>Agent evidence JSON</button></div>
        <p>{plan.status === "COMPLETE" ? "Agent review finished. This is a proposal, not independent proof that every file must change." : "The plan still needs review. Inspect the issues before using its recommendations."}</p>
        {!!plan.review_issues?.length && <div className={styles.issues}><h4>Review issues</h4><ul>{plan.review_issues.map((issue, i) => <li key={i}>{issue}</li>)}</ul></div>}
        {plan.plans.map((file, i) => <article key={`${file.file}:${file.symbol}:${i}`}><header><b>{file.file}</b><span>{file.action}</span></header><small>{file.symbol || "whole file"} · {file.module} / {file.layer}</small><p>{file.summary}</p>{!!file.proposed_changes?.length && <><h4>Proposed changes</h4><ul>{file.proposed_changes.map((change, n) => <li key={n}>{change}</li>)}</ul></>}<h4>Implementation steps</h4><ol>{file.steps.map((step, n) => <li key={n}>{step}</li>)}</ol>{!!file.acceptance_checks?.length && <><h4>Acceptance checks</h4><ul>{file.acceptance_checks.map((check, n) => <li key={n}>{check}</li>)}</ul></>}<small>Evidence: {file.evidence_ids.join(", ")}</small></article>)}
      </>}
    </div>}
  </section>;
}
