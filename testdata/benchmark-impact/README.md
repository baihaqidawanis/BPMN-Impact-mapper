# Synthetic positive and negative controls

This pack freezes two positive change scenarios against a tiny invented source
repository, plus the existing display-label negative control. The truth JSON
was written from the fixture author's intended edits **before** inspecting
retrieval rankings. It lives outside `repository/`, which is the only indexed
source folder. The subsequent BM25 probe was used only to confirm exact change
keys and the method-independent input fingerprint for the manifest.

For `TASK-CONFIRM`, the intended software change is to add a confirmation step
in `web/request-form.ts` and update `web/request-form.test.ts`. The task does
not demand a new backend validation rule. The two sequence-flow records
describe how that one insertion connects to existing tasks; they have empty
independent impact sets.

For `GW-APPROVAL-LIMIT`, the intended software change is to apply the approval
threshold in `internal/workflow/approval.go` and update
`tests/approval_test.go`. The existing approval flow otherwise requires
approval unconditionally. The three connecting sequence-flow records have
empty independent impact sets. `internal/customer/approval.go` is a deliberate
similar-word decoy; it handles a different domain. A retrieved decoy is a
false positive under this fixture contract.

The negative control changes only a BPMN flow's display label and has no
required source edits. Its expected set is empty. All three cases are checked
for exact change coverage by the benchmark evaluator.

Run from the mapper root:

```powershell
$env:GOCACHE = (Join-Path (Get-Location) '.gocache')
go run .\cmd\bpmn-impact-mapper benchmark --manifest testdata\benchmark-impact\manifest.json --output outputs\benchmark-impact
```

The runner checks the SHA-256 locks, input fingerprint, and byte-for-byte
reproducibility across two repetitions for BM25, Semantic, Fusion, and Hybrid.
Only timing lives outside the deterministic benchmark report. These labels
demonstrate the measurement pipeline; they are not independent review or an
accuracy claim for Simple-ERP or other repositories.

The verified control run produced benchmark ID
`a4b2c567147ed8ae70e778c8770cd54d` and identical deterministic
`benchmark-report.json` SHA-256
`2d5c15b04ff8889248d7cfe7649cfaad7b037296eaf19ca01d73221c5a9035b9`
before and after enforcing complete ground-truth coverage. On the main changed
element, BM25 ranked the task's two expected files 4 and 2; Semantic, Fusion,
and Hybrid ranked them 3 and 2. For the gateway, all methods ranked the
approval function 1; only Hybrid retrieved the approval test, at rank 6. The
gateway's connecting flow records also produced unrelated candidates despite
their empty expected sets. No method got the fixture's required-action labels
right, so the action macro-F1 was 0 in this control. The aggregate Recall@5
includes the supporting flow cases, which is why it is much lower than the
main-element recall; read per-change files before interpreting the aggregate.
