# BPMN Impact Mapper UI direction

Owner direction: preserve the current charcoal sidebar, blue actions and light workspace. Improve the whole interface without changing analysis or agent behavior. Haqi will judge the visual result directly.

Design read: a working tool for comparing processes, inspecting source evidence and reviewing proposed code changes. Content and provenance take priority over decoration. Current direction: ENERGY 2 / RHYTHM 3 / MOTION 2.

- Charcoal navigation identifies the workspace. Blue identifies actions and selection, never confidence or approval.
- Keep system typography for familiar, readable forms. Monospace is reserved for paths and source excerpts. Body information uses 13-14px; metadata uses 12px.
- Large, tightly spaced system-sans headings establish hierarchy. The earlier serif direction was rejected as too rigid. No downloaded font is required.
- Setup and review occupy separate stages of the page. Expanded setup uses a full-width three-part form; completed analysis folds it into a summary with an explicit edit action. Results and agent plans receive the full content width. Inputs remain mounted so collapsing does not discard file/state selections.
- Use flat rows inside a single change group. Cards distinguish independent process changes or canvases, not every sentence, score or evidence item.
- The report is one reading surface with ruled change sections, not separate cards for every result. The plan uses label/content columns like a technical review document. Overview prioritizes the active analysis and saved reports over decorative metrics.
- Use 12px field radii, rounded action buttons and 24px major surfaces. Rounded surfaces identify substantial work areas; do not wrap every label or paragraph in a card.
- Headers identify each screen; file paths, aligned ranks and readable excerpts are the recurring product motif. Genuine numerical results remain unchanged.
- Keep before and after Canvas in separate full-width panels. Boundaries separate the diagrams without reducing their editing area.
- Preserve all existing navigation and handlers. Actions use visible button boundaries, keyboard focus and adequate touch targets. New assets and dependencies need an explicit purpose; the owner approved GSAP, Lenis and React Bits exploration on 18 November.
- Desktop, intermediate widths and mobile stack deliberately. Tables and excerpts scroll within their own containers. Reduced motion disables nonessential transitions.

Verification: source review, TypeScript and encoding checks. Haqi requested direct source changes and will perform visual acceptance; do not claim browser visual approval or a completed Anti Slop visual gate without that review.

Reference update, 18 November: inspected root image.png comparing antislop with and without direction. Transfer its intentional composition and hierarchy to a technical workspace, not its illustrated marketing hero. Overview uses a large, quiet opening beside an actual current-analysis panel, followed by saved reports with a supporting library column. Charcoal navigation is inset from the canvas edge; the empty Mapping state separates explanation from live input readiness. No new decorative assets are required. These decisions preserve ENERGY 1 / RHYTHM 2 / MOTION 1.

Owner correction, 18 November: restore the earlier full-height, flush-left charcoal sidebar; the floating/inset navigation was rejected. Keep collapsed and mobile offsets consistent with this version. Other UI revisions still need owner acceptance; do not interpret passing static checks as a positive design review.

## Change Explorer composition

Mapping results are now an inspector rather than a long stream of repeated change cards. The report header and truthful counts establish context; a narrow, sticky process-change navigator sits directly beside a separate source-evidence reading surface. Selecting a process change changes the visible context without changing ranking or exports. A Show all changes control preserves the full-report reading mode, and By file retains the existing alternative. Tablet navigation becomes a two-column index above the reading surface, then one column on phones. The numbered process context, file paths and code excerpts are the interface's identity; the chosen sidebar is retained rather than redesigned again.

## Motion, 18 November

Owner-requested structural redesign supersedes the earlier serif and small-radius direction: large centered introduction, one wide current-work surface and quieter supporting lists. Analysis setup uses spacious rows with section labels at left and controls at right. Before/after files share one comparison row. The duplicate illustrated setup card becomes a compact readiness strip. Results remain a denser inspector for reading source. Keep the full-height charcoal sidebar, blue accent and native Canvas controls. No fabricated preview or decorative asset. Visual acceptance remains pending.

Owner-approved library exploration raises MOTION to 2 only for deliberate result navigation. GSAP briefly introduces the changed reading surface (6px, 240ms). The local FadeContent component interprets the React Bits pattern with original, reduced implementation rather than importing the component collection. Lenis is loaded only when a clicked change needs to bring its result into view; its animation loop is bounded to 700ms and user input interrupts it. Wheel/touch smoothing stays disabled, and BPMN Canvas is outside this behavior. Reduced-motion users receive immediate navigation and no fade. Agent polling, numbers and progress are not animated or remounted. This is motion refinement, not a replacement for the forthcoming Stitch layout direction. Visual acceptance remains pending.
