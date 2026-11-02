# File-level fusion experiment

The report/evaluation unit is a file, while previous production RRF fused
file/function candidates and diversified files only after fusion. Two source
retrievers could agree on a file but choose different functions, losing the
agreement vote. Multiple functions from one file also consumed ranking slots
ahead of other files before display diversification.

The candidate variant deduplicates each retrieval list by file before assigning
RRF positions and merges votes by file. Each branch uses its first-ranked
candidate from that file; duplicate function count adds no vote. Original
evidence locations survive, but the intermediate function name is cleared:
agreement on a file is not proof of agreement on a function. The existing
conservative source-backed function resolver can still identify a function.
Sequence-flow candidates retain their existing artifact eligibility policy.

There are no file-name/role/framework boosts, test penalties, new weights,
model changes, or changes to `k=60`. Method metadata includes `ranking_unit`.
The optional `fusion.Strategy.FileLevelFusion` flag isolates the variant;
the candidate production Hybrid is the only wiring changed for the experiment.
BM25, Semantic and plain Fusion remain unchanged comparison methods.

The existing benchmark command requires all four comparison methods. A
Hybrid-only before/after probe therefore uses ordinary CLI analyze/evaluate,
not a weakened benchmark validator. For each of the five scenarios in
`staff-consolidated-2026-10-29-manifest.json`, it runs analyze twice with the
same repo/name/revision/BPMNs and top ten, compares full prediction hashes,
and checks the fingerprint against the completed consolidated baseline.
Only afterward is the unchanged reference hash checked and the CLI evaluate
command called. Outputs are in `outputs/staff-file-fusion-probe/<scenario>/run-2/`;
run-1 is retained for reproducibility checking. No four-method benchmark ID is
claimed for this narrower probe.

Generic regression tests check invariance to one versus twenty functions in
the same file, no mutation of input candidates, and no invented function
agreement across retrievers. The full Go suite and vet passed before the
real-model probe. Existing source-derived labels stay unchanged and exposed;
results are development evidence, not held-out accuracy.

Final verification: the complete synthetic four-method benchmark
`e623fcc6194446583075003112afec83` in
`outputs/benchmark-impact-file-fusion/` reproduced all twelve method/scenario
results over two repetitions (24 analyses). Hybrid expected-file ranks remain
3/2 on confirmation-task form/test and 1/6 on gateway code/test; the label
negative control remains empty. Full `go test -p 1 ./...`, `go vet ./...`,
changed-file gofmt and `git diff --check` passed. The file-level variant is
therefore retained as production Hybrid; plain Fusion is unchanged. No browser
test was needed for this backend-only ranking change. Simple-ERP/POC TA retain
their 232/6 pre-existing dirty entries. No commit or push was performed.

## Metric ceiling

The four positive scenarios contain ten reference file/change pairs. With a
fixed denominator of ten per scenario, macro Precision@10 cannot exceed
10/(4*10)=0.25 on this reference; baseline Hybrid already reaches that ceiling.
Precision@5 has a ceiling of 10/(4*5)=0.5. Ranking improvements should therefore
be judged by Precision@5, Recall@5 and MRR while preserving Recall@10. A
separate experiment would be needed to measure precision among an adaptively
returned set; truncating output alone does not improve fixed-denominator P@10.

## Staff outcome

All five scenario pairs completed twice with identical prediction hashes and
matching baseline input fingerprints. The unchanged label-control reference
still returns no file candidates. The comparison is in
`outputs/staff-file-fusion-probe/comparison.csv`; per-run canonical reports and
Go evaluation reports are retained in the scenario directories.

| Primary positive-case metric | Previous Hybrid | File-level Hybrid |
| --- | ---: | ---: |
| Precision@5 | 0.4000 | 0.4500 |
| Recall@5 | 0.7917 | 0.8750 |
| Recall@10 | 1.0000 | 1.0000 |
| MRR | 0.7083 | 0.8333 |

| Reference file | Previous rank | New rank |
| --- | ---: | ---: |
| Inventory condition backend store | 4 | 2 |
| PR price backend store | 8 | 6 |
| Inventory reason integration test | 2 | 1 |
| PR confirmation form | 6 | 5 |

Other reference-file positions are unchanged. The price backend is still
outside five, so the method does not reach the reference's P@5 ceiling (0.5).
P@10 remains at its 0.25 ceiling. No file-specific boost, test penalty or
reference-label revision was used to obtain these results. These exposed
development cases support the ranking correction but do not establish
generalization. Required actions remain VERIFY, not implementation proof.
