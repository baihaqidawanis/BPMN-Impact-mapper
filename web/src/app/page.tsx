"use client";

import { useEffect, useState } from "react";
import { AnalysisForm } from "@/components/AnalysisForm";
import { BpmnEditor } from "@/components/BpmnEditor";
import { BpmnLibrary } from "@/components/BpmnLibrary";
import { InterpretationGuide } from "@/components/InterpretationGuide";
import { ReportView } from "@/components/ReportView";
import { ComparisonView } from "@/components/ComparisonView";
import { EvaluationDashboard } from "@/components/EvaluationDashboard";
import { BlindReviewPack } from "@/components/BlindReviewPack";
import { AnalysisHistory } from "@/components/AnalysisHistory";
import { ReviewPanel } from "@/components/ReviewPanel";
import { PlanningContextView } from "@/components/PlanningContextView";
import { BatchAnalysis } from "@/components/BatchAnalysis";
import { BenchmarkSummary } from "@/components/BenchmarkSummary";
import { analysisRepository, rememberAnalysisRepository } from "@/lib/analysis-repository";
import { WorkspaceShell, type WorkspaceView } from "@/components/WorkspaceShell";
import { WorkspaceOverview } from "@/components/WorkspaceOverview";
import { runAnalysis } from "@/features/analysis/run-analysis";
import { createAnalysis } from "@/lib/api";
import { RND_ENABLED } from "@/lib/features";
import { useSavedAnalysis } from "@/lib/useSavedAnalysis";
import type { Report } from "@/lib/types";
import styles from "./page.module.css";

