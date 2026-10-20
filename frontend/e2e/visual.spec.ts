import { test, expect } from "@playwright/test";

test.describe("Visual Regression Tests", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to homepage, wait for DOM content and fonts to settle
    await page.goto("/");
    await page.waitForLoadState("domcontentloaded");
    await page.waitForSelector("body > *", { state: "attached" });
    await page.evaluate(() => document.fonts.ready);
  });

  test("home page full layout matches baseline", async ({ page }) => {
    await expect(page).toHaveScreenshot("home-page-layout.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.05,
      animations: "disabled",
    });
  });

  test("site header matches baseline visual snapshot", async ({ page }) => {
    const header = page.locator("header");
    await expect(header).toBeVisible();
    await expect(header).toHaveScreenshot("site-header.png", {
      animations: "disabled",
    });
  });

  test("hero section matches baseline visual snapshot", async ({ page }) => {
    const hero = page.locator("main .text-center").first();
    await expect(hero).toBeVisible();
    await expect(hero).toHaveScreenshot("hero-section.png", {
      animations: "disabled",
    });
  });

  test("mobile viewport layout matches baseline", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.reload();
    await page.waitForLoadState("domcontentloaded");
    await page.waitForSelector("body > *", { state: "attached" });
    await page.evaluate(() => document.fonts.ready);

    await expect(page).toHaveScreenshot("home-page-mobile.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.05,
      animations: "disabled",
    });
  });
});
