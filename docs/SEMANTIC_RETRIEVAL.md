# Semantic Retrieval Experimental Baseline

## Status

Semantic Retrieval is implemented as an isolated offline experimental method.
It remains separately selectable for controlled evaluation. Experimental RRF
fusion is implemented, but no accuracy improvement is claimed before evaluation
against independently reviewed hidden ground truth.
Inclusion in the final thesis comparison remains advisor-gated.

This is a defensible undergraduate-thesis method because it addresses vocabulary
mismatch between BPMN activity descriptions and source-code identifiers. Its
benefit remains a hypothesis to test against hidden, independently reviewed
ground truth. A successful synthetic example proves functionality, not accuracy
or generalization.

## Research position

The target automatic comparison is:

1. BM25-only: deterministic lexical retrieval;
2. Semantic-only: embedding similarity without exact-word dependence;
3. BM25 + Semantic rank fusion;
4. Graph-Aware Hybrid: fusion plus exact, structural, and source-relation evidence.

Semantic Retrieval stays independently measurable even though experimental fusion
is now available. It is not RAG: retrieval ranks candidates, while RAG additionally sends
retrieved context to a generative model.

## Implemented method contract

- Construct a bounded query from change type, old/new label, element type,
  actor/lane, and immediate graph neighbors.
- Construct allowed repository documents at file and symbol granularity.
- Encode queries and documents with one pinned local embedding model.
- Rank by cosine similarity with deterministic tie-breaking.
- Emit only indexed files/symbols and bounded source excerpts as evidence.
- Classify results as `VERIFY`; similarity alone does not prove code must change.
- Record provider and model identity in each report. Model checksum, hardware,
  and latency belong in the frozen experiment manifest and remain follow-up work.
- Keep repository content local; do not call an external embedding API.

The implementation uses cached `all-MiniLM-L6-v2` through
`sentence-transformers`, with offline-only loading, normalized embeddings,
cosine similarity, a default `0.25` threshold, and deterministic file/symbol
tie-breaking. It excludes stable IDs from queries and emits only `VERIFY`.
That model choice is not yet validated for source code or Indonesian BPMN
labels. A smoke run proves operability only, not accuracy.

## Implemented rank fusion

`fusion` runs BM25-only and Semantic-only against the same changes and indexed
artifacts. Candidates are identified by normalized repository-relative file and
symbol. Their raw BM25 and cosine values are deliberately not added because the
two scales have different meanings. Instead, Reciprocal Rank Fusion uses:

`RRF(d) = 1 / (60 + rank_bm25(d)) + 1 / (60 + rank_semantic(d))`

An absent rank contributes zero. The implementation normalizes by the maximum
two-list contribution `2 / 61`, sorts ties by file and symbol, keeps Top-K, and
records both contributing ranks as `RRF_FUSION` evidence. The normalized value
is rank consensus, not probability. `fusion` always emits `VERIFY`.

`hybrid` applies the same RRF mechanism to the existing graph-aware ranking and
Semantic ranking. A candidate introduced only by Semantic is forced to
`VERIFY`; a stronger action is retained only when the graph-aware method already
supported it. On an equal RRF score, a graph-supported candidate precedes a
Semantic-only candidate before the file/symbol tie-break. This prevents
embedding similarity from inventing or displacing stronger causal evidence.

The Python adapter serializes model workers per server process and retains only
the most recent input hash and vectors in memory. Therefore the web comparison
can reuse identical embeddings across Semantic, Fusion, and Hybrid without
persisting repository text.

## Recent literature (2022-2024)

1. B. Di Martino, C. Colucci Cante, A. Esposito, and M. Graziano,
   *A tool for the semantic annotation, validation and optimization of business
   process models*, Software: Practice and Experience, 53(5), 1174-1195,
   2023. DOI: [10.1002/spe.3184](https://doi.org/10.1002/spe.3184).
   The work uses word embeddings and cosine similarity to match BPMN labels
   with ontology concepts. It does not retrieve source-code impacts.

2. O. L. Vega-Marquez, H. Duarte, and V. Collazos, *Mapping Study on
   Traceability Between BPMN Models and Source Code*, IJASEIT, 12(1),
   218-230, 2022. DOI:
   [10.18517/ijaseit.12.1.16170](https://doi.org/10.18517/ijaseit.12.1.16170).
   This establishes BPMN-to-source traceability as a research area and reports
   graph techniques for versioning and impact analysis. It is a mapping study,
   not validation of this algorithm.

3. A. Majidzadeh, M. Ashtiani, and M. Zakeri-Nasrabadi, *Multi-type
   requirements traceability prediction by code data augmentation and
   fine-tuning MS-CodeBERT*, Computer Standards & Interfaces, 90, 103850,
   2024. DOI:
   [10.1016/j.csi.2024.103850](https://doi.org/10.1016/j.csi.2024.103850).
   It supports semantic text-to-code traceability, but uses supervised
   fine-tuning and Java-focused datasets rather than lightweight unsupervised
   retrieval.

Together these papers support BPMN semantic comparison, the BPMN-to-source
traceability problem, and semantic natural-language-to-code matching. None
directly proves BPMN-change-to-source-impact accuracy.

## Evaluation gate

- Freeze repository revision, BPMN mutations, model, configuration, and Top-K.
- Include vocabulary-mismatch cases without stable task IDs in source.
- Compare methods on byte-identical inputs.
- Create ground truth only after inference using blind review and adjudication.
- Report Recall@k, Precision@k, MRR/MAP, file/module/layer accuracy, action
  classification, unresolved rate, false discovery proportion, latency, and
  reproducibility.
- Do not claim cross-language or cross-system generalization from Simple-ERP.

The prototype was user-authorized on 2026-10-10. Final thesis inclusion and
using fusion as the thesis treatment still require advisor approval plus hidden-ground-truth
evaluation.
