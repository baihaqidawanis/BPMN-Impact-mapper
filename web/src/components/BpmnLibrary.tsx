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
  const [repositoryFilter, setRepositoryFilter] = useState("");
  const [roleFilter, setRoleFilter] = useState("");
  const [levelFilter, setLevelFilter] = useState("");
  const [upload, setUpload] = useState<File | null>(null);
  const [source, setSource] = useState("upload");
  const [loaded, setLoaded] = useState(false);
  const [opened, setOpened] = useState("");
  const selected = source === "baseline" ? baseline : source === "changed" ? edited : upload;

  useEffect(() => {
    if (!active || loaded) return;
    let current = true;
    listDiagrams()
      .then(values => {
        if (current) {
          setEntries(values);
          setLoaded(true);
        }
      })
      .catch(err => {
        if (current) setError(err.message);
      });
    return () => {
      current = false;
    };
  }, [active, loaded]);

  async function refresh() {
    setBusy(true);
    setError("");
    try {
      setEntries(await listDiagrams());
      setLoaded(true);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function store(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    const fields = new FormData(event.currentTarget);
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const entry = await saveDiagram(
        {
          repository: String(fields.get("repository")),
          process: String(fields.get("process")),
          role: String(fields.get("role")),
          level: Number(fields.get("level")),
          version: String(fields.get("version")),
        },
        selected
      );
      setEntries(await listDiagrams());
      setNotice(`Saved ${entry.process}, level ${entry.level}, version ${entry.version}. Original XML preserved.`);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function use(id: string, target: "baseline" | "changed" | "download") {
    setBusy(true);
    setError("");
    try {
      const file = diagramFile(await loadDiagram(id));
      if (target === "download") {
        const url = URL.createObjectURL(file);
        const anchor = document.createElement("a");
        anchor.href = url;
        anchor.download = file.name;
        anchor.click();
        setTimeout(() => URL.revokeObjectURL(url), 1000);
      } else {
        onUse(file, target);
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const visible = entries.filter(
    e =>
      `${e.repository} ${e.process} ${e.role} L${e.level} ${e.version}`.toLowerCase().includes(filter.trim().toLowerCase()) &&
      (!repositoryFilter || e.repository === repositoryFilter) &&
      (!roleFilter || e.role === roleFilter) &&
      (levelFilter === "" || e.level === Number(levelFilter))
  );

  function clearFilters() {
    setFilter("");
    setRepositoryFilter("");
    setRoleFilter("");
    setLevelFilter("");
  }

  if (opened) {
    return (
      <section className={styles.library} aria-label="BPMN library">
        <BpmnLibraryDetail key={opened} initialID={opened} entries={entries} onClose={() => setOpened("")} onBaseline={file => onUse(file, "baseline")} />
      </section>
    );
  }

  return (
    <section className={styles.library} aria-label="BPMN library">
      <div className={styles.layout}>
        <div className={styles.catalog}>
          <div className={styles.toolbar}>
            <label className={styles.searchField}>
              <span>Find diagrams</span>
              <input type="search" value={filter} onChange={e => setFilter(e.target.value)} placeholder="Search process, repository, role, version…" />
            </label>
            <div className={styles.filterGroup}>
              <label htmlFor="library-repository-filter" className={styles.srOnly}>Repository</label>
              <select id="library-repository-filter" aria-label="Repository" value={repositoryFilter} onChange={e => setRepositoryFilter(e.target.value)}>
                <option value="">All repositories</option>
                {[...new Set(entries.map(e => e.repository))].sort().map(value => (
                  <option key={value}>{value}</option>
                ))}
              </select>
              <label htmlFor="library-role-filter" className={styles.srOnly}>Role</label>
              <select id="library-role-filter" aria-label="Role" value={roleFilter} onChange={e => setRoleFilter(e.target.value)}>
                <option value="">All roles</option>
                {[...new Set(entries.map(e => e.role).filter(Boolean))].sort().map(value => (
                  <option key={value}>{value}</option>
                ))}
              </select>
              <label htmlFor="library-level-filter" className={styles.srOnly}>Level</label>
              <select id="library-level-filter" aria-label="Level" value={levelFilter} onChange={e => setLevelFilter(e.target.value)}>
                <option value="">All levels</option>
                {[0, 1, 2, 3].map(level => (
                  <option key={level} value={level}>Level {level}</option>
                ))}
              </select>
              <button type="button" onClick={refresh} disabled={busy}>Refresh</button>
            </div>
          </div>

          <div className={styles.filterSummary}>
            <small aria-live="polite">
              Showing {visible.length} of {entries.length} diagrams.
            </small>
            <button
              type="button"
              className={styles.textBtn}
              disabled={!filter && !repositoryFilter && !roleFilter && levelFilter === ""}
              onClick={clearFilters}
            >
              Clear filters
            </button>
          </div>

          {error && <p role="alert" className={styles.error}>{error}</p>}
          {notice && <p role="status" className={styles.notice}>{notice}</p>}
          {loaded && visible.length === 0 && (
            <p className={styles.empty}>
              {entries.length ? "No matching diagrams." : "No saved diagrams yet. Add a baseline on the right to reuse it across sessions."}
            </p>
          )}

          <div className={styles.list}>
            {visible.map(entry => (
              <article key={entry.id} className={styles.entry}>
                <div className={styles.entryMain}>
                  <div className={styles.entryInfo}>
                    <h4>
                      <button type="button" className={styles.titleLink} disabled={busy} onClick={() => setOpened(entry.id)}>
                        {entry.process}
                      </button>
                    </h4>
                    <p>
                      {entry.repository} · {entry.role || "No role"} · L{entry.level} · <code>{entry.version}</code>
                      {entry.findings.length === 0 && <span className={styles.cleanTag}> · Valid</span>}
                    </p>
                  </div>
                  <div className={styles.actions}>
                    <button type="button" aria-label="View process and details" disabled={busy} onClick={() => setOpened(entry.id)}>Details</button>
                    <button type="button" aria-label="Use as baseline" disabled={busy} onClick={() => use(entry.id, "baseline")}>Baseline</button>
                    <button type="button" aria-label="Use as changed BPMN" disabled={busy || !hasBaseline} onClick={() => use(entry.id, "changed")}>Changed</button>
                    <button type="button" aria-label="Download BPMN" title="Download BPMN" disabled={busy} onClick={() => use(entry.id, "download")}>↓</button>
                  </div>
                </div>
                {entry.findings.length > 0 && (
                  <details className={styles.checks}>
                    <summary>{entry.findings.length} modelling check{entry.findings.length === 1 ? "" : "s"}</summary>
                    {entry.findings.map((f, i) => (
                      <p key={i}>
                        <strong>{f.code}</strong>
                        {f.element_id && <code> {f.element_id}</code>} — {f.message}
                      </p>
                    ))}
                  </details>
                )}
              </article>
            ))}
          </div>
        </div>

        <form className={styles.form} onSubmit={store}>
          <h3>Save a diagram</h3>
          <label>
            Diagram source
            <select value={source} onChange={e => setSource(e.target.value)}>
              <option value="upload">Upload a file</option>
              <option value="baseline" disabled={!baseline}>Current baseline</option>
              <option value="changed" disabled={!edited}>Current changed BPMN</option>
            </select>
          </label>
          {source === "upload" && (
            <label>
              BPMN file
              <input type="file" accept=".bpmn,.xml" onChange={e => setUpload(e.target.files?.[0] ?? null)} required />
            </label>
          )}
          {selected && <small className={styles.fileChosen}>{selected.name}</small>}
          <label>
            Repository label
            <input name="repository" required maxLength={160} placeholder="Simple-ERP" />
          </label>
          <label>
            Process name
            <input name="process" required maxLength={160} placeholder="Submit stock adjustment" />
          </label>
          <label>
            Role (optional)
            <input name="role" maxLength={160} placeholder="Staff" />
          </label>
          <div className={styles.row}>
            <label>
              Process level
              <select name="level" defaultValue="3">
                {[0, 1, 2, 3].map(n => (
                  <option key={n} value={n}>Level {n}</option>
                ))}
              </select>
            </label>
            <label>
              Version label
              <input name="version" required maxLength={160} placeholder="baseline-v1" />
            </label>
          </div>
          <button type="submit" disabled={busy || !selected}>
            {busy ? "Working…" : "Save diagram"}
          </button>
        </form>
      </div>
    </section>
  );
}
