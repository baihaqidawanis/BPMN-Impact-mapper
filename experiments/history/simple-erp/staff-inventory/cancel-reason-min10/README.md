# Staff cancellation reason: prepared, not scored

Requirement: cancellation needs at least ten Unicode code points after
trimming outer whitespace. Existing state/version/ownership rules remain;
other actions do not acquire this length requirement.

- `baseline.bpmn`: exact copy of the validated Staff cancellation baseline.
- `changed.bpmn`: one existing user-task label states the stricter rule.
- `mutation.json`: reproducible attribute mutation with baseline hash.
- `frozen-inputs.json`: BPMN/reference/source hashes and preparation status.
- Reference: `docs/ground-truth/staff-inventory-cancel-reason-min10-reference.json`
  and its rationale Markdown, outside the analyzed repository.

The mapper parser/diff confirmed exactly one `MODIFY_TASK_LABEL` on
`role_staff_l3_adj_cancel_input`; the task remains a `userTask`. BPMN-DI is
identical to baseline, so nodes and lines remain present. New label wrapping
has not been visually checked in bpmn.io. No mapper ranking or accuracy result
exists for this case yet. Do not use it as a presentation of measured accuracy.

For a later web experiment, repository path is the local Simple-ERP root,
repository name `Simple-ERP`, old/new files are this pair, method is production
Hybrid and top candidates ten. Revision can be empty for current local files;
the hashes identify the paired BPMNs and inspected source files, not the whole
dirty repository. Record the mapper input fingerprint at inference and verify
these hashes before using the reference. Export the JSON before comparing it to the
reference. Never upload the reference as mapping configuration or method input.

The source-only reference lists three existing targets and is provisional.
This operation was not used for previous ranking experiments, but its source
repository and related inventory files were already inspected: it is a fresh
development input, not independent blind or cross-repository validation.
