"use client";

import {useEffect,useRef,useState,type ReactNode} from "react";
import styles from "./WorkspaceShell.module.css";

export type WorkspaceView="overview"|"mapping"|"canvas"|"library"|"history"|"research"|"guide";
const labels:Record<WorkspaceView,string>={overview:"Overview",mapping:"Analysis",canvas:"BPMN Canvas",library:"BPMN Library",history:"History",research:"Research",guide:"Guide"};
const paths:Record<WorkspaceView,string>={overview:"M4 4h6v6H4z M14 4h6v6h-6z M4 14h6v6H4z M14 14h6v6h-6z",mapping:"M5 3h10l4 4v14H5z M14 3v5h5 M8 12h8 M8 16h5",canvas:"M3 8h6v8H3z M15 8h6v8h-6z M9 12h6",library:"M4 4h4v16H4z M10 4h4v16h-4z M17 5l3 14",history:"M3 4v5h5 M3 9a9 9 0 1 1 1 9 M12 7v5l3 2",research:"M8 3h8 M10 3v6L5 18q-1 3 2 3h10q3 0 2-3l-5-9V3 M8 15h8",guide:"M4 4h5q3 0 3 3q0-3 3-3h5v15h-5q-3 0-3 2q0-2-3-2H4z M12 7v14"};

export function WorkspaceShell({view,onNavigate,research,busy,children}:{view:WorkspaceView;onNavigate:(view:WorkspaceView)=>void;research:boolean;busy:boolean;children:ReactNode}){
  const [open,setOpen]=useState(false);
  const [collapsed,setCollapsed]=useState(false);
  const [mobile,setMobile]=useState(false);
  const toggle=useRef<HTMLButtonElement>(null);
  const navigation=useRef<HTMLElement>(null);
  useEffect(()=>{
    const media=window.matchMedia("(max-width:760px)");
    function update(){setMobile(media.matches);if(!media.matches)setOpen(false);}
    update();media.addEventListener("change",update);return()=>media.removeEventListener("change",update);
  },[]);
  useEffect(()=>{
    if(!open)return;
    navigation.current?.querySelector<HTMLButtonElement>('button[aria-current="page"]')?.focus();
    function escape(event:KeyboardEvent){if(event.key==="Escape"){setOpen(false);toggle.current?.focus();}}
    window.addEventListener("keydown",escape);return()=>window.removeEventListener("keydown",escape);
  },[open]);
  function go(next:WorkspaceView){onNavigate(next);setOpen(false);}
  function button(id:WorkspaceView){return <button type="button" key={id} aria-label={labels[id]} title={collapsed&&!mobile?labels[id]:undefined} className={id==="canvas"?styles.child:undefined} aria-current={view===id?"page":undefined} data-parent-active={id==="mapping"&&view==="canvas"||undefined} onClick={()=>go(id)}><svg viewBox="0 0 24 24" aria-hidden="true"><path d={paths[id]}/></svg><span>{labels[id]}</span></button>;}
  return <div className={`${styles.layout} ${collapsed&&!mobile?styles.collapsed:""}`}>
    <a href="#workspace-content" className={styles.skip}>Skip to content</a>
    {open && <button type="button" aria-label="Close navigation" className={styles.backdrop} onClick={()=>setOpen(false)}/>}
    <aside className={`${styles.sidebar} ${open?styles.open:""}`}>
      <div className={styles.brand}><span className={styles.mark} aria-hidden="true"><svg viewBox="0 0 24 24"><rect x="3" y="8" width="6" height="8" rx="1"/><path d="M9 12h6"/><rect x="15" y="8" width="6" height="8" rx="1"/></svg></span><div><strong>Impact Mapper</strong><small>BPMN · Source code</small></div><button type="button" className={styles.closeMenu} aria-label="Close sidebar" onClick={()=>{setOpen(false);toggle.current?.focus();}}><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M6 18L18 6"/></svg></button></div>
      <nav id="workspace-navigation" ref={navigation} className={styles.navigation} aria-label="Workspace navigation">
        {button("overview")}{button("mapping")}{button("canvas")}{button("library")}{button("history")}
        <div className={styles.secondary}>{research&&<><span className={styles.caption}>Research workspace</span>{button("research")}</>}{button("guide")}</div>
      </nav>
      <div className={styles.local}><span/>Local workspace<small>Source repositories stay read-only</small></div>
    </aside>
    <div className={styles.main}>
      <header className={styles.topbar}>
        <button ref={toggle} type="button" className={styles.menu} aria-label="Toggle navigation" aria-expanded={mobile?open:!collapsed} aria-controls="workspace-navigation" title={mobile?"Open navigation":collapsed?"Expand sidebar":"Collapse sidebar"} onClick={()=>mobile?setOpen(!open):setCollapsed(!collapsed)}><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16"/></svg></button>
        <span>{view==="canvas"?"Analysis / BPMN Canvas":labels[view]}</span><small>{busy?"Analysis in progress":"Local processing"}</small>
      </header>
      <main id="workspace-content" className={styles.content}>{children}</main>
    </div>
  </div>;
}
