"use client";
import { useEffect, useState } from "react";
import { checkAgent, exportAgentDraft, getAgent, startAgent, stopAgent, type AgentCapability, type AgentSession, type ReviewMode } from "@/lib/agents";
import { planDownloadURL, type PlanningContext } from "@/lib/planning";
import { Disclosure } from "./Disclosure";
import { AgentSettings } from "./AgentSettings";
import { AgentProgress } from "./AgentProgress";
import { AgentPlanResults } from "./AgentPlanResults";
import { ExportAction } from "./ExportAction";
import styles from "./QwenReview.module.css";

const active = (status?: string) => status === "STARTING" || status === "RUNNING";

export function QwenReview({
  context,
  repositoryPath = "",
  onRepositoryPath,
}: {
  context: PlanningContext;
  repositoryPath?: string;
  onRepositoryPath?: (path: string) => void;
}) {
  const [capability, setCapability] = useState<AgentCapability | null>(null);
  const [checking, setChecking] = useState(false);
  const [settingsDirty, setSettingsDirty] = useState(false);
  const [repository, setRepository] = useState(repositoryPath);
  const [editingRepository, setEditingRepository] = useState(false);
  useEffect(() => {
    if (repositoryPath) {
      setRepository(repositoryPath);
      setEditingRepository(false);
    }
  }, [repositoryPath]);
  const [mode, setMode] = useState<ReviewMode>("light");
  const [changeIndex, setChangeIndex] = useState(-1);
  const [savedChangeIndex, setSavedChangeIndex] = useState(-1);
  const [sessionID, setSessionID] = useState("");
  const [session, setSession] = useState<AgentSession | null>(null);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const [statusError, setStatusError] = useState("");
  const [pollVersion, setPollVersion] = useState(0);
  const storageKey = `bpmn-agent-session:v2:${context.analysis_id}`;

  useEffect(() => {
    let mounted = true;
    void checkAgent()
      .then(value => {
        if (mounted) setCapability(value);
      })
      .catch(() => {
        if (mounted) setError("Model status unavailable. Check the backend connection.");
      });
    const query = new URLSearchParams(window.location.search);
    const linkedSession = query.get("agentSession");
    const isLinked = query.get("analysis") === context.analysis_id && linkedSession && /^[a-f0-9]{32}$/.test(linkedSession);
    if (isLinked) setSessionID(linkedSession);
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) ?? "null") as { id: string; index: number; mode?: ReviewMode } | null;
      if (saved && typeof saved.id === "string" && (!isLinked || saved.id === linkedSession) && Number.isInteger(saved.index) && saved.index >= -1 && saved.index < context.changes.length) {
        setSessionID(saved.id);
        setChangeIndex(saved.index);
        setSavedChangeIndex(saved.index);
        if (saved.mode) setMode(saved.mode);
      }
    } catch {
      /* Optional browser storage. */
    }
    return () => {
      mounted = false;
    };
  }, [storageKey, context.changes.length]);

  useEffect(() => {
    if (!sessionID) return;
    let mounted = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const controller = new AbortController();
    async function poll() {
      try {
        const value = await getAgent(sessionID, controller.signal);
        if (!mounted) return;
        if (value.analysis_id !== context.analysis_id) throw new Error("This session belongs to another analysis.");
        setStatusError("");
        setSession(value);
        if (active(value.status)) timer = setTimeout(poll, 1200);
      } catch (reason) {
        if (mounted && !controller.signal.aborted) {
          setStatusError(`${reason instanceof Error ? reason.message : "Session status unavailable."} Retrying status in 3 seconds; the displayed progress may be outdated.`);
          timer = setTimeout(poll, 3000);
        }
      }
    }
    void poll();
    return () => {
      mounted = false;
      controller.abort();
      if (timer) clearTimeout(timer);
    };
  }, [sessionID, pollVersion, context.analysis_id]);

  async function check() {
    setChecking(true);
    setError("");
    try {
      setCapability(await checkAgent());
    } catch {
      setError("Model check failed. Check backend, VPN and SSH tunnel.");
    } finally {
      setChecking(false);
    }
  }

  async function run(resume = false) {
    setStarting(true);
    setError("");
    try {
      const runMode = resume ? session?.mode ?? mode : mode;
      const index = resume ? session?.change_index ?? savedChangeIndex : changeIndex;
      const value = await startAgent(context.analysis_id, repository.trim(), index, resume ? sessionID : undefined, runMode);
      onRepositoryPath?.(repository.trim());
      setEditingRepository(false);
      setSession(value);
      setSessionID(value.session_id);
      setPollVersion(n => n + 1);
      setMode(runMode);
      setChangeIndex(index);
      setSavedChangeIndex(index);
      try {
        localStorage.setItem(storageKey, JSON.stringify({ id: value.session_id, index, mode: runMode }));
      } catch {
        /* Optional restoration. */
      }
      if (!resume) {
        const url = new URL(window.location.href);
        if (url.searchParams.has("agentSession")) {
          url.searchParams.delete("agentSession");
          window.history.replaceState(window.history.state, "", url);
        }
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not start AI review.");
    } finally {
      setStarting(false);
    }
  }

  function downloadContext() {
    if (!session?.context) return;
    const url = URL.createObjectURL(new Blob([JSON.stringify(session.context, null, 2)], { type: "application/json" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `agent-context-${sessionID}.json`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  const busy = starting || active(session?.status);
  const plan = session?.plan;
  const configurationChanged =
    !!session?.provider &&
    !!capability?.provider &&
    (session.provider.model !== capability.provider.model ||
      session.provider.prompt_version !== capability.provider.prompt_version ||
      session.provider.provider !== capability.provider.provider);

  return (
    <section className={styles.panel} data-testid="qwen-review">
      <header className={styles.topCommand}>
        <div className={styles.topTitle}>
          <h3>Change plan</h3>
          <div className={styles.topMeta}>
            <span className={styles.statusDot} data-state={capability ? (capability.available ? "ready" : "unavailable") : "checking"}>
              {capability ? (capability.available ? "Connected" : "Unavailable") : checking || !error ? "Checking…" : "Unavailable"}
            </span>
            {capability?.provider?.model && <span className={styles.modelInline}>{capability.provider.model}</span>}
            <button type="button" className={styles.inlineLink} onClick={check} disabled={checking || busy}>
              {checking ? "Checking…" : "Check model"}
            </button>
            {sessionID && (
              <span className={styles.sessionTag}>
                Session <code>{sessionID.slice(0, 8)}</code>
              </span>
            )}
          </div>
        </div>
        <div className={styles.topActions}>
          {active(session?.status) && (
            <button
              type="button"
              className={styles.danger}
              onClick={() => {
                void stopAgent(sessionID)
                  .then(() => setPollVersion(n => n + 1))
                  .catch(reason => setError(reason.message));
              }}
            >
              Stop review
            </button>
          )}
          {(session?.status === "PAUSED" || (session?.status === "FAILED" && session.context)) && (
            <button
              type="button"
              className={styles.resume}
              disabled={starting || checking || settingsDirty || configurationChanged || !repository.trim() || !capability?.available}
              onClick={() => void run(true)}
            >
              Resume review
            </button>
          )}
          {sessionID && (
            <button
              type="button"
              className={styles.secondaryBtn}
              onClick={() => {
                setError("");
                setPollVersion(n => n + 1);
              }}
            >
              Refresh status
            </button>
          )}
          <button
            type="button"
            className={styles.primary}
            disabled={busy || checking || settingsDirty || !repository.trim() || !capability?.available || !context.changes.length}
            onClick={() => void run()}
          >
            {busy ? "Review running…" : session ? "Start new AI review" : "Run AI review"}
          </button>
        </div>
      </header>

      {error && <p className={styles.error} role="alert">{error}</p>}
      {statusError && <p className={styles.error} role="status">{statusError}</p>}

      <div className={styles.workbench}>
        <aside className={styles.controlRail} aria-label="AI review controls">
          <fieldset className={styles.scopeChoice} disabled={busy}>
            <legend>Review mode</legend>
            <div className={styles.scopeList}>
              {(
                [
                  { id: "focused", name: "Focused (Standard)", detail: "All mapped candidates + lookups" },
                  { id: "light", name: "Light", detail: "Top files & tests, expands if needed" },
                  { id: "ultra-light", name: "Ultra Light", detail: "Fastest · 4 lookups per batch" },
                  { id: "full", name: "Full repository", detail: "Scans all eligible source files" },
                ] as const
              ).map(option => (
                <label key={option.id}>
                  <input type="radio" name={`scope-${context.analysis_id}`} checked={mode === option.id} onChange={() => setMode(option.id)} />
                  <span>
                    <b>{option.name}</b>
                    <small>{option.detail}</small>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          <div className={styles.railBody}>
            {!context.process_context && (
              <p className={styles.inlineNotice}>No BPMN diagrams attached to this older report.</p>
            )}
            {capability?.enabled === false && (
              <p className={styles.inlineNotice}>Start backend with <code>--agent-model</code> to enable AI review.</p>
            )}
            {configurationChanged && (
              <p className={styles.inlineNotice}>Model config changed since this checkpoint. Start a new session.</p>
            )}

            <div className={styles.fields}>
              <label>
                BPMN changes to review
                <select
                  value={changeIndex}
                  onChange={event => setChangeIndex(Number(event.target.value))}
                  disabled={busy || !context.changes.length}
                >
                  <option value={-1}>All BPMN changes ({context.changes.length})</option>
                  {context.changes.map((change, index) => (
                    <option key={index} value={index}>
                      {change.change_type.replaceAll("_", " ")} · {change.element_name || change.element_id}
                      {change.element_name && change.element_name !== change.element_id ? " (" + change.element_id + ")" : ""}
                    </option>
                  ))}
                </select>
              </label>
              {repositoryPath && !editingRepository ? (
                <div className={styles.repository}>
                  <span>Repository</span>
                  <code>{repositoryPath}</code>
                  <button type="button" disabled={busy} onClick={() => setEditingRepository(true)}>
                    Change path
                  </button>
                </div>
              ) : (
                <label>
                  Repository path for AI review
                  <input
                    value={repository}
                    onChange={e => setRepository(e.target.value)}
                    disabled={busy}
                    placeholder="Path on the backend machine"
                    autoComplete="off"
                  />
                </label>
              )}
            </div>

            {capability?.enabled && <AgentSettings disabled={busy} onDirty={setSettingsDirty} onApplied={() => void check()} />}

            <Disclosure className={styles.accessDetails} title="Read-only access scope" openLabel="Info" closeLabel="Hide">
              <p className={styles.scope}>
                {mode === "full"
                  ? "Reads eligible source files in chunks; excludes secrets, docs, and lockfiles. Never executes or edits code."
                  : "Reads ranked candidates and bounded source lookups for the detected BPMN changes. Never executes or edits code."}
              </p>
            </Disclosure>
          </div>
        </aside>

        <div className={styles.stage}>
          {!session ? (
            <div className={styles.emptyStage}>
              <h4>Ready to draft a change plan</h4>
              <p>
                Choose the BPMN changes and mode, then click <b>Run AI review</b>.
              </p>
            </div>
          ) : (
            <div className={styles.result}>
              <header className={styles.runHeader} data-status={session.status}>
                <div>
                  <h4 aria-live="polite">
                    {session.status.replaceAll("_", " ").toLowerCase()}
                    {session.role && <span> · {session.role}</span>}
                  </h4>
                  {session.message && <p>{session.message}</p>}
                </div>
                <dl>
                  {(
                    [
                      ["Turns", session.turns],
                      ["Tool calls", session.tools],
                      ["Revisions", session.revisions],
                    ] as const
                  ).map(([label, value]) => (
                    <div key={label}>
                      <dt>{label}</dt>
                      <dd>{value ?? 0}</dd>
                    </div>
                  ))}
                </dl>
              </header>

              {!!session.issues?.length && (
                <details className={styles.issues} open={session.status === "PAUSED" || session.status === "FAILED"}>
                  <summary>Session issues ({session.issues.length})</summary>
                  <ul>
                    {session.issues.map((issue, i) => (
                      <li key={i}>{issue}</li>
                    ))}
                  </ul>
                </details>
              )}

              <AgentProgress session={session} />

              {plan && (
                <>
                  {!!plan.review_issues?.length && (
                    <details className={styles.issues}>
                      <summary>Review issues ({plan.review_issues.length})</summary>
                      <p>These findings remain unresolved. Read them before using the plan.</p>
                      <ul>
                        {plan.review_issues.map((issue, i) => (
                          <li key={i}>{issue}</li>
                        ))}
                      </ul>
                    </details>
                  )}
                  <AgentPlanResults
                    plan={plan}
                    actions={
                      <div className={styles.actions} aria-label="Change plan downloads">
                        <ExportAction variant="primary" href={planDownloadURL(plan.plan_id, "md")}>
                          Download plan Markdown
                        </ExportAction>
                        <ExportAction href={planDownloadURL(plan.plan_id, "json")}>Plan JSON</ExportAction>
                        {session.context && <ExportAction onClick={downloadContext}>Agent evidence JSON</ExportAction>}
                      </div>
                    }
                  />
                </>
              )}

              {!plan && session.context && (
                <div className={styles.actions}>
                  <ExportAction onClick={downloadContext}>Agent evidence JSON</ExportAction>
                  <button
                    type="button"
                    className={styles.secondaryBtn}
                    disabled={busy}
                    onClick={() => {
                      void exportAgentDraft(sessionID)
                        .then(value => setSession(current => (current ? { ...current, plan: value } : current)))
                        .catch(reason => setError(reason instanceof Error ? reason.message : "Draft export failed."));
                    }}
                  >
                    View partial plan
                  </button>
                </div>
              )}

              <details className={styles.auditGroup}>
                <summary>Diagnostics &amp; logs ({session.source_activity?.length ?? 0} lookups)</summary>
                <div className={styles.auditLogs}>
                  {!!plan?.change_coverage?.length && (
                    <details>
                      <summary>BPMN change coverage ({plan.change_coverage.length})</summary>
                      {plan.change_coverage.map(change => {
                        const unit = session.context?.changes?.find(c => c.element_id === change.element_id && c.change_type === change.change_type);
                        return (
                          <div key={`${change.change_type}/${change.element_id}`} className={styles.coverageItem}>
                            <p>
                              <b>
                                {change.change_type.replaceAll("_", " ")} / {change.element_id}
                              </b>{" "}
                              · {change.plans} {change.plans === 1 ? "proposal" : "proposals"} · {change.status.replaceAll("_", " ")}
                            </p>
                            {!!unit?.supporting_flow_ids?.length && <small>Flows: {unit.supporting_flow_ids.join(", ")}</small>}
                          </div>
                        );
                      })}
                    </details>
                  )}
                  {session.context?.repository_overview && (
                    <details>
                      <summary>Repository README context</summary>
                      <p>{session.context.repository_overview.summary}</p>
                      <small>{session.context.repository_overview.file}{session.context.repository_overview.truncated ? " (truncated to 8 KiB)" : ""}</small>
                    </details>
                  )}
                  <details>
                    <summary>Source access log ({session.source_activity?.length ?? 0} lookups)</summary>
                    {!session.source_activity?.length && <p>No source-tool lookup recorded yet.</p>}
                    <ol>
                      {session.source_activity?.map((item, i) => (
                        <li key={i}>
                          <b>
                            {session.context?.changes?.[item.change_index]?.element_id || `Change ${item.change_index + 1}`} · {item.tool.replaceAll("_", " ")}
                          </b>
                          {item.query && <p>Query: {item.query}</p>}
                          {item.file && (
                            <p>
                              {item.file}
                              {item.function ? ` / ${item.function}` : ""}
                            </p>
                          )}
                          <p>{item.files?.join(", ") || "No files returned"}</p>
                          {item.error && <p>Error: {item.error}</p>}
                        </li>
                      ))}
                    </ol>
                  </details>
                  {!!session.events?.length && (
                    <details>
                      <summary>Agent activity ({session.events.length})</summary>
                      <ol>
                        {session.events.map(event => (
                          <li key={event.sequence}>
                            {event.role} — {event.kind.toLowerCase().replaceAll("_", " ")}
                          </li>
                        ))}
                      </ol>
                    </details>
                  )}
                  {!!session.decisions?.length && (
                    <details>
                      <summary>File recommendations ({session.decisions.length})</summary>
                      {session.decisions.map((decision, i) => (
                        <article key={`${decision.file}:${decision.symbol}:${i}`}>
                          <b>{decision.file}</b>
                          <small>
                            {decision.symbol || "whole file"} · {decision.verdict}
                          </small>
                          <p>{decision.rationale}</p>
                          <small>Evidence: {(decision.evidence_ids ?? []).join(", ")}</small>
                        </article>
                      ))}
                    </details>
                  )}
                </div>
              </details>
              <small className={styles.sessionFoot}>
                Session <code>{sessionID}</code> · checkpoint auto-saved
              </small>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
