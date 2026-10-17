# Performance Testing

kor-AssetForge runs a load and performance suite at 10x expected load to catch
regressions before they reach production. The suite lives in
`backend/tests/performance/` and is driven nightly by
[`.github/workflows/performance.yml`](../.github/workflows/performance.yml).

## What runs

Each scenario in the suite runs concurrently against a live backend and asserts
two things:

- a **minimum success rate**, and
- a **maximum P95 latency**.

Results are logged per scenario, for example:

```
=== Assets/List 1000-concurrent 30s ===
  Total requests : 123456
  Success rate   : 99.9%
  Avg latency    : 12.3 ms
  P50 latency    : 9.0 ms
  P95 latency    : 25.0 ms
  P99 latency    : 40.0 ms
```

## Baselines

Recorded baselines live in
[`backend/tests/performance/baselines.json`](../backend/tests/performance/baselines.json).
Each entry holds `avg_ms`, `p50_ms`, `p95_ms`, `p99_ms`, and `success_rate`
for one scenario keyed by its scenario name.

Baselines are intentionally conservative budgets. Update them only after an
intentional, reviewed performance change.

## Regression detection

The regression gate is implemented by
[`backend/tests/performance/compare.go`](../backend/tests/performance/compare.go).
It parses the captured load-test output, compares it to the baselines, and
applies the default policy:

| Rule              | Default                          |
| ----------------- | -------------------------------- |
| P95 latency       | may not exceed baseline × **1.20** |
| Average latency   | may not exceed baseline × **1.25** |
| Absolute success  | must stay ≥ **90%**              |
| Success-rate drop | ≤ **1.0** point below baseline    |

## Running locally

```bash
# Start the backend (with Postgres + Redis) first, then:
cd backend
go test ./tests/performance/... -v -timeout 30m 2>&1 | tee perf-output.txt

# Compare the captured output against the baselines:
PERF_OUTPUT_FILE=perf-output.txt PERF_REPORT_FILE=perf-report.md \
  go test ./tests/performance/... -run TestCompareToBaselines -v
```

Against a custom target:

```bash
PERF_BASE_URL=http://staging:8080 go test ./tests/performance/... -v
```

## Nightly pipeline

The nightly workflow:

1. Starts Postgres, Redis, and the backend server.
2. Runs the full performance suite and captures the output.
3. Runs the regression gate and writes a Markdown report.
4. Publishes the report to the GitHub Actions **job summary** (dashboard) and
   uploads `perf-output.txt`, `perf-report.md`, and `server.log` as artifacts.
5. On regression, opens (or comments on) an issue labelled `performance` and
   fails the job.

## Environment variables

| Variable            | Purpose                                             |
| ------------------- | --------------------------------------------------- |
| `PERF_BASE_URL`     | Base URL of the server under test.                  |
| `DATABASE_URL`      | Postgres DSN used to seed test data.                |
| `JWT_SECRET`        | Must match the server's JWT secret.                 |
| `PERF_OUTPUT_FILE`  | Captured load-test log for the comparison gate.     |
| `PERF_BASELINES_FILE` | Override the baselines file (default `baselines.json`). |
| `PERF_REPORT_FILE`  | Where the Markdown comparison report is written.    |
