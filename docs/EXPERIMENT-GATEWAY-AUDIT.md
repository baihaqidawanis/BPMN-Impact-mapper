# Experimental gateway adjacency audit

Checked 2026-11-04. Scope: experimental BPMN files excluding filenames containing before.

19 direct gateway-to-gateway flows in 12 files. The active current-staff-v2 pack has none. Earlier cardinality checks did not test this adjacency rule across historical packs.

| File | Flow | Source | Target |
|---|---|---|---|
| `experiments/history/simple-erp/current-staff/add-and/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-and/after.bpmn` | `InvalidMasters` | `MasterDecision` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-task/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-xor/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-xor/after.bpmn` | `ReasonMissing` | `ReasonDecision` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/combined/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/combined/after.bpmn` | `InvalidMasters` | `MasterDecision` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/combined/after.bpmn` | `ReasonProvided` | `ReasonDecision` | `MasterChecksSplit` |
| `experiments/history/simple-erp/current-staff/combined/after.bpmn` | `ReasonMissing` | `ReasonDecision` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/unchanged-control/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/p2p-transfer/checks/after.bpmn` | `role_staff_l3_pr_submit_Flow_6` | `role_staff_l3_pr_submit_valid` | `role_staff_l3_pr_submit_result_merge` |
| `experiments/history/simple-erp/p2p-transfer/checks/after.bpmn` | `SubmitChecksFailed` | `SubmitEligibility` | `role_staff_l3_pr_submit_result_merge` |
| `experiments/history/simple-erp/p2p-transfer/unchanged-control/after.bpmn` | `role_staff_l3_pr_submit_Flow_6` | `role_staff_l3_pr_submit_valid` | `role_staff_l3_pr_submit_result_merge` |
| `experiments/history/simple-erp/selection-challenges/change-condition/after.bpmn` | `role_staff_l3_pr_copy_Flow_6` | `role_staff_l3_pr_copy_valid` | `role_staff_l3_pr_copy_result_merge` |
| `experiments/history/simple-erp/selection-challenges/remove-check/after.bpmn` | `role_staff_l3_pr_copy_Flow_6` | `role_staff_l3_pr_copy_valid` | `role_staff_l3_pr_copy_result_merge` |
| `experiments/history/simple-erp/selection-challenges/unchanged-control/after.bpmn` | `role_staff_l3_pr_copy_Flow_6` | `role_staff_l3_pr_copy_valid` | `role_staff_l3_pr_copy_result_merge` |
| `experiments/history/simple-erp/staff-single-end/after.bpmn` | `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `ResponseMerge` |
| `experiments/history/simple-erp/staff-single-end/after.bpmn` | `ReasonPresent` | `ReasonDecision` | `MasterChecksSplit` |
| `experiments/history/simple-erp/structural-staff/add-or/after.bpmn` | `ZeroDeltaBypass` | `ResearchSplit` | `ResearchJoin` |

## Interpretation

Gateway adjacency is a project modeling convention; inserting a task creates a new activity and must be justified by the requirement. Do not silently rewrite frozen experiment pairs or references.

- Plain XOR result merges: consider removing the redundant merge and routing alternatives to the existing response activity in a newly versioned pair.
- Reason decision to parallel split: explicitly define preparation of input references, as in current-staff-v2 combined.
- ZeroDeltaBypass in add-or: an intentional bypass from split to join. A replacement activity needs a defined no-change behavior; do not invent validation just for visual conformity.
- Unchanged controls must remain byte-identical before/after; normalize both together in a new pack.

Use experiments/simple-erp/current-staff-v2 for the next lab run. Historical metrics remain tied to their original inputs.

## Applied correction

Nine positive-scenario after files were updated at user request. Plain alternative-result XOR merges were removed; preparation before parallel checks and explicit zero-delta result handling were added where necessary. Before files and three historical unchanged-control pairs remain untouched. These three controls retain their historical adjacency; all other after files now have zero gateway-to-gateway edges.

18 before/after files imported and rendered without warnings or missing elements. Manifest/snapshot after hashes were updated where present; old evaluation scores were not rewritten and must not be reused for changed inputs. Added/removed notation nodes can change the diff against an old before. Existing references do not constitute complete labels for these new notation changes.

- `experiments/history/simple-erp/current-staff/add-and/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_adj_submit_result_merge
- `experiments/history/simple-erp/current-staff/add-task/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_adj_submit_result_merge
- `experiments/history/simple-erp/current-staff/add-xor/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_adj_submit_result_merge
- `experiments/history/simple-erp/current-staff/combined/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_adj_submit_result_merge; Expand implicit handling: PrepareMasterCheckInputs
- `experiments/history/simple-erp/p2p-transfer/checks/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_pr_submit_result_merge
- `experiments/history/simple-erp/selection-challenges/change-condition/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_pr_copy_result_merge
- `experiments/history/simple-erp/selection-challenges/remove-check/after.bpmn`: Remove redundant alternative-result merge role_staff_l3_pr_copy_result_merge
- `experiments/history/simple-erp/staff-single-end/after.bpmn`: Remove redundant alternative-result merge ResponseMerge; Expand implicit handling: PrepareMasterCheckInputs
- `experiments/history/simple-erp/structural-staff/add-or/after.bpmn`: Expand implicit handling: UseZeroDeltaResult
