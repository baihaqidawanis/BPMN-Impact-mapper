import snapshot from "../../public/benchmarks/staff-2026-10-29.json";
import experiments from "../../public/benchmarks/retrieval-studies-2026-10-30.json";
import styles from "./BenchmarkSummary.module.css";
import { AgentEvaluationSummary } from "./AgentEvaluationSummary";
import { ModelSelectionSummary } from "./ModelSelectionSummary";

const names: Record<string, string> = { bm25: "BM25", semantic: "Semantic", fusion: "BM25 + Semantic", hybrid: "Hybrid" };
const percent = (value: number) => `${(value * 100).toFixed(1)}%`;
const variants: Record<string, string> = { full: "Current Hybrid", "condition-focus": "Condition words", "generic-windows": "All text windows", "code-windows": "Code windows" };
const cases: Record<string, string> = { "selection-challenge-remove-check": "Remove product check", "selection-challenge-change-condition": "Allow copying cancelled PR" };

export function BenchmarkSummary() {
  const hybrid = snapshot.results.find(r => r.method === "hybrid");
  return <section className={styles.panel} aria-label="Saved benchmark results">
    <header><div><span className={styles.label}>Saved snapshot · {snapshot.date}</span><h3>Benchmark results</h3><p>{snapshot.repository} / {snapshot.role} · {snapshot.positive_scenarios} scenarios + {snapshot.negative_controls} control</p></div><a href="/benchmarks/staff-2026-10-29.json" download>Download results JSON</a></header>
    {hybrid && (
      <div className={styles.highlights}>
        <div><span>Hybrid Recall@10</span><b>{percent(hybrid.recall_at_10)}</b></div>
        <div><span>Hybrid MRR</span><b>{hybrid.mrr.toFixed(3)}</b></div>
        <div><span>Hybrid MAP</span><b>{hybrid.map.toFixed(3)}</b></div>
        <div><span>No-impact controls</span><b>0 FP</b></div>
      </div>
    )}
    <div className={styles.tableWrap} tabIndex={0} aria-label="Benchmark comparison table"><table>
      <thead><tr><th scope="col">Method</th><th scope="col">Precision@5</th><th scope="col">Recall@5</th><th scope="col">Recall@10</th><th scope="col">MAP</th><th scope="col">MRR</th><th scope="col">R-precision</th></tr></thead>
      <tbody>{snapshot.results.map(row => <tr key={row.method} className={row.method === "hybrid" ? styles.hybrid : undefined}><th scope="row">{names[row.method] ?? row.method}</th><td>{percent(row.precision_at_5)}</td><td>{percent(row.recall_at_5)}</td><td>{percent(row.recall_at_10)}</td><td>{row.map.toFixed(3)}</td><td>{row.mrr.toFixed(3)}</td><td>{percent(row.r_precision)}</td></tr>)}</tbody>
    </table></div>
    <details><summary>Metric definitions &amp; caveats</summary><dl>
      <div><dt>Precision@5</dt><dd>Share of first 5 candidates matching reference.</dd></div>
      <div><dt>Recall@5 / @10</dt><dd>Share of expected files found within top 5 or 10.</dd></div>
      <div><dt>MAP / MRR / R-precision</dt><dd>Ranking quality across all expected files, first match rank, and cutoff at reference size R.</dd></div>
    </dl><p className={styles.note}>Provisional development references; scores average the four positive scenarios and do not change when you run new mappings.</p></details>
    <details className={styles.experiments}><summary>AI review case diagnostic (Add Task)</summary><AgentEvaluationSummary /></details>
    <details className={styles.experiments}><summary>Lab model selection study</summary><ModelSelectionSummary /></details>
    <details className={styles.experiments}>
      <summary>Latest retrieval experiments · {experiments.date}</summary>
      <section aria-label="Recent retrieval experiments">
        <p>{experiments.analysis_runs} repeated analysis runs on two change cases and an unchanged control. These are per-case Hybrid experiments, separate from the four-method benchmark above.</p>
        <div className={styles.tableWrap} tabIndex={0} aria-label="Recent experiment comparison"><table>
          <thead><tr><th scope="col">Case / variant</th><th scope="col">Precision@5</th><th scope="col">Recall@5</th><th scope="col">Recall@10</th><th scope="col">MAP</th><th scope="col">MRR</th><th scope="col">R-precision</th></tr></thead>
          <tbody>{experiments.results.map(row => <tr key={`${row.scenario}-${row.variant}`}><th scope="row">{cases[row.scenario]}<small className={styles.variant}>{variants[row.variant]}</small></th><td>{percent(row.precision_at_5)}</td><td>{percent(row.recall_at_5)}</td><td>{percent(row.recall_at_10)}</td><td>{row.map.toFixed(3)}</td><td>{row.mrr.toFixed(3)}</td><td>{percent(row.r_precision)}</td></tr>)}</tbody>
        </table></div>
        <p>No experimental variant was adopted. Condition-word queries and all-text windows reduced some results; code windows did not recover the missing frontend file. The production Hybrid is unchanged.</p>
        <p>References remain provisional source reviews, and these cases were already used during development. They are not independent validation. No-impact controls returned no changes. Running your own mapping does not calculate Precision or Recall without an expected-file reference.</p>
        <a href="/benchmarks/retrieval-studies-2026-10-30.json" download>Download experiment results JSON</a>
      </section>
    </details>
  </section>;
}
