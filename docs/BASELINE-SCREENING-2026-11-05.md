# Screening mapping BPMN current Simple-ERP

Review 2026-11-05. Scope: three modules (P2P, Inventory, Master Data/Configuration) and three roles. Static screening only; ERP source/BPMN and Qwen outputs were not modified.

## Findings

| Check | Result |
|---|---|
| L3 process coverage | 72 / 72 processes parsed |
| Activity-to-source coverage | 354 / 354 current activity IDs still mapped |
| Missing/renamed activity IDs or labels | 0 |
| Mapped source/test hashes | 34 / 34 match historical snapshot |
| Evidence match at saved source line | No missing or moved evidence found |
| Missing files among snapshot inputs | 0 |
| Changed BPMN hashes | 87 / 112 differ from old snapshot |
| Changed L3 flow snapshots | 72 / 72 differ |
| Obsolete historical controls | 72 result_merge gateways absent from current XML |
| L3 invalid flow endpoints | 0 |
| L3 start/end and gateway convention checks | 72 / 72 passed selected structural checks |
| L0-L2 hierarchy | 40 diagrams; 139 decomposition links checked; 0 broken links found |
| L0-L2 selected structural checks | 0 flagged diagrams |

The differences in flow metadata consistently include removal of result_merge and rerouting existing outcomes to the existing response activity. Activity IDs, labels, source files and saved evidence still match. Hash drift alone does not prove a business behavior change. This is a stale snapshot issue; the historical mapping package must not be claimed to reflect current flow metadata verbatim.

## Per role and module

| Role | Module | L3 processes | Current activities mapped | Historical flow tables needing refresh |
|---|---|---:|---:|---:|
| staff | inventory | 8 | 36 | 8 |
| staff | master-data | 1 | 3 | 1 |
| staff | p2p | 10 | 50 | 10 |
| super-admin | inventory | 11 | 53 | 11 |
| super-admin | master-data | 5 | 21 | 5 |
| super-admin | p2p | 19 | 101 | 19 |
| supervisor | inventory | 6 | 26 | 6 |
| supervisor | master-data | 3 | 13 | 3 |
| supervisor | p2p | 9 | 51 | 9 |

## Adjustment submit and current Qwen plan

Use the [refreshed adjustment-submit mapping](../experiments/simple-erp/baseline-traceability/adj-submit-current.md) for the presentation. Its complete current node/flow set, evidence lines and hashes were checked. The existing UI -> handler -> service -> store submit path supports the mapped DRAFT/version guards, SUBMITTED update, audit and HTTP response. Error exits after the gateway remain an abstraction: not every SQL/audit/commit failure is drawn separately. Do not claim runtime one-to-one proof.

The Qwen change plan is a proposal for the hypothetical after requirement, not the current-before mapping. Its saved status remains NEEDS_REVIEW and manual quality assessment 55/100. Against the unchanged provisional three-file ReasonReview reference, selection precision is 50%, recall 100%, F1 66.67%. The same old model output was assessed; no new inference was run.

## What is safe to present

- The scoped baseline mapping exists across 72 L3 processes and still references current source correctly at the checked lines.
- The current before and hypothetical after have distinct purposes; the latter introduces reason validation and needs frontend/backend/test changes.
- Current diagram structure passed the selected automatic checks. That does not verify all gateway semantics, all RBAC behavior, or all process outcomes.
- Historical flow tables need refresh after diagram cleanup. This screening deliberately preserves old artifacts and model output for traceability.
- Qwen is under refinement; existing evidence supports showing its weaknesses, not claiming a finished developer-ready plan.

## Evidence and limitations

Detailed machine-readable screening is stored locally at `outputs/baseline-screening-2026-11-05/screening.json`. Checks compare current XML activity IDs/labels, flow endpoints and controls with the historical snapshot; source hashes and evidence match text at saved lines. Decomposition calledElement QNames are resolved using declared XML namespaces and checked against the actual target namespace and process ID. These are mechanical consistency checks, not an independent semantic review of every business rule. ERP code/tests were not executed and no lab request was made.
