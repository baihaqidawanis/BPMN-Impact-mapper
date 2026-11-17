# Offline agent verification — 12 November 2026

Scope: orchestration and export contracts. No lab inference, VPN connection, application server, browser, dependency download, or case-study source edits were used.

## Added regression coverage

`internal/planning/offline_pipeline_test.go` checks the complete scripted agent entry point against four persistent Planner failures:

- Missing file plans for an Analyst-supported target.
- Invented evidence identifiers.
- A file not verified by the Analyst.
- Whitespace-only acceptance checks.

Each failure must return an error without a COMPLETE session, accepted draft, or Reviewer call. Output repair is bounded to at most one attempt. Existing validation already satisfies these cases; no production protocol or model configuration change was necessary.

The export regression roundtrips the canonical JSON into Markdown and checks deterministic output, provisional status, review issues, current and required behavior, rationale, assumptions, implementation steps, proposed changes, acceptance checks, and source evidence references. The content is synthetic and is not a new Simple-ERP ground truth.

## Interpretation

These tests establish that invalid scripted proposals do not bypass the existing guards and that the renderer retains supplied technical detail. They cannot establish whether an actual model reads source correctly, chooses the correct files, or writes good technical steps. No Precision, Recall, MAP, MRR, or model-quality score is inferred from them.

## Next live comparison

1. Restart the backend and start a fresh agent session after the source-tool identity change. Reuse an intact saved analysis when its source snapshot still matches.
2. Keep the same inventory ADD_TASK scenario, repository snapshot, candidate list, model configuration, context size, and budget.
3. Save both agent evidence JSON and the change-plan Markdown. Record role/model, tool activity, retries, timing, and completion status.
4. Check whether the agent distinguishes a literal empty argument from a variable or computed expression and validates its meaning against the function declaration.
5. Compare selected files with the provisional reference separately from plan quality. Inspect rejected or NO_ACTION targets too. Do not treat COMPLETE as proof of correctness or Markdown order as a relevance ranking.

Verification commands: `go test ./internal/planning ./internal/llm ./internal/agentjobs ./internal/agenttools -count=1`, `go vet` on the same four packages, and `git diff --check`.
