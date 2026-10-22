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
4. BPMN-Context-Aware Hybrid Retrieval: BPMN-neighbour context in BM25, fused with Semantic retrieval, plus bounded explanatory evidence.

Semantic Retrieval stays independently measurable even though experimental fusion
is now available. It is not RAG: retrieval ranks candidates, while RAG additionally sends
retrieved context to a generative model.

## Implemented method contract

- Construct a bounded query from changed labels, state/data/condition text.
  For a named added/removed task, use its own responsibility plus at most one
  short process phrase shared by its documentation and an adjacent BPMN label;
  omit the rest of the instruction paragraph. If no shared phrase exists,
  use a token common to two adjacent labels. Retain full neighbors for
  topology-only changes and otherwise unlabeled elements. Include actor/lane on role changes;
  use element type/reason only when a useful label is absent. Generated BPMN
  IDs and generic change-type boilerplate are not embedded.
- Construct allowed repository documents at file and symbol granularity.
- Embed a bounded first passage per artifact and, within the 5,001-text
  request limit, a bounded tail and middle passage for long artifacts. Select
  the best cosine score per artifact and show a bounded excerpt from that passage.
- Encode queries and documents with one pinned local embedding model.
- Rank by cosine similarity with deterministic tie-breaking.
- Emit only indexed files/symbols and bounded source excerpts as evidence.
- Classify results as `VERIFY`; similarity alone does not prove code must change.
- Record provider/model, embedded-adapter SHA-256, Python and
  `sentence-transformers` versions, embedding dimension, and a deterministic
  loaded-model artifact SHA-256 in each report. Hardware identity remains
  experiment-run metadata rather than canonical prediction input.
- Keep repository content local; do not call an external embedding API.

The implementation uses cached `all-MiniLM-L6-v2` through
`sentence-transformers`, with offline-only loading, normalized embeddings,
cosine similarity, a default `0.25` threshold, and deterministic file/symbol
tie-breaking. It excludes stable IDs from queries and emits only `VERIFY`.
That model choice is not yet validated for source code or Indonesian BPMN
labels. A smoke run proves operability only, not accuracy. The model artifact
fingerprint hashes loaded weights, module configuration, tokenizer vocabulary,
dimension, selected model name, and library version without exposing a local
cache path.

Each passage is limited to 600 characters before repository metadata is added.
This reduces silent truncation by the model's token window, but character count
cannot guarantee a specific number of wordpieces. For long symbols/files, at
most three fixed passages (head, tail, middle) are represented; code between
these samples can still be missed. Every artifact gets its head before extra
passages are allocated round-robin. If the embedding-text budget prevents an
extra passage, the report emits `SEMANTIC_PASSAGE_LIMIT`. The excerpt selects
the line with most query-word overlap within the best passage, or its centre
if there is no overlap. It is an inspection aid, not an attribution of the
cosine score to that one line. Cosine values and RRF agreement remain ranking
signals, not calibrated impact probabilities. No DPR or code-specific model is
installed or claimed here.

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

For file-level reports, standalone BM25 and Semantic and both fused methods
display the highest-ranked artifact per file before Top-K. Fusion still uses
the complete artifact-level BM25/Semantic lists internally before this display
selection, so RRF arithmetic is unchanged. Reports from older engine versions
may contain duplicate files and are not directly comparable at a fixed Top-K.

`hybrid` applies the same RRF mechanism to BPMN-context BM25 and Semantic
ranking. Context BM25 includes neighbouring BPMN labels; the standalone BM25
baseline does not. Exact task IDs, nearby IDs, file paths, function names, and
code-pattern hints remain bounded explanatory evidence, not independently
weighted ranking branches. Path/function words can already occur in indexed
BM25 and Semantic text; another weight would double-count them. A newly added
task ID is expected to be absent in the old source and is not penalized.
All retrieved Hybrid candidates are `VERIFY`: neither agreement nor an exact
ID alone proves that a code edit is required. A separate versioned reranker
could be tested later against blinded ground truth, not silently added to the
current method.

The Python adapter serializes model workers per server process and retains only
the most recent input hash and vectors in memory. Therefore the web comparison
can reuse identical embeddings across Semantic, Fusion, and Hybrid without
persisting repository text.

The Go-to-Python request is UTF-8 JSON and is decoded from stdin bytes, not
from the Windows console code page. This prevents non-ASCII source text from
becoming surrogate-escaped characters rejected by the tokenizer. Build caches
such as `.gocache` are excluded from repository inventory; they are not source
evidence and needlessly enlarge the embedding corpus.

Semantic Retrieval is not a vector database and is not RAG. For each bounded
analysis, the local model embeds the BPMN-change query and allowed indexed
source artifacts, then calculates cosine similarity in process memory. A vector
database would only make repeated or very large retrieval faster; it is not
required to make the similarity semantic. The workflow becomes RAG-like only
when selected, bounded evidence is subsequently supplied to a generative LLM
for planning.

The web performs an explicit capability check before a method that needs
embeddings. The check embeds one fixed, non-repository sentence using the same
configured adapter. Successful readiness is cached for the server lifetime;
failures are not cached. The public response exposes only provider/model identity
and a bounded generic message, while detailed runtime errors remain in the local
server log. This is an operational preflight, not an accuracy measurement.

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

The local worker now bounds JSON input size, serializes one embedding request at
a time, limits numerical-library threads, and uses small embedding batches.
An in-process cache retains up to four corpus/query embedding results within a
64 MiB vector budget, avoiding repeated Python model launches when comparing
methods on identical inputs. The cache is discarded on server restart and does
not change the computed vectors or ranking.
Memory and input-limit failures are surfaced to the API without local paths or
subprocess output; the UI does not silently substitute another method. These
controls reduce resource spikes but do not guarantee that the local model fits
every machine. The Research batch UI is exploratory and may be stopped; it does
not compute accuracy or replace the hidden-ground-truth benchmark.

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
