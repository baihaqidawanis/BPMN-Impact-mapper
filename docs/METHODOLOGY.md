# Methodology

## Interpretation boundary

The method identifies candidates, not proven causal impacts. A stable BPMN ID found in source is direct traceability evidence; BM25 and lexical similarity remain retrieval evidence. A high score is not empirical probability.

## Graph matching and change taxonomy

Stable IDs are matched first. Unmatched elements are compared only within the same BPMN type using normalized label, actor/lane, degree, and neighboring labels. A unique score of at least `0.65` is accepted; an equal best match abstains with `UNRESOLVED_MATCH`.

Implemented taxonomy:

- task/gateway additions and removals;
- task label and actor/lane changes;
- gateway outgoing-condition changes;
- sequence-flow add/remove/rewire/name/condition changes;
- data-requirement additions/removals;
- data-state additions/changes;
- terminal outcome changes;
- unresolved matches.

## BM25

Tokenization lowercases Unicode letters/digits and splits punctuation, snake/kebab separators, and camel-case boundaries. Domain stop words are not removed. BM25 uses `k1=1.2`, `b=0.75`, and:

```text
IDF(q) = ln(1 + (N - df(q) + 0.5) / (df(q) + 0.5))
```

Documents and ties use stable artifact IDs. Within each change query, BM25 is normalized as `documentScore / maximumScore`.

The BM25 baseline uses only this normalized score, requires positive lexical evidence, and labels every result `VERIFY`. It does not inherit trace-ID action classification from the hybrid method.

## Configuration-driven baseline

A versioned frozen JSON file maps exact change selectors (`change_type` and/or `element_id`) to clean repository-relative file/symbol targets. Targets are emitted only if present in the allowed index. Missing targets produce warnings. `confidence=1` means exact rule-and-index match, not probability that the software truly requires change. Rule preparation must happen before evaluation and its human effort must be measured independently to avoid disguising ground truth as configuration.

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

Candidates below `0.20` are omitted. A candidate with exact trace evidence and score at least `0.70` receives the change-derived action (`ADD`, `REMOVE`, or `MODIFY`); other retained candidates receive `VERIFY`. BM25-only candidates can never cross the candidate threshold under default weights, preventing weak lexical results from appearing without another signal. No candidate produces an unresolved reason. Top candidates within `0.01` produce a tie warning.

## Action semantics

- `ADD`: strong evidence links an added BPMN requirement to the candidate context.
- `MODIFY`: strong evidence links a modified BPMN element to existing implementation.
- `REMOVE`: strong evidence links a removed BPMN requirement to existing implementation.
- `VERIFY`: relevance is supported, but a necessary implementation change is not proven.
- `NO_ACTION`: reserved for future evidence-backed negative rules; never inferred from absence.
- `UNRESOLVED`: explicit abstention; represented by unresolved impacts in Phase 1.

Human review remains mandatory, especially for `ADD`/`REMOVE`, because a candidate file may be a location to modify rather than a file literally created/deleted.

## LLM boundary

The provider contract receives only a filtered change and its evidence-bearing candidates. `NoLLMProvider` is the only Phase 1 implementation and performs no network call. Future providers may interpret or rerank evidence but must not invent paths/symbols, ingest an entire repository, or override the evidence requirement.
