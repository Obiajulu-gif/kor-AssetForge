# Database Connection Pool Tuning Guide

This document provides guidance on configuring, calculating, and optimizing PostgreSQL connection pooling for **kor-AssetForge** across development, staging, and production deployment environments.

- **Related Documentation:**
  - [`DATABASE_REPLICATION.md`](./DATABASE_REPLICATION.md) — Read replicas and query routing
  - [`SCALING_GUIDE.md`](./SCALING_GUIDE.md) — Horizontal scaling and performance guidelines
  - [`OBSERVABILITY.md`](./OBSERVABILITY.md) — Metrics, tracing, and log aggregation
  - [`../backend/config/config.go`](../backend/config/config.go) — Database pool configuration implementation

---

## 1. Overview & Architecture

kor-AssetForge uses Go's standard `database/sql` connection pool underneath GORM and raw SQL migration drivers. By default, an unconfigured connection pool can open an unlimited number of concurrent connections, quickly exhausting PostgreSQL's `max_connections` limit and degrading database throughput due to context-switching overhead.

### Key Pool Parameters

| Environment Variable | Config Field | Default | Description |
|----------------------|--------------|---------|-------------|
| `DB_MAX_OPEN_CONNS` | `MaxOpen` | `25` | Maximum number of open connections to the database (active + idle). |
| `DB_MAX_IDLE_CONNS` | `MaxIdle` | `10` | Maximum number of connections retained in the idle pool. |
| `DB_CONN_MAX_LIFETIME` | `MaxLifetime` | `30m` | Maximum amount of time a connection may be reused before being closed. |
| `DB_CONN_MAX_IDLE_TIME` | `MaxIdleTime` | `5m` | Maximum amount of time a connection may sit idle before being closed. |
| `DB_CONNECT_RETRIES` | `ConnectRetry` | `3` | Number of initial connection retry attempts on startup. |
| `DB_CONNECT_RETRY_WAIT` | `ConnectWait` | `1s` | Backoff delay between startup connection retries. |

---

## 2. Pool Sizing Formula

A common misconception is that larger connection pools always increase performance. In relational databases like PostgreSQL, each active connection consumes dedicated memory (work_mem, maintenance_work_mem, stack size) and competes for CPU and disk I/O.

### 2.1 The Classic PostgreSQL Connection Formula

Derived from PostgreSQL performance research (HikariCP / PostgreSQL benchmarks):

$$\text{Pool Size} = ((\text{Core Count} \times 2) + \text{Effective Spindle Count})$$

* For a database server with 4 CPU cores and an SSD (effective spindle count = 1), the optimal total active database connections typically fall between **9 and 15**.

### 2.2 Sizing Across Multi-Instance Deployments

When running multiple instances (replicas/pods) of the kor-AssetForge API service behind a load balancer, total database connections must not exceed PostgreSQL's `max_connections` budget:

$$\text{Total App Connections} = \sum_{i=1}^{N_{\text{instances}}} (\text{DB\_MAX\_OPEN\_CONNS}_i) \le (\text{max\_connections} - \text{superuser\_reserved} - \text{worker\_pool})$$

Where:
- $\text{superuser\_reserved}$: Typically `3` to `5` connections reserved for DBAs and monitoring agents (`superuser_reserved_connections` in `postgresql.conf`).
- $\text{worker\_pool}$: Connections allocated to async workers, cron schedulers, or replication monitors.

#### Example Calculation:
- PostgreSQL `max_connections` = `100`
- Superuser reserved = `5`
- Asynchronous batch jobs = `10`
- Remaining budget for API instances = `85`
- If running 4 API instances: $\text{DB\_MAX\_OPEN\_CONNS} \le \lfloor 85 / 4 \rfloor = 21$ per instance.

---

## 3. Environment Profiles & Recommended Configurations

### 3.1 Development (`dev`)
Designed for local workstations with minimal resource usage and fast restart times.

```env
# Local Development (.env)
DATABASE_URL=postgresql://postgres:password@localhost:5432/assetforge?sslmode=disable
DB_MAX_OPEN_CONNS=5
DB_MAX_IDLE_CONNS=2
DB_CONN_MAX_LIFETIME=10m
DB_CONN_MAX_IDLE_TIME=2m
DB_CONNECT_RETRIES=2
DB_CONNECT_RETRY_WAIT=500ms
```

- **Rationale:** Keeps developer machines light; avoids leaking hundreds of connections across hot reloads.

### 3.2 Staging / QA (`staging`)
Simulates moderate multi-user concurrency and validates connection recycling behavior.

```env
# Staging Environment
DATABASE_URL=postgresql://app:staging_secret@db.staging.internal:5432/assetforge?sslmode=require
DB_MAX_OPEN_CONNS=15
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=15m
DB_CONN_MAX_IDLE_TIME=5m
DB_CONNECT_RETRIES=3
DB_CONNECT_RETRY_WAIT=1s
```

- **Rationale:** High enough to handle automated regression suites and testing load, low enough to detect connection leak regressions.

### 3.3 Production (`production`)
Tuned for high throughput, resilience against database failovers, and low latency under traffic spikes.

