"use client";
import { useEffect, useState, useId } from "react";
import { applyAgentSettings, getAgentSettings, getAgentModels, type AgentSettings as Settings } from "@/lib/agents";
import styles from "./AgentSettings.module.css";

export function AgentSettings({ disabled, onApplied, onDirty }: { disabled:boolean; onApplied:()=>void; onDirty:(dirty:boolean)=>void }) {
 const catalogID=useId();
 const [models,setModels]=useState<string[]>([]);
 const [saved,setSaved]=useState<Settings|null>(null);
 const [draft,setDraft]=useState<Settings|null>(null);
 const [saving,setSaving]=useState(false);
 const [error,setError]=useState("");
 const [notice,setNotice]=useState("");
 useEffect(()=>{let live=true; void getAgentSettings().then(v=>{if(live){setSaved(v);setDraft(v);}}).catch(()=>{if(live)setError("Model settings are unavailable. Restart the updated backend with --agent-model.");});void getAgentModels().then(v=>{if(live)setModels(v.models);}).catch(()=>{/* Manual installed alias remains available. */});return()=>{live=false;};},[]);
 const dirty=JSON.stringify(saved)!==JSON.stringify(draft);
 useEffect(()=>{onDirty(dirty||saving);},[dirty,saving,onDirty]);
 async function apply(){if(!draft)return;setSaving(true);setError("");setNotice("");try{const v=await applyAgentSettings(draft);setDraft(v);setSaved(v);setNotice("Applied. Start a new AI session. Existing checkpoints keep their original model identity.");onApplied();}catch(e){setError(e instanceof Error?e.message:"Settings were not applied.");}finally{setSaving(false);}}
 return <details className={styles.settings}><summary>Model settings</summary>
 <p>Applies to this backend while it is running. Restarting restores command-line defaults. Metadata checks do not run inference or download models.</p>
 <datalist id={catalogID}>{models.map(name=><option key={name} value={name}/>)}</datalist>
 {draft&&<fieldset disabled={disabled||saving}>
 <legend className={styles.srOnly}>Agent generation settings</legend>
 <div className={styles.roles}>{(["analyst","planner","reviewer"] as const).map(role=><div key={role} className={styles.role}>
 <h4>{role}</h4>
 <label>Model<input list={catalogID} aria-label={`${role} model`} value={draft[role].model} placeholder="Installed local Ollama alias" onChange={e=>setDraft({...draft,[role]:{...draft[role],model:e.target.value}})}/></label>
 <label>Thinking<select aria-label={`${role} thinking`} value={draft[role].thinking?"on":"off"} onChange={e=>setDraft({...draft,[role]:{...draft[role],thinking:e.target.value==="on"}})}><option value="off">Off</option><option value="on">On</option></select></label>
 <label>Generation profile<select aria-label={`${role} profile`} value={draft[role].profile} onChange={e=>setDraft({...draft,[role]:{...draft[role],profile:e.target.value as "baseline"|"coding"}})}><option value="baseline">Baseline</option><option value="coding">Coding</option></select></label>
 </div>)}</div>
 <div className={styles.budgets}><label>Context tokens<input aria-label="Agent context tokens" type="number" min={4096} max={32768} value={draft.context} onChange={e=>setDraft({...draft,context:Number(e.target.value)})}/></label><label>Output tokens<input aria-label="Agent output tokens" type="number" min={32} max={8192} value={draft.output} onChange={e=>setDraft({...draft,output:Number(e.target.value)})}/></label></div>
 <p>Output includes thinking tokens. Thinking on can increase latency and use the output budget before a complete answer. More context also uses more memory; settings do not guarantee accuracy.</p>
 <div className={styles.actions}><button type="button" disabled={!dirty} onClick={()=>void apply()}>{saving?"Checking models...":"Apply settings"}</button><button type="button" disabled={!dirty} onClick={()=>{setDraft(saved);setError("");setNotice("");}}>Discard changes</button></div>
 </fieldset>}
 {disabled&&<p>Settings are locked while a review is running.</p>}
 {dirty&&<p>Unapplied changes. Apply or discard before starting a review.</p>}
 {notice&&<p role="status">{notice}</p>}{error&&<p role="alert">{error}</p>}
 </details>;
}
