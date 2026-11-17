# Live lab model probe - 9 November 2026

## Scope

A bounded Analyst comparison was run through the actual Go Ollama adapter, followed by bounded 9B Planner/Reviewer diagnostics. This is one scenario, not a general model benchmark or end-to-end application evaluation. No ground-truth/reference files were supplied to either model. No analyzed repository source was executed or edited.

Models were loaded sequentially, starting from an idle Ollama runtime. Both used context 32768, output budget 4096, temperature 0 and thinking disabled. Analyst system/user/schema/options hashes matched across models for each variant; only model identity and timing differed. The first variant contained the existing backend target with repaired source evidence; the second contained the original five-target backend/test/documentation batch and full before/after process context.

## Observed Analyst results

| Model | Single backend target | Five-target batch |
|---|---|---|
| qwen3.5:9b-q8_0 | RELEVANT, valid references | Backend and integration test RELEVANT; three documents WEAK; engine validation passed |
| bpmn-qwen-coder:30b-q4_k_m | RELEVANT, valid references | All five RELEVANT; invalid evidence reference for a seven-item target; adapter rejected |

9B took approximately 20.28 s cold and 13.30 s warm. Coder took approximately 16.67 s cold and 10.93 s warm. These are single observations including loading where applicable, not latency benchmarks. Coder cited index 7 for a candidate whose available evidence indices were 0-6. The strict validator correctly rejected it; references were not silently repaired or promoted.

This supports preferring 9B for the Analyst role on this example. It does not establish that 9B is better for every role, language or BPMN scenario, or that Coder is generally inferior.

## Planner and Reviewer are still unresolved

Three bounded 9B follow-through diagnostics reused its actual validated decisions, without extra source or reference answers:

1. The first proposed the correct backend/test files, but inline quoted code fragments were broken into incomplete prose/array items. Reviewer rejected the plan.
2. A declared prose-only feedback control improved readability but still produced an uncertain signature/locking proposal. Reviewer rejected it. This control was test-only; it was not silently added to the production Analyst comparison.
3. One allowed revision reached the Planner again but exceeded its context budget before a further inference request. The draft still contained technical errors, including a nil check for a Go string and confusion between the service and store transition entry points. This is not an approved developer plan.

Ten actual inference requests were made in total. No full repository review or complete seven-batch mapping run was performed. The sample does not include the frontend target; do not compute full scenario recall from these bounded follow-through results. No precision/recall dashboard or historical benchmark was updated.

## Narrow program repair after the probes

The revision Planner was carrying source excerpts and long rationales for WEAK targets that it was forbidden to plan. Production v16 now supplies Planner evidence/decisions only for RELEVANT targets, preserving all changed requirements and process graphs. Original immutable target indices are retained even when earlier candidates are omitted. A regression checks that target index 1 is not accidentally renumbered to 0. Canonical reports, mapping ranking and source evidence remain unchanged.

The captured failed Planner revision now reaches the local stub with input 17,372 bytes and total request 24,120 bytes, below the conservative 28,160-byte budget. This is a transport repair, not proof of corrected model reasoning. A live v16 plan was not run.

## Model inventory observed

Local generation models include Qwen3.5 9B Q8, Qwen3-Coder 30B Q4, Qwen3.5 35B-A3B Q4, Qwen3 14B Q4, Qwen3.5 4B Q8, Qwen2.5 3B and GLM-4.7-Flash Q4. A Qwen3 embedding model is a retrieval model, not the planning agent. No 27B alias was registered in Ollama at inspection; the earlier GGUF folder name alone does not verify its identity or quality. Other models were not loaded or tested.

## Artifacts and recommendation

Private raw requests/responses, hashes and validation sessions remain under `.data/model-probes/20261109T134301/` (ignored by Git). Do not publish raw source-bearing payloads without review. Paired results are in `results.json`; three 9B follow-through folders preserve the unsuccessful drafts and review feedback.

Use 9B as the current Analyst baseline, retain strict validation and NEEDS_REVIEW status for unsuccessful plans. Do not declare the entire AI pipeline ready. Evaluate Planner and Reviewer separately on frozen, evidence-complete inputs before selecting a model for those roles. Do not switch to the unverified 27B solely because its parameter count is larger.
