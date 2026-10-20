import { defineConfig, devices } from "@playwright/test";

const PORT = process.env.PORT ?? "3000";
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://127.0.0.1:${PORT}`;
const IS_CI = !!process.env.CI;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: IS_CI,
  retries: IS_CI ? 2 : 0,
  workers: IS_CI ? 1 : undefined,
  timeout: IS_CI ? 60_000 : 30_000,
  expect: {
    timeout: IS_CI ? 10_000 : 5_000,
  },
  reporter: IS_CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL,
    trace: "on-first-retry",
    actionTimeout: IS_CI ? 15_000 : 10_000,
    navigationTimeout: IS_CI ? 30_000 : 15_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: process.env.PLAYWRIGHT_BASE_URL
    ? undefined
    : {
        command: "npm run start",
        url: baseURL,
        reuseExistingServer: !IS_CI,
        timeout: IS_CI ? 240_000 : 120_000,
      },
});
