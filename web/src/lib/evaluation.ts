import type { ReviewerAnnotation, ReviewerCase } from "./types";

export interface GroundTruth { version: "1"; cases: ReviewerCase[] }
export interface Reconciliation {
  agreed_ground_truth: GroundTruth;
  disagreements: { change_type: string; element_id: string; reviewer_a: ReviewerCase["expected"] | null; reviewer_b: ReviewerCase["expected"] | null; reason: string }[];
}
export interface EvaluationResult {
  analysis_id: string;
  method: string;
  metrics: {
    recall: { at_1: number; at_3: number; at_5: number };
    precision: { at_1: number; at_3: number; at_5: number };
    mrr: number; map: number; file_f1: number; file_precision: number; file_recall: number;
    module_accuracy: number; layer_accuracy: number; required_action_accuracy: number;
    unresolved_rate: number; false_positive_rate: number; evaluated_changes: number;
  };
}
export interface EvaluationBundle {
  input_fingerprint: string;
  ground_truth: GroundTruth;
  results: EvaluationResult[];
  markdown: string;
}

export async function evaluationRequest<T>(route: string, body: unknown): Promise<T> {
  const response = await fetch("/backend/api/v1/" + route, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error?.message ?? "Evaluation request failed.");
  return data as T;
}

export async function readAnnotation(file: File): Promise<ReviewerAnnotation> {
  if (file.size > 512 * 1024) throw new Error("Each annotation must be at most 512 KiB.");
  const value = JSON.parse(await file.text());
  if (!value || value.version !== "1" || typeof value.reviewer_id !== "string" || !Array.isArray(value.cases)) throw new Error("Select a version-1 reviewer annotation JSON.");
  return value as ReviewerAnnotation;
}

export function downloadEvaluation(value: unknown, filename: string) {
  const text = typeof value === "string" ? value : JSON.stringify(value, null, 2) + "\n";
  const url = URL.createObjectURL(new Blob([text], { type: typeof value === "string" ? "text/markdown" : "application/json" }));
  const link = document.createElement("a");
  link.href = url; link.download = filename;
  document.body.appendChild(link); link.click(); link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
