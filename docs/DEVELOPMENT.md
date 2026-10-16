# Backend development

Use Go 1.25 or newer. Review [Deployment](DEPLOYMENT.md) before running a server;
`JWT_SECRET` and `SOROBAN_ASSET_TOKEN_CONTRACT_ID` are required.

## SQL safety

Keep SQL structure static and pass every external value separately. GORM uses
`?` placeholders, while `database/sql` with PostgreSQL uses `$1`, `$2`, etc.

```go
db.Raw("SELECT * FROM users WHERE email = ?", email).Scan(&users)
db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
sqlDB.QueryContext(ctx, "SELECT id FROM users WHERE email = $1", email)
db.Raw("SELECT DATE_TRUNC(CAST(? AS text), recorded_at) FROM valuation_histories", "month").Scan(&rows)
```

Never build SQL with concatenation, `fmt.Sprintf`, or request-provided SQL
fragments. Do not put placeholders inside quotes. `Order`, `Select`, `Table`,
and other SQL-structure APIs also need static text or explicit allowlists;
placeholders bind values, not SQL keywords or table/column identifiers.

Include filtering, grouping, and ordering in a complete parameterized `Raw`
statement. Chaining `Where` or `Group` after `Raw` does not reliably add those
clauses to the raw statement. Alternatively use the GORM query builder throughout.

The audit for #289 found that the analytics handler already bound its timestamp
and migration SQL was embedded at build time. Valuation date granularity now uses
a bind parameter, rate-limit history binds the endpoint filter in the complete
query, and performance-test cleanup uses bound patterns. The migrations execute
only trusted `go:embed` scripts: complete DDL/seed programs cannot themselves be
passed as query-value placeholders. Keep these files under code review and never
replace embedded content with request/environment-provided SQL or interpolation.

## Static analysis and review

Install the CI-pinned analyzer in an isolated Python environment:

```bash
python3 -m venv /tmp/assetforge-sql-tools
/tmp/assetforge-sql-tools/bin/pip install semgrep==1.168.0
PATH="/tmp/assetforge-sql-tools/bin:$PATH" ./scripts/check_sql_safety.sh
```

The repository's Semgrep taint rule detects formatted, concatenated, and
request-derived query text passed to GORM `Raw`/`Exec` and `database/sql` query,
execution, and preparation methods, including context variants. It follows local assignments and verifies
positive and negative fixtures before scanning production and test code. Passing
concatenated values as separate bind arguments is allowed. The only suppression
documents trusted embedded seed SQL. This is a focused check, not a complete
interprocedural SQL-injection proof; review all sources of dynamic query text.

The PR template requires reviewers to check parameter binding, allowlists for SQL
structure, the embedded migration trust boundary, and a passing SQL scan. Any new
suppression must document the trusted source and be reviewed explicitly.

## Testing

```bash
cd backend
go test $(go list ./... | grep -v /tests/performance)
go vet ./...
```

PostgreSQL query regressions use `SQL_TEST_DATABASE_URL`. Point it at a disposable
database and run the normal tests; these tests use transactions and roll back
their schema/data. They verify date truncation and literal handling of malicious
endpoint filters against PostgreSQL itself. Without this variable only those
PostgreSQL-specific tests skip. CI sets it using its PostgreSQL service.

Load tests under `backend/tests/performance` require a running configured server
and a matching `JWT_SECRET`; they are separate from the normal unit test run.
The older `backend/tests/integration` suite is currently excluded by its existing
`ignore` build constraint and does not contribute integration coverage.
