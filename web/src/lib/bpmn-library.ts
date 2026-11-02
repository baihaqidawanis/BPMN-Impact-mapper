export interface DiagramMetadata {
  repository: string;
  process: string;
  role: string;
  level: number;
  version: string;
}
export interface DiagramEntry extends DiagramMetadata {
  id: string;
  xml?: string;
  findings: { code: string; category: string; element_id?: string; message: string }[];
  activities?: {id:string;name:string;type:string}[];
}
export interface DiagramLink { parent_id:string; element_id:string; target_id:string }
const endpoint = "/backend/api/v1/bpmn-library";
async function responseJSON(response: Response) {
  const payload = await response.json();
  if (!response.ok) throw new Error(payload.error?.message || "BPMN library could not be accessed.");
  return payload;
}
export async function listDiagrams(): Promise<DiagramEntry[]> {
  return (await responseJSON(await fetch(endpoint, { cache: "no-store" }))).entries;
}
export async function saveDiagram(meta: DiagramMetadata, file: File): Promise<DiagramEntry> {
  if (file.size > 8 * 1024 * 1024) throw new Error("BPMN file must be at most 8 MiB.");
  // Preserve a UTF-8 BOM if present, and reject invalid UTF-8 instead of
  // silently replacing bytes in a saved baseline.
  let xml: string;
  try { xml = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(await file.arrayBuffer()); }
  catch { throw new Error("Save the BPMN file as UTF-8 XML before importing it."); }
  return responseJSON(await fetch(endpoint, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ...meta, xml }) }));
}
export async function loadDiagram(id: string): Promise<DiagramEntry> {
  return responseJSON(await fetch(`${endpoint}/${encodeURIComponent(id)}`, { cache: "no-store" }));
}
export async function diagramLinks(id:string):Promise<DiagramLink[]> {
  return (await responseJSON(await fetch(`${endpoint}/${encodeURIComponent(id)}/links`,{cache:"no-store"}))).links;
}
export async function setDiagramLink(id:string,elementID:string,targetID:string):Promise<DiagramLink[]> {
  return (await responseJSON(await fetch(`${endpoint}/${encodeURIComponent(id)}/links`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({element_id:elementID,target_id:targetID})}))).links;
}
export function diagramFile(entry: DiagramEntry): File {
  if (typeof entry.xml !== "string") throw new Error("Diagram XML is unavailable.");
  return new File([entry.xml], `library-${entry.id.slice(0, 12)}.bpmn`, { type: "application/xml" });
}
