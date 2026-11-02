import type { ChangePlan, FullReviewProgress, PlanningContext, ProviderIdentity } from "./planning";

export interface AgentCapability { enabled: boolean; available: boolean; provider?: ProviderIdentity; message: string }
export interface AgentSession {
  mode?: "focused" | "full"; full_review?: FullReviewProgress;
  session_id: string; analysis_id: string; status: string; role: string;
  turns: number; tools: number; revisions: number; provider: ProviderIdentity;
  message?: string; plan?: ChangePlan; context?: PlanningContext;
  decisions?: Array<{ file: string; symbol?: string; verdict: string; rationale: string; evidence_ids: string[] }>;
  events?: Array<{ sequence: number; role: string; kind: string }>;
  issues?: string[];
}
async function responseJSON<T>(response: Response): Promise<T> {
  const data = await response.json();
  if (!response.ok) throw new Error(data.error?.message ?? `Request failed (${response.status}).`);
  return data as T;
}
export async function checkAgent(): Promise<AgentCapability> {
  return responseJSON(await fetch("/backend/api/v1/capabilities/agents/check", { method: "POST" }));
}
export async function startAgent(analysisID: string, repositoryPath: string, changeIndex: number, sessionID?: string, mode: "focused" | "full" = "focused"): Promise<AgentSession> {
  return responseJSON(await fetch(`/backend/api/v1/analyses/${encodeURIComponent(analysisID)}/agent-sessions`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ repository_path: repositoryPath, change_index: changeIndex, session_id: sessionID, mode }),
  }));
}
export async function getAgent(id: string, signal: AbortSignal): Promise<AgentSession> {
  return responseJSON(await fetch(`/backend/api/v1/agent-sessions/${encodeURIComponent(id)}`, { cache: "no-store", signal }));
}
