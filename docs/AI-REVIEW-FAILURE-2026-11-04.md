# AI review failure: actual session audit

Analysis: 1b460114e1d139606933f7b1af417e1e. Session: 7a015a183913b523094e8ef14adedc31. Plan: 1f86b437adbe02a247e9a0b14135f545.

## Confirmed findings

- Both parsed BPMN graphs are present in the saved process_context. XML is parsed by Go; the model receives semantic graph data, not a visual screenshot.
- Scope is focused, with 20 per-change child sessions, 153 total turns and 123 tool calls. This is targeted source retrieval, not an exhaustive file scan.
- Nineteen children stopped at the context guard while still in Analyst. One reached Reviewer with a migration recommendation. That reviewer flagged schema-only evidence as insufficient for control-flow implementation.
- Final combined review also exceeded its request budget. Plan status is NEEDS_REVIEW, with 19 requirements unplanned. This output is incomplete, not evidence that only the migration is affected.
- Hybrid rankings remain unchanged; partial plan export exposes only surviving drafts, so its one-file list must not be compared as a completed replacement of retrieval.

## Applied corrections

- Factor identical before/after elements and flows into shared graph entries. Provide complete ID aliases for repeated long IDs. No node, flow, condition or source excerpt is removed.
- Use candidate-relative evidence locations instead of repeating the same file path. Saved evidence and source binding remain unchanged.
- Factor repeated JSON action-schema properties for more than three targets. The engine still strictly checks each change/file/function/evidence tuple; enum membership alone grants no authority.
- Explicitly instruct planners that BPMN is a blueprint and schema existence alone does not justify a migration.
- Provider identity bumped to v5. Create a new session after backend restart.

## Verification and limits

The 20 captured child turns replay successfully through the real HTTP adapter to a localhost stub, including its context guards; no lab inference was made. Lossless graph transport has a reconstruction/immutability regression test. Adapter, planning, agentjobs and API tests pass.

This verifies these captured inputs, not arbitrary later evidence growth or actual model judgment. The final all-change reviewer still has a bounded context and may mark a large combined proposal NEEDS_REVIEW. Full-repository scan was not performed in this focused session. Do not claim full repository coverage or validated model accuracy.

## Where BPMN context is visible

- Mapping: expand "BPMN context recorded for this change".
- impact report JSON: changes[].context (process name, predecessors, successors, branch terms, branch changes, condition delta, related flows, baseline anchors where applicable).
- AI session JSON: session.bundle.process_context.before / after contain complete semantic graphs, including element names/types/actors and flow endpoints/conditions. Original XML is stored separately with the analysis in bpmn-inputs.json.
- Retrieval context is derived from BPMN topology and used to construct search queries; it is not a separate accuracy percentage or proof of a code dependency.

## Source access log

Planning now exposes Source access log: per-change source-tool name, query, requested file/function, distinct returned file paths, tool errors, truncation and paging metadata. It includes completed focused child batches and the active child; older stored child observations are restored when available. The API field is source_activity. Source excerpts and hidden reasoning are not added to this summary. Search hits are distinct from read-tool excerpts; neither proves complete-file comprehension. Initial seeds are already in the evidence context and are not falsely counted as tool reads. Full-scan chunk coverage is shown separately.

The application uses in-memory BPMN graphs for parsing, graph diff and context. It does not persist a unified BPMN/source knowledge graph in Neo4j. A source tool can retrieve files beyond the initial seed candidates, subject to explicit discovery/tool/context bounds. The observed 123 tool calls are not 123 unique files.

Actual old-session trace count: 23 search_source calls, 100 read/inspect/function calls, six lookup errors, 38 distinct returned paths and 34 distinct paths with successful read-tool results. These are bounded excerpts, not full-file or reasoning-completeness claims. Inventory store, application service, inventory frontend panel and operations integration tests were among the returned read paths.

Browser source-log display and refresh restoration test passes (19.5 seconds) after an initial Chrome tab closure before navigation; backend regression, vet, typecheck and encoding checks pass.
