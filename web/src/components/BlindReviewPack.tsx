"use client";

import { useState } from "react";
import type { Report, ReviewerAnnotation } from "@/lib/types";
import { downloadEvaluation, evaluationRequest } from "@/lib/evaluation";
import styles from "./BlindReviewPack.module.css";

interface Manifest {
  pack_id: string;
  input_fingerprint: string;
  repository: { name: string; revision: string };
  case_digest: string;
  predictions: { method: string; analysis_id: string; sha256: string }[];
}

interface PackResponse {
  manifest: Manifest;
  reviewer_a: ReviewerAnnotation;
  reviewer_b: ReviewerAnnotation;
  guide_markdown: string;
}

export function BlindReviewPack({ reports }: { reports: Report[] }) {
  const [reviewerA, setReviewerA] = useState("reviewer-a");
  const [reviewerB, setReviewerB] = useState("reviewer-b");
  const [pack, setPack] = useState<PackResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function prepare() {
    setBusy(true); setError(""); setPack(null);
    try {
      const value = await evaluationRequest<PackResponse>("review-packs", {
        analysis_ids: reports.map(report => report.analysis_id),
        input_fingerprint: reports[0].input_fingerprint,
        reviewer_a: reviewerA,
        reviewer_b: reviewerB,
      });
      setPack(value);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Review pack could not be prepared.");
    } finally { setBusy(false); }
  }

  return <details className={styles.panel} data-testid="blind-review-pack">
    <summary>Blind reviewer pack <span>Prepare independent review without showing mapper candidates</span></summary>
    <div className={styles.body}>
      <p>This post-inference tool uses the {reports.length} immutable report(s) in the current comparison. Reviewers receive BPMN changes only. Keep the coordinator manifest separate.</p>
      <div className={styles.reviewers}>
        <label>Reviewer A ID<input value={reviewerA} disabled={busy} onChange={event => { setReviewerA(event.target.value); setPack(null); }} /></label>
        <label>Reviewer B ID<input value={reviewerB} disabled={busy} onChange={event => { setReviewerB(event.target.value); setPack(null); }} /></label>
      </div>
      <button type="button" disabled={busy || !reviewerA.trim() || !reviewerB.trim() || reviewerA.trim() === reviewerB.trim()} onClick={() => void prepare()}>{busy ? "Preparing…" : "Prepare blind review pack"}</button>
      {error && <p role="alert" className={styles.error}>{error}</p>}
      {pack && <section className={styles.ready} aria-live="polite">
        <div><span>Pack ready</span><code>{pack.manifest.pack_id}</code><small>{pack.manifest.repository.name} · {pack.manifest.repository.revision}</small></div>
        <p>Send each reviewer their own template and the guide. Do not send the coordinator manifest until reviews are complete.</p>
        <div className={styles.downloads}>
          <button type="button" onClick={() => downloadEvaluation(pack.reviewer_a, `reviewer-a-${pack.manifest.pack_id}.json`)}>Reviewer A template</button>
          <button type="button" onClick={() => downloadEvaluation(pack.reviewer_b, `reviewer-b-${pack.manifest.pack_id}.json`)}>Reviewer B template</button>
          <button type="button" onClick={() => downloadEvaluation(pack.guide_markdown, `review-guide-${pack.manifest.pack_id}.md`)}>Blind review guide</button>
          <button type="button" className={styles.manifest} onClick={() => downloadEvaluation(pack.manifest, `coordinator-manifest-${pack.manifest.pack_id}.json`)}>Coordinator manifest</button>
        </div>
      </section>}
    </div>
  </details>;
}
