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
    latestPlan: "7059db9f",
    latestTotal: "75 / 100",
    summaryNote: "Reconciled 2-reviewer strict score for 7059db9f (GPT 75 / Gemini 75; 0 critical errors; doc-wide 100% P / 100% R; 0 domain literals). Best Skip baseline.",
    scores: {
      1: { score: "5 / 10", note: "Per-line check & +2/-2 -> SUBMITTED correct; lines 34/56/78 still query stock movements (caught by validator at line 142)." },
      2: { score: "10 / 10", note: "All-zero -> NO_ADJUSTMENT (skip approval); +2/-2 -> SUBMITTED; stale/invalid precondition -> 409." },
      3: { score: "5 / 10", note: "DRAFT + version guard, audit & commit in one transaction; in-tx query source still names stock movements." },
      4: { score: "10 / 10", note: "Submit -> approval -> deferred stock posting sequence preserved cleanly across all sections." },
      5: { score: "5 / 10", note: "Lines 48/70 accurately describe TransitionAdjustment (120-138); line 26 still says it queries stock movements." },
      6: { score: "10 / 10", note: "Exact 2 runtime files only (store.go + operations_test.go); zero extra files." },
      7: { score: "2.5 / 5", note: "Specifies 000013_inventory_adjustment_status.up.sql on inventory_adjustments.status + rollback script; existing NO_ADJUSTMENT row handling not detailed." },
      8: { score: "10 / 10", note: "Concrete runnable test function TestInventoryApprovalZeroDeltaBypass and observable outputs across all branches." },
      9: { score: "10 / 10", note: "Adds separate zero-delta & +2/-2 cases and explicitly keeps TestInventoryApprovalRejectsChangedSnapshot assertions unchanged (line 105)." },
      10: { score: "5 / 10", note: "Ordered end-to-end with 000013 migration, audit/commit, and regression-safe tests; query step still needs inventory_adjustment_lines." },
      11: { score: "2.5 / 5", note: "Outcomes and tests align cleanly; minor mismatch between line 32 (counted_quantity vs snapshot_quantity) and line 34 (stock movements)." },
    },
    runs: [
      { run: "Run 1", planId: "481a2859", score: "20 / 100", recall: "100% (2/2)", status: "Critical error" },
      { run: "Run 2", planId: "96bdb04a", score: "25 / 100", recall: "100% (2/2)", status: "Critical error" },
      { run: "Run 3", planId: "e2d734bd", score: "65 / 100", recall: "50% (1/2)", status: "Critical error (store withheld)" },
      { run: "Run 4", planId: "637b4c44", score: "60 / 100*", recall: "100% (2/2 doc)", status: "Needs table + migration fix" },
      { run: "Run 5", planId: "9ca98b59", score: "55 / 100*", recall: "100% (2/2 case & doc)", status: "ZeroDeltaDecision 100% P/R; missing migration" },
      { run: "Run 6", planId: "27d7b37e", score: "52.5 / 100*", recall: "100% (2/2 doc)", status: "Doc 100% P/R; +2/-2 inverted & stock_balances" },
      { run: "Run 7", planId: "9a3ee1f2", score: "65 / 100*", recall: "100% (2/2 doc)", status: "Reconciled (GPT 65 / Gemini 70); 000013 migration; clean MD" },
      { run: "Run 8", planId: "8379adab", score: "70 / 100*", recall: "100% (2/2 doc)", status: "Reconciled (GPT 70); deferred posting fixed" },
      { run: "Run 9 (Latest)", planId: "7059db9f", score: "75 / 100*", recall: "100% (2/2 doc)", status: "Reconciled (GPT 75 / Gemini 75); 0 domain literals; regression tests preserved" },
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
