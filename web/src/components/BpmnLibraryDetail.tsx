"use client";

import {useEffect,useState} from "react";
import {BpmnReadOnly} from "./BpmnReadOnly";
import {diagramFile,diagramLinks,loadDiagram,setDiagramLink,type DiagramEntry,type DiagramLink} from "@/lib/bpmn-library";
import styles from "./BpmnLibrary.module.css";

export function BpmnLibraryDetail({initialID,entries,onClose,onBaseline}:{initialID:string;entries:DiagramEntry[];onClose:()=>void;onBaseline:(file:File)=>void}){
  const [trail,setTrail]=useState<string[]>([initialID]);
  const id=trail[trail.length-1];
  const [entry,setEntry]=useState<DiagramEntry|null>(null);
  const [links,setLinks]=useState<DiagramLink[]>([]);
  const [selected,setSelected]=useState("");
  const [target,setTarget]=useState("");
  const [error,setError]=useState("");
  const [notice,setNotice]=useState("");
  const [busy,setBusy]=useState(false);
  useEffect(()=>{
    let current=true;setEntry(null);setSelected("");setTarget("");setError("");setNotice("");
    Promise.all([loadDiagram(id),diagramLinks(id)]).then(([diagram,relationships])=>{if(current){setEntry(diagram);setLinks(relationships);}}).catch(err=>{if(current)setError(err.message);});
    return()=>{current=false;};
  },[id]);
  function choose(elementID:string){
    if(!entry?.activities?.some(a=>a.id===elementID))return;
    setSelected(elementID);setTarget(links.find(l=>l.element_id===elementID)?.target_id||"");setNotice("");
  }
  async function save(){
    setBusy(true);setError("");setNotice("");
    try {setLinks(await setDiagramLink(id,selected,target));setNotice(target?"Detail link saved for this diagram version.":"Detail link removed.");}
    catch(err){setError((err as Error).message);}finally{setBusy(false);}
  }
  const activity=entry?.activities?.find(a=>a.id===selected);
  const link=links.find(l=>l.element_id===selected);
  const child=entries.find(e=>e.id===link?.target_id);
  const choices=entries.filter(e=>entry && e.repository===entry.repository && e.level>entry.level);
  return <section className={styles.detail} aria-label="Process detail browser">
    <div className={styles.heading}><button type="button" onClick={onClose}>Back to library</button><span>Read-only process navigation</span></div>
    <nav aria-label="Process breadcrumb" className={styles.actions}>{trail.map((item,index)=><button type="button" key={item} disabled={busy||index===trail.length-1} onClick={()=>setTrail(trail.slice(0,index+1))}>{entries.find(e=>e.id===item)?.process||"Diagram"}</button>)}</nav>
    {error && <p role="alert" className={styles.error}>{error}</p>}
    {notice && <p role="status">{notice}</p>}
    {!entry && !error && <p role="status">Opening process...</p>}
    {entry && <>
      <div className={styles.heading}><div><h3>{entry.process}</h3><p>{entry.repository} / {entry.role||"No role"} / Level {entry.level} / {entry.version}</p></div><button type="button" onClick={()=>onBaseline(diagramFile(entry))}>Use this diagram as baseline</button></div>
      <div className={styles.detailLayout}>
        <BpmnReadOnly xml={entry.xml!} onSelect={choose}/>
        <aside className={styles.form} aria-label="Activity details">
          <h4>Process activity</h4><p>Select an activity in the diagram or use this list. Diagram files stay unchanged.</p>
          <label>Selected activity<select value={selected} onChange={e=>choose(e.target.value)}><option value="">Choose an activity</option>{entry.activities?.map(a=><option key={a.id} value={a.id}>{a.name||a.id}</option>)}</select></label>
          {activity && <><small>{activity.type} / {activity.id}</small>
            {child?<><p>Linked detail: {child.process} / Level {child.level} / {child.version}</p><button type="button" disabled={busy} onClick={()=>setTrail([...trail,child.id])}>Open process detail</button></>:<p>{link?"Linked diagram is unavailable. Choose another saved version.":"Detail has not been linked."}</p>}
            {entry.level<3 && <><label htmlFor="library-detail-target">Detail diagram</label><select id="library-detail-target" value={target} onChange={e=>setTarget(e.target.value)}><option value="">No linked detail</option>{choices.map(e=><option key={e.id} value={e.id}>{e.process} / L{e.level} / {e.role||"No role"} / {e.version}</option>)}</select>
              {!choices.length && <small>Save a deeper-level diagram with the same repository label in Library first.</small>}
              <button type="button" disabled={busy} onClick={save}>{busy?"Saving...":"Save detail link"}</button><small>Explicit navigation only. This does not propagate changes or affect mapping scores.</small></>}
          </>}
        </aside>
      </div>
    </>}
  </section>;
}
