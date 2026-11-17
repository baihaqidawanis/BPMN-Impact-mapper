import snapshot from "../../public/benchmarks/lab-model-selection-2026-11-10.json";
import styles from "./BenchmarkSummary.module.css";

export function ModelSelectionSummary() {
  return <section className={styles.experiments} aria-label="Agent model selection evidence">
    <header><div><span className={styles.label}>Model experiments / {snapshot.date}</span>
      <h3>Why these models?</h3><p>{snapshot.status}</p>
    </div><a href="/benchmarks/lab-model-selection-2026-11-10.json" download>Download model study JSON</a></header>
    <p>Case: {snapshot.case}. These are candidates for further testing, not a claim that the complete AI pipeline is ready.</p>
    <p><strong>Current conclusion:</strong> {snapshot.conclusion}</p>
    <dl>{snapshot.roles.map(item => <div key={item.role}>
      <dt>{item.role} <code>{item.model}</code></dt>
      <dd>{item.reason}</dd><dd><strong>Limit:</strong> {item.limitation}</dd>
    </div>)}</dl>
    <details><summary>Reviewer control check</summary>
      <p>{snapshot.control_gate.note}</p>
      <div className={styles.tableWrap} tabIndex={0} aria-label="Reviewer control qualification"><table>
        <thead><tr><th scope="col">Model / configuration</th><th scope="col">Valid / required</th><th scope="col">Correct / required</th><th scope="col">Passed full suite</th></tr></thead>
        <tbody>{snapshot.control_gate.results.map(row => <tr key={`${row.study}:${row.model}`}><th scope="row">{row.model}<small className={styles.variant}>{row.configuration}</small></th><td>{row.summary.valid_decisions}/{row.summary.expected_controls}</td><td>{row.summary.correct_decisions}/{row.summary.expected_controls}</td><td>{row.summary.passed_all_controls ? "Yes" : "No"}</td></tr>)}</tbody>
      </table></div>
    </details>
    {snapshot.studies.map(study => <details key={study.role}>
      <summary>{study.role}: measured results</summary>
      <p>{study.date} / {study.protocol}. Study ID: <code>{study.artifact}</code>.</p>
      <div className={styles.tableWrap} tabIndex={0} aria-label={`${study.role} model comparison`}><table>
        <thead><tr><th scope="col">Installed model</th>{study.columns.map(column => <th scope="col" key={column}>{column}</th>)}</tr></thead>
        <tbody>{study.rows.map(row => <tr key={row.model}><th scope="row"><code>{row.model}</code></th>{row.cells.map((cell, i) => <td className={styles.studyCell} key={i}>{cell}</td>)}</tr>)}</tbody>
      </table></div><p>{study.note}</p>
    </details>)}
    <details><summary>Complete pipeline trial: {snapshot.pipeline.status.replaceAll("_", " ").toLowerCase()}</summary>
      <p>{snapshot.pipeline.date} / {snapshot.pipeline.protocol} / {snapshot.pipeline.mode}.</p>
      <p>Tested configuration: {snapshot.pipeline.models}.</p>
      <p>{snapshot.pipeline.turns} orchestration turns, {snapshot.pipeline.tools} source tool calls, {snapshot.pipeline.revisions} revisions. Turns are not an inference count.</p>
      <p>{snapshot.pipeline.inferences} inference calls logged in this run: {snapshot.pipeline.inference_seconds.toFixed(1)}s total, including {snapshot.pipeline.loading_seconds.toFixed(1)}s loading weights. These are observed times for one run, not a repeated speed benchmark.</p>
      <p>{snapshot.pipeline.result} Study ID: <code>{snapshot.pipeline.artifact}</code>.</p>
    </details>
    <details><summary>Earlier complete-case attempts</summary>{snapshot.past_runs.map(run => <div key={run.artifact}>
      <p><strong>{run.protocol}: {run.status}</strong> / <code>{run.artifact}</code></p>
      <p>{run.calls} inference calls, {run.model_seconds.toFixed(1)}s total including {run.loading_seconds.toFixed(1)}s loading. {run.result}</p>
    </div>)}</details>
    <details><summary>What these results do and do not measure</summary><ul>{snapshot.limitations.map(note => <li key={note}>{note}</li>)}</ul></details>
  </section>;
}
