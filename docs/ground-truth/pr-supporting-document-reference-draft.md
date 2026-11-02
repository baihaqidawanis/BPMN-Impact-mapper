# PR supporting document: scenario and provisional impact reference

Status: **provisional reference, not blind ground truth**. Created 2026-10-22.
Do not feed this document to repository retrieval or prediction. The author has
already seen mapper rankings for this scenario, so this cannot independently
establish Precision/Recall. Confirm it against a separately implemented change
or reviewers who have not seen the mapper report before exporting evaluation JSON.

## Frozen scenario

- Case-study repository: Simple-ERP working tree; no Simple-ERP files are changed here.
- Baseline: `C:/Simple-ERP/docs/bpm-haqi/level-3-p2p/l3-pr-create.bpmn`.
- Changed BPMN: `outputs/manual-inputs/edited-process.bpmn`, element `Activity_0l7249z` (arsip upload dari root)
  (`ADD_TASK`, visible label: `Upload required supporting PDF`).
- Staff or Super Admin supplies exactly one PDF of at most 1 MiB as part of
  creating a purchase request. Missing, invalid, or oversized input fails
  creation without a partial DRAFT. Accepted bytes and metadata persist in
  the same database transaction as the new DRAFT. For this reference design,
  use a one-to-one attachment table and a multipart create request. No new
  gateway, approval, state, external file store, or BPMN runtime is required.
- The three unlabelled rewired sequence flows are supporting edits of the
  inserted task, not three separate implementation changes.

## Core existing implementation files to review

| File (relative to Simple-ERP) | Why | Expected action |
| --- | --- | --- |
| `frontend/features/purchasing/purchase-request-panel.tsx` | Add required file selection, client-side feedback, and multipart submission. The current form sends only purpose and product lines. | MODIFY |
| `backend/internal/purchasing/adapter/http/handler.go` | `Handler.create` currently decodes JSON only; it must parse a bounded upload, enforce authorization, and reject missing/invalid files. | MODIFY |
| `backend/internal/purchasing/application/service.go` | The create operation and store contract must carry the attachment and enforce the required input before persistence. | MODIFY |
| `backend/internal/purchasing/adapter/postgres/store.go` | Persist the attachment with the purchase request in the existing create transaction; rollback on failure. | MODIFY |

These are code-impact hypotheses for the chosen persisted-upload behavior, not
labels copied from mapper output or a verified implementation diff. One BPMN
task can require several software-layer changes, but the BPMN edit alone does
not prove that every surrounding artifact must change.

## Conditional or non-code impacts; not core file-level positives

- `frontend/lib/api.ts`: only if the existing shared helper is extended to
  send FormData. A separate upload client is also possible.
- `api/openapi.yaml`: update if the maintained API contract is part of the
  implementation scope. It is documentation/contract impact, not proof that
  the OpenAPI file itself contains the upload implementation.

The schema change should be an **added migration pair**, not a modification of
the historical `backend/migrations/000005_operations.up.sql`. New filenames
must be chosen during implementation, so they cannot honestly be asserted as
existing-file positives in a file-level ground-truth JSON yet.

## Required verification, but exact test filenames are not frozen

Test missing PDF, wrong type, oversize, rollback, successful persistence,
authorization, and form submission. Existing HTTP, integration, and frontend
tests may be updated or new test files may be added. Current call sites in
`backend/test/integration/operations_test.go`, `finance_test.go`, and
`reporting_test.go` will need adaptation **if** the create signature is changed
incompatibly. Do not label these as certain file-level positives before a
reference implementation fixes that design choice.

## Not required by the stated scenario

- `backend/internal/purchasing/domain/purchase_request.go`: existing DRAFT
  transition rules need not change merely to require an attachment at create.
- Sales and supplier-invoice implementations: no changed responsibility in
  those modules is specified.
- A new gateway or altered approval/submit transition: the existing invalid
  request path is sufficient.

## Evaluation protocol

Freeze the BPMN and source revision first. Build and review an implementation
diff in a separate authorized Simple-ERP branch or obtain blind reviewer
judgments. Record required existing files, new files, actions, disagreements,
and rationale. Only then produce the evaluation tool's canonical ground-truth
JSON and compare it with a mapper report generated from the **pre-change**
repository. Never index this draft, the implementation diff, or the final
ground truth during inference.
