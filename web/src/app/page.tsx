"use client";

import { useState } from "react";
import { AnalysisForm } from "@/components/AnalysisForm";
import { InterpretationGuide } from "@/components/InterpretationGuide";
import { ReportView } from "@/components/ReportView";
import { createAnalysis } from "@/lib/api";
import type { Report } from "@/lib/types";
import styles from "./page.module.css";

export default function Home() {
  const [report, setReport] = useState<Report | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(formData: FormData) {
    setLoading(true);
    setError("");
    try {
      setReport(await createAnalysis(formData));
    } catch (requestError) {
      setReport(null);
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
          <p className={styles.eyebrow}>Evidence-led change analysis</p>
          <h1>BPMN Impact Mapper</h1>
        </div>
        <div className={styles.status}><span /> Local analysis engine</div>
      </header>

      <section className={styles.hero}>
        <div>
          <p className={styles.kicker}>From process change to reviewable software evidence</p>
          <h2>Map what changed. Trace what may be affected.</h2>
          <p>Compare two BPMN specifications against a source repository. Every candidate is ranked with explicit evidence, score components, and uncertainty.</p>
        </div>
        <div className={styles.flow} aria-label="Analysis flow">
          <span>BPMN diff</span><b>→</b><span>Source index</span><b>→</b><span>Impact evidence</span>
        </div>
      </section>
			<InterpretationGuide />

      <div className={styles.workspace}>
        <AnalysisForm loading={loading} onSubmit={handleSubmit} />
        <section className={styles.resultRegion} aria-live="polite">
          {loading && <div className={styles.stateCard}><div className={styles.spinner} /><h3>Analyzing repository</h3><p>Parsing BPMN, indexing allowed files, and ranking evidence…</p></div>}
          {!loading && error && <div className={`${styles.stateCard} ${styles.error}`}><strong>Analysis could not be completed</strong><p>{error}</p></div>}
          {!loading && !error && !report && <div className={styles.stateCard}><div className={styles.emptyGlyph}>↗</div><h3>Your impact report will appear here</h3><p>Configure the analysis and provide both BPMN versions to begin.</p></div>}
          {!loading && report && <ReportView report={report} />}
        </section>
      </div>
    </main>
  );
}
