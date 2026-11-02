# Workspace navigation and presentation

The interface uses React/Next.js, CSS Modules and inline SVG icons. Bootstrap,
an icon package, downloaded fonts and new dependencies were not added. Existing
SF/system font fallbacks and neutral/blue palette remain.

## Navigation

- Overview is the initial page. It shows actual saved report/diagram counts,
  the selected analysis, a saved-report sample and entry points to Analysis and
  Library. Reports have no timestamp in their summary contract, so the sample
  is not labelled as chronologically recent. No accuracy numbers are invented.
- Analysis retains the repository/BPMN inputs, mapping results, Canvas access,
  and Planning. The repository path continues to be carried privately to review.
- BPMN Canvas is a child item under Analysis. Drafts and undo state remain
  mounted across navigation; a new-analysis action explicitly clears inputs and
  asks before discarding an unapplied Canvas draft.
- Library, History and Guide remain available. Research has its own separated
  sidebar group and continues to respect the existing RND presentation flag.
  That flag is not authentication or API authorization.

The topbar contains a straight three-line menu button and page title. Desktop
navigation collapses to an icon rail and expands again. Mobile navigation opens
over the page and closes using its X button, backdrop, Escape or a navigation
selection. Escape/X return focus to the menu button. Sidebar icons share an
18px box and 1.5px stroke. The logo is a process-connection SVG, distinct from
the menu affordance.

View selection is recorded as a `view` query parameter; report selection remains
the existing `analysis` ID. Opening an old report URL without `view` still opens
Analysis. A fresh `/` opens Overview. `/?view=analysis` directly opens inputs.
No repository paths enter these URLs. Section navigation returns scroll to top.
This remains the existing client workspace, not a new multi-route backend.

## Result display

The report retains grouping by BPMN change or file, category filters, all raw
changes in downloads, and all evidence. Candidate rows and action badges are
more compact; completion and metric typography are quieter. Technical evidence,
formula explanations and context remain available through disclosure controls.
Retrieval, rankings, thresholds, scores and evaluation calculations are unchanged.

Existing saved analyses, private path restoration, Canvas exports, hierarchy
links, and Qwen checkpoints keep their existing persistence boundaries. Uploaded
files/unapplied Canvas drafts still require download or explicit library save
before page refresh. No model inference is triggered by Overview/navigation.

## Verification

Browser checks cover Overview, desktop collapse/expand, mobile X/Escape and
focus return, saved report restoration, new analysis, mapping evidence, Canvas
editing/undo/export, Library and hierarchy navigation. Screenshots are emitted
under `web/test-results/overview-*/` and `workspace-flow-*/`.

The first regression pass exposed two identical Canvas button names; the
in-workspace tab now says Canvas and the sidebar item says BPMN Canvas. Narrow
input panels now stack the demo selector and action so the selector has room.
No case-study files, dependencies, model configuration or core analysis methods
were changed in this work.

Final verification: 13 distinct browser cases passed across targeted runs,
including the corrected Canvas case. The final Overview visual run passed;
desktop/mobile screenshots were inspected. Typecheck, UTF-8 guard and diff check
pass. Next development indicator is hidden through documented devIndicators=false;
compile/runtime errors remain visible. Restart the frontend dev server for this
configuration update. This UI increment requires no backend restart.

## Collection follow-up

History now opens directly as a full-page report list. Search matches repository,
revision, method and analysis ID; a method selector combines with search. Refresh,
clear filters, visible/total count and empty/error states are available. IDs help
distinguish reports for the same repository. No fabricated timestamps/recency,
new ordering assumptions or modifications to saved report content were added.
Library adds repository/role/level selectors alongside existing free-text search.

## Final visual tone pass

[DONE] CSS-only readability/interaction polish: 14px body base with 1.5 line
height, white topbar, restrained hover/focus states, consistent metadata and
larger mobile navigation targets. Original SF/system font and palette preserved.
No function, schema or score change. Overview/analysis/mobile browser test passes
(8.5s), mobile screenshot inspected; TypeScript, encoding and diff checks pass.
