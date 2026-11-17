import type { AgentSession } from "@/lib/agents";
import styles from "./AgentProgress.module.css";

function AgentIcon({ role }: { role: string }) {
  return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {role === "Analyst" ? <><circle cx="10" cy="10" r="5.5"/><path d="m14 14 5 5M8 10h4M10 8v4"/></> : role === "Planner" ? <><rect x="5" y="4" width="14" height="17" rx="2"/><path d="M9 2v4m6-4v4M9 10h6m-6 4h6m-6 4h3"/></> : <><path d="m12 3 8 3v6c0 4-4 7-8 9-4-2-8-5-8-9V6z"/><path d="m8 12 3 3 5-6"/></>}
  </svg>;
}

export function AgentProgress({ session }: { session: AgentSession }) {
  const p = session.full_review;
  const f = session.focused_review;
  const live = session.status === "RUNNING" || session.status === "STARTING";
  const focusedDone = !!f && f.batches_total > 0 && f.batches_reviewed === f.batches_total;
  const scanDone = p ? p.chunks_total > 0 && p.chunks_reviewed === p.chunks_total : f ? f.candidates_total > 0 && f.candidates_processed === f.candidates_total : session.role === "planner" || session.role === "reviewer";
  const plansDone = p ? scanDone && p.batches_total > 0 && p.batches_reviewed === p.batches_total : f ? focusedDone : session.role === "reviewer";
  const reviewApproved = p ? p.consistency_review === "approved" : session.status === "COMPLETE";
  const reviewFlagged = !live && (session.status === "NEEDS_REVIEW" || p?.consistency_review === "needs_review");
  const stages = [
    { name: "Analyst", description: "Inspect source and identify affected files.", done: scanDone, active: live && (p ? p.phase === "scanning" : session.role === "analyst"), flagged: false, value: p?.chunks_reviewed, total: p?.chunks_total, detail: p ? `${p.chunks_reviewed}/${p.chunks_total} source chunks` : f ? `${f.candidates_processed}/${f.candidates_total} initial candidates processed` : "Source inspection" },
    { name: "Planner", description: "Draft technical steps and acceptance checks.", done: plansDone, active: live && (p ? p.phase === "planning" : session.role === "planner"), flagged: !live && !!f?.batches_needing_review, value: p?.batches_reviewed, total: p?.batches_total, detail: p ? `${p.batches_reviewed}/${p.batches_total} planning batches (includes batch review)` : f ? `${f.batches_reviewed}/${f.batches_total} batches processed` : "Developer change plan" },
    { name: "Reviewer", description: "Check the plan against source evidence.", done: reviewApproved, active: live && (p ? p.phase === "consistency_review" : session.role === "reviewer"), flagged: reviewFlagged, value: undefined, total: undefined, detail: p?.consistency_review?.replaceAll("_", " ") || "Final consistency check" },
  ];
  return <ol className={styles.stages} aria-label="AI agent progress">{stages.map((stage, index) => {
    const state = stage.active ? "active" : stage.flagged ? "review" : stage.done ? "done" : live ? "waiting" : "stopped";
    return <li key={stage.name} data-state={state}>
      <div className={styles.track}><span className={styles.icon}><AgentIcon role={stage.name}/></span><span className={styles.connector}/></div>
      <header><small>0{index + 1}</small><b>{stage.name}</b><span className={styles.status}>{state === "active" ? "Working" : state === "review" ? "Needs review" : state === "done" ? (f && stage.name !== "Reviewer" ? "Processed" : "Finished") : state === "waiting" ? "Waiting" : "Not completed"}</span></header>
      <p>{stage.description}</p>
      {stage.total ? <progress aria-label={`${stage.name} progress`} max={stage.total} value={stage.value ?? 0}/> : stage.active ? <progress aria-label={`${stage.name} progress`}/> : null}
      <span className={styles.detail}>{stage.detail}</span>
    </li>;
  })}</ol>;
}
