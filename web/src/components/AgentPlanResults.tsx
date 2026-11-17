import type { ChangePlan } from "@/lib/planning";
import styles from "./QwenReview.module.css";

type FilePlan = ChangePlan["plans"][number];
const list = (values?: string[]) => Array.isArray(values) ? values.filter(value => typeof value === "string") : [];

function FileAssessment({ file }: { file: FilePlan }) {
  const unchanged = file.action === "NO_ACTION";
  return <article>
    <header><b>{file.file}</b><span>{unchanged ? "No change proposed" : file.action}</span></header>
    <small>{file.symbol || "whole file"} · {file.module} / {file.layer}</small>
    <p>{file.summary}</p>
    {unchanged && <p>This target was reviewed. It is excluded from the proposed edit list; its no-change conclusion still depends on the cited evidence.</p>}
    {([ ["Current behavior", file.current_behavior], ["Required behavior", file.target_behavior], ["Design rationale", file.design_rationale] ] as const).map(([title, value]) => value && <div key={title}><h4>{title}</h4><p>{value}</p></div>)}
    {!!list(file.assumptions).length && <div><h4>Assumptions and open decisions</h4><ul>{list(file.assumptions).map((value, i) => <li key={i}>{value}</li>)}</ul></div>}
    {!unchanged && <>
      {!!list(file.proposed_changes).length && <div><h4>Proposed changes</h4><ul>{list(file.proposed_changes).map((value, i) => <li key={i}>{value}</li>)}</ul></div>}
      {!!list(file.steps).length && <div><h4>Implementation steps</h4><ol>{list(file.steps).map((value, i) => <li key={i}>{value}</li>)}</ol></div>}
    </>}
    {!!list(file.acceptance_checks).length && <div><h4>{unchanged ? "Existing-contract checks" : "Acceptance checks"}</h4><ul>{list(file.acceptance_checks).map((value, i) => <li key={i}>{value}</li>)}</ul></div>}
    <small>Evidence: {list(file.evidence_ids).join(", ")}</small>
  </article>;
}

export function AgentPlanResults({ plan }: { plan: ChangePlan }) {
  const files = Array.isArray(plan.plans) ? plan.plans.filter(file => file && typeof file.file === "string") : [];
  const edits = files.filter(file => file.action !== "NO_ACTION");
  const unchanged = files.filter(file => file.action === "NO_ACTION");
  const findings = Array.isArray(plan.review_findings) ? plan.review_findings.filter(item => item && typeof item.file === "string") : [];
  return <>
    {edits.length > 0 ? <h4>Proposed edits · {new Set(edits.map(file => file.file)).size} files</h4> : <div className={styles.issues} role="status"><h4>No source edits were proposed</h4><p>{unchanged.length ? "The reviewed targets have no-change assessments below. This does not prove the entire repository needs no changes." : "Read the review issues. An empty result does not mean no files need changes."}</p></div>}
    {edits.map((file, i) => <FileAssessment key={`${file.file}:${file.element_id}:${i}`} file={file} />)}
    {unchanged.length > 0 && <details><summary>{unchanged.length} reviewed targets with no edit proposed</summary>{unchanged.map((file, i) => <FileAssessment key={`${file.file}:${file.element_id}:${i}`} file={file} />)}</details>}
    {findings.length > 0 && <details><summary>Reviewer checks</summary><p>These are model conclusions, not independent proof of correctness.</p>{findings.map((item, i) => <article key={`${item.file}:${item.element_id}:${i}`}><b>{item.file}</b><small>{item.element_id} · {item.basis === "source_and_proposal" ? "Source and proposal checked" : "Proposal consistency only"}</small>{(item.source_excerpt || item.implementation_excerpt) && <details><summary>Compared source and plan</summary>{item.source_excerpt && <p>Source: <code className={styles.comparisonExcerpt}>{item.source_excerpt}</code></p>}{item.implementation_excerpt && <p>Plan item: <code className={styles.comparisonExcerpt}>{item.implementation_excerpt}</code></p>}</details>}<table className={styles.auditTable}><thead><tr><th>Check</th><th>Result</th><th>Finding</th></tr></thead><tbody>{(Array.isArray(item.checks) ? item.checks : []).filter(Boolean).map((check, n) => <tr key={n}><th>{check.criterion.replaceAll("_", " ")}</th><td>{check.status}</td><td>{check.finding}</td></tr>)}</tbody></table><small>Evidence: {list(item.evidence_ids).join(", ") || "No source cited in this consistency check"}</small></article>)}</details>}
  </>;
}
