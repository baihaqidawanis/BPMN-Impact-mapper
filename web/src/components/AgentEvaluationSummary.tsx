import snapshot from "../../public/benchmarks/agent-add-task-2026-11-05.json";
import styles from "./BenchmarkSummary.module.css";

const percent = (value: number) => `${(value * 100).toFixed(1)}%`;

export function AgentEvaluationSummary() {
  return <section className={styles.experiments} aria-label="Saved AI review evaluation">
    <header><div><span className={styles.label}>Single-case diagnostic / {snapshot.date}</span>
      <h3>Hybrid + AI review</h3>
      <p>Add task: require a reason before stock adjustment submit.</p>
    </div><a href="/benchmarks/agent-add-task-2026-11-05.json" download>Download AI evaluation JSON</a></header>
    <p>Model: {snapshot.model}. Session status: <strong>{snapshot.session_status}</strong>. These are draft file proposals, not an approved change plan. This saved result is separate from the four-scenario benchmark above.</p>
    {snapshot.cases.map(item => <div key={`${item.change_type}:${item.element_id}`}>
      <div className={styles.tableWrap} tabIndex={0} aria-label="AI file selection comparison"><table>
        <thead><tr><th scope="col">Output</th><th scope="col">Selected files</th><th scope="col">Reference matches</th><th scope="col">Precision</th><th scope="col">Recall</th><th scope="col">F1</th></tr></thead>
        <tbody>{item.hybrid_display.map(row => <tr key={row.cutoff}><th scope="row">{row.label}</th><td>{row.selected}</td><td>{row.true_positive}</td><td>{percent(row.precision)}</td><td>{percent(row.recall)}</td><td>{percent(row.f1)}</td></tr>)}
          <tr className={styles.hybrid}><th scope="row">Hybrid + AI draft selection</th><td>{item.agent.selected}</td><td>{item.agent.true_positive}</td><td>{percent(item.agent.precision)}</td><td>{percent(item.agent.recall)}</td><td>{percent(item.agent.f1)}</td></tr>
        </tbody>
      </table></div>
      <p>Top 5 and Top 10 are user-selected display limits, not automatic confidence thresholds. AI proposed {item.agent.selected} files and retained all {item.expected_files.length} reference files. That improves selection precision over the displayed Top 10 in this case; it does not establish general superiority.</p>
      <details><summary>Ranking metrics and reference files</summary>
        <p>Hybrid AP: {item.hybrid_ranking.average_precision.toFixed(3)}. Reciprocal rank: {item.hybrid_ranking.reciprocal_rank.toFixed(3)}. R-precision: {percent(item.hybrid_ranking.r_precision)}. These describe the deterministic ranking for this one change.</p>
        <p>AI MAP / MRR / R-precision: unavailable. The AI draft is a file selection without a declared relevance ranking; Markdown order is not a ranking. AP becomes MAP when averaged across evaluated queries; reciprocal rank becomes MRR the same way.</p>
        <p>Reference files:</p><ul>{item.expected_files.map(file => <li key={file}><code>{file}</code></li>)}</ul>
        <p>Additional AI proposals outside the reference:</p><ul>{item.agent.files.filter(file => !item.expected_files.includes(file)).map(file => <li key={file}><code>{file}</code></li>)}</ul>
      </details>
    </div>)}
    <p className={styles.note}>Evaluation covers ADD_TASK / ReasonReview only. The two sequence-flow changes are not labelled by this reference. The three-file reference is a provisional source review, not independent ground truth; additional files count against precision here but are not independently proven irrelevant. No new model request is sent when viewing this result.</p>
  </section>;
}
