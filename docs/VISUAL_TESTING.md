# Visual Regression Testing Guide

This document describes the visual regression testing setup for kor-AssetForge using Playwright.

---

## 1. Overview

Visual regression testing prevents accidental layout shifts, CSS breakages, typography shifts, and component regressions across releases. Playwright compares pixel-by-pixel screenshots of rendered pages and components against committed baseline images stored in `frontend/e2e/snapshots/`.

---

## 2. Configuration & Settings

Visual testing settings are configured in `frontend/playwright.config.ts`:

- **Snapshot Directory**: `frontend/e2e/snapshots/`
- **Screenshot Capture**: Enabled automatically on failures and on demand.
- **Diff Tolerance**:
  - `maxDiffPixelRatio: 0.05` (allows up to 5% non-critical pixel anti-aliasing variations)
  - `animations: "disabled"` (prevents false positives caused by mid-flight CSS/JS transitions)

---

## 3. Running Visual Regression Tests

To run the visual test suite:

```bash
cd frontend
npx playwright test e2e/visual.spec.ts
```

To run with Playwright UI mode:
```bash
cd frontend
npx playwright test e2e/visual.spec.ts --ui
```

---

## 4. Updating Baseline Snapshots

When an intentional design update or layout refactor is introduced, the baseline snapshots must be updated.

To regenerate and update baselines:

```bash
cd frontend
npx playwright test e2e/visual.spec.ts --update-snapshots
```

Review the updated snapshot files in `frontend/e2e/snapshots/` using `git diff`, and commit the updated PNG images alongside your code changes.

---

## 5. Reviewing Visual Diff Failures in CI

When a visual test fails in CI (`.github/workflows/ci.yml`):
1. Navigate to the failed GitHub Actions workflow run under the `e2e-frontend` job.
2. Download the uploaded `playwright-report` and `test-results` artifacts.
3. Open `index.html` in the report. Playwright displays an interactive 3-way visual diff:
   - **Expected**: Baseline image
   - **Actual**: Screen captured in CI
   - **Diff**: Highlighted pixel differences in red.
