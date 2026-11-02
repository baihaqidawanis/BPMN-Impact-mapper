# Staff P2P retrieval diagnostic — 2026-10-27

## What was tested

The inserted-task scenario in this directory, with the read-only Staff L3
baseline documented in README. Three full offline Hybrid runs share input
fingerprint `69915fc4cf937663735157b288cbd922`. This controls their input corpus
and BPMN pair, but does not freeze the current working tree for future runs.

| Probe | First result | PR form position | Important limitation |
| --- | --- | --- | --- |
| MiniLM, previous query | login page | 3 | Only two semantic candidates passed the threshold; the PR form appeared in the lexical list only |
| E5, previous query + required model prefixes | dashboard E2E test | 3 | Different cosine distribution; the MiniLM threshold is not calibrated for E5 |
| MiniLM, explicit process-name context | dashboard E2E test | 4 | Purchasing-related results appear, but tests and the purchase-order form outrank the PR form |

PR form: `frontend/features/purchasing/purchase-request-panel.tsx`.
It is a plausible review target, not an independently proven ground-truth label.
The table deliberately reports ranks, not a claim of improved accuracy.
E5 plus the new process context was **not** tested in this comparison.

A later read-only source review found that the dashboard and operations E2E
tests at ranks 1–2 both create PR fixtures, so they would need PDF fixtures if
the attachment became mandatory. They are test-maintenance hits, although the
dashboard feature itself does not change. The purchase-order form and generic
purchasing route page remain unsupported for this requirement. See the
[provisional source review](../../../docs/ground-truth/staff-p2p-attachment-source-review.md)
for the chosen implementation design and omitted core files.

Local generated reports (ignored outputs, not committed research data):

- `outputs/staff-p2p-add-document-hybrid-final/impact-report.json`
- `outputs/staff-p2p-e5-probe/impact-report.json`
- `outputs/staff-p2p-minilm-process-context/impact-report.json`

## Changes and rationale

1. Explicit conditions take priority over branch labels in gateway comparison.
   Renaming a branch without changing its condition no longer invents a
   gateway-condition change. Labels remain a fallback for absent conditions.
2. Generated preview trees and BPMN generator trees under `scripts` are excluded
   by generic ingestion rules. No ERP-specific ranking rule was introduced.
3. Fusion reports sparse semantic support. An RRF number near one still means
   rank agreement, not confidence that a file must change.
4. The parser retains the process name. Recognized task insertion/removal
   contexts carry it to context-aware BM25 and Hybrid semantic retrieval.
   The semantic process-name anchor is whitespace-normalized and capped at
   160 characters. Standalone Semantic and plain BM25 keep their baseline queries.
5. Known multilingual-E5 profiles receive `query:` / `passage:` prefixes,
   including every head/middle/tail passage. Evidence remains original source
   text. Model/profile/query versions are recorded for reproducibility.
6. The engine identity was bumped for the changed diff/context behavior.

The E5 input convention was checked against its locally cached model card
(`intfloat/multilingual-e5-small`, snapshot
`614241f622f53c4eeff9890bdc4f31cfecc418b3`). No model was downloaded.

## Engineering controls and boundaries

The opt-in `TestLocalBilingualRetrievalSmoke` has three authored concepts,
each queried in English and Indonesian, over six short synthetic documents.
It logs expected-file ranks without asserting a research accuracy threshold.
Observed at the same configured cosine cutoff of 0.25:

| Model | Expected file at rank 1 | English controls | Indonesian controls |
| --- | --- | --- | --- |
| MiniLM | 3/6 | All three rank 1 | All three unresolved |
| multilingual-E5-small | 6/6 | All three rank 1 | All three rank 1 |

This comparison is confounded by different model score distributions and the
tiny, deliberately simple corpus. It is a reason to investigate E5, not to
replace the default automatically or claim 100% real-world accuracy.

Enable only with both models already cached:

```powershell
$env:BPMN_IMPACT_LOCAL_MODEL_SMOKE='1'; go test ./internal/semantic -run TestLocalBilingualRetrievalSmoke -v -count=1
```

