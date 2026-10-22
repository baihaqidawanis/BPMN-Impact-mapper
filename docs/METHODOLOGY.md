# Methodology

## Current production ranking (engine 3.5d)

The production Hybrid has two ranking branches: BPMN-context BM25 and
Semantic cosine retrieval. RRF combines their positions with equal votes
(`k=60`); file diversity is applied to the displayed Hybrid list. The lexical
query includes the changed element plus applicable predecessor/successor
labels; long task documentation is replaced by a short shared process anchor
for named task additions/removals. The plain BM25 research baseline omits
neighbour labels, keeping BPMN-context retrieval as a distinct comparison.

Direct changed-element IDs, neighbouring IDs, function names, file paths, and
conservative code-pattern clues are **explanatory evidence only** in this
method. They do not carry independent weights and cannot lower BM25 or cosine.
The source index still contains path/function text, so BM25 can naturally
retrieve on those words; they are not counted a second time. A new BPMN ID
missing from old source is expected and causes no score penalty. Hybrid
currently labels retrieved candidates `VERIFY`; rank consensus cannot prove a
required edit. `confidence`/`total` retain the RRF ordering value for JSON
compatibility, not a calibrated probability. The six-signal weighted
`retrieval.Hybrid` remains a legacy experimental implementation for tests and
old reports, but is no longer wired to the production `hybrid` method.

The dated notes below document earlier iterations; where they describe a
45% ID weight, code-evidence ranking, or source-relation propagation, they
refer to that legacy implementation. Compare only reports produced by the
same engine version; evaluate changes against independent ground truth before
claiming an accuracy improvement.

Current Semantic indexing samples at most three bounded passages per artifact:
head, tail, and middle, within the global embedding budget. The best passage
sets the cosine score; its bounded evidence excerpt is chosen for inspection,
not claimed as a line-level explanation of the model. This changes rankings and
analysis identity, so compare only same-engine reports. The next research gate
is a frozen, independently labeled multi-scenario benchmark. Only if an
ablation shows benefit should exact-ID/structural evidence become an optional,
versioned reranking variant. Do not treat it as a third score now: absent IDs
are normal for additions, and path/function text already enters retrieval.

### SQL table evidence precision (2026-10-22)

`DATABASE_TABLE` requires BPMN query-word overlap with the extracted table
identifier itself. Merely mentioning query words in a selected column,
predicate, variable, or string value no longer earns structural credit. For
this and other structural hints, the query comes from the changed BPMN element;
adjacent tasks remain lexical/BM25 context but cannot masquerade as direct
structural evidence. A table receives strength proportional to the fraction
of its name tokens matched by that changed-element query, with common word
inflections recognized. This is not a SQL parser; complex joins and dynamic
SQL may abstain. `AUDIT_ACTION` now requires overlap with a quoted uppercase
action identifier, and `API_ROUTE` with the literal route path; unrelated
function arguments no longer earn these structural signals. `STATE_STATUS`
uses the status-assignment fragment, and `TEST_NAME` the matched test
declaration, rather than arbitrary words elsewhere on the line. Dynamic
actions and routes may abstain. BM25 and Semantic rankings are unchanged. No
file, module, or business term is hardcoded. Engine identity is
`engine-3.4s-structured-span-evidence`.

### Literal cross-layer HTTP evidence (2026-10-22)

The Hybrid code-evidence index now recognizes a narrow cross-language relation:
an explicit-method JS/TS `api` or `fetch` call with a literal path can link to
one unique Go route registration with the same HTTP method and path, provided
the native Go indexer resolves the registered handler. This uses generic
syntax patterns and repository-local source; it does not depend on a BPMN
runtime, stable task ID in code, framework folder, or case-study vocabulary.
Dynamic paths, ambiguous duplicate registrations, test files, unsupported
client syntax, and unresolved handlers abstain. `HTTP_ROUTE_LINKS` reports
that this limited extractor ran, not that every endpoint was understood. No
JavaScript/TypeScript AST support or universal framework support is claimed.

