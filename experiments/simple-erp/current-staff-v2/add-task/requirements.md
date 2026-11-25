# add-task

Require non-empty reason before adjustment submit.

Submit currently only checks DRAFT/version. Reason field exists but UI submits empty reason. Master active checks during create do not ensure they remain active at submit. Operations integration tests exercise existing submit/guards. No schema or new endpoint required.

Reference scope: one selected primary change; other changes not labelled or counted as negatives.
Reference status: provisional source-derived diagnostic; prepared before inference, not independent ground truth.

## Reading the generated plan

These checks were added after inspecting an earlier run. Use them for a manual review of the proposal. Keep the original three-file reference for file precision and recall.

| Check | What a developer should be able to implement from the plan |
| --- | --- |
| Submit in the UI | The panel already has a reason input. Make its purpose clear for submit, validate it, and send its value in the submit request. Currently only reject, cancel and reverse forward the entered reason. |
| Submit on the server | Enforce the requirement on the inventory submit path before changing status. A direct API caller must receive an error for an empty reason and the adjustment must remain in DRAFT. |
| Existing contract | The HTTP handler reads `Reason`, the service forwards it, and the store already writes it. A proposal for a new endpoint, parameter or table-wide constraint needs a separate requirement. |
| Regression tests | Cover empty reason on submit and a valid reason reaching SUBMITTED. Update existing successful inventory submit calls that pass an empty string. Purchasing and sales tests exercise different workflows. |

Record each check as met, partial, contradicted or missing, with a quote from the plan. A file appearing in the output earns no credit by itself on these checks.