This tiny diagnostic cannot establish multilingual superiority, Recall@k on
ERP code, or generalization. No held-out ground truth was created from these
rankings. Default MiniLM, thresholds, and equal-weight RRF remain unchanged.
No source in Simple-ERP was edited or executed. BPMN diagram layout has static
DI checks, not a new browser-level visual sign-off in this session.

## Next decision

Review the Staff scenario requirement once, then freeze the source snapshot
and independently label expected implementation files. Compare models on
additional unseen scenarios before changing defaults. Do not filter login,
README, or tests merely because they ranked poorly in this exposed example.

## Verification and handoff

- `go test ./...`: passed after the engine identity/context changes.
- Additional process-name baseline-isolation test: passed with focused
  retrieval/diff/semantic/app package tests.
- `go vet ./...`: passed on the final Go changes.
- `gofmt -l` on changed/new Go files: empty.
- `git -c core.autocrlf=false diff --check`: passed.
- Both complete Staff CLI probes (E5 and process-context MiniLM): completed.
- Baseline BPMN SHA-256 still matches the README preparation hash.

Web end-to-end smoke (opt-in case study) uploads the same Staff baseline and
three changed BPMN files through the actual Next.js form and Go API. The
inserted-task run verified the ADD_TASK and grouped flow details, evidence on
each displayed candidate, canonical JSON/Markdown downloads, and Saved analyses
reload. The gateway condition and branch-label runs verified their distinct
diff taxonomies through the web with BM25. All three browser tests passed.

```powershell
$env:BPMN_IMPACT_STAFF_REPO='C:\Simple-ERP'
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'
cd C:\BPMN-Impact-Mapper\web
npx playwright test tests/e2e/staff-p2p-local.spec.ts --config playwright.config.ts
```

The environment variable is opt-in because the external case-study checkout
is not a portable fixture. The browser test never writes to that checkout.

The first Hybrid web run indexed 198 files and returned the same five leading
files/scores as the earlier CLI Hybrid run. A CLI rerun with no explicit
ignore flags confirmed the same leading ranking, but initially produced a
different fingerprint from the web: HTTP omitted ingest limits while CLI
supplied their default values. The engine now normalizes effective ingest
limits before hashing. On the Staff gateway-control scenario, web and CLI
subsequently produced the same analysis ID
`18fcdbbe0f824b6dedc99204365ea2bf`, input fingerprint
`b13d2155391f77d6fcd8628020f6e5ea`, and byte-identical canonical JSON
and Markdown. This is an interface parity check, not mapping accuracy evidence.

After the identity fix, the full Go test suite, vet, web TypeScript check,
formatting check, and diff whitespace check passed. The Hybrid add-task browser
run preceded the identity fix; that fix changes report IDs/fingerprints, not
the candidate ranking formula. Both BM25 browser controls were rerun after it.

Restart the backend and rerun Analyze to use the updated engine. No frontend
code changed, so no frontend rebuild is required for this change. Existing
saved reports remain historical results; refreshing them does not rerun the
analysis. The optional E5 profile is available through the existing
`BPMN_IMPACT_SEMANTIC_MODEL` backend setting, but was not enabled by default.
No commit or push was made.

The later diff refinement now reports a label-only sequence-flow edit as
`sequence flow label changed` rather than the generic added/removed/modified
reason. The independent label-control CLI run confirms the taxonomy and
reason, but its five VERIFY candidates still include unrelated auth files.
The retrieval ranking was not tuned on this exposed example.

The subsequent method-independent flow-label eligibility rule is a narrow
pre-retrieval control: only when a sequence flow's display label changes while
its explicit condition and endpoints remain unchanged does retrieval require
an exact flow-ID or old/new label reference in the indexed artifact. With the
same Staff label-control input fingerprint
`f6095e7ad63be127073ee7be96cec810`, BM25 changed from five broad VERIFY
candidates to zero candidates and one explicit unresolved reason; the
production Hybrid likewise returns zero candidates and one unresolved reason
without invoking its offline embedding runtime. This is
abstention, **not** a claim that Simple-ERP needs no source change. The Staff
add-document control still yields one ADD_TASK with five top-K candidates;
the changed-budget condition control still yields five sequence-flow
candidates and an unresolved gateway-condition record at top-K 5. Those two
behavioral cases are outside the new gate. These are exposed-case diagnostics,
not Precision/Recall improvements or model calibration.
