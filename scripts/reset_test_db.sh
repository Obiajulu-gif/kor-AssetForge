#!/usr/bin/env bash
# scripts/reset_test_db.sh
# Resets the test database between integration test runs (#432).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"

DB_URL="${SQL_TEST_DATABASE_URL:-${DATABASE_URL:-postgresql://postgres:password@localhost:5432/testdb?sslmode=disable}}"

echo "=========================================================="
echo "Resetting test database: ${DB_URL%%:*}://***@${DB_URL##*@}"
echo "=========================================================="

if [[ "$DB_URL" == *":memory:"* ]] || [[ "$DB_URL" == *".sqlite"* ]] || [[ "$DB_URL" == *".db"* ]]; then
    echo "SQLite/In-memory database detected. Cleaning file artifacts..."
    rm -f "$BACKEND_DIR"/*.sqlite "$BACKEND_DIR"/*.db
    echo "✓ SQLite artifacts cleaned."
    exit 0
fi

if command -v psql >/dev/null 2>&1; then
    echo "Terminating active connections and dropping public schema..."
    psql "$DB_URL" -v ON_ERROR_STOP=1 <<-EOSQL
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = current_database() AND pid <> pg_backend_pid();

        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO public;
EOSQL

    echo "Running schema migrations up on clean database..."
    if [ -f "$REPO_ROOT/scripts/migrate.sh" ]; then
        "$REPO_ROOT/scripts/migrate.sh" up
    fi

    echo "✓ Test database successfully reset."
else
    echo "Notice: psql not found in PATH. Skipping direct Postgres schema drop."
fi
