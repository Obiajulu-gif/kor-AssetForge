import { test, expect } from "@playwright/test";

test.describe("Smoke Test", () => {
  test("sanity check environment and base URL loading", async ({ page }) => {
    await page.goto("/");
    expect(page.url()).toBeTruthy();
  });
});
