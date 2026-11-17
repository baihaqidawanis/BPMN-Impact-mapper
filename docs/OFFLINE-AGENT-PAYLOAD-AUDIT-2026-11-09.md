# Offline audit of agent model payloads

Date: 2026-11-09. Provider contract: v10. No lab/VPN/model request was made.
Only a localhost HTTP stub received requests; no case-study source was executed.

## Scope

Seven saved child checkpoints from failed session
f78cb1f48ec714fdbff8c82f7d8eccd3 were replayed using the current adapter.
Their original roles are five Planner and two Reviewer checkpoints. Separately,
each saved state was reconstructed for Analyst, Planner and Reviewer: 21 payload
checks. Those 21 are test variants, not 21 observed live model turns.

Checks inspect actual outbound HTTP payloads, not only an intermediate object:
selected-change counts, candidate/evidence indices, original before/after presence
facts, Planner action field exclusion, and request-budget accounting. Saved
feedback is included. Checkpoint/source snapshots are not modified.

## Reconstructed request sizes

All values below are bytes of system instruction + input JSON + response schema,
not token counts. The adapter applies a conservative tokenizer-independent byte
bound within its 32,768-token context setting, reserving 4,096 output tokens and
512 for the template. The resulting conservative request bound is 28,160 bytes.

| Checkpoint batch | Analyst | Planner | Reviewer |
|---|---:|---:|---:|
| 1 | 15,507 | 16,464 | 12,729 |
| 2 | 20,517 | 22,553 | 17,414 |
| 3 | 18,720 | 21,223 | 15,719 |
| 4 | 21,195 | 24,337 | 18,523 |
| 5 | 19,127 | 21,038 | 16,326 |
| 6 | 19,472 | 22,056 | 16,461 |
| 7 | 10,987 | 11,042 | 8,897 |

All 21 requests reach the local stub. Maximum input JSON is 17,700 bytes, below
the separate 24 KiB input cap. These measurements cover these specific states;
larger repositories or more evidence can still exceed the configured bounds.

## Invalid response regression coverage

Regression tests reject negative, fractional, string, boolean and unknown target
indices; negative/fractional/string/null/out-of-range evidence references; empty
references; mixed named/indexed identity; and invalid explicit legacy action
values modify, add and ADD_TASK. Indexed plan responses omit action and the host
binds the existing candidate action. Engine evidence/identity validation remains.

Other existing tests verify that omitted RELEVANT plans fail validation, WEAK
candidates may remain unresolved, unapproved drafts are withheld and an empty
combined draft does not start final model review. Markdown labels unapproved
results explicitly. Actual model reasoning quality is not determined here.

## Reproduce from PowerShell in the mapper root

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'
$env:BPMN_REPLAY_PATTERN='C:\BPMN-Impact-Mapper\.data\agent-sessions\f78cb1f48ec714fdbff8c82f7d8eccd3-focused-*.json'
$env:BPMN_REPLAY_ROLES='all'
go test ./internal/llm -run TestLocalCheckpointTransportReplay -count=1 -v
Remove-Item Env:BPMN_REPLAY_PATTERN, Env:BPMN_REPLAY_ROLES
```

Without BPMN_REPLAY_ROLES=all the test replays each checkpoint's original role.
Without BPMN_REPLAY_PATTERN the opt-in replay is skipped. Local checkpoints are
not committed, so another machine needs its own captured checkpoints.

## Limits

The stub returns transport fixtures, not a real reasoning result. Passing these
checks does not establish semantic plan correctness, response-schema compliance
by the live model, precision/recall improvement, lower GPU memory, or latency.
No benchmark scores or saved model outputs are overwritten. A fresh v10 Coder
Focused review for the same add-task case remains the next decisive test.
