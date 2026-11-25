"use client";

import { useEffect, useState } from "react";
import { applyAgentSettings, getAgentSettings, getAgentModels, type AgentSettings as Settings } from "@/lib/agents";
import styles from "./AgentSettings.module.css";

export function AgentSettings({
  disabled,
  onApplied,
  onDirty,
}: {
  disabled: boolean;
  onApplied: () => void;
  onDirty: (dirty: boolean) => void;
}) {
  const [open, setOpen] = useState(false);
  const [catalogError, setCatalogError] = useState("");
  const [loadingModels, setLoadingModels] = useState(false);
  const [models, setModels] = useState<string[]>([]);
  const [saved, setSaved] = useState<Settings | null>(null);
  const [draft, setDraft] = useState<Settings | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let live = true;
    void getAgentSettings()
      .then(v => {
        if (live) {
          setSaved(v);
          setDraft(v);
        }
      })
      .catch(() => {
        if (live) setError("Model settings are unavailable. Restart the updated backend with --agent-model.");
      });
    return () => {
      live = false;
    };
  }, []);

  useEffect(() => {
    if (disabled) return;
    let live = true;
    setLoadingModels(true);
    void getAgentModels()
      .then(v => {
        if (live) {
          setModels(v.models);
          setCatalogError("");
        }
      })
      .catch(() => {
        if (live) setCatalogError("Model list unavailable. Check the tunnel and refresh the list after the review finishes.");
      })
      .finally(() => {
        if (live) setLoadingModels(false);
      });
    return () => {
      live = false;
    };
  }, [disabled]);

  useEffect(() => {
    if (!open) return;
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open]);

  async function refreshModels() {
    setLoadingModels(true);
    try {
      const v = await getAgentModels();
      setModels(v.models);
      setCatalogError("");
    } catch {
      setCatalogError("Model list unavailable. Check the tunnel and try again.");
    } finally {
      setLoadingModels(false);
    }
  }

  const dirty = JSON.stringify(saved) !== JSON.stringify(draft);
  useEffect(() => {
    onDirty(dirty || saving);
  }, [dirty, saving, onDirty]);

  async function apply() {
    if (!draft) return;
    setSaving(true);
    setError("");
    setNotice("");
    try {
      const v = await applyAgentSettings(draft);
      setDraft(v);
      setSaved(v);
      setNotice("Applied. Start a new AI session. Existing checkpoints keep their original model identity.");
      onApplied();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Settings were not applied.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className={styles.settings}>
      <div className={styles.triggerRow}>
        <div>
          <span className={styles.triggerTitle}>Model settings</span>
          <small className={styles.triggerMeta}>
            {saved ? `${saved.context.toLocaleString()} ctx · ${saved.output.toLocaleString()} out` : "Local Ollama"}
          </small>
        </div>
        <button
          type="button"
          className={styles.triggerBtn}
          aria-expanded={open}
          onClick={() => setOpen(true)}
        >
          {dirty ? "Unsaved changes" : "Configure"}
        </button>
      </div>
      {dirty && !open && <p className={styles.warnInline}>Unapplied model settings. Open to apply or discard.</p>}

      {open && (
        <div
          className={styles.backdrop}
          role="presentation"
          onClick={event => {
            if (event.target === event.currentTarget) setOpen(false);
          }}
        >
          <div className={styles.modal} role="dialog" aria-modal="true" aria-labelledby="agent-settings-title">
            <header className={styles.modalHead}>
              <div>
                <h3 id="agent-settings-title">Model settings</h3>
                <p>Applies to this running backend only. Restarting restores command-line defaults.</p>
              </div>
              <button type="button" className={styles.closeBtn} aria-label="Close model settings" onClick={() => setOpen(false)}>
                <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
                  <path d="M5 5l10 10M15 5L5 15" />
                </svg>
              </button>
            </header>

            <div className={styles.modalBody}>
              <div className={styles.catalog}>
                <span>
                  {loadingModels
                    ? "Loading installed models…"
                    : models.length
                      ? `${models.length} local models available`
                      : "Current configuration shown"}
                </span>
                <button
                  type="button"
                  className={styles.textBtn}
                  disabled={disabled || saving || loadingModels}
                  onClick={() => void refreshModels()}
                >
                  Refresh model list
                </button>
              </div>
              {catalogError && <p className={styles.statusMsg} role="status">{catalogError}</p>}

              {draft && (
                <fieldset disabled={disabled || saving}>
                  <legend className={styles.srOnly}>Agent generation settings</legend>
                  <div className={styles.roles}>
                    {(["analyst", "planner", "reviewer"] as const).map(role => (
                      <div key={role} className={styles.role}>
                        <h4>{role}</h4>
                        <label>
                          Model
                          <select
                            aria-label={`${role} model`}
                            value={draft[role].model}
                            onChange={e => setDraft({ ...draft, [role]: { ...draft[role], model: e.target.value } })}
                          >
                            {[...new Set([draft[role].model, ...models])].filter(Boolean).map(name => (
                              <option key={name} value={name}>
                                {name}
                                {!models.includes(name) ? " (current)" : ""}
                              </option>
                            ))}
                          </select>
                        </label>
                        <label>
                          Thinking
                          <select
                            aria-label={`${role} thinking`}
                            value={draft[role].thinking ? "on" : "off"}
                            onChange={e => setDraft({ ...draft, [role]: { ...draft[role], thinking: e.target.value === "on" } })}
                          >
                            <option value="off">Off</option>
                            <option value="on">On</option>
                          </select>
                        </label>
                        <label>
                          Generation profile
                          <select
                            aria-label={`${role} profile`}
                            value={draft[role].profile}
                            onChange={e =>
                              setDraft({ ...draft, [role]: { ...draft[role], profile: e.target.value as "baseline" | "coding" } })
                            }
                          >
                            <option value="baseline">Baseline</option>
                            <option value="coding">Coding</option>
                          </select>
                        </label>
                      </div>
                    ))}
                  </div>

                  <div className={styles.budgets}>
                    <label>
                      Context tokens
                      <input
                        aria-label="Agent context tokens"
                        type="number"
                        min={4096}
                        max={32768}
                        value={draft.context}
                        onChange={e => setDraft({ ...draft, context: Number(e.target.value) })}
                      />
                    </label>
                    <label>
                      Output tokens
                      <input
                        aria-label="Agent output tokens"
                        type="number"
                        min={32}
                        max={8192}
                        value={draft.output}
                        onChange={e => setDraft({ ...draft, output: Number(e.target.value) })}
                      />
                    </label>
                  </div>
                  <p className={styles.hint}>
                    Output includes thinking tokens. Higher context uses more GPU memory; settings do not guarantee accuracy.
                  </p>
                </fieldset>
              )}

              {disabled && <p className={styles.statusMsg}>Settings are locked while a review is running.</p>}
              {dirty && <p className={styles.warnInline}>Unapplied changes. Apply or discard before starting a review.</p>}
              {notice && <p className={styles.okMsg} role="status">{notice}</p>}
              {error && <p className={styles.errMsg} role="alert">{error}</p>}
            </div>

            <footer className={styles.modalFoot}>
              <button
                type="button"
                className={styles.secondaryBtn}
                disabled={!dirty}
                onClick={() => {
                  setDraft(saved);
                  setError("");
                  setNotice("");
                }}
              >
                Discard changes
              </button>
              <button type="button" className={styles.primaryBtn} disabled={!dirty || disabled || saving} onClick={() => void apply()}>
                {saving ? "Checking models…" : "Apply settings"}
              </button>
            </footer>
          </div>
        </div>
      )}
    </div>
  );
}
