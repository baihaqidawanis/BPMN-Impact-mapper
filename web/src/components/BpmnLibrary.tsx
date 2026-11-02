"use client";

import { useEffect, useState } from "react";
import { diagramFile, listDiagrams, loadDiagram, saveDiagram, type DiagramEntry } from "@/lib/bpmn-library";
import styles from "./BpmnLibrary.module.css";
import { BpmnLibraryDetail } from "./BpmnLibraryDetail";

interface Props {
  active: boolean;
  baseline: File | null;
  edited: File | null;
  hasBaseline: boolean;
  onUse: (file: File, target: "baseline" | "changed") => void;
}

export function BpmnLibrary({ active, baseline, edited, hasBaseline, onUse }: Props) {
  const [entries, setEntries] = useState<DiagramEntry[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [filter, setFilter] = useState("");
  const [repositoryFilter,setRepositoryFilter]=useState("");
  const [roleFilter,setRoleFilter]=useState("");
  const [levelFilter,setLevelFilter]=useState("");
  const [upload, setUpload] = useState<File | null>(null);
  const [source, setSource] = useState("upload");
  const [loaded, setLoaded] = useState(false);
  const [opened,setOpened]=useState("");
  const selected = source === "baseline" ? baseline : source === "changed" ? edited : upload;
  useEffect(() => {
    if (!active || loaded) return;
    let current = true;
    listDiagrams().then(values => { if (current) { setEntries(values); setLoaded(true); } }).catch(err => { if (current) setError(err.message); });
    return () => { current = false; };
  }, [active, loaded]);

  async function refresh() {
    setBusy(true); setError("");
    try { setEntries(await listDiagrams()); setLoaded(true); } catch (err) { setError((err as Error).message); } finally { setBusy(false); }
  }
  async function store(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!selected) return;
    const fields = new FormData(event.currentTarget);
    setBusy(true); setError(""); setNotice("");
    try {
      const entry = await saveDiagram({ repository: String(fields.get("repository")), process: String(fields.get("process")), role: String(fields.get("role")), level: Number(fields.get("level")), version: String(fields.get("version")) }, selected);
      setEntries(await listDiagrams());
      setNotice(`Saved ${entry.process}, level ${entry.level}, version ${entry.version}. Original XML preserved.`);
    } catch (err) { setError((err as Error).message); } finally { setBusy(false); }
  }
  async function use(id: string, target: "baseline" | "changed" | "download") {
    setBusy(true); setError("");
    try {
      const file = diagramFile(await loadDiagram(id));
      if (target === "download") {
        const url = URL.createObjectURL(file); const anchor = document.createElement("a");
        anchor.href = url; anchor.download = file.name; anchor.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
      } else { onUse(file, target); }
    } catch (err) { setError((err as Error).message); } finally { setBusy(false); }
  }
  const visible = entries.filter(e => `${e.repository} ${e.process} ${e.role} L${e.level} ${e.version}`.toLowerCase().includes(filter.trim().toLowerCase()) && (!repositoryFilter||e.repository===repositoryFilter) && (!roleFilter||e.role===roleFilter) && (levelFilter===""||e.level===Number(levelFilter)));
  function clearFilters(){setFilter("");setRepositoryFilter("");setRoleFilter("");setLevelFilter("");}
  if(opened)return <section className={styles.library} aria-label="BPMN library"><BpmnLibraryDetail key={opened} initialID={opened} entries={entries} onClose={()=>setOpened("")} onBaseline={file=>onUse(file,"baseline")}/></section>;
  return <section className={styles.library} aria-label="BPMN library">
    <div className={styles.layout}>
      <form className={styles.form} onSubmit={store}>
        <h3>Save a diagram</h3>
        <p>Store one version with its process details. Saving never changes the original file or starts analysis.</p>
        <label>Diagram source<select value={source} onChange={e => setSource(e.target.value)}><option value="upload">Upload a file</option><option value="baseline" disabled={!baseline}>Current baseline</option><option value="changed" disabled={!edited}>Current changed BPMN</option></select></label>
        {source === "upload" && <label>BPMN file<input type="file" accept=".bpmn,.xml" onChange={e => setUpload(e.target.files?.[0] ?? null)} required /></label>}
        {selected && <small>{selected.name}</small>}
        <label>Repository label<input name="repository" required maxLength={160} placeholder="Simple-ERP" /></label>
        <label>Process name<input name="process" required maxLength={160} placeholder="Submit stock adjustment" /></label>
        <label>Role (optional)<input name="role" maxLength={160} placeholder="Staff" /></label>
        <div className={styles.row}><label>Process level<select name="level" defaultValue="3">{[0,1,2,3].map(n => <option key={n} value={n}>Level {n}</option>)}</select></label><label>Version label<input name="version" required maxLength={160} placeholder="baseline-v1" /></label></div>
        <button type="submit" disabled={busy || !selected}>{busy ? "Working..." : "Save diagram"}</button>
        <small>Stored in this backend workspace. A new XML or metadata combination creates a separate entry.</small>
      </form>
      <div className={styles.catalog}>
        <div className={styles.heading}><h3>Saved diagrams <span>({entries.length})</span></h3><button type="button" onClick={refresh} disabled={busy}>Refresh library</button></div>
        <label>Find diagrams<input type="search" value={filter} onChange={e => setFilter(e.target.value)} placeholder="Repository, process, role, level or version" /></label>
        <div className={styles.filters}>
          <label htmlFor="library-repository-filter">Repository</label><select id="library-repository-filter" value={repositoryFilter} onChange={e=>setRepositoryFilter(e.target.value)}><option value="">All repositories</option>{[...new Set(entries.map(e=>e.repository))].sort().map(value=><option key={value}>{value}</option>)}</select>
          <label htmlFor="library-role-filter">Role</label><select id="library-role-filter" value={roleFilter} onChange={e=>setRoleFilter(e.target.value)}><option value="">All roles</option>{[...new Set(entries.map(e=>e.role).filter(Boolean))].sort().map(value=><option key={value}>{value}</option>)}</select>
          <label htmlFor="library-level-filter">Level</label><select id="library-level-filter" value={levelFilter} onChange={e=>setLevelFilter(e.target.value)}><option value="">All levels</option>{[0,1,2,3].map(level=><option key={level} value={level}>Level {level}</option>)}</select>
        </div>
        <div className={styles.filterSummary}><small aria-live="polite">Showing {visible.length} of {entries.length} diagrams. All saved versions are included.</small><button type="button" disabled={!filter&&!repositoryFilter&&!roleFilter&&levelFilter===""} onClick={clearFilters}>Clear filters</button></div>
        {error && <p role="alert" className={styles.error}>{error}</p>}
        {notice && <p role="status">{notice}</p>}
        {loaded && visible.length === 0 && <p>{entries.length ? "No matching diagrams." : "No saved diagrams yet. Add a baseline to reuse it after refresh."}</p>}
        {visible.map(entry => <article key={entry.id} className={styles.entry}>
          <h4>{entry.process}</h4><p>{entry.repository} / {entry.role || "No role"} / Level {entry.level} / {entry.version}</p>
          <button type="button" disabled={busy} onClick={()=>setOpened(entry.id)}>View process and details</button>
          <div className={styles.actions}><button type="button" disabled={busy} onClick={() => use(entry.id,"baseline")}>Use as baseline</button><button type="button" disabled={busy || !hasBaseline} onClick={() => use(entry.id,"changed")}>Use as changed BPMN</button><button type="button" disabled={busy} onClick={() => use(entry.id,"download")}>Download BPMN</button></div>
          <details><summary>{entry.findings.length ? `${entry.findings.length} modelling checks to review` : "No local check findings"}</summary><p>Single start/end and no gateway directly to a gateway/end are project conventions. These checks do not prove business correctness or code alignment.</p>{entry.findings.map((f,i) => <p key={i}><strong>{f.code}</strong>{f.element_id && <code> {f.element_id}</code>}<br />{f.message}</p>)}</details>
        </article>)}
      </div>
    </div>
  </section>;
}
