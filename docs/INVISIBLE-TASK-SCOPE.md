# Invisible behavior as a change-impact scenario

This mapper reads BPMN blueprints and repository source. It does not discover invisible transitions from event logs or execute the process. A service task is not automatically an invisible transition in the process-mining sense.

The old POC (`C:\POC TA\README.md`, `docs/aturan-git-paper-vs-kita.md`) documents Skip, Redo/loop, XOR-AND-Stack and Non-free choice. Its Prime/Non-Prime terminology labels atomic and composite internal decisions in that POC; it should not be used as an automatic classification claim for arbitrary software. The POC also states no SWITCH counterpart was implemented.

## Current thesis scope

The selected prime patterns are Skip, Redo and Switch. Non-free choice (NFC) covers a later decision constrained by an earlier choice. The [experiment fixtures](../experiments/simple-erp/poc-patterns/README.md) contain before/after pairs for all four, alongside the historical XOR-AND-Stack fixture. Skip has a source-derived reference for one gateway. The other three fixtures have behavior checks for plan review and still need independently reviewed file references.

## Scenarios

| Behavior | BPMN before/after change | Expected source concern |
|---|---|---|
| Skip | Close an all-zero adjustment before the approval queue | Per-line comparison, stored status, version and audit in the submit transaction |
| Redo | Request revision and return the same PR to review after the creator resubmits | Separate requests, durable revision status, creator and version checks |
| Switch | Reroute an approve action to rejection when a PR product is inactive | Current product status, automatic rejection evidence and unchanged manual rejection path |
| Non-free choice | Lock the approval route at submit and enforce it on a later review command | Persisted route, role eligibility, prior PRs and tests across policy changes |

These are changes to automatic behavior represented by conditions, gateways, flows or explicitly named decision activities. They need not appear as ADD_TASK: condition edits and flow edits are meaningful impact inputs too. Current parser/diff can expose represented nodes/edges/conditions, but reliable retrieval and agent plans for each scenario still require experiments and scoped references. Do not describe these as already validated end-to-end.

If an automatic rule is absent from both BPMN versions and no supplied requirement describes its change, the input does not establish that requirement. Blank labels alone do not identify invisible behavior. Custom POC annotations are not a portable runtime contract for other repositories.

Within the current one-month deadline, prioritize one explicit skip-condition change after the existing add-task scenario works. Keep event-log process discovery, inferred invisible-task taxonomy and new execution engines out of the present implementation scope.
