# Local BPMN library

Open **BPMN Library** in the navigation. Choose an uploaded UTF-8 BPMN file,
the currently selected baseline, or the applied changed BPMN. Enter repository
label, process name, optional role, level 0-3 and version label, then save.
These labels describe the diagram; they do not select or read a source repository.

Entries are persisted on the backend under `.data/bpmn-library/` (or the selected
`--data-dir`). Restarting the backend or refreshing the browser does not remove
them. Back up this directory if moving the application. It is local workspace
data and must not be committed or added to the retrieval corpus.

Search the saved list and choose **Use as baseline** or **Use as changed BPMN**.
Select a baseline before choosing the changed version. Then open BPMN Canvas or
run Mapping. The XML is not normalized, renamed internally, translated or edited
on save. Downloads contain that saved XML. Metadata/content combinations have
SHA-256 identifiers; identical saves reuse an entry, while a new version label
or changed XML creates another entry. Existing versions remain available.

The current repository path is still configured in Workspace. Library entries
do not contain absolute source paths or automatically run mapping/Qwen. An
unapplied Canvas draft must first be applied or downloaded; saving the current
changed BPMN saves the applied version.

## Modelling checks

Import parses the BPMN with the existing bounded XML parser. Invalid XML is
rejected. Valid BPMN with structural, capability or project convention findings
can still be saved, with findings available in the entry details. The project
conventions include one start/end per scope and no direct gateway-to-gateway/end
edge. They are not universal BPMN rules and do not prove alignment with ERP.
Missing diagram layout is reported: textual BPMN can be mapped, but Canvas needs
diagram shapes and coordinates exported by an editor.

The library does not infer parent/child relationships from similar names or
levels. Explicit links are now supported as described below. Bulk process
bundles, deletion of diagrams and automatic propagation are not implemented.

## Boundaries

- `internal/bpmnlib`: metadata/entry contracts, store port, parsing and checks.
- `internal/filestore/bpmn_library.go`: atomic local file persistence; filenames
  use validated content IDs, not user metadata. Symlink entries are rejected.
- `internal/api/bpmn_library.go`: bounded JSON HTTP upload, list and detail.
- `web/src/lib/bpmn-library.ts`: HTTP client and XML file reconstruction.
- `web/src/components/BpmnLibrary.tsx`: upload, filter, checks and explicit reuse.

API: `GET/POST /api/v1/bpmn-library`, `GET /api/v1/bpmn-library/{id}`.
No new dependencies, database, VPN or LLM required.

## Open detail processes across levels

Save the parent and detail diagrams first. Open **View process and details** on
the parent. This viewer is read-only and supports pan, zoom and fit. Click an
activity or select it from **Selected activity**, choose a saved **Detail diagram**
and **Save detail link**. Both entries must share the exact repository label;
the target must have a deeper level. Cross-role detail references are allowed.
Links use explicit diagram version IDs; saving a new version does not silently
retarget an old relationship. Deeper levels must strictly increase, preventing
cycles. The server checks that the parent element is an actual activity.

**Open process detail** follows a saved link. The breadcrumb returns to the
parent; **Back to library** returns to the catalog. An unlinked activity has an
explicit empty state. Native embedded-subprocess drilldown controls are hidden
in this viewer so they cannot replace file navigation with an empty plane. The
BPMN plus marker and XML are preserved. **Use this diagram as baseline** explicitly
loads it into Workspace, where Canvas edits a separate copy using the existing
workflow. A missing layout produces a display error without modifying the XML.

Links are persisted separately under `bpmn-library/links/`. API:
`GET/PUT /api/v1/bpmn-library/{id}/links`. PUT takes `element_id` and `target_id`;
an empty target removes that activity link. Updating links never rewrites a BPMN,
changes ranking scores or initiates Qwen. Breadcrumb position resets on page
refresh; saved links remain and can be reopened from Library.

Bulk import is a later convenience for populating this library, not a requirement
for analysis. Production mapping remains one before/after pair. Multi-process
analysis would need explicit version pairing and separate evaluation; concatenating
XML files or rendering every diagram together is not implemented.

## Find saved diagrams

Free-text search covers repository/process/role/level/version. Repository, role
and level selectors combine with that search; Clear filters resets all four.
The visible/total count includes every saved version. Filters affect the catalog
only: no XML, links, mapping scores or saved entries are changed. Empty results
are distinguished from an empty library.
