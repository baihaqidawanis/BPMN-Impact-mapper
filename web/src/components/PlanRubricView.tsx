"use client";

import { useState } from "react";
import styles from "./PlanRubricView.module.css";

type ScenarioKey = "skip" | "redo" | "switch" | "nfc";

interface RubricCriterion {
  id: number;
  name: string;
  points: number;
  summary: string;
}

interface ScenarioScore {
  score: string;
  note: string;
}

interface ScenarioRun {
  run: string;
  planId: string;
  score: string;
  recall: string;
  status: string;
}

interface ScenarioData {
  key: ScenarioKey;
  label: string;
  latestPlan?: string;
  latestTotal?: string;
  summaryNote: string;
  scores?: Record<number, ScenarioScore>;
  runs?: ScenarioRun[];
}

const CRITERIA: RubricCriterion[] = [
  { id: 1, name: "Business predicate", points: 10, summary: "Matches the BPMN guard and evaluates data at the right granularity (e.g., per line, not an aggregate sum)." },
  { id: 2, name: "Branch outcomes", points: 10, summary: "Names the exact result and side effects for every changed path." },
  { id: 3, name: "Enforcement", points: 10, summary: "Preserves authorization, validation, persistence, and transaction boundaries." },
  { id: 4, name: "Existing behavior", points: 10, summary: "Keeps unaffected paths intact and explains which legacy behavior still applies." },
  { id: 5, name: "Current behavior", points: 10, summary: "Describes actual code behavior backed by verifiable source evidence." },
  { id: 6, name: "Implementation ownership", points: 10, summary: "Identifies the real runtime code path and owning function or component." },
  { id: 7, name: "Prerequisites", points: 5, summary: "Accounts for required schema migrations, API contracts, or config dependencies." },
  { id: 8, name: "Success checks", points: 10, summary: "Defines runnable acceptance cases with concrete inputs and observable outputs." },
  { id: 9, name: "Failure & regression checks", points: 10, summary: "Covers invalid inputs, edge cases (+delta/-delta), and existing regression tests." },
  { id: 10, name: "Implementation steps", points: 10, summary: "Orders concrete edits and dependencies so a developer can implement directly." },
  { id: 11, name: "Internal consistency", points: 5, summary: "Uses the same predicates, outcomes, and assumptions across all sections." },
];

const SCENARIOS: ScenarioData[] = [
  {
    key: "skip",
    label: "Skip",
    latestPlan: "637b4c44",
    latestTotal: "60 / 100*",
    summaryNote: "*Strict independent review (GPT-6.2); prior baseline was 20-37.5 / 100.",
    scores: {
      1: { score: "5 / 10", note: "Per-line check present, but queries stock_movements instead of inventory_adjustment_lines." },
      2: { score: "10 / 10", note: "All-zero -> NO_ADJUSTMENT (skip approval); nonzero -> SUBMITTED; invalid -> 409." },
      3: { score: "5 / 10", note: "Version & audit named, but lock -> validate -> persist order is not fully traced." },
      4: { score: "5 / 10", note: "SUBMITTED path kept, but extra edits on Adjustment.Complete and idempotency test lack proof." },
      5: { score: "5 / 10", note: "Accurately cites TransitionAdjustment (120-138), with minor inaccuracies on legacy test claims." },
      6: { score: "5 / 10", note: "Finds Store.TransitionAdjustment, but duplicates logic onto test-only Adjustment.Complete." },
      7: { score: "2.5 / 5", note: "Adds forward migration step, but uses wrong table name (adjustments) and taken number 000006." },
      8: { score: "10 / 10", note: "Verifies status, version increment, audit entry, skipped approval, and no stock posting." },
      9: { score: "5 / 10", note: "Includes stale-version 409 check, but omits explicit opposing nonzero (+2 / -2) case." },
      10: { score: "5 / 10", note: "Ordered end-to-end, but line query table and migration target need correction first." },
      11: { score: "2.5 / 5", note: "Branch outcomes align, but data source and owning layer differ between sections." },
    },
    runs: [
      { run: "Run 1", planId: "481a2859", score: "20 / 100", recall: "100% (2/2)", status: "Critical error" },
      { run: "Run 2", planId: "96bdb04a", score: "25 / 100", recall: "100% (2/2)", status: "Critical error" },
      { run: "Run 3", planId: "e2d734bd", score: "65 / 100", recall: "50% (1/2)", status: "Critical error (store withheld)" },
      { run: "Run 4 (Latest)", planId: "637b4c44", score: "60 / 100*", recall: "100% (2/2 doc)", status: "Needs table + migration fix" },
    ],
  },
  {
    key: "redo",
    label: "Redo",
    summaryNote: "Not evaluated yet. Run Ultra Light on the Redo BPMN pair to record its 11-criterion rubric score.",
  },
  {
    key: "switch",
    label: "Switch",
    summaryNote: "Not evaluated yet. Run Ultra Light on the Switch BPMN pair to record its 11-criterion rubric score.",
  },
  {
    key: "nfc",
    label: "Non-Free-Choice",
    summaryNote: "Not evaluated yet. Run Ultra Light on the Non-Free-Choice (NFC) BPMN pair to record its 11-criterion rubric score.",
  },
];

