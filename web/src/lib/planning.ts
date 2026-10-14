export interface PlanningEvidence {
  id: string;
  type: string;
  location: string;
  excerpt: string;
}

export interface PlanningCandidate {
  file: string;
  symbol?: string;
  module: string;
  layer: string;
  suggested_action: string;
  retrieval_score: number;
  evidence: PlanningEvidence[];
}

export interface PlanningChange {
  change_type: string;
  element_id: string;
  element_name?: string;
  reason?: string;
  candidates: PlanningCandidate[];
}

export interface PlanningContext {
  version: string;
  context_id: string;
  analysis_id: string;
  input_fingerprint: string;
  method: string;
  repository: { name: string; revision: string };
  changes: PlanningChange[];
}

export interface ProviderIdentity {
  provider: string;
  model: string;
  prompt_version: string;
}

export interface PlanSummary {
  plan_id: string;
  analysis_id: string;
  context_id: string;
  provider: ProviderIdentity;
  status: "DISABLED" | "COMPLETE" | "NEEDS_REVIEW";
  files: number;
}

export interface ChangePlan {
  plan_id: string;
  analysis_id: string;
  context_id: string;
  provider: ProviderIdentity;
  status: PlanSummary["status"];
  version: string;
  input_fingerprint: string;
  plans: Array<{
    file: string;
    symbol?: string;
    module: string;
    layer: string;
    action: string;
    summary: string;
    steps: string[];
    evidence_ids: string[];
  }>;
  review_issues: string[];
}

export async function getPlanningContext(analysisID: string): Promise<{ context: PlanningContext; bytes: number }> {
  const response = await fetch(`/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/planning-context.json`, { cache: "no-store" });
  if (!response.ok) throw new Error("Planning context could not be loaded.");
  const text = await response.text();
  return { context: JSON.parse(text) as PlanningContext, bytes: new TextEncoder().encode(text).byteLength };
}

export async function listPlans(analysisID: string): Promise<PlanSummary[]> {
  const response = await fetch(`/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/plans`, { cache: "no-store" });
  if (!response.ok) throw new Error("Saved plans could not be loaded.");
  const value = await response.json() as { plans: PlanSummary[] };
  return value.plans;
}

export async function createPlan(analysisID: string): Promise<ChangePlan> {
  const response = await fetch(`/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/plans`, { method: "POST" });
  if (!response.ok) {
    let message = `Planning request failed (${response.status}).`;
    try {
      const value = await response.json() as { error?: { message?: string } };
      if (value.error?.message) message = value.error.message;
    } catch {
      // Keep the bounded status message when the response is not JSON.
    }
    throw new Error(message);
  }
  return await response.json() as ChangePlan;
}

export function planDownloadURL(planID: string, format: "json" | "md"): string {
  return `/backend/api/v1/plans/${encodeURIComponent(planID)}/change-plan.${format}`;
}
