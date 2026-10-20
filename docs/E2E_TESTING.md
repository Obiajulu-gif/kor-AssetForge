# End-to-End (E2E) Testing Guide

This guide outlines best practices, configuration, and guidelines for writing and running reliable Playwright E2E tests for kor-AssetForge.

---

## 1. Timeout Architecture: CI vs. Local

E2E tests execute in both fast local environments and resource-constrained CI runners (e.g. GitHub Actions Ubuntu runners). To prevent false negatives while maintaining developer velocity locally, timeouts are tiered based on `process.env.CI`:

| Setting | Local Development | CI Environment | Purpose |
| :--- | :--- | :--- | :--- |
| **Overall Test Timeout** | `30s` | `60s` | Bounds test run time; prevents hanging on slow runners |
| **Expect Timeout** | `5s` | `10s` | Assertion retry ceiling (e.g., waiting for element visible) |
| **Action Timeout** | `10s` | `15s` | Element actions (clicks, fills, checks) |
| **Navigation Timeout** | `15s` | `30s` | Page navigation and redirect settlements |
| **Dev / WebServer Startup** | `120s` | `240s` | Next.js production build startup budget |

---

## 2. Custom Wait Helpers

Do **not** use arbitrary `page.waitForTimeout(3000)`. Flaky tests usually stem from racing against React hydration, async network fetching, or route transitions.

Use the reusable helper utilities located in `frontend/e2e/helpers.ts`:

```typescript
import { test, expect } from "@playwright/test";
import {
  waitForHydration,
  waitForNetworkIdle,
  waitForApiResponse,
  safeClick,
  waitForCondition,
} from "./helpers";

test("creates asset after server hydration", async ({ page }) => {
  await page.goto("/create-asset");

  // Wait for React hydration before clicking or interacting
  await waitForHydration(page);

  // Safely click interactive elements
  const submitBtn = page.getByRole("button", { name: /create asset/i });
  await safeClick(submitBtn);

  // Wait for specific backend endpoint response
  const response = await waitForApiResponse(page, "/api/v1/assets");
  expect(response.status()).toBe(201);
});
```

### Available Helper Utilities
- `waitForHydration(page, timeout?)`: Waits for `domcontentloaded` and ensures the root body elements are mounted.
- `waitForNetworkIdle(page, timeout?)`: Waits until there are no open network connections for 500ms, with automatic fallback for open sockets/SSE.
- `waitForApiResponse(page, urlPattern, timeout?)`: Waits for an explicit API network request and returns the Playwright `Response`.
- `safeClick(locator, options?)`: Waits for visibility and auto-scrolls the element into view before clicking.
- `waitForCondition(predicate, options?)`: Polls a custom condition until resolved or timeout is reached.

---

## 3. Running E2E Tests Locally

Run headlessly:
```bash
cd frontend
npm run test:e2e
```

Run with Playwright Interactive UI Mode:
```bash
cd frontend
npm run test:e2e:ui
```

Debug single spec file:
```bash
cd frontend
npx playwright test e2e/home.spec.ts --debug
```

---

## 4. CI Workflow Integration

Playwright tests run in `.github/workflows/ci.yml` under the `e2e-frontend` job.
- Job timeout is capped at **25 minutes**.
- Test run step timeout is set to **15 minutes**.
- Playwright HTML reports and failure traces are automatically saved and uploaded as GitHub Actions artifacts upon completion.
