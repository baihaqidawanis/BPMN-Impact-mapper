"use client";

import { useState } from "react";
import { AnalysisForm } from "@/components/AnalysisForm";
import { InterpretationGuide } from "@/components/InterpretationGuide";
import { ReportView } from "@/components/ReportView";
import { ComparisonView } from "@/components/ComparisonView";
import { EvaluationDashboard } from "@/components/EvaluationDashboard";
import { BlindReviewPack } from "@/components/BlindReviewPack";
import { AnalysisHistory } from "@/components/AnalysisHistory";
import { createAnalysis } from "@/lib/api";
import type { Report } from "@/lib/types";
import styles from "./page.module.css";

export default function Home() {
  const [report, setReport] = useState<Report | null>(null);
  const [comparison, setComparison] = useState<Report[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

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
        <div className={styles.status}><span /> Local engine ready</div>
      </header>

      <section className={styles.hero}>
        <div>
          <p className={styles.kicker}>BPMN change impact mapping</p>
          <h2>From process change to source evidence.</h2>
          <p>Compare two BPMN specifications against a repository. Review ranked files and symbols with explicit evidence, score components, and uncertainty.</p>
        </div>
        <div className={styles.flow} aria-label="Analysis flow">
          <span>BPMN diff</span><b>→</b><span>Source index</span><b>→</b><span>Impact evidence</span>
        </div>
      </section>
			<InterpretationGuide />

      <div className={styles.workspace}>
		<div className={styles.leftRail}>
			<AnalysisForm loading={loading} onSubmit={handleSubmit} />
			<AnalysisHistory refreshKey={report?.analysis_id ?? ""} onOpen={(saved) => { setReport(saved); setComparison([]); setError(""); }} />
		</div>
        <section className={styles.resultRegion} aria-live="polite">
          {loading && <div className={styles.stateCard}><div className={styles.spinner} /><h3>Analyzing repository</h3><p>Parsing BPMN, indexing allowed files, and ranking evidence…</p></div>}
          {!loading && error && <div className={`${styles.stateCard} ${styles.error}`}><strong>Analysis could not be completed</strong><p>{error}</p></div>}
          {!loading && !error && !report && <div className={styles.stateCard}><div className={styles.emptyGlyph}>↗</div><h3>Your impact report will appear here</h3><p>Configure the repository and provide both BPMN versions to begin.</p></div>}
          {!loading && report && <>
				{comparison.length > 1 && <ComparisonView reports={comparison} selectedMethod={report.method} onSelect={setReport} />}
				<ReportView report={report} />
				<BlindReviewPack key={`pack-${(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")}`} reports={comparison.length ? comparison : [report]} />
				<EvaluationDashboard key={(comparison.length ? comparison : [report]).map(r => r.analysis_id).join("|")} reports={comparison.length ? comparison : [report]} />
			</>}
        </section>
      </div>
    </main>
  );
}

function cloneFormData(source: FormData): FormData {
	const result = new FormData();
	for (const [name, value] of source.entries()) result.append(name, value);
	return result;
}
