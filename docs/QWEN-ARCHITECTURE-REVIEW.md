# Qwen integration review - 2026-11-03

## Current input and execution

New web analyses retain the original uploaded before/after XML privately alongside the report. Reopening restores the pair in Canvas. Planning receives the parsed full process structures (elements, actors, process/parent IDs, gateway properties, conditions, flows and adjacency), not BPMN DI coordinates or a screenshot. Unchanged process elements remain available as context. Canonical shareable impact JSON excludes the private XML/graph snapshot. Source and BPMN are untrusted data, not instructions.

Older reports without retained XML cannot reconstruct the complete process from a diff. Upload both originals and run mapping again. Never attach a guessed pair to a historical report.

Full review sends each approved source chunk against all changed elements, retaining per-change evidence. Hybrid is an initial lead, not a final decision. No secret, ignored dependency, build artifact or excluded evaluation corpus should enter inference. Context overflow is rejected explicitly; full process context can exceed the current model request budget for large inputs. This is not solved by silently dropping nodes.

## Three roles

1. Analyst: read approved source, classify relevance per change and cite actual source evidence. Distinguish observed behavior from assumptions.
2. Planner: propose developer steps, intended behavior and acceptance checks while retaining file/change evidence and uncertainty. Produce a proposal, never execute or edit source.
3. Reviewer: check evidence, missing guards, contradictions and specificity; request bounded revisions. Same-model review is not independent validation.

These are sequential agent roles coordinated by a Go state machine with checkpointed tool actions and bounded revisions. They share one configured Qwen model. No separate model instances or LangGraph dependency are required by this architecture. Full-scan classification has no source tools because complete chunks are supplied; later planning can request approved source evidence.

## Plan format

Canonical plan JSON drives Markdown: identity and model version; status; source coverage; review issues; per-change coverage; recommendations grouped by file while preserving each change identity, layer and action; cited evidence; proposed behavior; implementation steps; acceptance checks. Keep recommendations for separate changes traceable even when they concern the same file. COMPLETE means protocol review finished, not proven correctness or executed tests.

## Remaining gaps

- Reviewer now checks the assembled plan after batch reviews. Its checkpoint preserves an approved/issues result; transport failures can resume and invalid output is limited to three attempts. Conflicts produce NEEDS_REVIEW rather than automatic global rewrites. This remains model review, not independent proof of correctness.
- The full scan can be expensive, and all-change structured responses may exceed budgets. Measure real latency and failure rates on the lab model rather than assuming scale.
- Full-context handling is tested offline; current combined-scenario model quality is not yet verified on the lab.
- Precision/recall needs a separate provisional or independent reference; the model must not see evaluation answer keys. Compare unchanged deterministic ranking and agent recommendations separately.

## Connected test

Restart backend, create a fresh analysis with the combined before/after pair, open Planning and check full process context is included. Run one full review, inspect cited source against the real requirement, download plan Markdown and JSON. If interrupted, Stop preserves checkpoint; refresh restores status without starting inference. Do not change repository inputs while resuming a bound session.
