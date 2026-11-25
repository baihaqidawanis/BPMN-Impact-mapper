import styles from "./MappingEmptyState.module.css";

export function MappingEmptyState({ repositoryPath, baselineName, editedName }: {
  repositoryPath: string; baselineName?: string; editedName?: string;
}) {
  const inputs = [
    { title: "Source repository", value: repositoryPath.trim(), fallback: "Enter the local repository path." },
    { title: "Baseline BPMN", value: baselineName, fallback: "Choose the original process." },
    { title: "New BPMN", value: editedName, fallback: "Upload a changed version or edit in Canvas." },
  ];
  const ready = inputs.filter(input => !!input.value).length;
  return (
    <div className={styles.empty}>
      <div className={styles.intro}>
        <span className={styles.kicker}>Ready check</span>
        <h3>Set up your comparison</h3>
        <p>Select a local repository snapshot and two BPMN versions to map process edits to source candidates.</p>
        <span className={styles.note} role="status">{ready === 3 ? "Inputs ready. Run impact analysis to continue." : `${ready} of 3 inputs ready`}</span>
      </div>
      <ol className={styles.steps} aria-label="Required analysis inputs">
        {inputs.map((input, i) => (
          <li key={input.title} data-ready={!!input.value}>
            <span className={styles.check}>
              {input.value ? (
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.75" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg>
              ) : `0${i + 1}`}
            </span>
            <div>
              <b>{input.title}<span>{input.value ? "Ready" : "Required"}</span></b>
              <small>{input.value || input.fallback}</small>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
