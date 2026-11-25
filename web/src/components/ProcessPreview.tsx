import { useState } from "react";
import styles from "./ProcessPreview.module.css";

export function ProcessPreview() {
  const [after, setAfter] = useState(true);
  return (
    <figure className={styles.preview}>
      <figcaption>
        <div><small>Example process</small><h2>Stock adjustment</h2></div>
        <div className={styles.switcher} role="group" aria-label="Preview version">
          <button type="button" aria-pressed={!after} onClick={() => setAfter(false)}>Before</button>
          <button type="button" aria-pressed={after} onClick={() => setAfter(true)}>After</button>
        </div>
      </figcaption>
      <ol className={styles.flow} aria-label="Process steps">
        <li>Prepare draft</li>
        {after && <li className={styles.added}><small>Added task</small>Validate reason</li>}
        <li>Submit adjustment</li>
      </ol>
    </figure>
  );
}