A frontend candidate must cite the exact client-call line via BM25/label words,
or that line must contain at least two distinct meaningful query words (common
inflections count), before the relation may suggest the Go handler. Existing handler and
one direct static service-call candidate gain located evidence without a score
increase. An absent handler can enter at the frontend evidence score because
the exact method+path link transfers the source candidate's relevance without
an additional ranking penalty. One further static call to a service enters
at half that score, subject to the usual threshold. Both are `VERIFY`, never
proof of a required edit. These factors are uncalibrated review heuristics,
not probabilities. This relation is used
only in code-evidence Hybrid, not BM25-only or Semantic-only. Engine identity
is `engine-3.4o-http-route-score`; previous saved analyses do not
change retroactively.

### Inserted-task context and supporting flows (2026-10-22)

For `ADD_TASK` with a verified one-predecessor/one-successor insertion,
path/function matching also considers tokens shared by the new task label and
its adjacent BPMN labels. At least two shared words are required. This uses
the existing path (10%) and function/name (15%) weights, not a new task-ID
match or a claim that neighbouring source code must change. BM25 and semantic
queries remain unchanged. The direct-ID contribution remains zero when the
generated ID does not occur in current source; all such candidates remain
`VERIFY` unless independent strong evidence exists. No business vocabulary or
Simple-ERP path is embedded in this rule.

All four public methods now display the highest-ranked function/class/component
candidate per file before applying Top-K. This makes file-level Top-K slots
comparable and stops several functions from one file consuming the review list.
BM25 and Semantic retain their full artifact rankings as the internal inputs to
Fusion/Hybrid RRF; only standalone display and final fused display are
file-diverse. No BM25, cosine, or RRF score is increased by this selection.
Selection does not prove that a newly visible file needs a change. Symbol-level
retrieval comparisons remain a separate research question, not measured by the
current file-level evaluator.

A plain, unlabelled old sequence flow replaced by two plain flows around the
new task is retained as three raw `MODIFY_SEQUENCE_FLOW` records in canonical
JSON with `part_of` pointing to the new task ID. Repository retrieval runs
once for that primary task. The web and Markdown reports summarize the three
flow edits under it. Labelled/conditional flows and other topology changes
still receive independent retrieval. Thus raw diff counts and mapped change
counts differ by design; the result must not be read as three extra software
changes. Engine identity is `engine-3.4m-report-unit-tie`.

Method comparison, saved-analysis unresolved counts, and blind reviewer packs
use independent changes as evaluation units; the supporting flow records are
still exported in canonical JSON. Fused candidate-tie warnings refer to the
two displayed RRF scores after file-diverse selection, not ties in an
underlying retriever. File/change overlap is Jaccard set overlap against the
Hybrid list, not precision or recall.

These changes improve focus and report usability; they do not establish
precision/recall without an adjudicated, withheld implementation truth set.

### Separate retrieval scores (2026-10-22)

Hybrid reports now retain `score_breakdown.evidence_score` before rank fusion.
`semantic` is cosine-based meaning similarity; `rrf` is normalized ranking
agreement. The legacy `confidence` and `total` fields still contain the final
method ranking score for compatibility, **not** a probability of required code
change. For a new task/gateway, an absent source ID is expected; the interface
labels it accordingly, but the existing 45% ID contribution remains zero.
This is an explanation/serialization change, not weight recalibration or an
accuracy claim. Engine identity `engine-3.4j-separated-score-explanation`
keeps saved analysis IDs distinct. Recalibration requires adjudicated ground
truth and comparison across controlled scenarios.

### Corpus and query correction (2026-10-21)

Default inventory now excludes process-specification trees under conventional
`docs/bpm`, `docs/bpm-*`, `docs/bpmn`, `docs/bpmn-*`, and
`docs/process-model*` paths; audit/review/test-plan and phase-design artifacts;
build/test outputs; and dependency lockfiles. BPMN input files remain separately readable
and are never prediction documents. Other naming schemes require explicit
`--ignore` rules. This is a leakage/noise policy, not a Simple-ERP mapping rule.

