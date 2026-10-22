# Source-relation probe (experimental)

`probe-relations` is an offline ablation diagnostic, **not** a fifth production
mapping method. It reads an existing canonical report and re-indexes a local
repository without executing its code. It never modifies the canonical JSON,
scores, required actions, or source repository.

```powershell
go run ./cmd/bpmn-impact-mapper probe-relations --repo C:\path\to\repository --report .data\analysis\impact-report.json --top-seeds 5
```

Output is JSON on stdout. `findings` contain repository-relative evidence paths;
`proposals` show an alternative **file order**, not new confidence values. The
saved report and current repository snapshot cannot be proven equal by this
command, so its warning must be retained when interpreting results. Re-run the
canonical analysis on a frozen snapshot before a controlled experiment.

The deliberately narrow rule is:

1. Take at most five displayed candidates per independent BPMN change.
2. Seed only a frontend HTTP call whose exact source line is cited by BM25 or
   label-word evidence in that candidate.
3. Follow a uniquely resolved literal method+path route to the indexed Go
   handler. The source index already abstains on duplicate registrations.
4. Follow at most one static method/interface call from that handler. No
   arbitrary multi-hop graph traversal, invented file, or inferred action.
5. In the proposal, move only files already present in the canonical result,
   directly after their cited seed in path order. Keep all original scores in
   the untouched report. Multiple possible links remain visible as findings;
   this rule is not a causal proof of a required edit.

The current Simple-ERP PDF-task development case yields the path
`purchase-request-panel.tsx:31` → `handler.go:22` (`Handler.create`) →
`handler.go:96` (`Service.CreateRequest`). The probe proposes the handler file
at rank 3 instead of 14 and service at rank 4 instead of 15. That puts the
four provisional core files in the proposed Top-5, but the reference was
written after seeing mapper output. It is **not independent accuracy evidence**.

Before production adoption, freeze multiple development and hidden scenarios,
compare the same file-level Top-K with/without the rule, inspect false route
links and missing/ambiguous edges, and report Recall@k, Precision@k, MRR,
latency, and per-change-type behavior. An empirical gain on one case is
insufficient. The probe is limited to literal JS/TS client calls and the native
Go relation extractor; other languages/frameworks correctly produce no path.
