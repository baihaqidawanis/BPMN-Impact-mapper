# Methodology

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

The BM25 baseline uses only this normalized score, requires positive lexical evidence, and labels every result `VERIFY`. It does not inherit trace-ID action classification from the hybrid method.

## Same-input comparison and reviewer boundary

The web comparison repeats byte-identical BPMN/repository/top-k inputs and requires one shared method-independent input fingerprint. BM25, Semantic, BM25 + Semantic RRF, and Graph-Aware Hybrid participate. Candidate count, unresolved count, set overlap, and ranked lists describe retrieval behavior only. They must not be reported as accuracy before reconciliation against hidden ground truth.

Reviewer export happens after inference and remains browser-local. A reviewer may accept a predicted file, change its expected action, add an omitted file manually, or record an empty negative case. Predictions are prompts rather than truth; independent reviewers must use frozen implementation evidence, and disagreements remain explicit through `reconcile`. Annotation JSON never becomes analyzer input.

## Evidence and score

The default weighted score is:

```text
0.45 exact stable trace ID
+ 0.15 symbol-token overlap
+ 0.10 path/module-token overlap
+ 0.10 label/domain-token overlap
+ 0.10 structured hint
+ 0.10 normalized BM25
```

Components are clipped to `[0,1]`, totals are rounded to four decimals, and the full breakdown is emitted. Structured hints currently recognize generic test, route, SQL/table, audit/trace, and uppercase state/status shapes. They are evidence labels, not framework-specific parsers.

Candidates below `0.20` are omitted. A candidate with exact trace evidence and score at least `0.70` receives the change-derived action (`ADD`, `REMOVE`, or `MODIFY`), except `MODIFY_TASK_LABEL`, which remains `VERIFY`: wording traceability alone does not prove a software-behavior change. Other retained candidates also receive `VERIFY`. BM25-only candidates can never cross the candidate threshold under default weights, preventing weak lexical results from appearing without another signal. If an accepted Go function/type/method candidate explains a file, its package-level candidate is suppressed while distinct matching symbols remain separate. No candidate produces an unresolved reason. Top candidates within `0.01` produce a tie warning.

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
stored in canonical report metadata. A model checksum, cross-machine numerical
reproducibility, and retrieval accuracy remain evaluation work. Semantic is not
part of the current Hybrid score; final thesis inclusion remains advisor-gated.
See [SEMANTIC_RETRIEVAL.md](SEMANTIC_RETRIEVAL.md).
