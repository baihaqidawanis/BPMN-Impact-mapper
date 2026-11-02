# Workspace readability pass

The existing navigation and mapping workflow are preserved. This pass changes
presentation in the workspace, analysis form, report and evaluation dashboard.

- Configuration uses a narrower desktop rail; reports gain horizontal space.
- Small 7-11px report labels and evidence text are enlarged, with clearer
  separation between titles, file paths, metadata and technical details.
- Surface shadows and corner radii are reduced. Neutral white/gray, dark text
  and blue actions retain the existing palette. The font stack uses native
  system UI (SF on Apple systems, Windows system fallback); no font download.
- Sample selection/button share a compact row. Upload controls stack their
  label and chooser, selected filenames can wrap, and lone candidate-limit
  controls use the available form width. Repository/revision inputs align.
- Revision explicitly states that it labels a run and does not select a Git
  commit. Hybrid copy describes ranking inputs rather than claiming that all
  explanatory source clues contribute a score.
- File paths wrap rather than being silently truncated. Expanded evidence uses
  the full available panel width. Export links wrap on narrow screens.
- Keyboard focus is visible; reduced-motion preference limits animations.
- Workspace titles use direct task names instead of promotional wording.
  Supporting flow text refers to the primary change, including gateways.

Verification: reviewed desktop/mobile fixture-report screenshots. Existing
report tests pass 3/3 (filters, file grouping, immutable report data and mobile
overflow). Isolated canvas draft/undo/redo/XML export/mapping passes 1/1.
Isolated Research reconciliation/evaluation passes 1/1. TypeScript and
`git diff --check` pass. Backend ranking, formulas and case-study source remain
unchanged; no backend test rerun was needed for styling/copy.

The full combined run hit browser `ERR_INSUFFICIENT_RESOURCES` while loading
canvas chunks, and the local embedding capability check reported Windows
OSError 1455 (paging file too small). It was stopped; targeted checks without
full-model inference passed afterward. This is not a full-suite pass or proof
that live semantic analysis will run under the same machine load. For the
presentation, check runtime readiness separately and keep saved reports ready.
No unrelated application was terminated and no system memory setting changed.
