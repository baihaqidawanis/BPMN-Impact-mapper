// Private, device-local convenience state. Paths never enter report JSON or
// shareable URLs. Key by analysis identity so different repositories cannot mix.
const currentSession = new Map<string, string>();
const key = (id: string) => `bpmn-analysis-repository:${id}`;

export function rememberAnalysisRepository(id: string, path: string) {
  if (!/^[a-f0-9]{32}$/.test(id) || !path.trim()) return;
  currentSession.set(id, path.trim());
  try { localStorage.setItem(key(id), path.trim()); } catch { /* Optional storage. */ }
}

export function analysisRepository(id: string): string {
  if (currentSession.has(id)) return currentSession.get(id)!;
  try { return localStorage.getItem(key(id)) ?? ""; } catch { return ""; }
}
