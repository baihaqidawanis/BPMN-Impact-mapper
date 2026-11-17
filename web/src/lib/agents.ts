import type { ChangePlan, FullReviewProgress, PlanningContext, ProviderIdentity } from "./planning";

export interface AgentCapability { enabled: boolean; available: boolean; provider?: ProviderIdentity; role_models?:Partial<Record<"analyst"|"planner"|"reviewer",ProviderIdentity>>; message: string }
export interface AgentSession {
  focused_review?: {outcome_counts_available?:boolean; batches_approved?:number; batches_needing_review?:number; batches_total:number; batches_reviewed:number; candidates_total:number; candidates_processed:number};
  mode?: "focused" | "full"; full_review?: FullReviewProgress;
  session_id: string; analysis_id: string; status: string; role: string;
  turns: number; tools: number; revisions: number; provider: ProviderIdentity;
  message?: string; plan?: ChangePlan; context?: PlanningContext;
  decisions?: Array<{ file: string; symbol?: string; verdict: string; rationale: string; evidence_ids: string[] }>;
  events?: Array<{ sequence: number; role: string; kind: string }>;
  source_activity?: Array<{ change_index: number; tool: string; query?: string; file?: string; function?: string; files: string[]; error?: string; truncated?: boolean; next_line?: number }>;
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

export async function stopAgent(id: string): Promise<void> {
 await responseJSON(await fetch(`/backend/api/v1/agent-sessions/${encodeURIComponent(id)}`, {method:"DELETE"}));
}

export interface AgentRoleSettings { model:string; thinking:boolean; profile:"baseline"|"coding" }
export interface AgentSettings { analyst:AgentRoleSettings; planner:AgentRoleSettings; reviewer:AgentRoleSettings; context:number; output:number }
export async function getAgentSettings():Promise<AgentSettings> {
 return responseJSON(await fetch("/backend/api/v1/agent-settings",{cache:"no-store"}));
}
export async function applyAgentSettings(settings:AgentSettings):Promise<AgentSettings> {
 return responseJSON(await fetch("/backend/api/v1/agent-settings",{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(settings)}));
}

export async function getAgentModels():Promise<{models:string[]}> {
 return responseJSON(await fetch("/backend/api/v1/agent-models",{cache:"no-store"}));
}

export async function exportAgentDraft(id:string):Promise<ChangePlan> {
 return responseJSON(await fetch(`/backend/api/v1/agent-sessions/${encodeURIComponent(id)}/draft-plan`,{method:"POST"}));
}
