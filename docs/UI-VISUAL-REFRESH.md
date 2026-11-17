# Workspace visual refresh

The workspace uses charcoal navigation, warm neutral surfaces and a restrained blue accent. The system font stack keeps SF Pro where available and falls back to the local OS font; no font download is required.

Analysis inputs come before sample workflows. File inputs, runtime status and method details have a clearer hierarchy. Reports use quieter badges and larger row spacing. Desktop inputs remain sticky; mobile navigation retains close, backdrop and Escape support. Reduced-motion preference is respected.

Presentation stays in component CSS modules and global design tokens. Analysis execution remains in the existing injected application use case; ranking, API contracts, saved analyses and model requests are unchanged.

Verification: eight Chrome tests passed covering overview/mobile navigation, library/history filters, five canvas cases and mapping-to-Planning restoration. Desktop screenshots were visually inspected. TypeScript and encoding checks passed. Real Qwen inference was not run during this presentation change.

## Softer surfaces following Form AM / RTWT (2026-11-03)

Read the reference apps' apple-kalbe-theme.css and rtwt-portal.css without editing or running their applications. Adopted the visual principles of rounded cards, low-opacity borders, subtle elevation and quieter neutral backgrounds. Mapper retains its blue accent and system font stack.

Shared CSS tokens now define 16px cards, 10px controls, inset radii, light shadows and eased interactions across Analysis, reports, Canvas, Library, History and agent panels. No extra component framework or font dependency. Mobile avoids animating content margin at the desktop breakpoint to prevent transient horizontal overflow. Reduced-motion settings remain respected.

Overall follow-up: raised tiny component text to an 11px metadata floor, made Guide content 13px and open on entry, shortened Overview copy and collapsed AI source-access explanations. Twelve browser cases pass across the workspace sections. Desktop Overview/Research/Planning screenshots inspected; no inference or scoring change.
