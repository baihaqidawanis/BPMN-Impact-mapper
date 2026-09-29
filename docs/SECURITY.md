# Security and ingestion policy

## Controls implemented

- repository roots are made absolute, checked with `Lstat`, canonicalized, and required to be directories;
- a symlink root is rejected and symlink entries are skipped;
- every walked path must remain relative to the canonical root;
- default limits: 10,000 files, 1 MiB per file, and 64 MiB total indexed bytes;
- NUL, invalid UTF-8, and high control-byte ratios classify binary content;
- deterministic directory/file order and context cancellation are enforced;
- XML input is limited to 8 MiB, depth 128, and 100,000 elements; directives are rejected;
- evidence excerpts are limited to 240 Unicode code points;
- report paths are repository-relative and slash-normalized;
- files are held as data and never executed.

## Default exclusions

Directories: `.git`, `node_modules`, `vendor`, `dist`, `build`, cache directories, and coverage.

Secrets: `.env*`, common key/certificate/keystore extensions, and credential/private-key filename markers.

Generated/binary: source maps, common minified assets, common generated Go/C# suffixes, and content detected as binary.

Leakage artifacts: process mapping, traceability coverage, checkpoint trees, release-candidate documents, audit/implementation plans, generated impact reports, and ground-truth paths/files.

Caller-supplied `--ignore` globs extend, but do not weaken, defaults.

Old/new BPMN files and the selected mapping configuration are dynamically excluded from the repository retrieval corpus when located inside the analyzed root. Ground-truth filename/path patterns are excluded by ingestion defaults. More importantly, the `analyze` request has no ground-truth field or evaluation dependency; hidden ground truth is accepted only by the separate post-inference `evaluate` command.

Benchmark manifests contain local paths and are not shareable result artifacts. Their input hashes are verified before analysis, except the hidden-ground-truth hash/content, which is verified and parsed only after inference. Artifact writers canonicalize the output root, reject absolute/parent-relative artifact names, and keep predictions, evaluations, deterministic benchmark summaries, volatile timing, and reconciliation outputs separate.

## Threats and residual risks

The controls reduce path traversal, symlink escape, parser/size denial of service, accidental secret ingestion, data leakage, and analyzed-code execution. Residual risks remain:

- Windows junction/reparse-point behavior is not identical to Unix symlinks;
- a file can change between metadata inspection and reading;
- secrets in innocuously named text files can still be indexed;
- allowed XML/text can contain confidential business data;
- local HTTP exposure would permit local-path analysis, so embedding applications must add authentication and path allow-lists before serving the handler remotely.

Operate on a frozen, least-privilege copy. Review the allowed corpus before distributing reports.
