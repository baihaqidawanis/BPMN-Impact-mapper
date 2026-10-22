import type { AnalysisMethod, Change, Report } from "@/lib/types";
import { methodDifferences } from "@/lib/method-differences";
import { candidateKeys, changeKey, comparisonChanges, overlapPercent, unresolvedComparisonChanges } from "@/lib/method-comparison";
import styles from "./ComparisonView.module.css";

interface Props {
  reports: Report[];
  selectedMethod: AnalysisMethod;
  onSelect: (report: Report) => void;
}

export function ComparisonView({ reports, selectedMethod, onSelect }: Props) {
  const hybrid = reports.find((report) => report.method === "hybrid");
  const differences = methodDifferences(reports);
  const sharedFingerprint = new Set(reports.map((report) => report.input_fingerprint)).size === 1;
  const changeKeys = Array.from(new Set(reports.flatMap((report) => comparisonChanges(report).map(changeKey)))).sort();

  return (
    <section className={styles.comparison} data-testid="method-comparison">
      <header>
        <div><p>Controlled retrieval comparison</p><h2>Same input, different methods</h2></div>
        <span className={sharedFingerprint ? styles.valid : styles.invalid}>{sharedFingerprint ? "Input fingerprint verified" : "Input mismatch"}</span>
      </header>
      <p className={styles.disclaimer}>Counts cover independent BPMN changes. Supporting flow edits remain in JSON but are not separate mapping failures. Overlap is shared file/change matches divided by the union with Hybrid; it is not accuracy.</p>
      <div className={styles.methods}>
        {reports.map((report) => {
          const candidates = candidateKeys(report);
          const unresolved = unresolvedComparisonChanges(report);
          const overlap = hybrid && report.method !== hybrid.method ? `${overlapPercent(candidates, candidateKeys(hybrid))}%` : hybrid ? "Reference" : "Unavailable";
          return <button type="button" aria-pressed={selectedMethod === report.method} onClick={() => onSelect(report)} key={report.method}>
            <span>{methodLabel(report.method)}</span>
							<b>{candidates.size}</b><small>file/change matches</small>
            <dl><div><dt>Unresolved changes</dt><dd>{unresolved}</dd></div><div><dt>Hybrid overlap</dt><dd>{overlap}</dd></div></dl>
          </button>;
        })}
      </div>
      <details className={styles.matrix}>
        <summary>Compare top candidates per BPMN change</summary>
        <div className={styles.tableWrap}><table><thead><tr><th>Change</th>{reports.map((report) => <th key={report.method}>{methodLabel(report.method)}</th>)}</tr></thead>
          <tbody>{changeKeys.map((key) => <tr key={key}><th>{key.replace("|", " · ")}</th>{reports.map((report) => <td key={report.method}>{topCandidates(comparisonChanges(report).find((change) => changeKey(change) === key))}</td>)}</tr>)}</tbody>
        </table></div>
      </details>
      <details className={styles.matrix}>
        <summary>Which files did each method find?</summary>
        <p className={styles.disclaimer}>Presence is compared per BPMN change within the returned Top-K. Missing does not mean irrelevant.</p>
        {differences === null ? <p>Matching input fingerprints are required before comparing files.</p> : <div className={styles.tableWrap}><table><thead><tr><th>BPMN change</th><th>File</th><th>Found by</th><th>Coverage</th></tr></thead><tbody>{differences.map(row => <tr key={`${row.change}|${row.file}`}><td>{row.change}</td><td>{row.file}</td><td>{row.methods.map(method => methodLabel(method as AnalysisMethod)).join(", ")}</td><td>{row.methods.length === reports.length ? "All compared methods" : row.methods.length === 1 ? "Only this method" : "Some methods"}</td></tr>)}</tbody></table>{differences.length === 0 && <p>No files returned by the compared methods.</p>}</div>}
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

function topCandidates(change?: Change) {
  if (!change || change.impacts.length === 0) return <span className={styles.none}>No related file found</span>;
  return <ol>{change.impacts.slice(0, 3).map((impact) => <li key={`${impact.file}|${impact.symbol}`}><b>{impact.file}</b><span>{impact.symbol || "whole file"}</span></li>)}</ol>;
}