BM25 and the deterministic branch of Hybrid now query changed element names,
state/data terms, and bounded predecessor/successor names. Actor words are
included for actor-role changes. Generated element/flow IDs and change-type
labels no longer contribute fragmented lexical terms; exact changed-element
IDs remain eligible for `TRACE_ID` evidence and the 45% direct-trace signal.
Unlabelled elements fall back to their IDs. Consequently, a newly generated
task ID absent from code cannot boost an unrelated file merely through shared
ID fragments. BM25-only remains a lexical baseline and does not infer actions.

Semantic-only and the Semantic side of RRF use the same focus principle:
the changed task's own label dominates a named add/remove embedding query.
At most one short process phrase shared by task documentation and adjacent
labels is appended; full instruction paragraphs are omitted. When the
documentation has no shared phrase, one token common to two adjacent labels
can provide context. The deterministic/BM25 branch still uses neighboring
labels but excludes the long instruction paragraph for named add/remove tasks;
structural hints use the changed label and same short anchor. Generic
`ADD_TASK`/`userTask` boilerplate and actor words for non-role changes are
omitted; gateway direction and state/data text remain where relevant. This is
query construction, not a new embedding model or a calibrated weight.

BM25/label evidence now cites the source line with the most distinct query
terms, or the path when only path tokens match. This changes explanations,
not BM25 arithmetic. Analysis identity is `engine-3.4i-focused-retrieval-queries`
to prevent collisions with previously saved reports. The source corpus and
query have changed, so old and new reports must not be treated as a controlled
method comparison even if BPMN inputs match. No accuracy improvement is
claimed without hidden, adjudicated ground truth.

### Structured evidence correction (2026-10-19)

Structured hints now require recognizable syntax and a query token within the
same displayed source line (maximum 240 characters). SQL elsewhere in a function
cannot label a task-ID assignment as DATABASE_TABLE. Test-directory membership,
the word `trace`, and arbitrary uppercase IDs alone do not establish a test name,
audit action, or state. Patterns cover a conservative subset of SQL, route
declarations, test declarations, audit calls, and state/status assignments.
These remain text heuristics; multiline statements and framework-specific forms
can be missed, and comments embedded in other syntax are not fully parsed.
Weights are unchanged; unsupported structured clues now contribute zero.

Engine identity is bumped to `engine-3.4e-line-evidence` so reruns do not collide
with saved reports from the earlier algorithm. Historical reports are preserved.
Compare methods using fresh reports from the same engine version.

The web Code/Tests/Docs filter uses reported layer plus generic path/extension
conventions. Code includes remaining configuration/schema files. This is a display
classification, not a language/framework guarantee. File grouping retains every
change/function association and original rank, without averaging scores or actions.
Filters do not alter retrieval, exports, or evaluation inputs.

## Interpretation boundary

The method identifies candidates, not proven causal impacts. A stable BPMN ID found in source is direct traceability evidence; BM25 and lexical similarity remain retrieval evidence. A high score is not empirical probability.

## Graph matching and change taxonomy

Stable IDs are matched first. Unmatched elements are compared only within the same BPMN type using normalized label, actor/lane, degree, and neighboring labels. A unique score of at least `0.65` is accepted; an equal best match abstains with `UNRESOLVED_MATCH`.

Process and parent-subprocess scope constrain structural fallback: elements in different known processes or subprocesses are not paired merely because their labels are similar. The parser also retains boundary attachment, gateway direction/default flow, and event-definition metadata for later semantic rules.

Implemented taxonomy:

- task/gateway additions and removals;
- task label and actor/lane changes;
- gateway outgoing-condition changes;
- sequence-flow add/remove/rewire/name/condition changes;
- data-requirement additions/removals;
- data-state additions/changes;
- terminal outcome changes;
- unresolved matches.

### Composite context

