# Structural BPMN scenario coverage

These are controlled mutation specifications for the read-only Simple-ERP BPMN
baselines. They are method inputs, not ground truth and not product changes.
Generate each `new` BPMN below ignored `outputs/pilots/generated/coverage/`;
use the original Simple-ERP BPMN as `old` in the mapper.

| Scenario | Baseline | Primary taxonomy | Intent |
| --- | --- | --- | --- |
| `p2p-add-task` | Procure-to-Pay | `ADD_TASK` | Insert a purchase-request validation activity between submission and review. |
| `p2p-add-gateway` | Procure-to-Pay | `ADD_GATEWAY` | Insert a budget decision with approve/reject branches before the existing approval gateway. |

The diff will also report affected sequence flows because the topology changes.
That is expected: one controlled business change can have multiple atomic graph
diff records. Generated BPMN has no diagram-interchange shape for new elements;
the mapper analyses process semantics, not diagram layout.

Example:

```powershell
go run ./cmd/bpmn-impact-mapper mutate `
  --spec experiments/simple-erp/coverage/p2p-add-task.json `
  --baseline C:\Simple-ERP\docs\bpmn\l2-procure-to-pay.bpmn `
  --output outputs\pilots\generated\coverage\p2p-add-task.bpmn
```
