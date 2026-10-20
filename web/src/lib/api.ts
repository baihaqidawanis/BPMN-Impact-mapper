import type { APIErrorPayload, Report, ReportSummary } from "./types";

export interface SemanticCapability {
  available: boolean;
  provider?: string;
  model?: string;
  message: string;
}

export async function checkSemanticRuntime(): Promise<SemanticCapability> {
  const response = await fetch("/backend/api/v1/capabilities/semantic/check", {
    method: "POST",
    cache: "no-store",
  });
  if (!response.ok) throw new Error("Semantic runtime could not be checked.");
  return await response.json() as SemanticCapability;
}

export async function createAnalysis(formData: FormData, signal?: AbortSignal): Promise<Report> {
  const response = await fetch("/backend/api/v1/analyses", {
    method: "POST",
    body: formData,
    signal,
  });
  if (!response.ok) {
    let message = `Analysis request failed (${response.status}).`;
    try {
      const payload = (await response.json()) as APIErrorPayload;
      if (payload.error?.message) message = payload.error.message;
    } catch {
      // Keep the bounded generic error when the response is not JSON.
    }
    throw new Error(message);
  }
  return (await response.json()) as Report;
}

export function reportDownloadURL(analysisID: string, format: "json" | "md"): string {
  return `/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/report.${format}`;
}

export function planningContextURL(analysisID: string): string {
  return `/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/planning-context.json`;
}

export async function listAnalyses(): Promise<ReportSummary[]> {
  const response = await fetch("/backend/api/v1/analyses", { cache: "no-store" });
  if (!response.ok) throw new Error("Saved analyses could not be loaded.");
  const value = await response.json() as { analyses: ReportSummary[] };
  return value.analyses;
}

export async function getAnalysis(analysisID: string): Promise<Report> {
  const response = await fetch(`/backend/api/v1/analyses/${encodeURIComponent(analysisID)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("The selected analysis is unavailable.");
  return await response.json() as Report;
}