Atomic changes remain the canonical taxonomy. A deterministic topology pass may attach an optional context rather than replacing those changes:

- `INSERT_TASK`: one added task bridges predecessor and successor nodes that had a direct baseline flow;
- `REMOVE_TASK_WITH_REWIRE`: one removed task previously bridged nodes that are directly connected in the new graph;
- `ADD_GATEWAY_BRANCH`: one added gateway has multiple incoming or outgoing branches.

The context records bounded predecessor/successor identities and sorted related flow IDs. It explains a multi-edit operation but does not prove behavioral equivalence. Hybrid and BM25 queries use the context terms, so a newly inserted task can retrieve implementation expressed through its neighboring responsibilities even when its new label is absent from source.

## BM25

Tokenization lowercases Unicode letters/digits and splits punctuation, snake/kebab separators, and camel-case boundaries. Domain stop words are not removed. BM25 uses `k1=1.2`, `b=0.75`, and:

```text
IDF(q) = ln(1 + (N - df(q) + 0.5) / (df(q) + 0.5))
```

Documents and ties use stable artifact IDs. Within each change query, BM25 is normalized as `documentScore / maximumScore`.

Query construction is change-aware: additions use the new element, removals use the old element, modifications use both, and recognized composite operations add predecessor/successor context. This avoids mixing removed wording into pure-addition queries while preserving both sides of a modification.

Generic process words in a BPMN label or its neighbors can also occur in
unrelated modules. BM25 normalization and rank fusion are relative rankings, so
they can place such cross-module matches high even without a causal dependency.
These remain `VERIFY`; there is no Simple-ERP-specific suppression rule. Any
new generic negative evidence or weight change must be evaluated against
independent, hidden ground truth before it becomes the product default.

The BM25 baseline uses only this normalized score and the changed element's own text (not graph-neighbour labels), requires positive lexical evidence, and labels every result `VERIFY`. It does not inherit trace-ID action classification from the legacy weighted method.

## Same-input comparison and reviewer boundary

The web comparison repeats byte-identical BPMN/repository/top-k inputs and requires one shared method-independent input fingerprint. BM25, Semantic, BM25 + Semantic RRF, and BPMN-Context-Aware Hybrid Retrieval participate. Candidate count, unresolved count, set overlap, and ranked lists describe retrieval behavior only. They must not be reported as accuracy before reconciliation against hidden ground truth.

Reviewer export happens after inference and remains browser-local. A reviewer may accept a predicted file, change its expected action, add an omitted file manually, or record an empty negative case. Predictions are prompts rather than truth; independent reviewers must use frozen implementation evidence, and disagreements remain explicit through `reconcile`. Annotation JSON never becomes analyzer input.

## Legacy weighted evidence score (not the current production Hybrid)

The earlier experimental score was:

```text
0.45 exact stable trace ID
+ 0.15 symbol-token overlap
+ 0.10 path/module-token overlap
+ 0.10 label/domain-token overlap
+ 0.10 structured hint
+ 0.10 normalized BM25
```

Components are clipped to `[0,1]`, totals are rounded to four decimals, and the full breakdown is emitted. Structured hints currently recognize generic test, route, SQL/table, audit/trace, and uppercase state/status shapes. They are evidence labels, not framework-specific parsers.

That legacy strategy omits candidates below `0.20` and may classify exact-trace candidates using the `0.70` threshold. It is retained for regression comparison only. These thresholds and weights do not describe the current production Hybrid. Current Hybrid, standalone BM25, and standalone Semantic return `VERIFY` for retrieved candidates; no candidate produces an unresolved reason.

## Action semantics

- `ADD`: strong evidence links an added BPMN requirement to the candidate context.
- `MODIFY`: strong evidence links a modified BPMN element to existing implementation.
- `REMOVE`: strong evidence links a removed BPMN requirement to existing implementation.
- `VERIFY`: relevance is supported, but a necessary implementation change is not proven.
- `NO_ACTION`: reserved for future evidence-backed negative rules; never inferred from absence.
- `UNRESOLVED`: explicit abstention; represented by unresolved impacts in Phase 1.

