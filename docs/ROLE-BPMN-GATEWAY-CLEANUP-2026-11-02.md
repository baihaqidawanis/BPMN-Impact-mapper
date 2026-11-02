# Gateway/result activity cleanup

Status: **applied with explicit user approval**. The initial automatic review
block was resolved by the user's explicit approval of the reviewed BPMN update.
87 canonical BPMNs changed; all 112 hashes match their reviewed versions.
179 application source hashes remain unchanged. Chrome import tests passed
112/112 (3.4m); the strict audit and projected graph/DI verifier passed for all
prepared files. A fresh strict audit of the actual canonical files also has zero
findings; see [post-apply hashes/checks](ROLE-BPMN-POST-CLEANUP-2026-11-02.json).

The user requested gateway/task cleanup of the BPMN documents. Backend,
frontend, database and configuration source are excluded from the write scope.

The stricter precheck identified 157 direct gateway-to-gateway edges and one
gateway-to-end edge in 87 of 112 role diagrams. The earlier audit's description
of all 158 as gateway pairs was corrected.

## Modelling decision

86 diagrams contain a plain XOR result merge immediately before an existing
result activity. They have no parallel/inclusive/event gateways, nested scopes
or boundary events. The merge has multiple alternative incoming flows, exactly
one unconditional outgoing flow, no default and no extension semantics.

Remove this redundant result merge and its sole outgoing edge. Redirect its
incoming edges to the existing result activity, with labels and conditions
unchanged. Alternative paths now merge implicitly at that activity. A task with
multiple incoming sequence flows does not synchronize concurrent tokens; this
transformation is restricted to the reviewed alternative outcome patterns.

Before: decision -> result merge -> existing result activity -> end.

After: decision -> existing result activity -> end.

This deliberately avoids inventing 157 new tasks merely to separate gateway
symbols. It does not preserve an explicit join symbol at those locations. If an
explicit join symbol is mandatory for the supervisor's modelling rubric, this
choice must be reviewed; inserting a real branch activity requires separately
identified behaviour, rather than a filler task.

The one Staff AP L2 edge `PO CLOSED` previously led directly to the end. It now
leads through `Periksa hasil posting pembayaran supplier` in the Staff lane.
This represents viewing existing finance output/status, not a new approval,
request, state mutation or mandatory acknowledgement click. Source evidence:
`frontend/features/finance/finance-workspace.tsx` displays transaction result
messages (command handler around lines 64-74) and payment status (around 225-231).
No new PO-status fetch or UI field is claimed by this task.

## Verification and reproducibility

Preparation and backups: `outputs/gateway-activity-2026-11-02/`. The manifest
records all 112 before/after hashes, removed merge IDs, and the one added task.

```powershell
python experiments/simple-erp/verify_result_merge_cleanup.py outputs/gateway-activity-2026-11-02
python scripts/audit_bpmn_conventions.py outputs/gateway-activity-2026-11-02/prepared
```

The verifier preserves every old business node's ID/type/attributes, namespace
declarations, actor lane, flow label/condition/default and projected connectivity.
Only the documented merge nodes/edges and AP presentation activity are treated
as changes. Every remaining node has one shape and lane, every flow has a
nonzero diagram edge, IDs are unique and cleanup is idempotent. Diagram edges
are extended to the existing activity coordinates; business node placement is
unchanged except the AP end is moved to make room for the new visible task.

This is static graph/source alignment evidence, not a proof of all runtime
paths, human behaviour, concurrency correctness or full 1:1 ERP equivalence.
The scope is the 112 canonical role diagrams; legacy diagrams and frozen
experiment pairs are not overwritten. Use both files from an existing frozen
experiment pair together. Mixing its old changed file with this newly cleaned
canonical baseline introduces unrelated structural differences.

The existing ERP diagram generator is not modified; regenerating from its old
templates may restore explicit result merges. Recheck regenerated documents.
Historical traceability snapshot hashes will need explicit refresh if this
baseline update; old experiment reports remain historical evidence.
