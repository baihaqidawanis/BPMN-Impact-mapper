# BPMN Impact Mapper - Baseline Methods

## Research position

The primary experiment compares four automatic retrieval configurations on the
same frozen repository, BPMN pair, Top-K, and independently reviewed ground
truth. They retrieve candidate software impacts; they do not prove causality or
generate source-code changes.

| Method | Implemented mechanism | Safe interpretation |
|---|---|---|
| BM25 | Deterministic lexical ranking over bounded file/symbol documents | Exact vocabulary baseline; all candidates are `VERIFY` |
| Semantic | Offline sentence embeddings plus cosine similarity | Vocabulary-mismatch baseline; all candidates are `VERIFY` |
| Fusion | Reciprocal Rank Fusion (RRF) over BM25 and Semantic ranks | Lexical-semantic rank consensus; all candidates are `VERIFY` |
| BPMN-Context-Aware Hybrid Retrieval | RRF over deterministic BPMN-context/evidence ranking and Semantic, plus conservative Go source-relation propagation | Main treatment; strong actions survive only when independently supported by deterministic evidence |

Configuration-Driven JSON remains an optional legacy comparator. It is not a
primary automatic method because preparing its mappings requires manual prior
knowledge of the analyzed software.

## 1. BM25 lexical baseline

The BPMN change becomes a query containing its change type, relevant old/new
labels, element type, actor, and immediate topology context. Repository files
and native Go symbols become documents. Tokenization splits case and common
identifier separators. BM25 uses `k1=1.2`, `b=0.75`, stable document ordering,
and deterministic file/symbol tie-breaking.

BM25 is grounded in the probabilistic relevance framework described by
Robertson and Zaragoza, *The Probabilistic Relevance Framework: BM25 and
Beyond* (2009), DOI: <https://doi.org/10.1561/1500000019>.

## 2. Semantic retrieval baseline

The same bounded BPMN change and repository artifacts are encoded with the
locally cached `all-MiniLM-L6-v2` sentence-transformer model. Cosine similarity
ranks candidates. Stable BPMN IDs are deliberately excluded from the semantic
query so this baseline tests meaning and vocabulary mismatch rather than exact
trace lookup. The default threshold is `0.25`; similarity is not probability.

Relevant foundations:

- Di Martino et al., *A tool for the semantic annotation, validation and
  optimization of business process models* (2023), DOI:
  <https://doi.org/10.1002/spe.3184>;
- Majidzadeh et al., *Multi-type requirements traceability prediction by code
  data augmentation and fine-tuning MS-CodeBERT* (2024), DOI:
  <https://doi.org/10.1016/j.csi.2024.103850>.

These papers support BPMN semantic matching and semantic requirement-to-code
traceability. Neither paper validates this mapper's exact algorithm.

## 3. BM25 + Semantic RRF

Fusion runs BM25 and Semantic independently, then combines their ranks:

```text
RRF(d) = 1 / (60 + rank_bm25(d)) + 1 / (60 + rank_semantic(d))
```

Raw BM25 and cosine values are not added because their scales have different
meanings. The implementation follows Cormack, Clarke, and Buettcher,
*Reciprocal Rank Fusion Outperforms Condorcet and Individual Rank Learning
Methods* (SIGIR 2009), DOI: <https://doi.org/10.1145/1571941.1572114>.

## 4. BPMN-Context-Aware Hybrid Retrieval

The deterministic side combines stable trace IDs, symbols, paths/modules,
labels/domain terms, structural hints, BM25, BPMN neighborhood context, and the
implemented conservative Go call/type relations. It is fused with Semantic by
RRF. A Semantic-only candidate remains `VERIFY`; Semantic similarity cannot
promote a candidate to `ADD`, `MODIFY`, or `REMOVE`.

The research problem is supported by Vega-Marquez et al., *Mapping Study on
Traceability Between BPMN Models and Source Code* (2022), DOI:
<https://doi.org/10.18517/ijaseit.12.1.16170>. The study identifies graph use as
an emerging technique for versioning and impact analysis. BECIA also discusses
dependency/traceability-based change-impact propagation:
<https://doi.org/10.1007/s41870-023-01471-9>.

BPMN-Context-Aware Hybrid Retrieval is a synthesis implemented by this project,
not a claim that one cited paper proposed the same pipeline. “BPMN context”
means the changed element's nearby predecessor/successor, actor/lane, gateway,
and incoming/outgoing flow. It is not a Neo4j graph database, a unified BPMN-
source knowledge graph, or graph traversal across the repository.

## Evaluation boundary

- Compare all methods on byte-identical inputs.
- Freeze repository revision, BPMN files, model/runtime identity, and Top-K.
- Keep ground truth outside inference and construct it through blind review and
  explicit adjudication.
- Report Recall@K, Precision@K, MRR/MAP, file/module/layer/action accuracy,
  unresolved rate, false discovery proportion, latency, and reproducibility.
- Treat synthetic scenarios as functionality controls, not evidence of
  real-system accuracy or cross-language generalization.

The intended contribution is an auditable BPMN-change-to-source-impact pipeline
that combines established retrieval and traceability techniques, emits bounded
evidence, and abstains when support is insufficient.
