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
			<p>Composite labels group related topology edits and add predecessor/successor context; atomic BPMN changes remain visible.</p>
        </div>
        <div className={styles.group}>
          <h3>Methods</h3>
					<p><b>BM25</b> matches exact terms. <b>Semantic</b> matches meaning using local embeddings. <b>Fusion</b> combines their ranks with RRF. <b>Hybrid</b> fuses Semantic with graph-aware trace, structure, BM25, and static source-relation evidence.</p>
					<p>RRF and cosine scores describe retrieval relevance or rank consensus, not the probability that a file must change.</p>
        </div>
      </div>
    </details>
  );
}
