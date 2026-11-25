import { useEffect, useState } from "react";
import type { AgentSession } from "@/lib/agents";
import styles from "./AgentProgress.module.css";

function formatDuration(ms?: number): string {
  if (ms === undefined) return "—";
  const seconds = Math.floor(ms / 1000);
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60]
    .map(value => String(value).padStart(2, "0"))
    .join(":");
}

function TelemetryStrip({ session, live }: { session: AgentSession; live: boolean }) {
  const recorded = session.active_ms;
  const [clock, setClock] = useState(0);
  useEffect(() => {
    if (!live || recorded === undefined) return;
    const started = performance.now();
    const timer = window.setInterval(() => setClock(recorded + performance.now() - started), 250);
    return () => window.clearInterval(timer);
  }, [live, recorded]);

  const f = session.focused_review;
  const p = session.full_review;
  const elapsed = recorded !== undefined ? (live ? Math.max(recorded, clock) : recorded) : undefined;
  const timeDisplay = formatDuration(elapsed);

  const done = f?.batches_reviewed ?? 0;
  const remaining = (f?.batches_total ?? 0) - done;
  const estimate =
    live && recorded !== undefined && recorded > 0 && done >= 2 && remaining > 0
      ? Math.max(1, Math.round(((recorded / done) * remaining) / 60000))
      : undefined;

  const batchValue = f ? `${f.batches_reviewed}/${f.batches_total}` : p ? `${p.batches_reviewed}/${p.batches_total}` : "—";
  const batchSub = f
    ? f.combined_review_only
      ? `${f.batches_prepared ?? 0} prepared · ${f.batches_needing_review ?? 0} issues`
      : f.outcome_counts_available
        ? `${f.batches_approved ?? 0} approved · ${f.batches_needing_review ?? 0} issues`
        : "Batches"
    : p
      ? p.phase.replaceAll("_", " ")
      : "";

  const coverageValue = f
    ? `${f.candidates_processed}/${f.candidates_total}`
    : p
      ? `${p.files_reviewed}/${p.files_total}`
      : "—";
  const coverageSub = f ? "Candidates" : p ? `${p.chunks_reviewed}/${p.chunks_total} chunks` : "";

  return (
    <dl className={styles.telemetry} aria-label="Session timing and progress">
      <div className={styles.metric} data-paused={session.status === "PAUSED" || undefined}>
        <dt>Active time</dt>
        <dd>{timeDisplay}</dd>
        <small>{session.status === "PAUSED" ? "Paused" : live ? "Running" : "Elapsed"}</small>
      </div>
      {live && (
        <div className={styles.metric}>
          <dt>Est. remaining</dt>
          <dd>{estimate !== undefined ? `~${estimate} min` : "—"}</dd>
          <small>{estimate !== undefined ? "To final review" : done > 0 && remaining === 0 ? "Final review" : "After 2 batches"}</small>
        </div>
      )}
      <div className={styles.metric}>
        <dt>Batches</dt>
        <dd>{batchValue}</dd>
        {batchSub && <small>{batchSub}</small>}
      </div>
      <div className={styles.metric}>
        <dt>Coverage</dt>
        <dd>{coverageValue}</dd>
        {coverageSub && <small>{coverageSub}</small>}
      </div>
    </dl>
  );
}

function AgentIcon({ role }: { role: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {role === "Analyst" ? (
        <>
          <circle cx="10" cy="10" r="5.5" />
          <path d="m14 14 5 5M8 10h4M10 8v4" />
        </>
      ) : role === "Planner" ? (
        <>
          <rect x="5" y="4" width="14" height="17" rx="2" />
          <path d="M9 2v4m6-4v4M9 10h6m-6 4h6m-6 4h3" />
        </>
      ) : (
        <>
          <path d="m12 3 8 3v6c0 4-4 7-8 9-4-2-8-5-8-9V6z" />
          <path d="m8 12 3 3 5-6" />
        </>
      )}
    </svg>
  );
}

