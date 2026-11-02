// Build-time presentation flag, not authorization for API or static-file access.
export const RND_ENABLED = process.env.NEXT_PUBLIC_ENABLE_RND !== "false";
