"use client";

import { useState } from "react";
import { AnalysisForm } from "@/components/AnalysisForm";
import { BpmnEditor } from "@/components/BpmnEditor";
import { InterpretationGuide } from "@/components/InterpretationGuide";
import { ReportView } from "@/components/ReportView";
import { ComparisonView } from "@/components/ComparisonView";
import { EvaluationDashboard } from "@/components/EvaluationDashboard";
import { BlindReviewPack } from "@/components/BlindReviewPack";
import { AnalysisHistory } from "@/components/AnalysisHistory";
import { ReviewPanel } from "@/components/ReviewPanel";
import { PlanningContextView } from "@/components/PlanningContextView";
import { createAnalysis } from "@/lib/api";
import type { Report } from "@/lib/types";
import styles from "./page.module.css";

export default function Home() {
  const [report, setReport] = useState<Report | null>(null);
  const [comparison, setComparison] = useState<Report[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [view, setView] = useState<"mapping" | "canvas" | "planning" | "history" | "research" | "guide">("mapping");
  const [baseline, setBaseline] = useState<File | null>(null);
  const [edited, setEdited] = useState<File | null>(null);
  const [editorInput, setEditorInput] = useState<File | null>(null);
  const [canvasDirty, setCanvasDirty] = useState(false);

  function navigate(next: typeof view) {
    if (next === "canvas" && baseline && !editorInput) setEditorInput(edited || baseline);
    setView(next);
  }

  function changeBaseline(file: File | null) {
    setBaseline(file); setEdited(null); setEditorInput(null); setCanvasDirty(false);
  }

  function changeEdited(file: File | null) {
    setEdited(file); setEditorInput(null); setCanvasDirty(false);
  }

  async function handleSubmit(formData: FormData) {
    setLoading(true);
    setError("");
    setComparison([]);
    try {
			const compare = formData.get("compare_methods") === "on";
			formData.delete("compare_methods");
			if (!compare) {
				setReport(await createAnalysis(formData));
				return;
			}
			const methods: Report["method"][] = ["bm25", "semantic", "fusion", "hybrid"];
			const reports: Report[] = [];
			for (const method of methods) {
				const request = cloneFormData(formData);
				request.set("method", method);
				request.delete("mapping_config_path");
				reports.push(await createAnalysis(request));
			}
			if (new Set(reports.map((item) => item.input_fingerprint)).size !== 1) {
				throw new Error("Comparison reports did not share the same input fingerprint.");
			}
			setComparison(reports);
			setReport(reports.find((item) => item.method === "hybrid") ?? reports[0]);
    } catch (requestError) {
      setReport(null);
			setComparison([]);
      setError(requestError instanceof Error ? requestError.message : "Analysis failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.brandMark} aria-hidden="true"><span /><span /><span /></div>
        <div>
          <p className={styles.eyebrow}>Deterministic impact analysis</p>
          <h1>BPMN Impact Mapper</h1>
        </div>
        <div className={styles.status}><span />{loading ? "Analysis in progress" : "Local workspace"}</div>
      </header>
      <nav className={styles.navigation} aria-label="Workspace navigation">
        {([ ["mapping", "Mapping"], ["canvas", "BPMN Canvas"], ["planning", "Planning"], ["history", "Saved analyses"], ["research", "Research"], ["guide", "Guide"] ] as const).map(([id, label]) =>
          <button key={id} type="button" aria-current={view === id ? "page" : undefined} onClick={() => navigate(id)}>{label}{id === "canvas" && baseline && <span className={styles.dot} aria-hidden="true" />}</button>
        )}
      </nav>
      <div className={styles.sectionTitle}><div><h2>{view === "mapping" ? "Map process changes." : view === "canvas" ? "Shape the next process." : view === "planning" ? "Inspect planning context." : view === "history" ? "Previous analyses." : view === "research" ? "Evaluate the method." : "Understand the evidence."}</h2><p>{view === "mapping" ? "Choose a repository and compare BPMN versions. Edit your process in Canvas whenever needed." : view === "canvas" ? "Work on a copy of your baseline, then use it for impact mapping." : view === "planning" ? "Review the bounded evidence package before any model is allowed to plan." : view === "history" ? "Open a saved report without running the analysis again." : view === "research" ? "Prepare independent labels and measure retrieval without feeding answers into inference." : "How methods, evidence and scores should be interpreted."}</p></div>{view === "mapping" && report && <span className={styles.reportBadge}>{report.changes.length} changes · {report.method}</span>}</div>

      <div hidden={view !== "mapping"}>
      {canvasDirty && <div className={styles.draftNotice} role="status"><span>Canvas has unapplied edits. Mapping will use the previously selected BPMN.</span><button type="button" onClick={() => navigate("canvas")}>Review canvas draft</button></div>}
      <div className={styles.workspace}>
		<div className={styles.leftRail}>
			<AnalysisForm loading={loading} onSubmit={handleSubmit} baseline={baseline} edited={edited} onBaseline={changeBaseline} onEdited={changeEdited} onOpenEditor={() => navigate("canvas")} />
		</div>
        <section className={styles.resultRegion} aria-live="polite">
          {loading && <div className={styles.stateCard}><div className={styles.spinner} /><h3>Analyzing repository</h3><p>Parsing BPMN, indexing allowed files, and ranking evidence…</p></div>}
          {!loading && error && <div className={`${styles.stateCard} ${styles.error}`}><strong>Analysis could not be completed</strong><p>{error}</p></div>}
          {!loading && !error && !report && <div className={styles.stateCard}><div className={styles.emptyGlyph}>↗</div><h3>Your impact report will appear here</h3><p>Configure the repository and provide both BPMN versions to begin.</p></div>}
          {!loading && report && <>
				{comparison.length > 1 && <ComparisonView reports={comparison} selectedMethod={report.method} onSelect={setReport} />}
				<ReportView report={report} />
			</>}
        </section>
      </div>
      </div>
      <div hidden={view !== "canvas"}>
        {baseline && editorInput ? <BpmnEditor baseline={baseline} initial={editorInput} active={view === "canvas"} onDirty={setCanvasDirty} onClose={() => navigate("mapping")} onApply={file => { setEdited(file); navigate("mapping"); }} /> : <section className={styles.canvasEmpty}>
          <p className={styles.kicker}>Start from an existing process</p><h3>Bring your baseline BPMN.</h3><p>Upload the original process here. Its IDs and original XML are preserved while you edit a separate version.</p>
          <label className={styles.upload}>Choose baseline BPMN<input type="file" accept=".bpmn,.xml" aria-label="Canvas baseline BPMN" onChange={event => { const file = event.target.files?.[0]; if (file) { changeBaseline(file); setEditorInput(file); } }} /></label>
          <button type="button" onClick={() => navigate("mapping")}>Or configure a mapping first</button>
          <small>Drafts stay available across tabs in this session. Download them before refreshing.</small>
        </section>}
      </div>
      <div hidden={view !== "history"} className={styles.historyPage}>
        <AnalysisHistory refreshKey={report?.analysis_id ?? ""} onOpen={(saved) => { setReport(saved); setComparison([]); setError(""); navigate("mapping"); }} />
      </div>
      <div hidden={view !== "planning"} className={styles.planningPage}>
        {report ? <PlanningContextView key={report.analysis_id} analysisID={report.analysis_id} /> : <section className={styles.stateCard}><h3>No analysis selected</h3><p>Run a mapping or open a saved analysis to inspect its bounded planning context.</p><button type="button" onClick={() => navigate("mapping")}>Go to Mapping</button></section>}
      </div>
      <div hidden={view !== "research"} className={styles.researchPage}>
        {!report ? <section className={styles.stateCard}><h3>No frozen analysis selected</h3><p>Run a mapping or open a saved analysis before preparing reviewer artifacts.</p><button type="button" onClick={() => navigate("mapping")}>Go to Mapping</button></section> : <>
          <section className={styles.researchIntro}><span>Optional research workflow</span><h3>Predictions stay immutable.</h3><p>Annotation, reconciliation, and metrics are post-inference tools. They never change candidate retrieval or the canonical impact report.</p></section>
          <ReviewPanel key={`review-${report.analysis_id}`} report={report} />
          <BlindReviewPack key={`pack-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
          <EvaluationDashboard key={`evaluation-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
        </>}
      </div>
      <div hidden={view !== "guide"} className={styles.guidePage}><InterpretationGuide /><section className={styles.stateCard}><h3>One baseline, two ways to compare</h3><p>Upload both BPMN files in Mapping, or open a baseline in BPMN Canvas and edit a copy. Use edited BPMN transfers the current draft to Mapping. Changing the baseline starts a new canvas; switching tabs preserves edits and undo history.</p><p>Scores describe retrieval relevance or agreement, not certainty that a file needs changing. Inspect evidence before deciding on an action.</p></section></div>
    </main>
  );
}

function cloneFormData(source: FormData): FormData {
	const result = new FormData();
	for (const [name, value] of source.entries()) result.append(name, value);
	return result;
}
