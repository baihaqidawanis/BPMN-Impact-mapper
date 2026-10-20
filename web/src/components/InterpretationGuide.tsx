import styles from "./InterpretationGuide.module.css";

export function InterpretationGuide() {
  return (
    <details className={styles.guide}>
      <summary><span>How to read the report</span><small>Actions, ranking scores, and evidence</small><b>+</b></summary>
      <div className={styles.content}>
        <div className={styles.group}>
          <h3>Required action</h3>
          <p><b className={styles.modify}>MODIFY</b> Strong evidence indicates related implementation likely needs change.</p>
          <p><b className={styles.verify}>VERIFY</b> Relevant implementation found, but a required change is not proven.</p>
          <p><b className={styles.unresolved}>UNRESOLVED</b> Evidence is insufficient or matching is ambiguous.</p>
        </div>
        <div className={styles.group}>
          <h3>Ranking score</h3>
					<p>Hybrid combines positions from code-evidence and semantic search. A high combined score can occur without a direct BPMN ID match, even when both methods have weak evidence. It is not a probability that code must change.</p>
          <p>A new task may have no ID in existing code. An ID from a neighbouring task supports context only; it does not prove the new task is implemented.</p>
			<p>Composite labels group related topology edits and add predecessor/successor context; atomic BPMN changes remain visible.</p>
        </div>
        <div className={styles.group}>
          <h3>Methods</h3>
					<p><b>BM25</b> looks for the same words. <b>Semantic</b> looks for similar meaning using a local model. <b>Fusion</b> combines both rankings. <b>BPMN-Context-Aware Hybrid</b> also uses task IDs, code names, file paths, BPMN neighbours, and code clues.</p>
					<p>RRF and cosine scores describe retrieval relevance or rank consensus, not the probability that a file must change.</p>
          <p><b>Ground truth</b> must come from a frozen reference implementation or independent reviewers after predictions are complete. Mapper candidates cannot label themselves.</p>
        </div>
      </div>
    </details>
  );
}
