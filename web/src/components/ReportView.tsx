"use client";

import { useState } from "react";
import { planningContextURL, reportDownloadURL } from "@/lib/api";
import { groupResults, independentChanges, matchesCategory, type ResultCategory } from "@/lib/report-results";
import { fusionCalculation, hasFusion, rankingExplanation, traceExplanation, usesCodeEvidence } from "@/lib/report-explanation";
import type { AnalysisMethod, Change, Impact, Report, ScoreBreakdown } from "@/lib/types";
import styles from "./ReportView.module.css";

export function ReportView({ report }: { report: Report }) {
  const [category, setCategory] = useState<ResultCategory>("all");
  const [grouping, setGrouping] = useState<"change" | "file">("change");
  const groups = groupResults(report, category);
  const visibleCount = groups.reduce((sum, group) => sum + group.entries.length, 0);
  const candidateCount = report.changes.reduce((sum, change) => sum + change.impacts.length, 0);
  const unresolvedCount = report.changes.reduce((sum, change) => sum + (change.unresolved_impacts?.length ?? 0), 0);
  const visibleChanges = independentChanges(report);
  const groupedFlowCount = report.changes.length - visibleChanges.length;
  const partialBpmnSupport = report.warnings.some(warning => warning.code === "BPMN_PARTIAL_SUPPORT");
  return (
		<div className={styles.report} data-testid="report-view">
      <div className={styles.reportHeader}>
        <div><p>{partialBpmnSupport ? "Analysis finished with BPMN limitations" : "Analysis complete"}</p><h2>{report.repository.name}</h2><span>{report.repository.revision} · {report.method}{report.method_metadata?.model ? ` · ${report.method_metadata.model}` : ""}</span></div>
        <div className={styles.downloads}>
          <a href={reportDownloadURL(report.analysis_id, "json")}>JSON</a>
          <a href={reportDownloadURL(report.analysis_id, "md")}>Markdown</a>
          <a href={planningContextURL(report.analysis_id)} title="Bounded evidence that a future planner may receive">Planning context</a>
        </div>
      </div>
      <div className={styles.metrics}>
        <Metric value={report.changes.length} label={partialBpmnSupport ? "Detected BPMN changes" : "BPMN changes"} />
        <Metric value={candidateCount} label="Candidates" />
        <Metric value={unresolvedCount} label="Unresolved" tone={unresolvedCount ? "warn" : "ok"} />
        <Metric value={report.warnings.length} label="Warnings" tone={report.warnings.length ? "warn" : "ok"} />
      </div>
      {groupedFlowCount > 0 && <p className={styles.groupNote}>{visibleChanges.length} independent change{visibleChanges.length === 1 ? "" : "s"}; {groupedFlowCount} sequence-flow edit{groupedFlowCount === 1 ? "" : "s"} grouped with their primary change. Raw edits remain in JSON.</p>}
      {partialBpmnSupport && <p className={styles.warnings} role="status">Some BPMN behavior is outside the current comparison coverage. Zero detected changes does not prove the process is unchanged. Review the limitations below.</p>}
      {report.warnings.length > 0 && (
        <div className={styles.warnings}><strong>Analysis warnings</strong>{report.warnings.map((warning) => <p key={`${warning.code}-${warning.message}`}><b>{warning.code}</b> {warning.message}</p>)}</div>
      )}
      {!!report.source_analysis?.length && <details className={styles.warnings}>
        <summary>Source analysis coverage ({report.source_analysis.length} files)</summary>
        <p>Text-only files support lexical retrieval. Static calls cover a limited subset of Go; these capabilities do not measure accuracy.</p>
        {report.source_analysis.map((file) => <p key={file.file}><b>{file.language}</b> {file.file} — {file.capabilities.join(", ")}{file.fallback ? " (parser fallback)" : ""}</p>)}
      </details>}
      <section className={styles.resultTools} aria-label="Result display">
        <div role="group" aria-label="Filter results">{(["all", "code", "tests", "docs"] as const).map(value => <button type="button" aria-pressed={category === value} onClick={() => setCategory(value)} key={value}>{({all:"All", code:"Code", tests:"Tests", docs:"Docs"})[value]}</button>)}</div>
        <div role="group" aria-label="Group results"><button type="button" aria-pressed={grouping === "change"} onClick={() => setGrouping("change")}>By BPMN change</button><button type="button" aria-pressed={grouping === "file"} onClick={() => setGrouping("file")}>By file</button></div>
        <p role="status">Showing {visibleCount} of {candidateCount} matches across {groups.length} {groups.length === 1 ? "file" : "files"}. Filters change the view only; exports keep the full report.</p>
      </section>
      <div className={styles.changeList}>
        {grouping === "change" ? visibleChanges.map(({ change, index }) => <ChangeCard category={category} method={report.method} change={change} index={index} supportingFlows={report.changes.filter(item => item.part_of === (change.new_element?.id ?? change.old_element?.id))} key={`${change.change_type}-${change.new_element?.id ?? change.old_element?.id}-${index}`} />) : groups.map(group => <article className={styles.change} key={group.file}>
          <header className={styles.fileHeader}><h3>{group.file}</h3><p>{new Set(group.entries.map(entry => entry.changeIndex)).size} BPMN changes · {group.entries.length} matches. Positions refer to each original change ranking.</p></header>
          {Array.from(new Set(group.entries.map(entry => entry.changeIndex))).map(changeIndex => {
            const change = report.changes[changeIndex];
            const element = change.new_element ?? change.old_element;
            return <section key={changeIndex} aria-label={`Related change ${changeIndex + 1}`}><p className={styles.fileChange}>Change {changeIndex + 1}: {change.change_type.replaceAll("_", " ")} · {element?.name || element?.id}</p>
              {group.entries.filter(entry => entry.changeIndex === changeIndex).map((entry, index) => <ImpactRow method={report.method} changeType={change.change_type} impact={entry.impact} rank={entry.rank} key={index} />)}
            </section>;
          })}
        </article>)}
        {grouping === "file" && groups.length === 0 && <p className={styles.noImpact}>No files match this filter. Choose All to see the complete result.</p>}
        {grouping === "file" && report.changes.some(change => !change.part_of && (change.unresolved_impacts?.length || !change.impacts.length)) && <section className={styles.warnings} aria-label="Unresolved BPMN changes"><strong>Unresolved BPMN changes</strong>{report.changes.map((change, index) => !change.part_of && (change.unresolved_impacts?.length || !change.impacts.length) ? <p key={index}>Change {index + 1}: {(change.unresolved_impacts?.length ? change.unresolved_impacts : ["No candidate found"]).join("; ")}</p> : null)}</section>}
      </div>
      <footer className={styles.fingerprint}>Analysis ID <code>{report.analysis_id}</code><span>Input fingerprint <code>{report.input_fingerprint}</code></span></footer>
    </div>
  );
}

