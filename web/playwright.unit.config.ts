import { defineConfig } from "@playwright/test";

// Pure report logic: no browser, backend, or embedding model required.
export default defineConfig({
  testDir: "./tests/e2e",
  testMatch: "report-explanation.spec.ts",
  reporter: "line",
});
