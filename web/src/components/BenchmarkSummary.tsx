import snapshot from "../../public/benchmarks/staff-2026-10-29.json";
import experiments from "../../public/benchmarks/retrieval-studies-2026-10-30.json";
import styles from "./BenchmarkSummary.module.css";

const names: Record<string, string> = { bm25: "BM25", semantic: "Semantic", fusion: "BM25 + Semantic", hybrid: "Hybrid" };
const percent = (value: number) => `${(value * 100).toFixed(1)}%`;
const variants: Record<string, string> = { full: "Current Hybrid", "condition-focus": "Condition words", "generic-windows": "All text windows", "code-windows": "Code windows" };
const cases: Record<string, string> = { "selection-challenge-remove-check": "Remove product check", "selection-challenge-change-condition": "Allow copying cancelled PR" };

export function BenchmarkSummary() {
  return <section className={styles.panel} aria-label="Saved benchmark results">
    <header><div><span className={styles.label}>Saved test results · {snapshot.date}</span><h3>Benchmark results</h3><p>{snapshot.repository} / {snapshot.role} · {snapshot.positive_scenarios} change scenarios + {snapshot.negative_controls} no-impact control</p></div><a href="/benchmarks/staff-2026-10-29.json" download>Download results JSON</a></header>
    <p>These are saved results for this test set. They do not change when you select another repository or run a new mapping.</p>
    <div className={styles.tableWrap} tabIndex={0} aria-label="Benchmark comparison table"><table>
      <thead><tr><th scope="col">Method</th><th scope="col">Precision@5</th><th scope="col">Recall@5</th><th scope="col">Recall@10</th><th scope="col">MAP</th><th scope="col">MRR</th><th scope="col">R-precision</th></tr></thead>
      <tbody>{snapshot.results.map(row => <tr key={row.method} className={row.method === "hybrid" ? styles.hybrid : undefined}><th scope="row">{names[row.method] ?? row.method}</th><td>{percent(row.precision_at_5)}</td><td>{percent(row.recall_at_5)}</td><td>{percent(row.recall_at_10)}</td><td>{row.map.toFixed(3)}</td><td>{row.mrr.toFixed(3)}</td><td>{percent(row.r_precision)}</td></tr>)}</tbody>
    </table></div>
    <p className={styles.note}>The reference lists were prepared by inspecting source code and remain provisional. Scores above average the four positive scenarios. The separate no-impact control returned no candidates for all four methods. A high recall does not mean every returned file is relevant.</p>
    <details><summary>How to read the scores</summary><dl>
      <div><dt>Precision@5</dt><dd>How many of the first five candidates match the reference. With only two or three expected files per case, 100% is not possible for this test set.</dd></div>
      <div><dt>Recall@5 / @10</dt><dd>How many expected files were found within the first five or ten results.</dd></div>
      <div><dt>MAP</dt><dd>How well all expected files are ranked. Higher means they tend to appear earlier.</dd></div>
      <div><dt>MRR</dt><dd>How early the first expected file appears.</dd></div>
      <div><dt>R-precision</dt><dd>If a case has three expected files, check its first three results. This adapts the cutoff to the reference size.</dd></div>
    </dl><p>Each scenario/method ran twice. The full source tree was locally modified; the revision label alone does not freeze it. This is a development benchmark, not an independently verified accuracy claim. It does not measure live speed or Qwen performance.</p></details>
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
