# Workspace visual refresh

The workspace uses charcoal navigation, warm neutral surfaces and a restrained blue accent. The system font stack keeps SF Pro where available and falls back to the local OS font; no font download is required.

Analysis inputs come before sample workflows. File inputs, runtime status and method details have a clearer hierarchy. Reports use quieter badges and larger row spacing. Desktop inputs remain sticky; mobile navigation retains close, backdrop and Escape support. Reduced-motion preference is respected.

Presentation stays in component CSS modules and global design tokens. Analysis execution remains in the existing injected application use case; ranking, API contracts, saved analyses and model requests are unchanged.

Verification: eight Chrome tests passed covering overview/mobile navigation, library/history filters, five canvas cases and mapping-to-Planning restoration. Desktop screenshots were visually inspected. TypeScript and encoding checks passed. Real Qwen inference was not run during this presentation change.
