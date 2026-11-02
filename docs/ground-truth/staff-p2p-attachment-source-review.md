# Staff P2P attachment: source-based impact review

Status: **provisional implementation reference, not blind ground truth**.
Prepared from read-only Simple-ERP source inspection on 2026-10-27 and checked
again against the create path on 2026-10-28. The
reviewer had already seen mapper rankings, and no reference implementation
diff exists. Do not feed this file, its labels, or the implementation plan to
repository retrieval or claim research Precision/Recall from it.

## Snapshot and requirement

- Simple-ERP HEAD: `c934497`, with uncommitted changes. HEAD alone does not
  freeze the analyzed source. Mapper input fingerprint for the inspected
  add-task run: `f15a566582de451fc870b0cccb46e7b5` (web, Top-K 5, before
  ingest identity normalization). Source corpus: 198 indexed files.
- Old BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-create.bpmn`;
  SHA-256 `93f10bf8766b4a8d1849c3803b698fc85a3d1baa05ab4a7e3453078577137d0c`.
- New BPMN: `experiments/simple-erp/staff-p2p/role-staff-l3-pr-create-add-document.bpmn`
  in the mapper repository. Staff must provide exactly one valid PDF, at most
  1 MiB, when creating a PR. An invalid/missing file must leave no new DRAFT;
  accepted bytes and metadata commit with the PR.
- One concrete reference design is assumed for file-level review: the existing
  `POST /purchase-requests` accepts multipart form data, a new one-to-one
  PostgreSQL attachment table stores bytes/metadata, and the existing create
  transaction persists both records. No external object store is assumed.

This design choice is necessary to name exact files. The BPMN establishes
required behavior but does not prescribe multipart, PostgreSQL bytes, or a
particular frontend helper. This is source-based implementation judgment, not
a measured mapper result or independent ground truth.

## Existing code files that this design must modify

| Relative file | Source evidence and required change |
| --- | --- |
| `frontend/features/purchasing/purchase-request-panel.tsx` | The `create` handler at line 31 sends JSON with purpose and lines only. Add required PDF input, validation/feedback, and multipart submission. |
| `backend/internal/purchasing/adapter/http/handler.go` | `create` near lines 82–102 decodes JSON only. Parse a bounded multipart request, validate required PDF size/type, retain role checks, and pass the attachment onward. |
| `backend/internal/purchasing/application/service.go` | `Store.CreateRequest` and `Service.CreateRequest` near lines 36 and 54 carry only actor, purpose, lines, and request ID. Add the attachment to the application contract and reject missing input before persistence. |
| `backend/internal/purchasing/adapter/postgres/store.go` | `Store.CreateRequest` near lines 29–65 inserts DRAFT and lines in one transaction. Insert the attachment in that transaction and roll back all writes on failure. `Store.TransitionRequest` is not the relevant function. |
| `backend/internal/platform/idempotency/middleware.go` | `maxRequestBody` is exactly 1 MiB near line 16 and applies before the handler. A 1 MiB PDF plus multipart envelope would be rejected. Raise the bound for this request in a scoped way and preserve replay/body-hash behavior. |

These five existing files are core targets under the stated design. The
validation layer can vary, but input, HTTP, application, persistence, and
request-size responsibilities cannot be omitted. The idempotency middleware
reads the entire body and rejects requests over 1 MiB before the handler runs;
it should not gain an unbounded global limit.

## Design-dependent existing files

| Relative file | Judgment |
| --- | --- |
| `frontend/lib/api.ts` | **Conditional MODIFY.** The shared helper always sends `Content-Type: application/json`; change it if the PR form continues to use it for `FormData`. A dedicated upload request preserving credentials, CSRF, and idempotency can avoid this edit. Do not score it as an unconditional positive. |
| `api/openapi.yaml` | **Contract MODIFY** if the maintained OpenAPI specification is in scope. It currently describes the POST as JSON, but is not executable upload logic. |
| `docs/API-CONTRACT.md` | **Documentation VERIFY/MODIFY** if the explanatory API contract is updated. Exclude it from retrieval regardless. |

## Contract, tests, and new artifacts

- Existing tests that must be adapted if attachment is mandatory:
  `frontend/e2e/operations.spec.ts` and `frontend/e2e/dashboard.spec.ts`
  click **Buat PR** without choosing a file. The dashboard test uses PR
  creation to seed audit data; it is test-maintenance impact, not dashboard
  feature logic. `backend/test/integration/operations_test.go`,
  `finance_test.go`, and `reporting_test.go` directly call `CreateRequest`
  with the old signature and need attachment fixtures after the contract
  changes. Additional HTTP, persistence, middleware, frontend unit, and E2E
  tests should cover valid/invalid PDF, 1 MiB boundary, unauthorized access,
  duplicate request, and atomic rollback. They may be new files, so exact
  existing test paths are not all mandatory labels.
- New files to **ADD**: the next numbered migration up/down pair for a
  one-to-one attachment table. The filenames do not exist yet and cannot be
  retrieved as existing-file candidates. Do not edit historical migration
  `backend/migrations/000005_operations.up.sql`.

## Negative and unresolved controls

- No feature change required in `frontend/features/purchasing/purchase-order-panel.tsx`,
  `frontend/app/purchasing/[section]/page.tsx`, `frontend/app/login/page.tsx`,
  `backend/internal/purchasing/domain/purchase_request.go`, or the existing
  PR status-transition method. The attachment is required at **create**;
  purchase-order conversion, routing, login, and PR states are unchanged.
- `role-staff-l3-pr-create-sequence-label.bpmn` only changes the displayed
  failure-branch label while retaining the explicit condition. It has no
  established mandatory source-code edit. An earlier mapper run emitted five
  noisy VERIFY candidates; a later direct-reference gate abstained on this
  case. Neither outcome proves general mapping accuracy.
- `role-staff-l3-pr-create-gateway-condition.bpmn` introduces a PR budget
  limit, but neither the BPMN nor the current settings schema defines its
  value or source. `Store.CreateRequest` is a plausible enforcement point;
  an exact file list for budget configuration/storage is unresolved until
  that business rule is specified.

## Diagnostic comparison with the exposed Hybrid report

In the inspected add-task Top-10, ranks 1 and 2 are the dashboard and
operations E2E tests; both do create PRs and need fixtures. Rank 4 is the PR
form, rank 6 an integration test that calls `CreateRequest`, and rank 8
`Store.CreateRequest`. Those are plausible review targets. Rank 3 is the
purchase-order form, rank 5 the purchasing route page, rank 7 login, rank 9
goods receipt, and rank 10 a PR state-machine test; the new requirement does
not establish changes there. The HTTP handler, application contract, API
client, idempotency limit, and OpenAPI contract are absent from Top-10.
This is an earlier source-review diagnostic on an exposed case, **not** an accuracy
score or evidence that every selected file must change.

## How to turn this into evaluation ground truth

Freeze the full pre-change repository snapshot (including uncommitted files),
both BPMN hashes, and the implementation design. Have an independent reviewer
inspect requirements/source without seeing mapper candidates, or implement
the change in a separate authorized branch and adjudicate its diff. Record
existing-file MODIFY labels separately from new ADD files, tests, docs, and
legitimate NO_ACTION cases. Only then export canonical evaluation JSON and
measure Recall/Precision/MRR. Keep all review artifacts excluded from the
retrieval corpus.
