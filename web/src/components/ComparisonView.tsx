import type { AnalysisMethod, Change, Report } from "@/lib/types";
import styles from "./ComparisonView.module.css";

interface Props {
  reports: Report[];
  selectedMethod: AnalysisMethod;
  onSelect: (report: Report) => void;
}

export function ComparisonView({ reports, selectedMethod, onSelect }: Props) {
  const hybrid = reports.find((report) => report.method === "hybrid");
  const sharedFingerprint = new Set(reports.map((report) => report.input_fingerprint)).size === 1;
  const changeKeys = Array.from(new Set(reports.flatMap((report) => report.changes.map(changeKey)))).sort();

  return (
    <section className={styles.comparison} data-testid="method-comparison">
      <header>
        <div><p>Controlled retrieval comparison</p><h2>Same input, different methods</h2></div>
        <span className={sharedFingerprint ? styles.valid : styles.invalid}>{sharedFingerprint ? "Input fingerprint verified" : "Input mismatch"}</span>
      </header>
      <p className={styles.disclaimer}>Candidate counts and overlap describe retrieval behavior, not accuracy. Accuracy requires independent reconciled ground truth.</p>
      <div className={styles.methods}>
        {reports.map((report) => {
          const candidates = candidateKeys(report);
          const unresolved = report.changes.filter((change) => change.impacts.length === 0 || (change.unresolved_impacts?.length ?? 0) > 0).length;
          const overlap = hybrid ? overlapPercent(candidates, candidateKeys(hybrid)) : 0;
          return <button type="button" aria-pressed={selectedMethod === report.method} onClick={() => onSelect(report)} key={report.method}>
            <span>{methodLabel(report.method)}</span>
            <b>{candidates.size}</b><small>unique candidates</small>
            <dl><div><dt>Unresolved</dt><dd>{unresolved}</dd></div><div><dt>Hybrid overlap</dt><dd>{overlap}%</dd></div></dl>
          </button>;
        })}
      </div>
      <details className={styles.matrix}>
        <summary>Compare top candidates per BPMN change</summary>
        <div className={styles.tableWrap}><table><thead><tr><th>Change</th>{reports.map((report) => <th key={report.method}>{methodLabel(report.method)}</th>)}</tr></thead>
          <tbody>{changeKeys.map((key) => <tr key={key}><th>{key.replace("|", " · ")}</th>{reports.map((report) => <td key={report.method}>{topCandidates(report.changes.find((change) => changeKey(change) === key))}</td>)}</tr>)}</tbody>
        </table></div>
      </details>
    </section>
  );
}

function methodLabel(method: AnalysisMethod): string {
  if (method === "bm25") return "BM25";
	if (method === "semantic") return "Semantic";
	if (method === "fusion") return "BM25 + Semantic";
  if (method === "configuration") return "Configuration";
  return "Hybrid";
}

function changeKey(change: Change): string {
  return `${change.change_type}|${change.new_element?.id ?? change.old_element?.id ?? "unresolved"}`;
}

function candidateKeys(report: Report): Set<string> {
  return new Set(report.changes.flatMap((change) => change.impacts.map((impact) => `${changeKey(change)}|${impact.file}`)));
}

function overlapPercent(left: Set<string>, right: Set<string>): number {
  const union = new Set([...left, ...right]);
  if (union.size === 0) return 100;
  let shared = 0;
  for (const item of left) if (right.has(item)) shared++;
  return Math.round((shared / union.size) * 100);
}

function topCandidates(change?: Change) {
  if (!change || change.impacts.length === 0) return <span className={styles.none}>No candidate</span>;
  return <ol>{change.impacts.slice(0, 3).map((impact) => <li key={`${impact.file}|${impact.symbol}`}><b>{impact.file}</b><span>{impact.symbol || "file level"}</span></li>)}</ol>;
}