Human review remains mandatory, especially for `ADD`/`REMOVE`, because a candidate file may be a location to modify rather than a file literally created/deleted.

## Static Go relation propagation (3.4B–3.4D)

The indexing registry separates SourceAdapter from RelationExtractor. A shared SourceRelation model carries relation kind, indexed target ID and located evidence. Only the explicitly supported `CALLS`, `CALLS_METHOD`, and `INTERFACE_DISPATCH` kinds propagate. Register adapters in priority order, with versioned names when behavior changes. Generic text is the automatic last fallback. Analysis identity binds the ordered registry identity in `engine-3.4d`.

The canonical `source_analysis` section records actual processing per file. `TEXT_ONLY` means lexical indexing; `AST_SYMBOLS` means syntax extraction; `STATIC_CALLS` means free-function extraction; `TYPE_RELATIONS` means the conservative receiver/interface pass ran. None means full call-graph coverage. Malformed Go is recorded as generic text with `fallback=true`. Extractor failure retains source artifacts but does not advertise its capabilities. No probability is invented from capability metadata; confidence remains a heuristic relevance score. Cross-language accuracy still requires independent evaluation.

The allowed inventory supplies same-package free-function calls and calls through explicit imports whose module is present in an indexed `go.mod`. It also resolves receiver methods when a receiver, parameter, explicit local declaration, composite literal, or receiver struct field provides a repository-local named type. An interface call resolves only when exactly one indexed named type implements every supported interface method with matching normalized parameter/result types and variadic shape. Calls carry source line and bounded excerpt. Duplicate targets, shadowing, multiple implementations, embedded interfaces, generic/promoted dispatch, unsupported type expressions, replace/workspace directives, and implementations outside the inventory abstain. The analyzer never builds or executes the repository.

Hybrid seeds are accepted function/method candidates with exact trace evidence before top-k truncation. One relation in either direction can introduce a previously absent candidate, always `VERIFY`. Its dependency score and total equal `round(seed score * 0.5)` and must pass the normal candidate threshold. Only original seeds expand; cycles cannot produce further hops. Direct candidates keep their score; ties use deterministic file/symbol order. The coefficient is a documented conservative heuristic, not calibrated accuracy. BM25-only and Semantic-only do not propagate. Evidence distinguishes `CALLS_SYMBOL`/`CALLED_BY`, `CALLS_METHOD`/`CALLED_BY_METHOD`, and `INTERFACE_DISPATCH`/`IMPLEMENTS_INTERFACE` at the located call site.

## LLM boundary

The provider contract receives only a filtered change and its evidence-bearing candidates. `NoLLMProvider` is the only Phase 1 implementation and performs no network call. Future providers may interpret or rerank evidence but must not invent paths/symbols, ingest an entire repository, or override the evidence requirement.

## Semantic retrieval baseline

The isolated `semantic` strategy encodes a bounded BPMN-change query and bounded
file/symbol documents with locally cached `all-MiniLM-L6-v2`, then ranks cosine
similarity with deterministic file/symbol tie-breaking. Stable BPMN IDs are
excluded from this query so the experiment measures vocabulary matching rather
than exact trace matching. The default threshold is `0.25`; the score is a
heuristic similarity, not calibrated confidence or probability. Every retained
candidate is `VERIFY` and carries `SEMANTIC_SIMILARITY` evidence. No model error
falls back to another method.

The adapter invokes only its embedded trusted Python helper, forces offline
model loading, bounds documents and process output, and honors cancellation. It
never imports or executes code from the analyzed repository. Model/provider are
stored in canonical report metadata. Semantic participates in BPMN-Context-Aware
Hybrid through RRF; it cannot promote deterministic actions. Cross-machine
numerical reproducibility and retrieval accuracy remain evaluation work. See
[SEMANTIC_RETRIEVAL.md](SEMANTIC_RETRIEVAL.md).
