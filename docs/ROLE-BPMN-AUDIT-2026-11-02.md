# Role BPMN convention recheck — 2026-11-02

Read-only inspection of canonical `docs/bpm-haqi/roles/` in the case-study
repository. Complete relative filenames, SHA-256 hashes and flow IDs are recorded
in [the audit data](ROLE-BPMN-AUDIT-2026-11-02.json).

| Role | Files | Files with findings | Direct gateway adjacency findings |
|---|---:|---:|---:|
| Staff | 32 | 24 | 44 |
| Supervisor | 31 | 23 | 48 |
| Super Admin | 49 | 40 | 66 |
| Total | 112 | 87 | 158 |

No findings for multiple/missing start or end events in the inspected populated
process scopes, invalid XML or missing flow endpoints. The 158 findings comprise
157 gateway-to-gateway edges and one gateway-to-end edge (Staff AP Level 2).
The first report incorrectly grouped that one end edge with gateway pairs;
the checker now records target types explicitly. Gateway pairs commonly connect a
decision gateway leading to a common result merge. For example, Staff inventory
adjustment submission has `role_staff_l3_adj_submit_valid` leading directly to
`role_staff_l3_adj_submit_result_merge` via `role_staff_l3_adj_submit_Flow_6`.

The older role normalization report checked single start/end and an activity
after a join. It did not enforce a prohibition on every direct gateway pair.
Its historical pass therefore does not establish compliance with the stricter
rule requested now. No source or BPMN was changed during this audit.

These are research modelling conventions. A direct gateway pair can be valid
BPMN, and an arbitrary task inserted merely for visual compliance can misstate
the software. Review the business meaning of each branch before replacing or
inserting activities. This audit does not prove runtime equivalence, reachability,
deadlock freedom, complete diagram layout or 1:1 code alignment. It covers role
diagrams, not every legacy BPMN or mutation fixture in the repository.

Reproduce from the Mapper directory:

```powershell
python scripts/audit_bpmn_conventions.py C:\Simple-ERP\docs\bpm-haqi\roles
```

The script only reads its input directory. It prints JSON to stdout. The new
library also surfaces the strict direct gateway pair/end convention when saving
a diagram. It does not automatically rewrite the diagram.

## Subsequent resolution

The findings above describe the pre-cleanup snapshot. The user-approved
[cleanup](ROLE-BPMN-GATEWAY-CLEANUP-2026-11-02.md) has now been applied. A fresh
canonical audit of 112 role diagrams has zero strict convention findings.
