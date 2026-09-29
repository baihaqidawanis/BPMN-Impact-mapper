import styles from "./InterpretationGuide.module.css";

export function InterpretationGuide() {
  return (
    <details className={styles.guide}>
      <summary><span>How to read the report</span><small>Actions, confidence, and method boundaries</small><b>+</b></summary>
      <div className={styles.content}>
        <div className={styles.group}>
          <h3>Required action</h3>
          <p><b className={styles.modify}>MODIFY</b> Strong evidence indicates related implementation likely needs change.</p>
          <p><b className={styles.verify}>VERIFY</b> Relevant implementation found, but a required change is not proven.</p>
          <p><b className={styles.unresolved}>UNRESOLVED</b> Evidence is insufficient or matching is ambiguous.</p>
        </div>
        <div className={styles.group}>
          <h3>Confidence</h3>
          <p>A normalized result of explicit signals such as stable IDs, symbols, paths, lexical rank, and structure. It is not probability or correctness proof.</p>
        </div>
        <div className={styles.group}>
          <h3>Methods</h3>
          <p><b>Hybrid</b> combines traceability, structural, and BM25 evidence. <b>BM25</b> is a lexical baseline. <b>Configuration</b> follows frozen explicit mappings.</p>
        </div>
      </div>
    </details>
  );
}