function Metric({ value, label, tone }: { value: number; label: string; tone?: "warn" | "ok" }) {
  return <div className={`${styles.metric} ${tone ? styles[tone] : ""}`}><b>{value}</b><span>{label}</span></div>;
}

function ChangeCard({ change, index, method, category, supportingFlows }: { change: Change; index: number; method: AnalysisMethod; category: ResultCategory; supportingFlows: Change[] }) {
  const element = change.new_element ?? change.old_element;
  const visible = change.impacts.map((impact, index) => ({ impact, rank: index + 1 })).filter(entry => matchesCategory(entry.impact, category));
  return (
    <article className={styles.change}>
      <div className={styles.changeHead}>
        <span className={styles.index}>{String(index + 1).padStart(2, "0")}</span>
        <div><div className={styles.changeType}>{change.change_type.replaceAll("_", " ")}</div><h3>{element?.name || element?.id || "Unresolved element"}</h3><p>{element?.id} · {element?.type}{element?.actor ? ` · ${element.actor}` : ""}</p></div>
        <span className={styles.impactCount}>{visible.length} / {change.impacts.length} matches</span>
      </div>
      {change.reason && <p className={styles.reason}>{change.reason}</p>}
		{change.context && <div className={styles.context}><b>{change.context.pattern.replaceAll("_", " ")}</b><span>{contextSummary(change)}</span></div>}
      <details className={styles.contextDetails}>
        <summary>BPMN context recorded for this change</summary>
        {change.context ? <>
          <dl>
            <div><dt>Process</dt><dd>{change.context.process_name || "Not recorded"}</dd></div>
            <div><dt>Directly before</dt><dd>{change.context.predecessors?.map(item => item.name || item.id).join(", ") || "Not recorded"}</dd></div>
            <div><dt>Directly after</dt><dd>{change.context.successors?.map(item => item.name || item.id).join(", ") || "Not recorded"}</dd></div>
            <div><dt>Branch conditions / labels</dt><dd>{change.context.branch_terms?.join("; ") || "Not recorded"}</dd></div>
            <div><dt>Related flow IDs</dt><dd>{change.context.related_flow_ids?.join(", ") || "Not recorded"}</dd></div>
          </dl>
          {!!change.context.branch_changes?.length && <div>{change.context.branch_changes.map(branch => <div key={branch.flow_id}><b>{branch.flow_id}</b><p>Before: {branch.old_condition || branch.old_label || "No branch condition"}</p><p>After: {branch.new_condition || branch.new_label || "No branch condition"}</p></div>)}</div>}
          <p>These are recorded BPMN relationships, not an extra score. Hybrid BM25 uses available process, neighbour and branch text. Semantic builds its query separately; for a named added/removed task it uses the task meaning, selected supporting text and available process name rather than copying every neighbour label. Flow IDs describe structure, not extra ranking weights.</p>
        </> : <p>No structural context was recorded for this change. Do not assume that neighbouring tasks were used. The element text may still provide a search query.</p>}
      </details>
      {supportingFlows.length > 0 && <div className={styles.context}><b>FLOW DETAILS</b><span>{supportingFlows.length} related flow edit{supportingFlows.length === 1 ? "" : "s"}: {supportingFlows.map(item => item.new_element?.id ?? item.old_element?.id).join(", ")}. File candidates are listed once under this BPMN change.</span></div>}
      {visible.length > 0 ? <div className={styles.impacts}>{visible.map(({impact, rank}) => <ImpactRow method={method} changeType={change.change_type} impact={impact} rank={rank} key={`${impact.file}-${impact.symbol}-${rank}`} />)}</div> : <p className={styles.noImpact}>{change.impacts.length ? "No matches in this category. Choose All to see the other results." : "No evidence-bearing candidate crossed the retrieval threshold."}</p>}
      {!!change.related_files?.length && <details className={styles.structural}>
        <summary>Related source connections ({change.related_files.length})</summary>
        <p>Possible callers found through indexed source connections. These connections add no retrieval score and are not extra ranked predictions for Precision/Recall. A connection does not prove a change is required.</p>
        {change.related_files.map(file => <section key={file.file} aria-label={`Related file ${file.file}`}>
          <p><b>{file.file}</b> · {file.module} / {file.layer} · VERIFY · {file.ranked_position ? `Also in ranked list at ${file.ranked_position}` : "Not ranked"}</p>
          <p>Connected to {file.seed_function} in {file.seed_file} through {file.hops} source connections.</p>
          <details><summary>Show connection evidence</summary>{file.evidence.map((e, index) => <div key={index}><p><b>{e.type}</b> {e.location}</p><pre>{e.excerpt}</pre></div>)}</details>
        </section>)}
      </details>}
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

function ImpactRow({ impact, rank, method, changeType }: { impact: Impact; rank: number; method: AnalysisMethod; changeType: string }) {
  const trace = traceExplanation(impact, method, changeType);
  return (
		<details className={styles.impact} data-testid="impact-details">
      <summary>
        <span className={styles.rank}>{String(rank).padStart(2, "0")}</span>
        <span className={`${styles.action} ${styles[impact.required_action.toLowerCase()] ?? ""}`}>{impact.required_action}</span>
				<span className={styles.target}><b>{impact.file}</b><small>{impact.symbol || "whole file"} · {impact.module} / {impact.layer}</small></span>
        <span className={styles.confidence}><b>{impact.confidence.toFixed(3)}</b><small>{hasFusion(impact) ? "rank agreement" : "retrieval score"}</small></span>
        <span className={styles.chevron}>⌄</span>
      </summary>
      <div className={styles.impactBody}>
        <div><h4>Why this file or function?</h4>
          <div className={styles.traceStatus}><b>{trace.status}</b><p>{trace.detail}</p></div>
          <ul className={styles.reasons}>{plainReasons(impact).map((reason) => <li key={reason}>{reason}</li>)}</ul>
          <details className={styles.technical}><summary>Show technical evidence</summary>{impact.evidence.map((evidence, index) => <div className={styles.evidence} key={`${evidence.type}-${index}`}><b>{evidenceLabel(evidence.type)}</b><code>{evidence.location}</code><p>{evidence.excerpt}</p></div>)}</details>
        </div>
        <div><h4>How the result was ranked</h4>
          {method === "hybrid" && <p className={styles.scoreSummary}>{usesCodeEvidence(impact, method) ? "This saved report used the earlier six-signal code-evidence ranking. Semantic ranked separately; RRF combined their positions." : "BM25 uses the changed BPMN element and short process context; Semantic ranks by meaning. RRF combines their positions equally. IDs, names, paths, and code patterns explain candidates but do not add ranking weights."}</p>}
          {method === "fusion" && <p className={styles.scoreSummary}>BM25 and Semantic each rank files independently. RRF combines their ranking positions equally, not their raw scores.</p>}
          <dl className={styles.scoreCards}>
            <div><dt>{usesCodeEvidence(impact, method) ? "Code evidence" : "BM25 word match"}</dt><dd>{usesCodeEvidence(impact, method) ? impact.score_breakdown.evidence_score?.toFixed(3) : impact.evidence.some(item => item.type === "BM25") ? impact.score_breakdown.bm25?.toFixed(3) ?? "Not scored" : "Not scored"}</dd></div>
            <div><dt>Semantic similarity</dt><dd>{impact.evidence.some(item => item.type === "SEMANTIC_SIMILARITY") ? impact.score_breakdown.semantic?.toFixed(3) ?? "Not scored" : "Not scored"}</dd></div>
            <div><dt>Rank agreement</dt><dd>{hasFusion(impact) ? impact.score_breakdown.rrf?.toFixed(3) ?? impact.confidence.toFixed(3) : "Not combined"}</dd></div>
          </dl>
          <p className={styles.scoreSummary}>{rankingExplanation(impact)}</p>
          <details className={styles.technical}><summary>How these numbers are calculated</summary>
            <p>BM25 scores matching query words using their frequency, rarity across indexed documents, and document length. The score is divided by the highest BM25 score for this query; 1 means the highest word-match score, not 100% accuracy.</p>
            <p>Semantic similarity compares the embedding vectors of the query and indexed source passages using cosine similarity: dot(query, passage) / (length(query) × length(passage)). The search keeps the best passage match for each candidate. A score such as 0.4 is not 40% accuracy and is not directly comparable with BM25.</p>
            {fusionCalculation(impact) && <p>{fusionCalculation(impact)}</p>}
            {usesCodeEvidence(impact, method) && <p>This older report uses the weighted code-evidence score for the left ranking; its BM25 value is one of those signals.</p>}
            <p>Technical evidence lists the recorded source locations and excerpts. It explains why a candidate was retrieved, not proof that it must change. Precision, Recall and MAP require a separate reference list.</p>
          </details>
          <details className={styles.technical}><summary>Show search signals</summary><ScoreTable impact={impact} method={method} />
            <p>{usesCodeEvidence(impact, method) ? "These are legacy weighted search signals. Missing matches do not prove that a file is unrelated." : "BM25 and Semantic are separate search scores. IDs, function names, paths, and code patterns appear under technical evidence only; they do not subtract from either score."}</p>
            {usesCodeEvidence(impact, method) && <details><summary>Weights used by code-evidence search</summary><p>Changed-element ID 45%, function/class/component name 15%, file path 10%, query words 10%, code clues 10%, BM25 10%. These initial research weights order the code-evidence list only. A missing direct ID contributes zero; other signals are not rescaled. Neighbouring IDs add context evidence, not the 45% direct-ID contribution.</p>{hasFusion(impact) && <p>The final score combines the code-evidence and semantic positions using RRF (k=60). It does not use this weighted sum as a probability.</p>}</details>}
          </details>
        </div>
      </div>
    </details>
  );
}

function plainReasons(impact: Impact): string[] {
	const types = new Set(impact.evidence.map((item) => item.type));
	const reasons: string[] = [];
	if (types.has("SYMBOL_SCOPE_UNRESOLVED")) reasons.push("This file is a candidate, but the evidence does not identify one specific function yet.");
	if (types.has("SYMBOL") && impact.symbol) reasons.push(`The name ${impact.symbol} shares words with the search query.`);
	if (types.has("LABEL_TOKEN") || types.has("BM25")) reasons.push("Contains words from the BPMN change or its surrounding process.");
	if (types.has("PATH")) reasons.push("Located in a folder or module related to this process.");
	if (types.has("DATABASE_TABLE") || types.has("STATE_STATUS") || types.has("AUDIT_ACTION")) reasons.push("Text patterns suggest a data, status, or audit clue. Check the excerpt; this does not prove a database dependency.");
	if (types.has("SEMANTIC_SIMILARITY")) reasons.push("The embedding model found a similar passage; check whether its meaning fits this change.");
	return reasons.length > 0 ? reasons.slice(0, 4) : ["Returned by the selected retrieval method; inspect the technical evidence before deciding."];
}

function evidenceLabel(type: string): string {
	const labels: Record<string, string> = {
		TRACE_ID: "changed-element BPMN ID", CONTEXT_TRACE_ID: "neighbouring BPMN ID (context only)", SYMBOL: "function / class / code name", PATH: "related file path",
		LABEL_TOKEN: "matching BPMN words", BM25: "exact-word match", SEMANTIC_SIMILARITY: "semantic similarity",
		RRF_FUSION: "search positions", DATABASE_TABLE: "possible data clue (text pattern)", STATE_STATUS: "possible status clue", AUDIT_ACTION: "possible audit clue",
		SYMBOL_SCOPE_UNRESOLVED: "function not established",
	};
	return labels[type] ?? type;
}

function ScoreTable({ impact, method }: { impact: Impact; method: AnalysisMethod }) {
  const scores = impact.score_breakdown;
  const active = usesCodeEvidence(impact, method) ? ["symbol", "path", "lexical", "structured", "bm25"] : method === "bm25" ? ["bm25"] : method === "fusion" || method === "hybrid" ? ["bm25", "semantic"] : ["semantic", "dependency"];
  const entries = Object.entries(scores).filter(([name, value]) => active.includes(name) && (!(name === "semantic" || name === "dependency") || value > 0)) as [keyof ScoreBreakdown, number][];
  return <dl className={styles.signalList}>{usesCodeEvidence(impact, method) && <div><dt>Changed-element ID</dt><dd>{scores.trace_id > 0 ? scores.trace_id.toFixed(3) : "No direct match"}</dd></div>}{entries.map(([name, value]) => <div key={name}><dt>{scoreLabel(name)}</dt><dd>{name === "symbol" && !impact.symbol ? "No name extracted" : value === 0 ? "No contribution" : value.toFixed(3)}</dd></div>)}</dl>;
}

function scoreLabel(name: keyof ScoreBreakdown): string {
	const labels: Partial<Record<keyof ScoreBreakdown, string>> = {
		trace_id: "task ID", symbol: "function / class / component", path: "file path",
		lexical: "query words", structured: "code clues", bm25: "BM25 word match",
		semantic: "semantic similarity", rrf: "ranking agreement", dependency: "code relation", total: "combined score",
	};
	return labels[name] ?? name.replaceAll("_", " ");
}
