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
  return <div className={styles.empty}>
    <svg className={styles.diagram} viewBox="0 0 420 160" fill="none" aria-hidden="true">
      <path d="M43 80h42m100 0h42m46 0h47" stroke="#aab5bf" strokeWidth="1.5"/>
      <path d="m80 76 5 4-5 4m142-8 5 4-5 4m88-8 5 4-5 4" stroke="#aab5bf" strokeWidth="1.5"/>
      <circle cx="30" cy="80" r="13" fill="#fff" stroke="#aab5bf" strokeWidth="1.5"/>
      <rect x="85" y="51" width="100" height="58" rx="10" fill="#fff" stroke="#aab5bf" strokeWidth="1.5"/>
      <path d="M108 73h51m-51 12h35" stroke="#aab5bf" strokeWidth="2" strokeLinecap="round"/>
      <path d="m250 57 23 23-23 23-23-23z" fill="#fff" stroke="#aab5bf" strokeWidth="1.5"/>
      <path d="m245 75 10 10m0-10-10 10" stroke="#aab5bf" strokeWidth="1.5"/>
      <rect x="320" y="51" width="85" height="58" rx="10" fill="#edf3fb" stroke="#8eafda" strokeWidth="1.5"/>
      <path d="M345 80h35m-17-17v34" stroke="#497cb8" strokeWidth="1.5" strokeLinecap="round"/>
    </svg>
    <h3>Set up your comparison</h3>
    <p>Start with the source repository and two process versions. Your report will list possible impacts with evidence from the code.</p>
    <ol className={styles.steps} aria-label="Required analysis inputs">{inputs.map((input, i) => <li key={input.title} data-ready={!!input.value}>
      <span className={styles.check}>{input.value ? <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><path d="m3 8 3 3 7-7"/></svg> : `0${i + 1}`}</span>
      <div><b>{input.title}<span>{input.value ? "Ready" : "Required"}</span></b><small>{input.value || input.fallback}</small></div>
    </li>)}</ol>
    <span className={styles.note} role="status">{ready === 3 ? "Inputs ready. Run impact analysis to continue." : `${ready} of 3 inputs ready`}</span>
  </div>;
}