```env
# Production Environment (.env.prod)
DATABASE_URL=postgresql://app:prod_secret@primary.internal:5432/assetforge?sslmode=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=30m
DB_CONN_MAX_IDLE_TIME=5m
DB_CONNECT_RETRIES=5
DB_CONNECT_RETRY_WAIT=2s
```

- **Rationale:** 
  - `DB_MAX_OPEN_CONNS=25` supports high concurrency without saturating a standard 4-8 core database instance.
  - `DB_MAX_IDLE_CONNS=10` keeps pre-warmed connections ready for immediate reuse, avoiding TLS/TCP handshake latency.
  - `DB_CONN_MAX_LIFETIME=30m` ensures connections are recycled periodically, preventing stale connections during RDS/cloud multi-AZ failovers or IP changes.
  - `DB_CONN_MAX_IDLE_TIME=5m` releases unused connections during low traffic windows (e.g. night).

---

## 4. Monitoring & Observability

### 4.1 Go Runtime Connection Pool Metrics

kor-AssetForge exposes database pool metrics through Prometheus (`/metrics`) using `sql.DB.Stats()`:

| Metric Name | Type | Description |
|-------------|------|-------------|
| `go_db_connections_open` | Gauge | Total established connections (in-use + idle). |
| `go_db_connections_in_use` | Gauge | Connections currently executing a query or in a transaction. |
| `go_db_connections_idle` | Gauge | Connections waiting in the pool ready for reuse. |
| `go_db_connection_wait_count_total` | Counter | Total number of times queries waited for a free connection. |
| `go_db_connection_wait_duration_seconds_total` | Counter | Total time blocked waiting for a connection from the pool. |
| `go_db_connection_max_idle_closed_total` | Counter | Connections closed because `DB_MAX_IDLE_CONNS` was exceeded. |
| `go_db_connection_max_lifetime_closed_total` | Counter | Connections closed because `DB_CONN_MAX_LIFETIME` was reached. |

### 4.2 Key Alert Thresholds

1. **Connection Starvation:**
   - **Condition:** `rate(go_db_connection_wait_count_total[5m]) > 5`
   - **Meaning:** Queries are queueing up waiting for a connection. Increase `DB_MAX_OPEN_CONNS` if database CPU allows, or investigate slow queries.

2. **Pool Saturation:**
   - **Condition:** `go_db_connections_in_use / go_db_connections_open > 0.85` for > 3 minutes.
   - **Meaning:** 85%+ of open connections are constantly busy.

3. **PostgreSQL Server Saturation:**
   - Run in PostgreSQL:
     ```sql
     SELECT count(*), state FROM pg_stat_activity WHERE datname = 'assetforge' GROUP BY state;
     ```
   - If `count(*)` approaches `max_connections`, scale instances down or install an external pooler (e.g. PgBouncer).

---

## 5. Troubleshooting & Runbook

### Issue: "pq: remaining connection slots are reserved for non-replication superuser connections" or "too many clients already"
- **Cause:** Total connections across all application pods, background jobs, and migrations exceeded PostgreSQL `max_connections`.
- **Resolution:**
  1. Inspect current active connections:
     ```sql
     SELECT client_addr, count(*) FROM pg_stat_activity GROUP BY client_addr ORDER BY count(*) DESC;
     ```
  2. Reduce `DB_MAX_OPEN_CONNS` across API deployment manifests.
  3. Ensure migrations and one-off CLI tasks close their connections upon exit.
  4. For large-scale setups, deploy **PgBouncer** in transaction pooling mode.

### Issue: Connection Leaks (`InUse` constantly increases without decreasing)
- **Cause:** HTTP handlers opening `*sql.Rows` or `*gorm.DB` transactions without properly closing them (`rows.Close()`, `tx.Rollback()`, or unhandled error returns).
- **Resolution:**
  1. Check `go_db_connections_in_use` over time.
  2. Find long-running queries:
     ```sql
     SELECT pid, now() - query_start AS duration, query, state 
     FROM pg_stat_activity 
     WHERE state != 'idle' AND datname = 'assetforge' 
     ORDER BY duration DESC;
     ```
  3. Ensure every transaction has a `defer tx.Rollback()` pattern.

### Issue: "driver: bad connection" or TCP RST errors during Cloud DB Failover
- **Cause:** The database primary failed over (e.g., AWS RDS Multi-AZ failover), but Go's pool retained established TCP connections to the old IP address.
- **Resolution:**
  - Ensure `DB_CONN_MAX_LIFETIME` is set to a reasonable duration (e.g., `15m` to `30m`). Never set it to `0` in production.
  - Set `DB_CONN_MAX_IDLE_TIME` to `5m` so stale idle connections are pruned regularly.

### Issue: High Query Latency on Idle Systems (Cold Connection Handshake)
- **Cause:** `DB_MAX_IDLE_CONNS` is set too low (e.g. `0` or `1`), causing every incoming request to incur a new TCP connection and TLS handshake with PostgreSQL.
- **Resolution:**
  - Increase `DB_MAX_IDLE_CONNS` to at least `5` - `10` so idle connections remain available.
