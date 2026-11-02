"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getAnalysis } from "./api";
import type { Report } from "./types";

// Only the selection is in the URL; reports remain in backend storage.
export function useSavedAnalysis() {
  const [report, setValue] = useState<Report | null>(null);
  const [restoring, setRestoring] = useState(false);
  const [restoreNotice, setRestoreNotice] = useState("");
  const pending = useRef<AbortController | null>(null);
  const setReport = useCallback((value: Report | null) => {
    pending.current?.abort();
    pending.current = null;
    setRestoring(false);
    setRestoreNotice("");
    setValue(value);
    const url = new URL(window.location.href);
    if (value) url.searchParams.set("analysis", value.analysis_id);
    else url.searchParams.delete("analysis");
    window.history.replaceState(window.history.state, "", url);
  }, []);

  useEffect(() => {
    const id = new URL(window.location.href).searchParams.get("analysis");
    if (!id) return;
    if (!/^[a-f0-9]{32}$/.test(id)) {
      setRestoreNotice("This saved report link is invalid. Open a report from Saved analyses.");
      return;
    }
    const controller = new AbortController();
    pending.current = controller;
    setRestoring(true);
    void getAnalysis(id, controller.signal).then(saved => {
      if (controller.signal.aborted) return;
      setValue(saved);
      setRestoreNotice("Saved report reopened. Reselect BPMN files only when running a new analysis.");
    }).catch(() => {
      if (!controller.signal.aborted) setRestoreNotice("The saved report could not be reopened. Check the backend or open it from Saved analyses.");
    }).finally(() => {
      if (!controller.signal.aborted) { pending.current = null; setRestoring(false); }
    });
    return () => controller.abort();
  }, []);
  return { report, setReport, restoring, restoreNotice };
}
