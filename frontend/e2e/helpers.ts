import { Page, Locator, Response } from "@playwright/test";

const IS_CI = !!process.env.CI;

export const DEFAULT_TIMEOUTS = {
  short: IS_CI ? 10_000 : 5_000,
  medium: IS_CI ? 20_000 : 10_000,
  long: IS_CI ? 45_000 : 20_000,
  navigation: IS_CI ? 30_000 : 15_000,
};

/**
 * Wait for Next.js client-side hydration to complete by ensuring the React
 * root has rendered and DOM interactions can proceed safely.
 */
export async function waitForHydration(page: Page, timeout = DEFAULT_TIMEOUTS.medium): Promise<void> {
  await page.waitForLoadState("domcontentloaded", { timeout });
  // Wait for React to mount body contents
  await page.waitForSelector("body > *", { state: "attached", timeout });
}

/**
 * Wait for network activity to settle down (no active connections for at least 500ms).
 */
export async function waitForNetworkIdle(page: Page, timeout = DEFAULT_TIMEOUTS.medium): Promise<void> {
  try {
    await page.waitForLoadState("networkidle", { timeout });
  } catch {
    // If networkidle times out due to persistent polling or websockets, fallback to domcontentloaded
    await page.waitForLoadState("domcontentloaded", { timeout: DEFAULT_TIMEOUTS.short });
  }
}

/**
 * Wait until a specific API request completes with a matching URL pattern and returns response.
 */
export async function waitForApiResponse(
  page: Page,
  urlPattern: string | RegExp,
  timeout = DEFAULT_TIMEOUTS.medium
): Promise<Response> {
  return page.waitForResponse(
    (response) => {
      const url = response.url();
      if (typeof urlPattern === "string") {
        return url.includes(urlPattern);
      }
      return urlPattern.test(url);
    },
    { timeout }
  );
}

/**
 * Safely clicks an element after ensuring it is visible, enabled and scrolled into view.
 */
export async function safeClick(
  locator: Locator,
  options?: { timeout?: number; force?: boolean }
): Promise<void> {
  const timeout = options?.timeout ?? DEFAULT_TIMEOUTS.short;
  await locator.waitFor({ state: "visible", timeout });
  await locator.scrollIntoViewIfNeeded({ timeout });
  await locator.click({ timeout, force: options?.force });
}

/**
 * Repeatedly polls a predicate function until it returns true or times out.
 */
export async function waitForCondition(
  predicate: () => boolean | Promise<boolean>,
  options?: { timeout?: number; intervalMs?: number; description?: string }
): Promise<void> {
  const timeout = options?.timeout ?? DEFAULT_TIMEOUTS.medium;
  const interval = options?.intervalMs ?? 250;
  const start = Date.now();

  while (Date.now() - start < timeout) {
    if (await predicate()) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, interval));
  }

  throw new Error(`Timed out after ${timeout}ms waiting for condition: ${options?.description || "predicate"}`);
}
