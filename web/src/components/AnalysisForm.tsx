"use client";

import { ChangeEvent, FormEvent, useEffect, useState } from "react";
import type { AnalysisMethod } from "@/lib/types";
import styles from "./AnalysisForm.module.css";

interface Props {
  loading: boolean;
  onSubmit: (formData: FormData) => Promise<void>;
  baseline: File | null;
  edited: File | null;
  onBaseline: (file: File | null) => void;
  onEdited: (file: File | null) => void;
  onOpenEditor: () => void;
}

export function AnalysisForm({ loading, onSubmit, baseline, edited, onBaseline, onEdited, onOpenEditor }: Props) {
  const [method, setMethod] = useState<AnalysisMethod>("hybrid");
	const [compareMethods, setCompareMethods] = useState(false);
	const [repositoryPath, setRepositoryPath] = useState("");
	const [repositoryName, setRepositoryName] = useState("");
	const [revision, setRevision] = useState("");
	const [demoFiles, setDemoFiles] = useState<{ old: File; next: File } | null>(null);
	const [demoLoading, setDemoLoading] = useState(false);
	const [demoError, setDemoError] = useState("");
	const [demoScenario, setDemoScenario] = useState("comprehensive");
	useEffect(() => {
		if (demoFiles && baseline !== demoFiles.old) setDemoFiles(null);
	}, [baseline, demoFiles]);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
		const formData = new FormData(event.currentTarget);
		for (const name of ["repo_path", "repository_name", "revision"]) {
			formData.set(name, String(formData.get(name) ?? "").trim());
		}
		if (demoFiles) {
			formData.set("old_bpmn", demoFiles.old);
			formData.set("new_bpmn", demoFiles.next);
		}
		if (edited) formData.set("new_bpmn", edited);
		if (baseline) formData.set("old_bpmn", baseline);
		void onSubmit(formData);
  }

	async function loadSyntheticDemo() {
		onEdited(null);
		setDemoLoading(true);
		setDemoError("");
		try {
			const prefix = demoScenario === "comprehensive" ? "/demo/" : `/demo/${demoScenario}-`;
			const [oldResponse, newResponse] = await Promise.all([fetch(prefix + "old.bpmn"), fetch(prefix + "new.bpmn")]);
			if (!oldResponse.ok || !newResponse.ok) throw new Error("Synthetic demo fixtures are unavailable.");
			const files = {
				old: new File([await oldResponse.blob()], "synthetic-old.bpmn", { type: "application/xml" }),
				next: new File([await newResponse.blob()], "synthetic-new.bpmn", { type: "application/xml" }),
			};
			setDemoFiles(files);
			onBaseline(files.old);
			onEdited(files.next);
			setRepositoryPath("testdata\\repository");
			setRepositoryName(demoScenario === "comprehensive" ? "Synthetic Request Workflow" : `Synthetic ${demoScenario.replaceAll("-", " ")} workflow`);
			setRevision("fixture-v1");
			setMethod("hybrid");
			setCompareMethods(false);
		} catch (error) {
			setDemoError(error instanceof Error ? error.message : "Synthetic demo could not be loaded.");
		} finally {
			setDemoLoading(false);
		}
	}

	function useManualFiles(event: ChangeEvent<HTMLInputElement>) {
		setDemoFiles(null);
		onEdited(event.target.name === "new_bpmn" ? event.target.files?.[0] ?? null : null);
		if (event.target.name === "old_bpmn") onBaseline(event.target.files?.[0] ?? null);
	}

  return (
    <aside className={styles.panel}>
      <div className={styles.heading}>
        <span>01</span>
        <div><h2>Configure analysis</h2><p>All processing stays on this machine.</p></div>
      </div>
			<label className={styles.demoChoice}><span>Demo scenario</span><select value={demoScenario} disabled={loading || demoLoading} onChange={event => { setDemoScenario(event.target.value); setDemoFiles(null); }}><option value="comprehensive">Comprehensive changes</option><option value="insert-task">Insert task</option><option value="add-branch">Add gateway branch</option><option value="move-task">Move task</option><option value="remove-branch">Remove gateway branch</option><option value="unresolved">Ambiguous match</option></select></label>
			<button className={styles.demoButton} type="button" disabled={loading || demoLoading} onClick={() => void loadSyntheticDemo()}>
				<span aria-hidden="true">▶</span>{demoLoading ? "Loading demo…" : "Load synthetic demo"}
			</button>
			{demoError && <p className={styles.demoError}>{demoError}</p>}
      <form onSubmit={submit} className={styles.form}>
        <label className={styles.field}>
          <span>Repository path <b>*</b></span>
					<input name="repo_path" value={repositoryPath} onChange={(event) => setRepositoryPath(event.target.value)} required placeholder="C:\path\to\repository" autoComplete="off" />
          <small>Server-local path; the repository is read only.</small>
        </label>

        <div className={styles.grid}>
					<label className={styles.field}><span>Repository name</span><input name="repository_name" value={repositoryName} onChange={(event) => setRepositoryName(event.target.value)} placeholder="Example ERP" /></label>
					<label className={styles.field}><span>Revision</span><input name="revision" value={revision} onChange={(event) => setRevision(event.target.value)} placeholder="commit or snapshot" /></label>
        </div>

        <div className={styles.divider}><span>BPMN comparison</span></div>
        <label className={styles.fileField}>
          <span><b>Baseline BPMN</b><small>Old process specification</small></span>
					<input key={`old-${baseline?.name}-${baseline?.lastModified}`} name="old_bpmn" type="file" accept=".bpmn,.xml,text/xml,application/xml" required={!baseline} onChange={useManualFiles} />
          {baseline && <small>Loaded: {baseline.name}</small>}
        </label>
        <label className={styles.fileField}>
          <span><b>New BPMN</b><small>Changed process specification</small></span>
					<input key={`new-${baseline?.name}-${baseline?.lastModified}`} name="new_bpmn" type="file" accept=".bpmn,.xml,text/xml,application/xml" required={!demoFiles && !edited} onChange={useManualFiles} />
        </label>
				<button className={styles.canvasLink} type="button" disabled={loading || !baseline} onClick={onOpenEditor}>Edit baseline in canvas <span aria-hidden="true">↗</span></button>
				{edited && <p role="status">New BPMN ready: {edited.name}. Run analysis to update results.</p>}
				{demoFiles && <div className={styles.demoActive}><b>Synthetic demo ready</b><span>Neutral BPMN fixtures will be submitted; no case-study data is used.</span></div>}

        <div className={styles.grid}>
          <label className={styles.field}>
            <span>Mapping method</span>
            <select name="method" value={method} onChange={(event) => setMethod(event.target.value as AnalysisMethod)}>
              <option value="hybrid">Graph-aware deterministic hybrid</option>
								<option value="fusion">BM25 + Semantic fusion (RRF)</option>
              <option value="bm25">BM25 lexical baseline</option>
								<option value="semantic">Semantic embedding baseline</option>
            </select>
          </label>
          <label className={styles.field}><span>Top candidates</span><input name="top_k" type="number" min="1" max="100" defaultValue="5" /><small>Five keeps the review focused.</small></label>
        </div>
        <label className={styles.checkField}>
          <input name="compare_methods" type="checkbox" checked={compareMethods} onChange={(event) => setCompareMethods(event.target.checked)} />
			<span><b>Compare mapping methods</b><small>Runs BM25, Semantic, rank fusion, and Graph-Aware Hybrid on the same input.</small></span>
        </label>

        <button className={styles.submit} disabled={loading} type="submit">
          {loading ? "Analysis in progress…" : "Run impact analysis"}<span aria-hidden="true">→</span>
        </button>
        <p className={styles.note}>The mapper never executes or modifies repository source.</p>
      </form>
    </aside>
  );
}
