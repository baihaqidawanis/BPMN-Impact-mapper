# Interface motion

18 November 2026. Requested by Haqi while preparing a Google Stitch layout reference.

## Applied

- GSAP 3.15.1: reading-pane transition when the selected process change, grouping or filter changes. Animation lasts 240ms and translates at most 6px. Cleanup restores inline styles and handles reduced-motion preference changes.
- Lenis 1.3.27: imported on deliberate change navigation when the reading pane is outside the useful viewport. The scroll lasts 380ms; its requestAnimationFrame loop stops within 700ms. Wheel, touch and keyboard input interrupt it. There is no global perpetual animation loop or wheel/touch smoothing.
- React Bits: the FadeContent interaction is the reference for an original, compact local implementation. This is not a vendored React Bits component or an installation of its component collection. Noise, blur, ScrollTrigger and decorative background effects are not included.

Implementation: `web/src/components/motion/FadeContent.tsx`, used by `ReportView.tsx`. Content stays visible without JavaScript. The wrapper does not key/remount its children. Canvas, agent inference, checkpoints and backend rankings are outside this change.

## References

- [GSAP](https://github.com/greensock/GSAP) and [context cleanup](https://gsap.com/docs/v3/GSAP/gsap.context()/)
- [Lenis configuration and programmatic scrolling](https://github.com/darkroomengineering/lenis)
- [React Bits FadeContent](https://github.com/DavidHDev/react-bits/tree/main/src/content/Animations/FadeContent)

## Manual acceptance pending

The subsequent owner-requested layout pass changes Overview to a centered, large system-sans introduction and one wide current-analysis panel. Analysis setup now uses vertically separated fieldsets with side labels and a before/after upload pair. The duplicate initial readiness area is hidden while setup is open. Existing inputs and agent views remain mounted; no inference/session behavior was changed. The original full-height sidebar is retained. Overview has a finite 650ms GSAP entrance on navigation, disabled for reduced motion. Source checks passed: TypeScript, UTF-8 scan, all 23 CSS files parsed and diff whitespace validation. A browser rendering and interaction pass has not been performed.

Select changes from a long report, interrupt scrolling with wheel/keyboard, switch filters and By file, and repeat with reduced motion enabled. Check phone-width navigation and Canvas zoom/pan. Static checks do not establish visual quality or browser interaction correctness. No inference or heavy frontend build is needed to review these source changes.
