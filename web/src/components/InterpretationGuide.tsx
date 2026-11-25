import styles from "./InterpretationGuide.module.css";

export function InterpretationGuide({ expanded = false, flat = false }: { expanded?: boolean; flat?: boolean }) {
  const body = (
    <div className={styles.content}>
      <div className={styles.group}>
        <h3>Required action</h3>
        <p><b className={styles.modify}>MODIFY</b> Strong evidence indicates code likely needs change.</p>
        <p><b className={styles.verify}>VERIFY</b> Relevant code found; confirm whether edits are required.</p>
        <p><b className={styles.unresolved}>UNRESOLVED</b> Insufficient or ambiguous evidence.</p>
      </div>
      <div className={styles.group}>
        <h3>Ranking score</h3>
        <p>Hybrid combines BPMN-context BM25 and Semantic retrieval via RRF (equal rank votes, not raw score sums).</p>
        <p>Scores measure retrieval consensus, not change probability. Hybrid shows the top candidate per file before Top-K cutoff.</p>
        <p>Insert-task flow rewires group under the new task; every raw BPMN edit remains in canonical JSON.</p>
      </div>
      <div className={styles.group}>
        <h3>Methods</h3>
        <p><b>BM25</b> exact word match · <b>Semantic</b> embedding similarity · <b>Fusion</b> combines both · <b>Hybrid</b> adds BPMN context to BM25 + Semantic.</p>
        <p><b>Ground truth</b> must come from independent reviewers after inference; mapper candidates never label themselves.</p>
      </div>
    </div>
  );

  if (flat) {
    return (
      <section className={`${styles.guide} ${styles.flat}`} aria-label="How to read the report">
        {body}
      </section>
    );
  }

  return (
    <details className={styles.guide} open={expanded || undefined}>
      <summary><span>How to read the report</span><small>Actions, scores, and methods</small><b>+</b></summary>
      {body}
    </details>
  );
}
