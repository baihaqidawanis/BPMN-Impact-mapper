# Reviewer contract repair

## Observed failure

Session `1ef2e2701e0f9832d602ad52ca871a51` paused in focused batch 7. The child checkpoint identified Reviewer output with invented source references during a proposal-only pass and a missing/invalid review criterion. It had used 18/18 turns. Other batches had context/review issues; 6/7 processed was not 6/7 approved.

No checkpoint, source repository or historical model output was edited for this repair.

## Changes

- The experimental explicit-JSON review policy is now v27. Source-backed file reviews retain explicit JSON mode and unchanged host validation. Contract-only passes now use constrained schema decoding, including an empty evidence index array and every required criterion. Host validation still rejects fabricated references, incomplete checks and bare approval.
- Contract-only instructions explicitly distinguish cross-file future agreements from current-source validation. Omitted detailed tests are not treated as proof that tests are missing. A contract PASS does not establish source correctness.
- Invalid-output feedback identifies the plan index and failing criterion, or explains that evidence indices must be empty in a source-free pass.
- If invalid model output exhausts a child's turn budget, that child finishes NEEDS_REVIEW rather than repeatedly offering a resumable PAUSED state. No approval or extra budget is granted.
- A focused pause exposes the actual failing child role and its validation feedback.
- New focused checkpoints distinguish processed, approved and needs-review batches. Historical checkpoints without outcome counts do not have approval counts inferred. Final consistency approval remains separate.

## Verification and limits

Mocked tests cover constrained contract-only requests, rejection of invented references despite decoding constraints, criterion-specific feedback, exhausted-budget resume behavior, failing-role propagation and rejected-batch counts. Frontend type/encoding checks cover the updated progress display. No real model inference or browser/server startup was performed.

These changes strengthen the request/response contract and diagnostics. They do not prove improved source selection or technical plan correctness. Context limits and slow inference may still occur. The model can still make semantic mistakes in structurally valid output.

## Next live trial

Restart the backend, reopen the same saved analysis and start a NEW session because model protocol and focused orchestration identities changed. Keep the same models/scenario/candidate count. In Model settings, verify Reviewer thinking is Off and Apply succeeds. Retain the old session as diagnostic evidence; do not resume it under the new protocol.
