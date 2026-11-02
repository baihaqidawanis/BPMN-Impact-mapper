"use client";

import { useEffect, useRef, useState } from "react";
import type NavigatedViewer from "bpmn-js/lib/NavigatedViewer";
import styles from "./BpmnLibrary.module.css";

export function BpmnReadOnly({xml,onSelect}:{xml:string;onSelect:(id:string)=>void}) {
  const container=useRef<HTMLDivElement>(null);
  const viewer=useRef<NavigatedViewer|null>(null);
  const select=useRef(onSelect);select.current=onSelect;
  const [error,setError]=useState("");
  const [ready,setReady]=useState(false);
  useEffect(()=>{
    let disposed=false;let instance:NavigatedViewer|undefined;
    let observer:ResizeObserver|undefined;let resizeFrame=0;
    setReady(false);setError("");
    async function open(){
      try {
        const {default:Viewer}=await import("bpmn-js/lib/NavigatedViewer");
        if(disposed || !container.current)return;
        instance=new Viewer({container:container.current});viewer.current=instance;
        await instance.importXML(xml);if(disposed)return;
        instance.get<{zoom:(value:string)=>void}>("canvas").zoom("fit-viewport");
        instance.on("element.click",(event:{element:{id:string}})=>select.current(event.element.id));
        // Navigation between files is explicit; never let the native drilldown
        // affordance open an empty embedded subprocess behind our breadcrumbs.
        instance.on("element.dblclick",2000,()=>false);
        observer=new ResizeObserver(()=>{
          cancelAnimationFrame(resizeFrame);
          resizeFrame=requestAnimationFrame(()=>{
            if(disposed || !container.current || container.current.clientWidth===0)return;
            const canvas=instance?.get<{resized:()=>void;zoom:(value:string)=>void}>("canvas");
            canvas?.resized();canvas?.zoom("fit-viewport");
          });
        });
        observer.observe(container.current);
        setReady(true);
      } catch(err){if(!disposed)setError((err as Error).message);}
    }
    void open();
    return ()=>{disposed=true;observer?.disconnect();cancelAnimationFrame(resizeFrame);instance?.destroy();viewer.current=null;};
  },[xml]);
  function zoom(value:"in"|"out"|"fit"){
    const canvas=viewer.current?.get<{zoom:(value?:number|string)=>number}>("canvas");
    if(!canvas)return;
    if(value==="fit")canvas.zoom("fit-viewport");else canvas.zoom(canvas.zoom()*(value==="in"?1.2:1/1.2));
  }
  return <div className={styles.diagram}>
    <div className={styles.actions} aria-label="Read-only diagram tools"><span>View only</span><button type="button" disabled={!ready} onClick={()=>zoom("in")}>Zoom in</button><button type="button" disabled={!ready} onClick={()=>zoom("out")}>Zoom out</button><button type="button" disabled={!ready} onClick={()=>zoom("fit")}>Fit diagram</button></div>
    {error && <p role="alert">Diagram cannot be displayed: {error}. Its XML can still be downloaded or used for mapping.</p>}
    <div ref={container} className={styles.viewer} data-testid="library-diagram" />
  </div>;
}
