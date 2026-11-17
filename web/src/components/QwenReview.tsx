"use client";
import { useEffect, useState } from "react";
import { checkAgent, exportAgentDraft, getAgent, startAgent, stopAgent, type AgentCapability, type AgentSession } from "@/lib/agents";
import { planDownloadURL, type PlanningContext } from "@/lib/planning";
import { AgentSettings } from "./AgentSettings";
import { AgentProgress } from "./AgentProgress";
import { AgentPlanResults } from "./AgentPlanResults";
import styles from "./QwenReview.module.css";

const active = (status?: string) => status === "STARTING" || status === "RUNNING";
export function QwenReview({ context, repositoryPath = "", onRepositoryPath }: { context: PlanningContext; repositoryPath?: string; onRepositoryPath?: (path: string) => void }) {
  const [capability, setCapability] = useState<AgentCapability | null>(null);
  const [checking, setChecking] = useState(false);
  const [settingsDirty, setSettingsDirty] = useState(false);
  const [repository, setRepository] = useState(repositoryPath);
  const [editingRepository, setEditingRepository] = useState(false);
  useEffect(() => { if (repositoryPath) { setRepository(repositoryPath); setEditingRepository(false); } }, [repositoryPath]);
  const [mode, setMode] = useState<"focused" | "full">("full");
  const [changeIndex, setChangeIndex] = useState(-1);
  const [sessionID, setSessionID] = useState("");
  const [session, setSession] = useState<AgentSession | null>(null);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const [pollVersion, setPollVersion] = useState(0);
  const storageKey = `bpmn-agent-session:v2:${context.analysis_id}`;

  useEffect(() => {
    let mounted = true;
    void checkAgent().then(value => { if (mounted) setCapability(value); }).catch(() => { if (mounted) setError("Model status unavailable. Check the backend connection."); });
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) ?? "null") as { id: string; index: number; mode?: "focused" | "full" } | null;
      if (saved && typeof saved.id === "string" && Number.isInteger(saved.index) && saved.index >= -1 && saved.index < context.changes.length) { setSessionID(saved.id); setChangeIndex(saved.index); if (saved.mode) setMode(saved.mode); }
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
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Could not start AI review."); } finally { setStarting(false); }
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
  const configurationChanged = !!session?.provider && !!capability?.provider && (session.provider.model !== capability.provider.model || session.provider.prompt_version !== capability.provider.prompt_version || session.provider.provider !== capability.provider.provider);
  return <section className={styles.panel} data-testid="qwen-review">
    <header><div><span>{context.method === "hybrid" ? "Hybrid + AI Agents" : "Mapping + AI Agents"}</span><h3>Review process changes</h3></div><button type="button" onClick={check} disabled={checking || busy}>{checking ? "Checking…" : "Check model"}</button></header>
    <p>{context.process_context ? "Full before/after process structure is included. " : "Legacy report: full before/after process structure is unavailable; rerun mapping with both BPMN files. "}Generate a developer change plan, with source evidence and review.</p>
    <div className={styles.connection} role="status">{capability ? `${(capability.role_models ? "Agent models" : capability.provider?.model) ?? "Model"} · ${capability.available ? "Connected" : "Unavailable"}. ${capability.message}` : "Checking model connection…"}</div>
    {capability?.role_models && <dl className={styles.roleModels}>{(["analyst", "planner", "reviewer"] as const).map(role => <div key={role}><dt>{role}</dt><dd>{capability.role_models?.[role]?.model || "Not configured"}</dd></div>)}</dl>}
    {capability?.enabled === false && <p>Restart the Go backend with <code>--agent-model qwen3.5:9b-q8_0</code> to enable this feature.</p>}
    {capability?.enabled && <AgentSettings disabled={busy} onDirty={setSettingsDirty} onApplied={() => void check()} />}
    {configurationChanged && <p>Model configuration differs from this checkpoint. Start a new AI review; the previous session is preserved.</p>}
    <div className={styles.fields}>
      {repositoryPath && !editingRepository ? <div className={styles.repository}><span>Repository from mapping</span><code>{repositoryPath}</code><button type="button" disabled={busy} onClick={() => setEditingRepository(true)}>Change path</button></div> : <label>Repository path for AI review<input value={repository} onChange={e => setRepository(e.target.value)} disabled={busy} placeholder="Path on the backend machine" autoComplete="off" /><small>Older saved reports may need this once on this device.</small></label>}
      <div className={styles.repository}><span>Analysis scope</span><b>All {context.changes.length} BPMN changes</b><small>One combined review and change plan.</small></div>
    </div>
    <label className={styles.mode}>Review scope<select aria-label="Review scope" value={mode} disabled={busy} onChange={e => { setMode(e.target.value as "focused" | "full"); choose(changeIndex); }}><option value="full">Full repository review</option><option value="focused">Focused review</option></select></label>
    <details className={styles.accessDetails}><summary>What the agents can access</summary>
    {mode === "full" ? <p className={styles.scope}>Implementation files approved for bulk review are sent to the configured model in overlapping chunks against all BPMN changes. Bulk scanning skips Markdown/prose and dependency manifests. Approved source tools remain available for targeted follow-up. Secrets and ingestion-excluded files stay excluded. Hybrid provides initial leads; the final list is not limited to top candidates. This can take a long time. Keep the backend running; interrupted sessions can be resumed.</p> : <p className={styles.scope}>All BPMN changes are processed in bounded internal batches, then combined into one reviewed plan. All displayed mapping candidates for every change are included. Candidates are inspected in small batches with short initial excerpts; source tools can retrieve more evidence. Agents may look up additional approved source. Clicking Run sends this evidence to the lab model; repository code is never executed or edited.</p>}
    </details>
    <div className={styles.actions}>
      {active(session?.status) && <button type="button" onClick={() => { void stopAgent(sessionID).then(() => setPollVersion(n=>n+1)).catch(reason=>setError(reason.message)); }}>Stop review</button>}
      <button type="button" className={styles.primary} disabled={busy || checking || settingsDirty || !repository.trim() || !capability?.available || !context.changes.length} onClick={() => void run()}>{busy ? "Review running…" : session ? "Start new AI review" : "Run AI review"}</button>
      {(session?.status === "PAUSED" || session?.status === "FAILED" && session.context) && <button type="button" disabled={starting || checking || settingsDirty || configurationChanged || !repository.trim() || !capability?.available} onClick={() => void run(true)}>Resume review</button>}
      {sessionID && <button type="button" onClick={() => { setError(""); setPollVersion(n => n + 1); }}>Refresh status</button>}
    </div>
    {error && <p className={styles.error} role="alert">{error}</p>}
    {session && <div className={styles.result}>
      <AgentProgress session={session}/>
      {session.focused_review && <div className={styles.coverage}><h4>Focused candidate coverage</h4><progress aria-label="Processed focused candidates" max={session.focused_review.candidates_total || 1} value={session.focused_review.candidates_processed}/><span>{session.focused_review.candidates_processed}/{session.focused_review.candidates_total} initial candidates processed | {session.focused_review.batches_reviewed}/{session.focused_review.batches_total} batches processed</span>{session.focused_review.outcome_counts_available && <small>{session.focused_review.batches_approved ?? 0} batches approved; {session.focused_review.batches_needing_review ?? 0} need review. Final consistency approval is separate.</small>}<small>Every displayed candidate is included. Completion counts processed batches, not correctness; inspect source access and review issues.</small></div>}
      {session.context?.repository_overview && <details><summary>Repository background from README</summary><p>{session.context.repository_overview.summary}</p><small>Source: {session.context.repository_overview.file}. Background only; implementation claims need source evidence.{session.context.repository_overview.truncated ? " The README input was limited to its first 8 KiB." : ""}</small></details>}
      <div className={styles.status} aria-live="polite"><b>{session.status.replaceAll("_", " ")}</b><span>{session.role} · {session.turns ?? 0} turns · {session.tools ?? 0} tools · {session.revisions ?? 0} revisions</span></div>
      {session.message && <p>{session.message}</p>}
      {session.full_review && <div className={styles.coverage}><h4>Repository coverage</h4><progress aria-label="Reviewed source chunks" max={session.full_review.chunks_total || 1} value={session.full_review.chunks_reviewed} /><span>{session.full_review.files_reviewed}/{session.full_review.files_total} eligible files | {session.full_review.chunks_reviewed}/{session.full_review.chunks_total} chunks</span><small>{session.full_review.phase} | {session.full_review.batches_reviewed}/{session.full_review.batches_total} planning batches</small>{session.full_review.current_file && <small>{session.full_review.current_file}</small>}{!!session.full_review.failed_chunks && <small>Failed chunks: {session.full_review.failed_chunks}</small>}{session.full_review.consistency_review && <small>Final consistency review: {session.full_review.consistency_review.replaceAll("_", " ")}</small>}<small>Coverage measures processed source, not accuracy.</small></div>}
      <small>Session {sessionID}. Refreshing the page restores this session; it does not restart inference.</small>
      {!plan && !!session.issues?.length && <div className={styles.issues}><h4>Session issues</h4><ul>{session.issues.map((issue, i) => <li key={i}>{issue}</li>)}</ul></div>}
      {!plan && session.context && <div className={styles.actions}><button type="button" onClick={downloadContext}>Agent evidence JSON</button><button type="button" disabled={busy} onClick={() => {void exportAgentDraft(sessionID).then(value=>setSession(current=>current?{...current,plan:value}:current)).catch(reason=>setError(reason instanceof Error?reason.message:"Draft export failed."));}}>View partial plan</button></div>}
      <details><summary>Source access log ({session.source_activity?.length ?? 0} lookups)</summary>
        <p>Search results are possible matches. Read tools return bounded source excerpts, not entire files. This log records tool access, not proof of understanding. Full-scan coverage is shown separately.</p>
        {!session.source_activity?.length && <p>No completed source-tool lookup is recorded yet. Initial mapping evidence may still have been supplied.</p>}
        <ol>{session.source_activity?.map((item, i) => <li key={i}>
          <b>Change {item.change_index + 1}: {item.tool.replaceAll("_", " ")}</b>
          {item.query && <p>Query: {item.query}</p>}
          {item.file && <p>Requested: {item.file}{item.function ? ` / ${item.function}` : ""}</p>}
          <p>{item.tool === "search_source" ? "Search matches" : "Returned source"}: {item.files?.join(", ") || "No files returned"}</p>
          {item.error && <p>Lookup issue: {item.error}</p>}
          {item.truncated && <small>Result limited; more source may exist. </small>}
          {!!item.next_line && <small>Next source window starts at line {item.next_line}.</small>}
        </li>)}</ol>
      </details>
      {!!session.events?.length && <details><summary>Agent activity</summary><ol>{session.events.map(event => <li key={event.sequence}>{event.role} — {event.kind.toLowerCase().replaceAll("_", " ")}</li>)}</ol></details>}
      {!!session.decisions?.length && <details><summary>File recommendations after review</summary>{session.decisions.map((decision, i) => <article key={`${decision.file}:${decision.symbol}:${i}`}><b>{decision.file}</b><small>{decision.symbol || "whole file"} · {decision.verdict}</small><p>{decision.rationale}</p><small>Evidence: {(decision.evidence_ids ?? []).join(", ")}</small></article>)}</details>}
      {plan && <>
        <div className={styles.actions}><a href={planDownloadURL(plan.plan_id, "md")}>Download plan Markdown</a><a href={planDownloadURL(plan.plan_id, "json")}>Plan JSON</a><button type="button" onClick={downloadContext}>Agent evidence JSON</button></div>
        <p>{plan.status === "COMPLETE" ? "Agent review finished. This is a proposal, not independent proof that every file must change." : "The plan still needs review. Inspect the issues before using its recommendations."}</p>
        {!!plan.review_issues?.length && <div className={styles.issues}><h4>Review issues</h4><ul>{plan.review_issues.map((issue, i) => <li key={i}>{issue}</li>)}</ul></div>}
        {!!plan.change_coverage?.length && <details><summary>BPMN change coverage</summary><p>Planned means a proposal exists. Unresolved does not mean no impact.</p>{plan.change_coverage.map((change,i)=><p key={i}><b>{change.change_type.replaceAll("_"," ")} / {change.element_id}</b> ? {change.plans} file plans ? {change.status}</p>)}</details>}
        <AgentPlanResults plan={plan} />
      </>}
    </div>}
  </section>;
}
