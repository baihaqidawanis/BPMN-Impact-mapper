"use client";

import { useEffect, useRef, useState } from "react";
import type Modeler from "bpmn-js/lib/Modeler";
import type NavigatedViewer from "bpmn-js/lib/NavigatedViewer";
import { compareBpmnVisuals, type VisualDiff } from "@/lib/bpmnVisualDiff";
import styles from "./BpmnEditor.module.css";

type MarkerCanvas = {
  addMarker: (id: string, marker: string) => void;
  removeMarker: (id: string, marker: string) => void;
};

const emptyDiff: VisualDiff = { added: [], removed: [], modified: [] };

interface Props {
  baseline: File;
  initial: File;
  onApply: (file: File) => void;
  onClose: () => void;
  active: boolean;
  onDirty: (value: boolean) => void;
}

export function BpmnEditor({ baseline, initial, onApply, onClose, active, onDirty }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const beforeContainer = useRef<HTMLDivElement>(null);
  const modeler = useRef<Modeler | null>(null);
  const viewer = useRef<NavigatedViewer | null>(null);
  const refreshPreview = useRef<(() => Promise<void>) | null>(null);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dirty, setDirty] = useState(false);
  const [diff, setDiff] = useState<VisualDiff>(emptyDiff);
  const [focusAfter, setFocusAfter] = useState(false);
  const [showMarkers, setShowMarkers] = useState(true);

  useEffect(() => {
    if (active && ready) {
      const afterCanvas = modeler.current?.get<{ resized: () => void; zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas");
      const beforeCanvas = viewer.current?.get<{ resized: () => void; zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas");
      afterCanvas?.resized();
      beforeCanvas?.resized();
      afterCanvas?.zoom("fit-viewport", { x: 0, y: 0 });
      if (!focusAfter) beforeCanvas?.zoom("fit-viewport", { x: 0, y: 0 });
    }
  }, [active, ready, focusAfter]);

  useEffect(() => {
    let disposed = false;
    let instance: Modeler | undefined;
    let before: NavigatedViewer | undefined;
    let version = 0;
    let oldMarkers: Array<[string, string]> = [];
    let newMarkers: Array<[string, string]> = [];
    let baselineXml = "";
    function mark(currentXml: string) {
      if (!instance || !before || disposed) return;
      const changes = compareBpmnVisuals(baselineXml, currentXml);
      const beforeCanvas = before.get<MarkerCanvas>("canvas");
      const afterCanvas = instance.get<MarkerCanvas>("canvas");
      const beforeRegistry = before.get<{ get: (id: string) => unknown }>("elementRegistry");
      const afterRegistry = instance.get<{ get: (id: string) => unknown }>("elementRegistry");
      for (const [id, marker] of oldMarkers) if (beforeRegistry.get(id)) beforeCanvas.removeMarker(id, marker);
      for (const [id, marker] of newMarkers) if (afterRegistry.get(id)) afterCanvas.removeMarker(id, marker);
      oldMarkers = [...changes.removed.map(id => [id, "visual-removed"] as [string, string]), ...changes.modified.map(id => [id, "visual-modified"] as [string, string])];
      newMarkers = [...changes.added.map(id => [id, "visual-added"] as [string, string]), ...changes.modified.map(id => [id, "visual-modified"] as [string, string])];
      for (const [id, marker] of oldMarkers) if (beforeRegistry.get(id)) beforeCanvas.addMarker(id, marker);
      for (const [id, marker] of newMarkers) if (afterRegistry.get(id)) afterCanvas.addMarker(id, marker);
      setDiff(changes);
    }
    async function refresh() {
      const current = ++version;
      const { xml } = await instance?.saveXML({ format: false }) ?? {};
      if (xml && current === version && !disposed) mark(xml);
    }
    refreshPreview.current = refresh;
    async function load() {
      try {
        const [{ default: BpmnModeler }, { default: BpmnNavigatedViewer }] = await Promise.all([import("bpmn-js/lib/Modeler"), import("bpmn-js/lib/NavigatedViewer")]);
        if (disposed || !container.current || !beforeContainer.current) return;
        baselineXml = await baseline.text();
        before = new BpmnNavigatedViewer({ container: beforeContainer.current });
        viewer.current = before;
        await before.importXML(baselineXml);
        instance = new BpmnModeler({ container: container.current });
        modeler.current = instance;
        const { warnings } = await instance.importXML(await initial.text());
        if (disposed) return;
        await refresh();
        before.get<{ zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas").zoom("fit-viewport", { x: 0, y: 0 });
        instance.get<{ zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas").zoom("fit-viewport", { x: 0, y: 0 });
        setNotice(warnings.length ? `${warnings.length} import warnings. Check missing shapes or connections before applying.` : "Double-click to rename. Use the wrench to choose User Task or Service Task.");
        instance.on("commandStack.changed", () => { if (!disposed) { setDirty(true); onDirty(true); void refresh().catch(() => setError("Change preview could not be refreshed.")); } });
        setReady(true);
      } catch (cause) {
        if (!disposed) setError(cause instanceof Error ? cause.message : "Diagram could not be opened. Check XML and BPMN diagram coordinates.");
      }
    }
    void load();
    return () => { disposed = true; refreshPreview.current = null; instance?.destroy(); before?.destroy(); modeler.current = null; viewer.current = null; };
  }, [baseline, initial, onDirty]);

  async function exportFile() {
    if (!modeler.current || !ready) throw new Error("Editor is not ready.");
    const { xml } = await modeler.current.saveXML({ format: true });
    if (!xml) throw new Error("No BPMN XML was produced.");
    return new File([xml], "edited-process.bpmn", { type: "application/xml" });
  }

  async function apply() {
    try { onApply(await exportFile()); setDirty(false); onDirty(false); } catch (cause) { setError(String(cause)); }
  }

  async function download() {
    try {
      const file = await exportFile();
      const url = URL.createObjectURL(file);
      const link = document.createElement("a"); link.href = url; link.download = file.name; link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (cause) { setError(String(cause)); }
  }

  async function reset() {
    if (!window.confirm("Reset this draft to the original baseline?")) return;
    setReady(false);
    try {
      await modeler.current?.importXML(await baseline.text());
      modeler.current?.get<{ zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas").zoom("fit-viewport", { x: 0, y: 0 });
      await refreshPreview.current?.();
      setDirty(true); onDirty(true); setError(""); setReady(true);
    } catch (cause) { setError(String(cause)); }
  }

  function close() {
    onClose();
  }

  function zoom(factor: number) {
    const canvas = modeler.current?.get<{ zoom: (value?: number) => number }>("canvas");
    if (canvas) canvas.zoom(Math.max(0.1, Math.min(4, canvas.zoom() * factor)));
  }

  function zoomBefore(factor: number) {
    const canvas = viewer.current?.get<{ zoom: (value?: number) => number }>("canvas");
    if (canvas) canvas.zoom(Math.max(0.1, Math.min(4, canvas.zoom() * factor)));
  }

  return <div role="region" aria-label="Edit new BPMN">
    <section className={styles.editor}>
      <header><div><h2>Compare BPMN versions</h2><p>Original preserved · edit only the new version</p></div><button type="button" onClick={close}>Back to mapping</button></header>
      <nav aria-label="Diagram tools">
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ undo: () => void }>("commandStack").undo()}>Undo</button>
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ redo: () => void }>("commandStack").redo()}>Redo</button>
        <button type="button" disabled={!ready} onClick={() => zoom(1.25)}>Zoom in</button>
        <button type="button" disabled={!ready} onClick={() => zoom(0.8)}>Zoom out</button>
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas").zoom("fit-viewport", { x: 0, y: 0 })}>Fit diagram</button>
        <button type="button" disabled={!ready} onClick={() => zoomBefore(1.25)}>Before +</button>
        <button type="button" disabled={!ready} onClick={() => zoomBefore(0.8)}>Before −</button>
        <button type="button" disabled={!ready} onClick={() => viewer.current?.get<{ zoom: (value: string, center?: { x: number; y: number }) => void }>("canvas").zoom("fit-viewport", { x: 0, y: 0 })}>Fit before</button>
        <button type="button" disabled={!ready} onClick={() => void reset()}>Reset to baseline</button>
        <button type="button" disabled={!ready} onClick={() => void download()}>Download BPMN</button>
        <button className={styles.primary} type="button" disabled={!ready} onClick={() => void apply()}>Use edited BPMN</button>
      </nav>
      {error && <p role="alert" className={styles.error}>{error}</p>}
      <p role="status">{dirty ? "Unapplied changes. " : ""}{notice || "Opening diagrams…"}</p>
      <div className={styles.legend} aria-label="Visual change preview"><span><i className={styles.added} />Added {diff.added.length}</span><span><i className={styles.removed} />Removed {diff.removed.length}</span><span><i className={styles.modified} />Changed {diff.modified.length}</span><small>Quick visual preview by ID, label, lane and flow. The backend report is the authoritative semantic diff.</small><button type="button" aria-pressed={showMarkers} onClick={() => setShowMarkers(value => !value)}>{showMarkers ? "Hide markers" : "Show markers"}</button></div>
      <div className={`${styles.panels} ${focusAfter ? styles.focusAfter : ""} ${showMarkers ? styles.showMarkers : ""}`}>
        <div className={styles.pane}><div className={styles.paneTitle}><b>Before</b><span>{baseline.name} · view only</span></div><div ref={beforeContainer} className={styles.canvas} data-testid="bpmn-baseline-canvas" /></div>
        <div className={styles.pane}><div className={styles.paneTitle}><b>After</b><span>{initial.name} · editable</span><button type="button" onClick={() => setFocusAfter(value => !value)}>{focusAfter ? "Show before" : "Expand editor"}</button></div><div ref={container} className={styles.canvas} data-testid="bpmn-editor-canvas" /></div>
      </div>
      <footer>Edits and undo history stay here when switching tabs. Use edited BPMN to send this version to Mapping. Download before refreshing.</footer>
    </section>
  </div>;
}
