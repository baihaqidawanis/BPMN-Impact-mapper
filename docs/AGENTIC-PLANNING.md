# Evidence-grounded agentic planning

## Why no LangGraph yet

The first planning workflow is a bounded, acyclic state machine implemented in
Go. It has three explicit roles and no dynamic tool loop:

```text
canonical impact report
  -> bounded evidence bundle
  -> candidate verifier
  -> planner per software layer
  -> cross-layer reviewer
  -> validated planning JSON
  -> Markdown rendered from the same object
```

Adding a graph framework now would duplicate orchestration already expressible
with typed Go interfaces. Reconsider LangGraph or an equivalent only if the
workflow later needs durable checkpoints, pause/resume, dynamic branches,
human interrupts, or repeated tool calls that cannot be represented clearly by
the current state machine.

## Role boundaries

- **Deterministic mapper (not an agent):** parses BPMN, computes graph changes,
  indexes source, retrieves candidates, aggregates evidence, and abstains.
- **Candidate verifier:** labels every supplied candidate `RELEVANT`, `WEAK`, or
  `REJECT` and cites evidence IDs. It cannot add candidates.
- **Layer planner:** receives only accepted candidates for one indexed software
  layer and proposes file-level steps. It cannot write source code.
- **Cross-layer reviewer:** checks the complete draft for contradictions,
  omissions, and unsupported claims. It may revise only existing accepted
  targets.
- **Deterministic validator:** rejects invented BPMN identities, files, symbols,
  modules, layers, evidence IDs, invalid actions, duplicate targets, and missing
  plans for `RELEVANT` decisions.

Logical roles do not require different models or concurrent processes. One
provider can execute all roles with different prompts. This keeps the first
experiment able to compare single-model role decomposition without introducing
an orchestration framework as a confounder.

## Implemented pre-provider contracts

`internal/planning` now provides:

- bounded evidence packaging with configurable candidate/evidence/excerpt caps;
- deterministic evidence IDs;
- a deterministic context ID that changes when the bounded payload changes;
- provider ports for verification, per-layer planning, and review;
- an explicit Go orchestrator with cancellation and role-specific timeouts;
- hallucination and completeness validation;
- deterministic plan identity bound to provider, model, and prompt version;
- canonical JSON and Markdown rendering;
- an offline path through `llm.NoLLMProvider`.

No network call, API key, prompt, model SDK, or repository source execution is
part of this increment.

`context_id` is the first 32 hexadecimal characters of SHA-256 over the
canonical JSON bundle with `context_id` cleared. The validator recomputes it
before any enabled provider call. It identifies the exact bounded input but is
not an authentication token or digital signature.

The saved-report API exposes `planning-context.json`, and the report UI links to
it directly. This lets a user inspect the exact bounded payload before any model
provider is enabled.

The planning execution boundary is also active before provider integration.
`POST /api/v1/analyses/{analysis-id}/plans` runs the injected orchestrator and
persists canonical plan JSON/Markdown through `planstore.Store`. The default
server injects `NoLLMProvider`, so the stored record is explicitly `DISABLED`;
it proves identity, persistence, and failure-safe UI behavior without fabricating
model output. Repeating the same request is idempotent.

## Provider requirements for the next increment

A concrete provider must be injected at the composition root and configured
only through environment variables. It must receive the bounded evidence bundle,
return strict JSON, honor cancellation/timeouts, and never receive the entire
repository. Provider output is untrusted until the deterministic validator
accepts it. Failure must preserve the canonical impact report and leave the
deterministic workflow usable.

The core makes one call per role and does not retry automatically. A future HTTP
adapter may retry only transient transport failures, HTTP 429, and HTTP 5xx with
a small fixed cap. It must not retry schema, grounding, or validation failures,
and it must record the attempt count. This avoids silently multiplying model
calls or accepting a different answer after a deterministic contract failure.

Every enabled provider must declare a non-empty provider name, model identity,
and prompt version. Contract tests cover identity enforcement, timeout
propagation, deterministic context identity, invented-file rejection,
invented-evidence rejection, and required-plan completeness.

The initial implementation should use an OpenAI-compatible HTTP adapter so the
same boundary can serve Groq or a local compatible endpoint. Provider/model
identity and prompt version must be recorded in every change plan.
