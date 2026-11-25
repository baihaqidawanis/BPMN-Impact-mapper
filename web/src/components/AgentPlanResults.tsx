import type { ReactNode } from "react";
import type { ChangePlan } from "@/lib/planning";
import styles from "./QwenReview.module.css";

type FilePlan = ChangePlan["plans"][number];
const list = (values?: string[]) => (Array.isArray(values) ? values.filter(value => typeof value === "string") : []);

function FileAssessment({ file, showPath = false }: { file: FilePlan; showPath?: boolean }) {
  const unchanged = file.action === "NO_ACTION";
  return (
    <article>
      <header>
        <b>{showPath ? file.file : file.symbol || "Whole file"}</b>
        <span>{unchanged ? "No edit" : file.action}</span>
      </header>
      <small>
        {showPath && file.symbol && `${file.symbol} · `}
        {file.element_id ? [file.change_type?.replaceAll("_", " ").toLowerCase(), file.element_id].filter(Boolean).join(" / ") : file.layer}
      </small>
      <p>{file.summary}</p>
      {(file.current_behavior || file.target_behavior) && (
        <section className={styles.behaviorDiff} aria-label="Behavior comparison">
          {file.current_behavior && (
            <div className={styles.behaviorBefore}>
              <h4>Current behavior</h4>
              <p>{file.current_behavior}</p>
            </div>
          )}
          {file.target_behavior && (
            <div className={styles.behaviorAfter}>
              <h4>Required behavior</h4>
              <p>{file.target_behavior}</p>
            </div>
          )}
        </section>
      )}
      {file.design_rationale && (
        <div>
          <h4>Design rationale</h4>
          <p>{file.design_rationale}</p>
        </div>
      )}
      {!!list(file.assumptions).length && (
        <div>
          <h4>Assumptions and open decisions</h4>
          <ul>
            {list(file.assumptions).map((value, i) => (
              <li key={i}>{value}</li>
            ))}
          </ul>
        </div>
      )}
      {!unchanged && (
        <>
          {!!list(file.proposed_changes).length && (
            <div>
              <h4>Proposed changes</h4>
              <ul>
                {list(file.proposed_changes).map((value, i) => (
                  <li key={i}>{value}</li>
                ))}
              </ul>
            </div>
          )}
          {!!list(file.steps).length && (
            <div>
              <h4>Implementation steps</h4>
              <ol>
                {list(file.steps).map((value, i) => (
                  <li key={i}>{value}</li>
                ))}
              </ol>
            </div>
          )}
        </>
      )}
      {!!list(file.acceptance_checks).length && (
        <div>
          <h4>{unchanged ? "Existing-contract checks" : "Acceptance checks"}</h4>
          <ul>
            {list(file.acceptance_checks).map((value, i) => (
              <li key={i}>{value}</li>
            ))}
          </ul>
        </div>
      )}
      <small>Evidence: {list(file.evidence_ids).join(", ")}</small>
    </article>
  );
}

export function AgentPlanResults({ plan, actions }: { plan: ChangePlan; actions?: ReactNode }) {
  const files = Array.isArray(plan.plans) ? plan.plans.filter(file => file && typeof file.file === "string") : [];
  const edits = files.filter(file => file.action !== "NO_ACTION");
  const unchanged = files.filter(file => file.action === "NO_ACTION");
  const byFile = new Map<string, FilePlan[]>();
  for (const proposal of edits) {
    const entries = byFile.get(proposal.file);
    if (entries) entries.push(proposal);
    else byFile.set(proposal.file, [proposal]);
  }
  const findings = Array.isArray(plan.review_findings) ? plan.review_findings.filter(item => item && typeof item.file === "string") : [];
  return (
    <>
      {byFile.size > 0 ? (
        <section className={styles.fileList} aria-label="Proposed file edits">
          <header>
            <div>
              <h4>Proposed edits</h4>
              <span>
                {byFile.size} {byFile.size === 1 ? "file" : "files"} · {plan.status === "COMPLETE" ? "Review complete" : "Draft"}
              </span>
            </div>
            {actions}
          </header>
          {Array.from(byFile, ([path, proposals], index) => {
            const separator = path.lastIndexOf("/");
            return (
              <details key={path} className={styles.fileRow} open={index === 0}>
                <summary>
                  <span className={styles.fileName}>
                    <strong>{path.slice(separator + 1)}</strong>
                    {separator >= 0 && <code>{path.slice(0, separator + 1)}</code>}
                  </span>
                  <span className={styles.fileCount}>
                    {proposals.length} {proposals.length === 1 ? "proposal" : "proposals"}
                  </span>
                </summary>
                {proposals.map((file, i) => (
                  <FileAssessment key={`${file.element_id}:${file.symbol}:${i}`} file={file} />
                ))}
              </details>
            );
          })}
        </section>
      ) : (
        <div className={styles.issues} role="status">
          <h4>No source edits were proposed</h4>
          <p>{unchanged.length ? "Open the reviewed targets below to see why." : "Check the review issues for missing evidence."}</p>
          {actions}
        </div>
      )}
      {unchanged.length > 0 && (
        <details>
          <summary>{unchanged.length} reviewed targets with no edit proposed</summary>
          {unchanged.map((file, i) => (
            <FileAssessment key={`${file.file}:${file.element_id}:${i}`} file={file} showPath />
          ))}
        </details>
      )}
      {findings.length > 0 && (
        <details>
          <summary>Reviewer checks ({findings.length})</summary>
          <p>These are model conclusions, not independent proof of correctness.</p>
          {findings.map((item, i) => (
            <article key={`${item.file}:${item.element_id}:${i}`}>
              <b>{item.file}</b>
              <small>
                {item.element_id} · {item.basis === "source_and_proposal" ? "Source and proposal checked" : "Proposal consistency only"}
              </small>
              {(item.source_excerpt || item.implementation_excerpt) && (
                <details>
                  <summary>Compared source and plan</summary>
                  {item.source_excerpt && (
                    <p>
                      Source: <code className={styles.comparisonExcerpt}>{item.source_excerpt}</code>
                    </p>
                  )}
                  {item.implementation_excerpt && (
                    <p>
                      Plan item: <code className={styles.comparisonExcerpt}>{item.implementation_excerpt}</code>
                    </p>
                  )}
                </details>
              )}
              <div className={styles.tableWrap}>
                <table className={styles.auditTable}>
                  <thead>
                    <tr>
                      <th>Check</th>
                      <th>Result</th>
                      <th>Finding</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(Array.isArray(item.checks) ? item.checks : []).filter(Boolean).map((check, n) => (
                      <tr key={n}>
                        <th>{check.criterion.replaceAll("_", " ")}</th>
                        <td>{check.status}</td>
                        <td>{check.finding}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <small>Evidence: {list(item.evidence_ids).join(", ") || "No source cited in this consistency check"}</small>
            </article>
          ))}
        </details>
      )}
    </>
  );
}