export function AgentProgress({ session }: { session: AgentSession }) {
  const [expanded, setExpanded] = useState(false);
  const p = session.full_review;
  const f = session.focused_review;
  const live = session.status === "RUNNING" || session.status === "STARTING";
  const settled = !live && session.status !== "PAUSED";
  const focusedDone = !!f && f.batches_total > 0 && f.batches_reviewed === f.batches_total;
  const scanDone = p
    ? p.chunks_total > 0 && p.chunks_reviewed === p.chunks_total
    : f
      ? f.candidates_total > 0 && f.candidates_processed === f.candidates_total
      : session.role === "planner" || session.role === "reviewer";
  const plansDone = p ? scanDone && p.batches_total > 0 && p.batches_reviewed === p.batches_total : f ? focusedDone : session.role === "reviewer";
  const reviewApproved = p ? p.consistency_review === "approved" : session.status === "COMPLETE";
  const reviewFlagged = !live && (session.status === "NEEDS_REVIEW" || p?.consistency_review === "needs_review");

  const stages = [
    {
      name: "Analyst",
      done: scanDone,
      active: live && (p ? p.phase === "scanning" : session.role === "analyst"),
      flagged: false,
      value: p?.chunks_reviewed,
      total: p?.chunks_total,
      detail: p ? `${p.chunks_reviewed}/${p.chunks_total} chunks` : f ? `${f.candidates_processed}/${f.candidates_total} candidates` : "Source scan",
    },
    {
      name: "Planner",
      done: plansDone,
      active: live && (p ? p.phase === "planning" : session.role === "planner"),
      flagged: !live && !!f?.batches_needing_review,
      value: p?.batches_reviewed,
      total: p?.batches_total,
      detail: p ? `${p.batches_reviewed}/${p.batches_total} batches` : f ? `${f.batches_reviewed}/${f.batches_total} batches` : "Draft plan",
    },
    {
      name: "Reviewer",
      done: reviewApproved,
      active: live && (p ? p.phase === "consistency_review" : session.role === "reviewer"),
      flagged: reviewFlagged,
      value: undefined,
      total: undefined,
      detail: p?.consistency_review?.replaceAll("_", " ") || "Evidence check",
    },
  ];

  const compactStats = [
    session.active_ms !== undefined ? formatDuration(session.active_ms) : null,
    f ? `${f.candidates_processed}/${f.candidates_total} candidates` : p ? `${p.files_reviewed}/${p.files_total} files` : null,
    f ? `${f.batches_reviewed}/${f.batches_total} batches` : p ? `${p.batches_reviewed}/${p.batches_total} batches` : null,
  ].filter(Boolean);

  if (settled && !expanded) {
    return (
      <div className={styles.compactBar} aria-label="Session timing and progress">
        <div className={styles.compactStages} aria-label="AI agent progress">
          {stages.map(stage => {
            const state = stage.flagged ? "review" : stage.done ? "done" : "stopped";
            return (
              <span key={stage.name} className={styles.stagePill} data-state={state}>
                <i aria-hidden="true" />
                <b>{stage.name}</b>
              </span>
            );
          })}
        </div>
        <div className={styles.compactMeta}>
          {compactStats.map(item => (
            <span key={item}>{item}</span>
          ))}
          <button type="button" className={styles.detailsToggle} onClick={() => setExpanded(true)}>
            Pipeline details
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.wrapper}>
      <div className={styles.telemetryHead}>
        <TelemetryStrip key={session.session_id} session={session} live={live} />
        {settled && (
          <button type="button" className={styles.detailsToggle} onClick={() => setExpanded(false)}>
            Hide details
          </button>
        )}
      </div>
      <ol className={styles.stages} aria-label="AI agent progress">
        {stages.map(stage => {
          const total = stage.total ?? (stage.name === "Analyst" ? f?.candidates_total : stage.name === "Planner" ? f?.batches_total : undefined);
          const value = stage.value ?? (stage.name === "Analyst" ? f?.candidates_processed : f?.batches_reviewed);
          const paused = session.status === "PAUSED" && session.role === stage.name.toLowerCase();
          const state = paused ? "paused" : stage.active ? "active" : stage.flagged ? "review" : stage.done ? "done" : live ? "waiting" : "stopped";
          return (
            <li key={stage.name} data-state={state}>
              <div className={styles.track}>
                <span className={styles.icon}><AgentIcon role={stage.name} /></span>
                <span className={styles.connector} />
              </div>
              <header>
                <b>{stage.name}</b>
                <span className={styles.status}>
                  {state === "paused" ? "Paused" : state === "active" ? "Working" : state === "review" ? "Needs review" : state === "done" ? "Done" : state === "waiting" ? "Waiting" : "Stopped"}
                </span>
              </header>
              {total || stage.active ? (
                <div className={styles.progressTrack}>
                  <progress aria-label={`${stage.name} progress`} max={total} value={total ? value ?? 0 : undefined} />
                </div>
              ) : null}
              <span className={styles.detail}>{stage.detail}</span>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
