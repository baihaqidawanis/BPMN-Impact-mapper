import path from "node:path";
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "line",
  use: {
    baseURL: "http://127.0.0.1:13000",
    channel: "chrome",
    headless: true,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: "go run ./cmd/bpmn-impact-mapper-server --addr 127.0.0.1:18081 --data-dir .data/playwright",
      cwd: path.resolve(__dirname, ".."),
      url: "http://127.0.0.1:18081/api/v1/health",
      timeout: 120_000,
      reuseExistingServer: false,
    },
    {
      command: "npm run dev -- -p 13000",
      cwd: __dirname,
      env: { BPMN_MAPPER_BACKEND_URL: "http://127.0.0.1:18081" },
      url: "http://127.0.0.1:13000",
      timeout: 120_000,
      reuseExistingServer: false,
    },
  ],
});
