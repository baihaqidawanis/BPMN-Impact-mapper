"use client";

import { useEffect, useRef, useState } from "react";
import type Modeler from "bpmn-js/lib/Modeler";
import styles from "./BpmnEditor.module.css";

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
  const modeler = useRef<Modeler | null>(null);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (active && ready) modeler.current?.get<{ resized: () => void }>("canvas").resized();
  }, [active, ready]);

  useEffect(() => {
    let disposed = false;
    let instance: Modeler | undefined;
    async function load() {
      try {
        const { default: BpmnModeler } = await import("bpmn-js/lib/Modeler");
        if (disposed || !container.current) return;
        instance = new BpmnModeler({ container: container.current });
        modeler.current = instance;
        const { warnings } = await instance.importXML(await initial.text());
        if (disposed) return;
        instance.get<{ zoom: (value: string) => void }>("canvas").zoom("fit-viewport");
        setNotice(warnings.length ? `${warnings.length} import warnings. Check missing shapes or connections before applying.` : "Double-click to rename. Use the wrench to choose User Task or Service Task.");
        instance.on("commandStack.changed", () => { if (!disposed) { setDirty(true); onDirty(true); } });
        setReady(true);
      } catch (cause) {
        if (!disposed) setError(cause instanceof Error ? cause.message : "Diagram could not be opened. Check XML and BPMN diagram coordinates.");
      }
    }
    void load();
    return () => { disposed = true; instance?.destroy(); modeler.current = null; };
  }, [initial, onDirty]);

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
      modeler.current?.get<{ zoom: (value: string) => void }>("canvas").zoom("fit-viewport");
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

  return <div role="region" aria-label="Edit new BPMN">
    <section className={styles.editor}>
      <header><div><h2>Edit new BPMN</h2><p>Baseline: {baseline.name} · original preserved</p></div><button type="button" onClick={close}>Back to mapping</button></header>
      <nav aria-label="Diagram tools">
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ undo: () => void }>("commandStack").undo()}>Undo</button>
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ redo: () => void }>("commandStack").redo()}>Redo</button>
        <button type="button" disabled={!ready} onClick={() => zoom(1.25)}>Zoom in</button>
        <button type="button" disabled={!ready} onClick={() => zoom(0.8)}>Zoom out</button>
        <button type="button" disabled={!ready} onClick={() => modeler.current?.get<{ zoom: (value: string) => void }>("canvas").zoom("fit-viewport")}>Fit diagram</button>
        <button type="button" disabled={!ready} onClick={() => void reset()}>Reset to baseline</button>
        <button type="button" disabled={!ready} onClick={() => void download()}>Download BPMN</button>
        <button className={styles.primary} type="button" disabled={!ready} onClick={() => void apply()}>Use edited BPMN</button>
      </nav>
      {error && <p role="alert" className={styles.error}>{error}</p>}
      <p role="status">{dirty ? "Unapplied changes. " : ""}{notice || "Opening diagram…"}</p>
      <div ref={container} className={styles.canvas} data-testid="bpmn-editor-canvas" />
      <footer>Edits and undo history stay here when switching tabs. Use edited BPMN to send this version to Mapping. Download before refreshing.</footer>
    </section>
  </div>;
}
