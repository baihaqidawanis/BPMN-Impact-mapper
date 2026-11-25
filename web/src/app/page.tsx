"use client";
import * as Tabs from "@radix-ui/react-tabs";

import { MappingEmptyState } from "@/components/MappingEmptyState";

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
import { PlanRubricView } from "@/components/PlanRubricView";
import { analysisRepository, rememberAnalysisRepository } from "@/lib/analysis-repository";
import { WorkspaceShell, type WorkspaceView } from "@/components/WorkspaceShell";
import { WorkspaceOverview } from "@/components/WorkspaceOverview";
import { runAnalysis } from "@/features/analysis/run-analysis";
import { createAnalysis, getAnalysisBPMN } from "@/lib/api";
import { RND_ENABLED } from "@/lib/features";
import { useSavedAnalysis } from "@/lib/useSavedAnalysis";
import type { Report } from "@/lib/types";
import styles from "./page.module.css";

export default function Home() {
  const { report, setReport, restoring, restoreNotice } = useSavedAnalysis();
  const [inputGeneration, setInputGeneration] = useState(0);
  useEffect(() => {
    const query = new URL(window.location.href).searchParams;
    const target = query.get("view");
    if (target === "analysis") setView("mapping");
    else if (target && ["overview", "mapping", "canvas", "library", "history", "research", "guide"].includes(target)) setView(target as WorkspaceView);
    else if (query.has("analysis")) setView("mapping");
  }, []);
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
  const [bpmnNotice, setBpmnNotice] = useState("");
  useEffect(() => {
    if (!report) return;
    const controller = new AbortController();
    setBpmnNotice("");
    void getAnalysisBPMN(report.analysis_id, controller.signal)
      .then(pair => {
        if (controller.signal.aborted) return;
        if (!pair) {
          setBpmnNotice("This older report has no saved BPMN files. Upload both versions and run analysis again to restore Canvas and complete process context.");
          return;
        }
        const before = new File([pair.before_xml], "baseline.bpmn", { type: "application/xml" });
        const after = new File([pair.after_xml], "changed-process.bpmn", { type: "application/xml" });
        setBaseline(before);
        setEdited(after);
        setEditorInput(after);
        setCanvasDirty(false);
      })
      .catch(() => {
        if (!controller.signal.aborted) setBpmnNotice("Saved BPMN files could not be restored. Check the backend connection.");
      });
    return () => controller.abort();
  }, [report?.analysis_id]);

  const [repositoryPath, setRepositoryPath] = useState("");
  const [reportRepositoryPath, setReportRepositoryPath] = useState("");
  const [workspaceStage, setWorkspaceStage] = useState<"mapping" | "planning">("mapping");
  const [setupOpen, setSetupOpen] = useState(true);
  useEffect(() => { setSetupOpen(!report); }, [report?.analysis_id]);
  useEffect(() => { if (error) setSetupOpen(true); }, [error]);
  const [planningAnalysis, setPlanningAnalysis] = useState("");
  useEffect(() => {
    if (!report) return;
    const query = new URLSearchParams(window.location.search);
    if (query.get("analysis") === report.analysis_id && /^[a-f0-9]{32}$/.test(query.get("agentSession") ?? "")) {
      setWorkspaceStage("planning");
      setPlanningAnalysis(report.analysis_id);
    }
  }, [report?.analysis_id]);
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
    if (view !== "mapping") navigate("mapping", stage);
    requestAnimationFrame(() => window.scrollTo({ top: 0, behavior: "instant" }));
  }
  function repositoryForReport(path: string) {
    if (report) rememberAnalysisRepository(report.analysis_id, path);
    setReportRepositoryPath(path);
    setRepositoryPath(path);
  }

  function navigate(next: typeof view, stage: "mapping" | "planning" = "mapping") {
    if (next === "mapping") setWorkspaceStage(stage);
    if (next === "research" && !RND_ENABLED) return;
    if (next === "canvas" && baseline && !editorInput) setEditorInput(edited || baseline);
    setView(next);
    requestAnimationFrame(() => window.scrollTo(0, 0));
    const url = new URL(window.location.href);
    url.searchParams.set("view", next === "mapping" ? "analysis" : next);
    window.history.replaceState(window.history.state, "", url);
  }

  function changeBaseline(file: File | null) {
    setBaseline(file);
    setEdited(null);
    setEditorInput(null);
    setCanvasDirty(false);
  }

  function changeEdited(file: File | null) {
    setEdited(file);
    setEditorInput(null);
    setCanvasDirty(false);
  }

  async function handleSubmit(formData: FormData) {
    setReport(null);
    setWorkspaceStage("mapping");
    setPlanningAnalysis("");
    setLoading(true);
    setError("");
    setComparison([]);
    try {
      const result = await runAnalysis(formData, createAnalysis, (saved, path) => rememberAnalysisRepository(saved.analysis_id, path));
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

  const inAnalysisFlow = view === "mapping" || view === "canvas";
  const headerCenter = inAnalysisFlow ? (
    <div role="group" aria-label="Analysis workflow" className={styles.workflowSteps} id="analysis-workflow">
      <button type="button" aria-pressed={view === "mapping" && workspaceStage === "mapping"} onClick={() => workflow("mapping")}>Mapping</button>
      <button type="button" aria-pressed={view === "canvas"} onClick={() => navigate("canvas")}>Canvas</button>
      <button type="button" aria-pressed={view === "mapping" && workspaceStage === "planning"} disabled={!report || loading} onClick={() => workflow("planning")}>Planning</button>
    </div>
  ) : undefined;

  const headerActions = view === "mapping" && (report || !setupOpen) ? (
    <button
      type="button"
      className={styles.headerActionBtn}
      aria-expanded={setupOpen && workspaceStage === "mapping"}
      aria-controls="analysis-inputs"
      disabled={loading || batchRunning}
      onClick={() => {
        if (workspaceStage !== "mapping") setWorkspaceStage("mapping");
        setSetupOpen(!setupOpen || workspaceStage !== "mapping");
      }}
    >
      {setupOpen && workspaceStage === "mapping" ? "Hide inputs" : "Edit inputs"}
    </button>
  ) : undefined;

  return (
    <WorkspaceShell
      view={view}
      onNavigate={navigate}
      research={RND_ENABLED}
      busy={loading || batchRunning}
      breadcrumbDetail={inAnalysisFlow ? report?.repository.name : undefined}
      headerCenter={headerCenter}
      headerActions={headerActions}
    >
      <div hidden={view !== "overview"}>
        <WorkspaceOverview
          active={view === "overview"}
          current={report}
          onStart={() => {
            if (loading || batchRunning) return;
            if (canvasDirty && !window.confirm("Start a new analysis and discard the current canvas draft? Download it first if needed.")) return;
            setReport(null);
            setComparison([]);
            setError("");
            changeBaseline(null);
            setRepositoryPath("");
            setPlanningAnalysis("");
            setInputGeneration(v => v + 1);
            navigate("mapping");
          }}
          onContinue={() => navigate("mapping")}
          onOpen={saved => {
            setReport(saved);
            setComparison([]);
            setError("");
            navigate("mapping");
          }}
          onLibrary={() => navigate("library")}
          onHistory={() => navigate("history")}
        />
      </div>

      {view !== "overview" && view !== "mapping" && view !== "canvas" && (
        <div className={styles.sectionTitle}>
          <h1>
            {view === "library"
              ? "BPMN library"
              : view === "history"
                ? "Saved analyses"
                : view === "research"
                  ? "Method evaluation"
                  : "Guide"}
          </h1>
        </div>
      )}

      {bpmnNotice && (view === "mapping" || view === "canvas") && (
        <p role="status" className={styles.draftNotice}>{bpmnNotice}</p>
      )}

      <div hidden={view !== "mapping"} className={styles.mappingStage}>
        {restoreNotice && (
          <div className={styles.draftNotice} role="status">
            <span>{restoreNotice}</span>
            <button type="button" onClick={() => navigate("history")}>Saved analyses</button>
          </div>
        )}
        {canvasDirty && (
          <div className={styles.draftNotice} role="status">
            <span>Canvas has unapplied edits. Mapping will use the previously selected BPMN.</span>
            <button type="button" onClick={() => navigate("canvas")}>Review canvas draft</button>
          </div>
        )}
        <div className={styles.workspace}>
          <div id="analysis-inputs" className={styles.leftRail} hidden={!setupOpen || workspaceStage !== "mapping"}>
            <AnalysisForm
              key={inputGeneration}
              completedRepository={report?.repository}
              repositoryPath={repositoryPath}
              onRepositoryPath={setRepositoryPath}
              enableResearch={RND_ENABLED}
              loading={loading || batchRunning}
              onSubmit={handleSubmit}
              baseline={baseline}
              edited={edited}
              onBaseline={changeBaseline}
              onEdited={changeEdited}
              onOpenEditor={() => navigate("canvas")}
            />
          </div>
          <section className={styles.resultRegion} aria-label="Analysis results">
            <p className={styles.srOnly} role="status">
              {loading ? "Analyzing repository." : report ? `${report.changes.length} BPMN changes ready for review.` : ""}
            </p>
            <div id="mapping-results" hidden={workspaceStage !== "mapping"}>
              {restoring && !loading && (
                <div className={styles.stateCard} role="status">
                  <h3>Reopening saved report</h3>
                </div>
              )}
              {loading && (
                <div className={styles.stateCard}>
                  <div className={styles.spinner} />
                  <h3>Analyzing repository</h3>
                </div>
              )}
              {!loading && error && (
                <div className={`${styles.stateCard} ${styles.error}`}>
                  <strong>Analysis could not be completed</strong>
                  <p>{error}</p>
                </div>
              )}
              {!restoring && !loading && !error && !report && (
                <MappingEmptyState repositoryPath={repositoryPath} baselineName={baseline?.name} editedName={edited?.name} />
              )}
              {!loading && report && (
                <>
                  {comparison.length > 1 && <ComparisonView reports={comparison} selectedMethod={report.method} onSelect={setReport} />}
                  <ReportView report={report} onOpenPlanning={() => workflow("planning")} />
                </>
              )}
            </div>
            <div id="planning-results" hidden={workspaceStage !== "planning"}>
              {report && planningAnalysis === report.analysis_id && (
                <PlanningContextView
                  key={report.analysis_id}
                  analysisID={report.analysis_id}
                  repositoryPath={reportRepositoryPath}
                  onRepositoryPath={repositoryForReport}
                />
              )}
            </div>
          </section>
        </div>
      </div>

      <div hidden={view !== "canvas"}>
        {baseline && editorInput ? (
          <BpmnEditor
            baseline={baseline}
            initial={editorInput}
            active={view === "canvas"}
            onDirty={setCanvasDirty}
            onClose={() => navigate("mapping")}
            onApply={file => {
              setEdited(file);
              navigate("mapping");
            }}
          />
        ) : (
          <section className={styles.canvasEmpty}>
            <h3>Choose baseline BPMN</h3>
            <label className={styles.upload}>
              Choose baseline BPMN
              <input
                type="file"
                accept=".bpmn,.xml"
                aria-label="Canvas baseline BPMN"
                onChange={event => {
                  const file = event.target.files?.[0];
                  if (file) {
                    changeBaseline(file);
                    setEditorInput(file);
                  }
                }}
              />
            </label>
            <button type="button" onClick={() => navigate("mapping")}>Or configure a mapping first</button>
            <small>Drafts stay available across tabs in this session. Download them before refreshing.</small>
          </section>
        )}
      </div>

      <div hidden={view !== "library"}>
        <BpmnLibrary
          active={view === "library"}
          baseline={baseline}
          edited={edited}
          hasBaseline={!!baseline}
          onUse={(file, target) => {
            if (canvasDirty && !window.confirm("Replace the current canvas draft? Download it first if you need to keep it.")) return;
            if (target === "baseline") changeBaseline(file);
            else changeEdited(file);
            navigate("mapping");
          }}
        />
      </div>

      <div hidden={view !== "history"} className={styles.historyPage}>
        <AnalysisHistory
          refreshKey={report?.analysis_id ?? ""}
          onOpen={saved => {
            setReport(saved);
            setWorkspaceStage("mapping");
            setComparison([]);
            setError("");
            navigate("mapping");
          }}
        />
      </div>

      {RND_ENABLED && (
        <div hidden={view !== "research"} className={styles.researchPage}>
          <Tabs.Root defaultValue="benchmarks">
            <Tabs.List className={styles.researchNav} aria-label="Research tools">
              <Tabs.Trigger value="benchmarks">Benchmarks</Tabs.Trigger>
              <Tabs.Trigger value="experiments">Run experiments</Tabs.Trigger>
              <Tabs.Trigger value="evaluation">Evaluate a report</Tabs.Trigger>
              <Tabs.Trigger value="rubric">Plan rubric</Tabs.Trigger>
            </Tabs.List>
            <Tabs.Content value="benchmarks" className={styles.tabPanel} forceMount>
              <BenchmarkSummary />
            </Tabs.Content>
            <Tabs.Content value="experiments" className={styles.tabPanel} forceMount>
              <BatchAnalysis
                busy={loading}
                onBusy={setBatchRunning}
                onOpen={reports => {
                  setReport(reports.find(r => r.method === "hybrid") ?? reports[0]);
                  setWorkspaceStage("mapping");
                  setComparison(reports);
                  setError("");
                  navigate("mapping");
                }}
              />
            </Tabs.Content>
            <Tabs.Content value="evaluation" className={styles.tabPanel} forceMount>
              {!report ? (
                <section className={styles.stateCard}>
                  <h3>Choose a report to evaluate</h3>
                  <p>Run a mapping or open a saved analysis to check it against expected files.</p>
                  <button type="button" onClick={() => navigate("mapping")}>Go to Mapping</button>
                </section>
              ) : (
                <>
                  <ReviewPanel key={`review-${report.analysis_id}`} report={report} />
                  <BlindReviewPack key={`pack-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
                  <EvaluationDashboard key={`evaluation-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
                </>
              )}
            </Tabs.Content>
            <Tabs.Content value="rubric" className={styles.tabPanel} forceMount>
              <PlanRubricView />
            </Tabs.Content>
          </Tabs.Root>
        </div>
      )}

      <div hidden={view !== "guide"} className={styles.guidePage}>
        <section className={styles.guideIntro}>
          <div className={styles.guideHead}>
            <div>
              <h2>How it works</h2>
              <p>Compare two BPMN versions against a read-only local repository in three steps.</p>
            </div>
            <button type="button" className={styles.headerActionBtn} onClick={() => navigate("mapping")}>Open Analysis →</button>
          </div>
          <ol className={styles.guideSteps}>
            <li><b>1. Configure</b><span>Pick a repository and upload baseline + changed BPMN in Analysis.</span></li>
            <li><b>2. Inspect</b><span>Compare visual diffs in Canvas and review ranked code candidates in Mapping.</span></li>
            <li><b>3. Plan</b><span>Switch to Planning to generate an evidence-backed change plan.</span></li>
          </ol>
        </section>
        <InterpretationGuide flat />
      </div>
    </WorkspaceShell>
  );
}