export function PlanRubricView() {
  const [active, setActive] = useState<ScenarioKey>("skip");
  const scenario = SCENARIOS.find(item => item.key === active) ?? SCENARIOS[0];

  return (
    <section className={styles.wrap} data-testid="plan-rubric-view">
      <header className={styles.header}>
        <div>
          <h2>Change Plan Markdown Rubric</h2>
          <p>
            11-criterion evaluation framework from <code>docs/METHODOLOGY.md</code>. Passing target is{" "}
            <strong>&ge; 80 / 100</strong> with <strong>zero critical errors</strong>, scored by two independent
            reviewers and reconciled.
          </p>
        </div>
        <div className={styles.metrics}>
          <div>
            <span>Target score</span>
            <strong>&ge; 80 / 100</strong>
          </div>
          <div>
            <span>{scenario.label}{scenario.latestPlan ? ` (${scenario.latestPlan})` : ""}</span>
            <strong>{scenario.latestTotal ?? "Pending"}</strong>
          </div>
          <div>
            <span>Critical errors</span>
            <strong>Must be 0</strong>
          </div>
        </div>
      </header>

      <div className={styles.scenarioTabs} role="tablist" aria-label="Evaluation scenarios">
        {SCENARIOS.map(item => (
          <button
            key={item.key}
            type="button"
            role="tab"
            aria-selected={item.key === active}
            className={item.key === active ? styles.scenarioTabActive : styles.scenarioTab}
            onClick={() => setActive(item.key)}
          >
            {item.label}
            {item.latestTotal ? <span className={styles.tabMeta}>{item.latestTotal}</span> : <span className={styles.tabMeta}>Pending</span>}
          </button>
        ))}
      </div>

      <p className={styles.ruleNote}>
        <strong>Critical error gate:</strong> Any plan that reverses a business outcome, allows unauthorized mutation,
        reports success before a required write/commit, or omits a blocking schema prerequisite fails regardless of
        numeric score.
      </p>

      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>#</th>
              <th>Criterion</th>
              <th className={styles.num}>Max</th>
              <th>What earns full points</th>
              <th className={styles.num}>{scenario.label}{scenario.latestPlan ? ` (${scenario.latestPlan})` : ""}</th>
              <th>Assessment note</th>
            </tr>
          </thead>
          <tbody>
            {CRITERIA.map(item => {
              const scored = scenario.scores?.[item.id];
              return (
                <tr key={item.id}>
                  <td className={styles.idx}>{item.id}</td>
                  <th scope="row">{item.name}</th>
                  <td className={styles.num}>{item.points}</td>
                  <td>{item.summary}</td>
                  <td className={styles.num}>{scored?.score ?? "—"}</td>
                  <td className={styles.note}>{scored?.note ?? "Not evaluated yet."}</td>
                </tr>
              );
            })}
          </tbody>
          <tfoot>
            <tr>
              <th colSpan={2} scope="row">Total</th>
              <td className={styles.num}>100</td>
              <td>Full / half / zero points per criterion</td>
              <td className={styles.num}>{scenario.latestTotal ?? "—"}</td>
              <td className={styles.note}>{scenario.summaryNote}</td>
            </tr>
          </tfoot>
        </table>
      </div>

      {scenario.runs ? (
        <section className={styles.history}>
          <h3>{scenario.label} iteration history</h3>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Iteration</th>
                  <th>Plan ID</th>
                  <th className={styles.num}>Rubric score</th>
                  <th className={styles.num}>File recall</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {scenario.runs.map(row => (
                  <tr key={row.planId}>
                    <th scope="row">{row.run}</th>
                    <td><code>{row.planId}</code></td>
                    <td className={styles.num}>{row.score}</td>
                    <td className={styles.num}>{row.recall}</td>
                    <td className={styles.note}>{row.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : (
        <p className={styles.emptyHistory}>{scenario.summaryNote}</p>
      )}
    </section>
  );
}
