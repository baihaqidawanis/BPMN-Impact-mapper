"use client";

import { useRef, useState } from "react";
import type { Impact, Report, ReviewerAnnotation, ReviewerExpectedImpact } from "@/lib/types";
import styles from "./ReviewPanel.module.css";

const actions = ["ADD", "MODIFY", "REMOVE", "VERIFY", "NO_ACTION", "UNRESOLVED"];

interface ManualImpact extends ReviewerExpectedImpact { id: number }

export function ReviewPanel({ report }: { report: Report }) {
  const [reviewerID, setReviewerID] = useState("");
  const [selected, setSelected] = useState<Record<string, { checked: boolean; action: string }>>({});
  const [manual, setManual] = useState<Record<number, ManualImpact[]>>({});
  const [error, setError] = useState("");
  const nextID = useRef(1);

  function toggle(changeIndex: number, impact: Impact, checked: boolean) {
    const key = selectionKey(changeIndex, impact.file);
    setSelected((current) => ({ ...current, [key]: { checked, action: current[key]?.action ?? impact.required_action } }));
  }

  function setAction(changeIndex: number, impact: Impact, action: string) {
    const key = selectionKey(changeIndex, impact.file);
    setSelected((current) => ({ ...current, [key]: { checked: current[key]?.checked ?? false, action } }));
  }

  function addManual(changeIndex: number) {
    const item: ManualImpact = { id: nextID.current++, file: "", symbol: "", module: "", layer: "", required_action: "VERIFY" };
    setManual((current) => ({ ...current, [changeIndex]: [...(current[changeIndex] ?? []), item] }));
  }

  function updateManual(changeIndex: number, id: number, field: keyof ReviewerExpectedImpact, value: string) {
    setManual((current) => ({ ...current, [changeIndex]: (current[changeIndex] ?? []).map((item) => item.id === id ? { ...item, [field]: value } : item) }));
  }

  function removeManual(changeIndex: number, id: number) {
    setManual((current) => ({ ...current, [changeIndex]: (current[changeIndex] ?? []).filter((item) => item.id !== id) }));
  }

  function exportAnnotation() {
    setError("");
    if (!reviewerID.trim()) {
      setError("Reviewer ID is required.");
      return;
    }
    try {
      const cases = report.changes.map((change, changeIndex) => {
        const elementID = change.new_element?.id ?? change.old_element?.id ?? "";
        if (!elementID) throw new Error(`Change ${changeIndex + 1}: element ID is required for reviewer export.`);
        const expected: ReviewerExpectedImpact[] = [];
        for (const impact of uniqueFileImpacts(change.impacts)) {
          const choice = selected[selectionKey(changeIndex, impact.file)];
          if (choice?.checked) expected.push({ file: impact.file, symbol: impact.symbol, module: impact.module, layer: impact.layer, required_action: choice.action });
        }
        for (const item of manual[changeIndex] ?? []) {
          const file = item.file.trim().replaceAll("\\", "/");
          if (!validRelativePath(file)) throw new Error(`Change ${changeIndex + 1}: manual file must be a clean relative path.`);
          if (expected.some((value) => value.file === file)) throw new Error(`Change ${changeIndex + 1}: expected file ${file} is duplicated.`);
          expected.push({ file, symbol: item.symbol?.trim(), module: item.module?.trim(), layer: item.layer?.trim(), required_action: item.required_action });
        }
        expected.sort((a, b) => `${a.file}|${a.symbol}`.localeCompare(`${b.file}|${b.symbol}`));
        return { change_type: change.change_type, element_id: elementID, expected };
      });
      const annotation: ReviewerAnnotation = { version: "1", reviewer_id: reviewerID.trim(), cases };
      downloadJSON(annotation, `review-${safeFilename(reviewerID)}-${report.input_fingerprint.slice(0, 8)}.json`);
    } catch (exportError) {
      setError(exportError instanceof Error ? exportError.message : "Annotation could not be exported.");
    }
  }

  return (
    <details className={styles.panel} data-testid="review-panel">
      <summary><span><b>Reviewer annotation</b><small>Export ground-truth input locally; nothing is sent to inference.</small></span><span>Open</span></summary>
      <div className={styles.body}>
        <div className={styles.notice}><b>Independent judgment required.</b> Rankings are prompts for review, not automatic truth. Add missing files manually and leave a case empty only when no software impact is expected.</div>
        <label className={styles.reviewer}><span>Reviewer ID</span><input value={reviewerID} onChange={(event) => setReviewerID(event.target.value)} placeholder="reviewer-a" /></label>
        {report.changes.map((change, changeIndex) => <section className={styles.case} key={`${change.change_type}-${changeIndex}`}>
          <header><span>{String(changeIndex + 1).padStart(2, "0")}</span><div><b>{change.change_type.replaceAll("_", " ")}</b><small>{change.new_element?.id ?? change.old_element?.id}</small></div></header>
          <div className={styles.candidates}>
            {uniqueFileImpacts(change.impacts).map((impact) => {
              const choice = selected[selectionKey(changeIndex, impact.file)];
              return <div className={styles.candidate} key={impact.file}>
                <label><input type="checkbox" checked={choice?.checked ?? false} onChange={(event) => toggle(changeIndex, impact, event.target.checked)} /><span><b>{impact.file}</b><small>{impact.symbol || "file-level candidate"}</small></span></label>
                <select aria-label={`Expected action for ${impact.file}`} disabled={!choice?.checked} value={choice?.action ?? impact.required_action} onChange={(event) => setAction(changeIndex, impact, event.target.value)}>{actions.map((action) => <option key={action}>{action}</option>)}</select>
              </div>;
            })}
            {change.impacts.length === 0 && <p className={styles.empty}>No predicted candidate. Add expected impacts manually when appropriate.</p>}
          </div>
          {(manual[changeIndex] ?? []).map((item) => <div className={styles.manual} key={item.id}>
            <input aria-label="Missing expected file" value={item.file} onChange={(event) => updateManual(changeIndex, item.id, "file", event.target.value)} placeholder="relative/path/file.go" />
            <input aria-label="Missing expected symbol" value={item.symbol} onChange={(event) => updateManual(changeIndex, item.id, "symbol", event.target.value)} placeholder="Symbol (optional)" />
            <input aria-label="Missing expected module" value={item.module} onChange={(event) => updateManual(changeIndex, item.id, "module", event.target.value)} placeholder="Module (optional)" />
            <input aria-label="Missing expected layer" value={item.layer} onChange={(event) => updateManual(changeIndex, item.id, "layer", event.target.value)} placeholder="Layer (optional)" />
            <select aria-label="Missing expected action" value={item.required_action} onChange={(event) => updateManual(changeIndex, item.id, "required_action", event.target.value)}>{actions.map((action) => <option key={action}>{action}</option>)}</select>
            <button type="button" onClick={() => removeManual(changeIndex, item.id)}>Remove</button>
          </div>)}
          <button className={styles.add} type="button" onClick={() => addManual(changeIndex)}>+ Add missing expected impact</button>
        </section>)}
        {error && <p className={styles.error}>{error}</p>}
        <button className={styles.export} type="button" onClick={exportAnnotation}>Export reviewer JSON</button>
      </div>
    </details>
  );
}

function uniqueFileImpacts(impacts: Impact[]): Impact[] {
  const seen = new Set<string>();
  return impacts.filter((impact) => {
    if (seen.has(impact.file)) return false;
    seen.add(impact.file);
    return true;
  });
}

function selectionKey(changeIndex: number, file: string): string { return `${changeIndex}|${file}`; }

function validRelativePath(value: string): boolean {
  return value !== "" && !value.startsWith("/") && !/^[A-Za-z]:/.test(value) && !value.split("/").some((part) => part === "" || part === "." || part === "..");
}

function safeFilename(value: string): string { return value.trim().replace(/[^a-zA-Z0-9_-]+/g, "-").replace(/^-|-$/g, "") || "reviewer"; }

function downloadJSON(value: ReviewerAnnotation, filename: string) {
  const url = URL.createObjectURL(new Blob([`${JSON.stringify(value, null, 2)}\n`], { type: "application/json" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