export default function Home() {
  const { report, setReport, restoring, restoreNotice } = useSavedAnalysis();
  const [inputGeneration,setInputGeneration]=useState(0);
  useEffect(()=>{
    const query=new URL(window.location.href).searchParams;
    const target=query.get("view");
    if(target==="analysis")setView("mapping");
    else if(target && ["overview","mapping","canvas","library","history","research","guide"].includes(target))setView(target as WorkspaceView);
    else if(query.has("analysis"))setView("mapping");
  },[]);
  const [comparison, setComparison] = useState<Report[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [batchRunning, setBatchRunning] = useState(false);
  const [selectedView, setView] = useState<WorkspaceView>("overview");
  const view = selectedView === "research" && !RND_ENABLED ? "mapping" : selectedView;
  const [baseline, setBaseline] = useState<File | null>(null);
  const [edited, setEdited] = useState<File | null>(null);
  const [editorInput, setEditorInput] = useState<File | null>(null);
  const [canvasDirty, setCanvasDirty] = useState(false);

  const [repositoryPath, setRepositoryPath] = useState("");
  const [reportRepositoryPath, setReportRepositoryPath] = useState("");
  const [workspaceStage, setWorkspaceStage] = useState<"mapping" | "planning">("mapping");
  const [planningAnalysis, setPlanningAnalysis] = useState("");
  useEffect(() => {
    const path = report ? analysisRepository(report.analysis_id) : "";
    setReportRepositoryPath(path);
    if (report) setRepositoryPath(path);
    document.getElementById("analysis-inputs")?.scrollTo({ top: 0 });
  }, [report?.analysis_id]);

  function workflow(stage: "mapping" | "planning") {
    if (stage === "planning" && !report) return;
    setWorkspaceStage(stage);
    if (stage === "planning" && report) setPlanningAnalysis(report.analysis_id);
    requestAnimationFrame(() => document.getElementById("analysis-workflow")?.scrollIntoView({ block: "start" }));
  }
  function repositoryForReport(path: string) {
    if (report) rememberAnalysisRepository(report.analysis_id, path);
    setReportRepositoryPath(path);
    setRepositoryPath(path);
  }

  function navigate(next: typeof view) {
    if (next === "mapping") setWorkspaceStage("mapping");
    if (next === "research" && !RND_ENABLED) return;
    if (next === "canvas" && baseline && !editorInput) setEditorInput(edited || baseline);
    setView(next);
    requestAnimationFrame(()=>window.scrollTo(0,0));
    const url=new URL(window.location.href);
    url.searchParams.set("view",next==="mapping"?"analysis":next);
    window.history.replaceState(window.history.state,"",url);
  }

  function changeBaseline(file: File | null) {
    setBaseline(file); setEdited(null); setEditorInput(null); setCanvasDirty(false);
  }

  function changeEdited(file: File | null) {
    setEdited(file); setEditorInput(null); setCanvasDirty(false);
  }

  async function handleSubmit(formData: FormData) {
    setReport(null);
    setWorkspaceStage("mapping");
    setPlanningAnalysis("");
    setLoading(true);
    setError("");
    setComparison([]);
    try {
      const result=await runAnalysis(formData,createAnalysis,(saved,path)=>rememberAnalysisRepository(saved.analysis_id,path));
      setComparison(result.comparison);
      setReport(result.report);
    } catch (requestError) {
      setReport(null);
			setComparison([]);
      setError(requestError instanceof Error ? requestError.message : "Analysis failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <WorkspaceShell view={view} onNavigate={navigate} research={RND_ENABLED} busy={loading||batchRunning}>
      <div hidden={view!=="overview"}><WorkspaceOverview active={view==="overview"} current={report} onStart={()=>{
        if(loading||batchRunning)return;
        if(canvasDirty&&!window.confirm("Start a new analysis and discard the current canvas draft? Download it first if needed."))return;
        setReport(null);setComparison([]);setError("");changeBaseline(null);setRepositoryPath("");setPlanningAnalysis("");setInputGeneration(v=>v+1);navigate("mapping");
      }} onContinue={()=>navigate("mapping")} onOpen={saved=>{setReport(saved);setComparison([]);setError("");navigate("mapping");}} onLibrary={()=>navigate("library")} onHistory={()=>navigate("history")}/></div>
      <div className={styles.sectionTitle} hidden={view==="overview"}><div><h1>{view === "mapping" ? "Analysis workspace" : view === "canvas" ? "BPMN comparison" : view === "library" ? "BPMN library" : view === "history" ? "Saved analyses" : view === "research" ? "Method evaluation" : "Reading the report"}</h1><p>{view === "mapping" ? "Compare BPMN versions, inspect affected files, then prepare a change plan." : view === "canvas" ? "Work on a copy of your baseline, then use it for impact mapping." : view === "library" ? "Keep process versions locally and reuse them in Mapping or Canvas." : view === "history" ? "Open a saved report without running the analysis again." : view === "research" ? "Compare saved test results, run scenarios, and check mapping against expected files." : "How methods, evidence and scores should be interpreted."}</p></div>{view === "mapping" && report && <span className={styles.reportBadge}>{report.changes.length} changes  |  {report.method}</span>}</div>

      <div hidden={view !== "mapping"}>
      {restoreNotice && <div className={styles.draftNotice} role="status"><span>{restoreNotice}</span><button type="button" onClick={() => navigate("history")}>Saved analyses</button></div>}
      {canvasDirty && <div className={styles.draftNotice} role="status"><span>Canvas has unapplied edits. Mapping will use the previously selected BPMN.</span><button type="button" onClick={() => navigate("canvas")}>Review canvas draft</button></div>}
      <div className={styles.workspace}>
		<div id="analysis-inputs" className={styles.leftRail}>
			<AnalysisForm key={inputGeneration} completedRepository={report?.repository} repositoryPath={repositoryPath} onRepositoryPath={setRepositoryPath} enableResearch={RND_ENABLED} loading={loading || batchRunning} onSubmit={handleSubmit} baseline={baseline} edited={edited} onBaseline={changeBaseline} onEdited={changeEdited} onOpenEditor={() => navigate("canvas")} />
		</div>
        <section className={styles.resultRegion} aria-label="Analysis results">
          <p className={styles.srOnly} role="status">{loading ? "Analyzing repository." : report ? `${report.changes.length} BPMN changes ready for review.` : ""}</p>
          <div className={styles.workflowBar} id="analysis-workflow">
            <div role="group" aria-label="Analysis workflow" className={styles.workflowSteps}>
              <button type="button" aria-pressed={workspaceStage === "mapping"} onClick={() => workflow("mapping")}>Mapping</button>
              <button type="button" aria-pressed={false} onClick={()=>navigate("canvas")}>Canvas</button>
              <button type="button" aria-pressed={workspaceStage === "planning"} disabled={!report || loading} onClick={() => workflow("planning")}>Planning</button>
            </div>
            <span className={styles.workflowRepository}>{report?.repository.name || "No analysis selected"}</span>
          </div>
          <div id="mapping-results" hidden={workspaceStage !== "mapping"}>
          {restoring && !loading && <div className={styles.stateCard} role="status"><h3>Reopening saved report</h3><p>Loading the completed analysis. Mapping is not being run again.</p></div>}
          {loading && <div className={styles.stateCard}><div className={styles.spinner} /><h3>Analyzing repository</h3><p>Parsing BPMN, indexing allowed files, and ranking evidence...</p></div>}
          {!loading && error && <div className={`${styles.stateCard} ${styles.error}`}><strong>Analysis could not be completed</strong><p>{error}</p></div>}
          {!restoring && !loading && !error && !report && <div className={styles.stateCard}><span className={styles.kicker}>No analysis yet</span><h3>Your impact report will appear here</h3><p>Configure the repository and provide both BPMN versions to begin.</p></div>}
          {!loading && report && <>
				{comparison.length > 1 && <ComparisonView reports={comparison} selectedMethod={report.method} onSelect={setReport} />}
				<ReportView report={report} />
			</>}
          </div>
          <div id="planning-results" hidden={workspaceStage !== "planning"}>
            {report && planningAnalysis === report.analysis_id && <PlanningContextView key={report.analysis_id} analysisID={report.analysis_id} repositoryPath={reportRepositoryPath} onRepositoryPath={repositoryForReport} />}
          </div>
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
      <div hidden={view !== "library"}>
        <BpmnLibrary active={view === "library"} baseline={baseline} edited={edited} hasBaseline={!!baseline} onUse={(file, target) => {
          if (canvasDirty && !window.confirm("Replace the current canvas draft? Download it first if you need to keep it.")) return;
          if (target === "baseline") changeBaseline(file); else changeEdited(file);
          navigate("mapping");
        }} />
      </div>
      <div hidden={view !== "history"} className={styles.historyPage}>
        <AnalysisHistory refreshKey={report?.analysis_id ?? ""} onOpen={(saved) => { setReport(saved); setWorkspaceStage("mapping"); setComparison([]); setError(""); navigate("mapping"); }} />
      </div>
      {RND_ENABLED && <div hidden={view !== "research"} className={styles.researchPage}>
        <BenchmarkSummary />
        <BatchAnalysis busy={loading} onBusy={setBatchRunning} onOpen={reports=>{setReport(reports.find(r=>r.method==="hybrid")??reports[0]);setWorkspaceStage("mapping");setComparison(reports);setError("");navigate("mapping");}} />
        {!report ? <section className={styles.stateCard}><h3>Choose a report to evaluate</h3><p>Run a mapping or open a saved analysis to check it against expected files.</p><button type="button" onClick={() => navigate("mapping")}>Go to Mapping</button></section> : <>
          <section className={styles.researchIntro}><span>Check the results</span><h3>Compare predictions with expected files</h3><p>A reference lists the files expected to be affected. Evaluation checks how many the mapper found and how high it ranked them. It does not change the original mapping report.</p></section>
          <ReviewPanel key={`review-${report.analysis_id}`} report={report} />
          <BlindReviewPack key={`pack-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
          <EvaluationDashboard key={`evaluation-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
        </>}
      </div>}
      <div hidden={view !== "guide"} className={styles.guidePage}><InterpretationGuide /><section className={styles.stateCard}><h3>One baseline, two ways to compare</h3><p>Upload both BPMN files in Mapping, or open a baseline in BPMN Canvas and edit a copy. Canvas shows the original read-only above the editable new version; color outlines are a quick visual preview, while the backend report is the authoritative diff. Expand the editor for precise edits. Use edited BPMN transfers the current draft to Mapping. Changing the baseline starts a new canvas; switching tabs preserves edits and undo history.</p><p>Scores describe retrieval relevance or agreement, not certainty that a file needs changing. Inspect evidence before deciding on an action.</p></section></div>
    </WorkspaceShell>
  );
}
