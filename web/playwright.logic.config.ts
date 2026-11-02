import { defineConfig } from "@playwright/test";

// Pure report helpers: no browser, API server, embedding runtime or network.
export default defineConfig({
  testDir: "./tests/e2e",
  outputDir: "./test-results/logic",
  testMatch: ["report-explanation.spec.ts", "analysis-usecase.spec.ts"],
  workers: 1,
  reporter: "line",
});
