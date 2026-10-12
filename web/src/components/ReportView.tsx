import { reportDownloadURL } from "@/lib/api";
import type { Change, Impact, Report, ScoreBreakdown } from "@/lib/types";
import { ReviewPanel } from "./ReviewPanel";
import styles from "./ReportView.module.css";

export function ReportView({ report }: { report: Report }) {
  const candidateCount = report.changes.reduce((sum, change) => sum + change.impacts.length, 0);
  const unresolvedCount = report.changes.reduce((sum, change) => sum + (change.unresolved_impacts?.length ?? 0), 0);
  return (
		<div className={styles.report} data-testid="report-view">
      <div className={styles.reportHeader}>
        <div><p>Analysis complete</p><h2>{report.repository.name}</h2><span>{report.repository.revision} · {report.method}{report.method_metadata?.model ? ` · ${report.method_metadata.model}` : ""}</span></div>
        <div className={styles.downloads}>
          <a href={reportDownloadURL(report.analysis_id, "json")}>JSON</a>
          <a href={reportDownloadURL(report.analysis_id, "md")}>Markdown</a>
        </div>
      </div>
      <div className={styles.metrics}>
        <Metric value={report.changes.length} label="BPMN changes" />
        <Metric value={candidateCount} label="Candidates" />
        <Metric value={unresolvedCount} label="Unresolved" tone={unresolvedCount ? "warn" : "ok"} />
        <Metric value={report.warnings.length} label="Warnings" tone={report.warnings.length ? "warn" : "ok"} />
      </div>
      {report.warnings.length > 0 && (
        <div className={styles.warnings}><strong>Analysis warnings</strong>{report.warnings.map((warning) => <p key={`${warning.code}-${warning.message}`}><b>{warning.code}</b> {warning.message}</p>)}</div>
      )}
      {!!report.source_analysis?.length && <details className={styles.warnings}>
        <summary>Source analysis coverage ({report.source_analysis.length} files)</summary>
        <p>Text-only files support lexical retrieval. Static calls cover a limited subset of Go; these capabilities do not measure accuracy.</p>
        {report.source_analysis.map((file) => <p key={file.file}><b>{file.language}</b> {file.file} — {file.capabilities.join(", ")}{file.fallback ? " (parser fallback)" : ""}</p>)}
      </details>}
      <div className={styles.changeList}>
        {report.changes.map((change, index) => <ChangeCard change={change} index={index} key={`${change.change_type}-${change.new_element?.id ?? change.old_element?.id}-${index}`} />)}
      </div>
      <ReviewPanel key={report.analysis_id} report={report} />
      <footer className={styles.fingerprint}>Analysis ID <code>{report.analysis_id}</code><span>Input fingerprint <code>{report.input_fingerprint}</code></span></footer>
    </div>
  );
}

function Metric({ value, label, tone }: { value: number; label: string; tone?: "warn" | "ok" }) {
  return <div className={`${styles.metric} ${tone ? styles[tone] : ""}`}><b>{value}</b><span>{label}</span></div>;
}

function ChangeCard({ change, index }: { change: Change; index: number }) {
  const element = change.new_element ?? change.old_element;
  return (
    <article className={styles.change}>
      <div className={styles.changeHead}>
        <span className={styles.index}>{String(index + 1).padStart(2, "0")}</span>
        <div><div className={styles.changeType}>{change.change_type.replaceAll("_", " ")}</div><h3>{element?.name || element?.id || "Unresolved element"}</h3><p>{element?.id} · {element?.type}{element?.actor ? ` · ${element.actor}` : ""}</p></div>
        <span className={styles.impactCount}>{change.impacts.length} candidates</span>
      </div>
      {change.reason && <p className={styles.reason}>{change.reason}</p>}
		{change.context && <div className={styles.context}><b>{change.context.pattern.replaceAll("_", " ")}</b><span>{contextSummary(change)}</span></div>}
      {change.impacts.length > 0 ? <div className={styles.impacts}>{change.impacts.map((impact, impactIndex) => <ImpactRow impact={impact} rank={impactIndex + 1} key={`${impact.file}-${impact.symbol}-${impactIndex}`} />)}</div> : <p className={styles.noImpact}>No evidence-bearing candidate crossed the retrieval threshold.</p>}
      {(change.unresolved_impacts?.length ?? 0) > 0 && <div className={styles.unresolved}><b>Unresolved impact</b>{change.unresolved_impacts.map((item) => <p key={item}>{item}</p>)}</div>}
    </article>
  );
}

function contextSummary(change: Change) {
	const context = change.context;
	if (!context) return "";
	const predecessors = context.predecessors?.map((element) => element.name || element.id).join(", ");
	const successors = context.successors?.map((element) => element.name || element.id).join(", ");
	return [predecessors && `from ${predecessors}`, successors && `to ${successors}`].filter(Boolean).join(" ");
}

function ImpactRow({ impact, rank }: { impact: Impact; rank: number }) {
  return (
		<details className={styles.impact} data-testid="impact-details">
      <summary>
        <span className={styles.rank}>{String(rank).padStart(2, "0")}</span>
        <span className={`${styles.action} ${styles[impact.required_action.toLowerCase()] ?? ""}`}>{impact.required_action}</span>
        <span className={styles.target}><b>{impact.file}</b><small>{impact.symbol || "file-level candidate"} · {impact.module} / {impact.layer}</small></span>
        <span className={styles.confidence}><b>{Math.round(impact.confidence * 100)}%</b><small>{impact.score_breakdown.rrf ? "rank consensus" : "confidence"}</small></span>
        <span className={styles.chevron}>⌄</span>
      </summary>
      <div className={styles.impactBody}>
        <div><h4>Evidence</h4>{impact.evidence.map((evidence, index) => <div className={styles.evidence} key={`${evidence.type}-${index}`}><b>{evidence.type}</b><code>{evidence.location}</code><p>{evidence.excerpt}</p></div>)}</div>
        <div><h4>Score breakdown</h4><ScoreTable scores={impact.score_breakdown} /></div>
      </div>
    </details>
  );
}

function ScoreTable({ scores }: { scores: ScoreBreakdown }) {
  const entries = Object.entries(scores) as [keyof ScoreBreakdown, number][];
  return <div className={styles.scores}>{entries.map(([name, value]) => <div key={name}><span>{name.replaceAll("_", " ")}</span><i><em style={{ width: `${Math.min(100, Math.max(0, value * 100))}%` }} /></i><b>{value.toFixed(3)}</b></div>)}</div>;
}
