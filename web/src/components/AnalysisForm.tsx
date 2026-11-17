"use client";

import { ChangeEvent, FormEvent, useEffect, useState } from "react";
import { checkSemanticRuntime, type SemanticCapability } from "@/lib/api";
import type { AnalysisMethod } from "@/lib/types";
import styles from "./AnalysisForm.module.css";

interface Props {
  completedRepository?: { name: string; revision: string };
  repositoryPath?: string;
  onRepositoryPath?: (path: string) => void;
  enableResearch?: boolean;
  loading: boolean;
  onSubmit: (formData: FormData) => Promise<void>;
  baseline: File | null;
  edited: File | null;
  onBaseline: (file: File | null) => void;
  onEdited: (file: File | null) => void;
  onOpenEditor: () => void;
}

export function AnalysisForm({ completedRepository, repositoryPath: sharedPath, onRepositoryPath, enableResearch = true, loading, onSubmit, baseline, edited, onBaseline, onEdited, onOpenEditor }: Props) {
	const [researchSelected, setResearchMode] = useState(false);
  const researchMode = enableResearch && researchSelected;
  const [method, setMethod] = useState<AnalysisMethod>("hybrid");
	const [compareMethods, setCompareMethods] = useState(false);
	const [localPath, setLocalPath] = useState("");
  const repositoryPath = sharedPath ?? localPath;
  function setRepositoryPath(path: string) { setLocalPath(path); onRepositoryPath?.(path); }
	const [repositoryName, setRepositoryName] = useState("");
	const [revision, setRevision] = useState("");
  useEffect(() => {
    if (!completedRepository) return;
    setRepositoryName(completedRepository.name);
    setRevision(completedRepository.revision === "unspecified" ? "" : completedRepository.revision);
  }, [completedRepository]);
	const [semanticStatus, setSemanticStatus] = useState<SemanticCapability | null>(null);
	const [semanticChecking, setSemanticChecking] = useState(false);

  async function verifySemanticRuntime() {
		setSemanticChecking(true);
		try {
			const capability = await checkSemanticRuntime();
			setSemanticStatus(capability);
			return capability.available;
		} catch (error) {
			setSemanticStatus({
				available: false,
				message: error instanceof Error ? error.message : "Semantic runtime could not be checked.",
			});
			return false;
		} finally {
			setSemanticChecking(false);
		}
	}

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
		const formData = new FormData(event.currentTarget);
		for (const name of ["repo_path", "repository_name", "revision"]) {
			formData.set(name, String(formData.get(name) ?? "").trim());
		}
		if (edited) formData.set("new_bpmn", edited);
		if (baseline) formData.set("old_bpmn", baseline);
		const needsSemantic = (researchMode && compareMethods) || !researchMode || method !== "bm25";
		if (needsSemantic && !semanticStatus?.available && !(await verifySemanticRuntime())) return;
		await onSubmit(formData);
  }

  function useManualFiles(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    // Cancelling the picker preserves the process already in the workspace.
    if (!file) return;
    if (event.target.name === "old_bpmn") onBaseline(file);
    else onEdited(file);
  }

  return (
    <aside className={styles.panel}>
      <div className={styles.heading}>

        <div><h2>Configure analysis</h2><p>Repository and process versions.</p></div>
      </div>
      <form onSubmit={(event) => void submit(event)} className={styles.form}>
        <fieldset className={styles.section} disabled={loading || semanticChecking}><legend><span>01</span> Repository</legend>
        <label className={styles.field}>
          <span>Repository path <b>*</b></span>
					<input name="repo_path" value={repositoryPath} onChange={(event) => setRepositoryPath(event.target.value)} required placeholder="C:\path\to\repository" autoComplete="off" />
          <small>Server-local path; the repository is read only.</small>
        </label>

        <div className={styles.grid}>
					<label className={styles.field}><span>Repository name</span><input name="repository_name" value={repositoryName} onChange={(event) => setRepositoryName(event.target.value)} placeholder="Example ERP" /></label>
					<label className={styles.field}><span>Revision</span><input name="revision" value={revision} onChange={(event) => setRevision(event.target.value)} placeholder="commit or snapshot" /><small>Optional label; does not select a Git revision.</small></label>
        </div>

        </fieldset>
        <fieldset className={styles.section} disabled={loading || semanticChecking}><legend><span>02</span> Process versions</legend>
        <BpmnFileField name="old_bpmn" label="Baseline BPMN" hint="Original process" file={baseline} onChange={useManualFiles} />
        <BpmnFileField key={`${baseline?.name}-${baseline?.lastModified}`} name="new_bpmn" label="New BPMN" hint="Changed process" file={edited} onChange={useManualFiles} />
        <button className={styles.canvasLink} type="button" disabled={loading || !baseline} onClick={onOpenEditor}>Compare BPMN in canvas <span aria-hidden="true">&#8594;</span></button>
        {edited && <p className={styles.fileReady} role="status">New BPMN ready: {edited.name}. Run analysis to update results.</p>}

				</fieldset>
        <fieldset className={styles.section} disabled={loading || semanticChecking}><legend><span>03</span> Search settings</legend>
        {enableResearch && <div className={styles.modeSwitch} role="group" aria-label="Analysis mode">
					<button type="button" aria-pressed={!researchMode} onClick={() => { setResearchMode(false); setMethod("hybrid"); setCompareMethods(false); }}>Product</button>
					<button type="button" aria-pressed={researchMode} onClick={() => setResearchMode(true)}>Research</button>
				</div>}
				{researchMode ? <p className={styles.modeNote}><b>Research mode</b> exposes baselines and controlled method comparison. It does not change the production recommendation.</p> : <div className={styles.methodSummary}><span>Production method</span><b>BPMN-Context-Aware Hybrid Retrieval</b><small>BM25 and semantic rankings, combined with BPMN process context.</small></div>}
				<input type="hidden" name="method" value={researchMode ? method : "hybrid"} />
				<div className={styles.grid}>
					{researchMode && <label className={styles.field}>
						<span>Experimental method</span>
						<select aria-label="Experimental method" value={method} onChange={(event) => setMethod(event.target.value as AnalysisMethod)}>
							<option value="hybrid">BPMN-context-aware hybrid</option>
							<option value="fusion">BM25 + Semantic fusion (RRF)</option>
							<option value="bm25">BM25 lexical baseline</option>
							<option value="semantic">Semantic embedding baseline</option>
						</select>
					</label>}
					<label className={styles.field}><span>Top candidates</span><input name="top_k" type="number" min="1" max="100" defaultValue="10" /><small>Maximum results per change; not a relevance threshold.</small></label>
				</div>
				{researchMode && <label className={styles.checkField}>
					<input name="compare_methods" type="checkbox" checked={compareMethods} onChange={(event) => setCompareMethods(event.target.checked)} />
					<span><b>Compare mapping methods</b><small>Runs BM25, Semantic, rank fusion, and BPMN-Context-Aware Hybrid Retrieval on the same input.</small></span>
				</label>}

				{((researchMode && compareMethods) || !researchMode || method !== "bm25") && (
					<div className={styles.runtimeCheck} data-state={semanticStatus ? (semanticStatus.available ? "ready" : "unavailable") : "unchecked"}>
						<div>
							<b>Semantic runtime</b>
							<small role="status">{semanticChecking ? "Checking the local embedding model..." : semanticStatus?.message ?? "Required by this method; checked locally before analysis."}</small>
							{semanticStatus?.available && semanticStatus.model && <small>{semanticStatus.model}</small>}
						</div>
						<button type="button" disabled={loading || semanticChecking} onClick={() => void verifySemanticRuntime()}>{semanticChecking ? "Checking..." : "Check runtime"}</button>
					</div>
				)}

				</fieldset>
        <button className={styles.submit} disabled={loading || semanticChecking} type="submit">
          {loading ? "Analysis in progress..." : "Run impact analysis"}<span aria-hidden="true">&#8594;</span>
        </button>
        <p className={styles.note}>The mapper never executes or modifies repository source.</p>
      </form>
    </aside>
  );
}

function BpmnFileField({ name, label, hint, file, onChange }: {
  name: string; label: string; hint: string; file: File | null;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
}) {
  return <label className={styles.fileField}>
    <span><b>{label}</b><small>{hint}</small></span>
    <span className={styles.filePicker} data-loaded={!!file}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><path d="M6 3h8l4 4v14H6zM14 3v5h4M9 12h6m-6 4h4"/></svg>
      <span className={styles.fileName}>{file?.name || "Choose a BPMN file"}</span>
      <span className={styles.fileAction}>{file ? "Replace" : "Browse"}</span>
      <input name={name} type="file" aria-label={label} accept=".bpmn,.xml,text/xml,application/xml" required={!file} onChange={onChange}/>
    </span>
  </label>;
}
