import type { APIErrorPayload, Report } from "./types";

export async function createAnalysis(formData: FormData): Promise<Report> {
  const response = await fetch("/backend/api/v1/analyses", {
    method: "POST",
    body: formData,
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
