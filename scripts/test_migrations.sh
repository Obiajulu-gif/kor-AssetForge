#!/usr/bin/env bash
# scripts/test_migrations.sh
# Tests schema migrations up and down rollbacks for data integrity (#309).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"

DB_URL="${SQL_TEST_DATABASE_URL:-${DATABASE_URL:-postgresql://postgres:password@localhost:5432/testdb?sslmode=disable}}"

echo "=========================================================="
echo "kor-AssetForge Database Migration Rollback Test Suite"
echo "=========================================================="
echo "Target DB: ${DB_URL%%:*}://***@${DB_URL##*@}"
echo ""

# 1. Run the Go automated rollback integration test suite
echo "1. Running automated Go migration rollback tests..."
echo "----------------------------------------------------------"
cd "$BACKEND_DIR"
if SQL_TEST_DATABASE_URL="$DB_URL" go test -v ./tests/migrations/...; then
    echo "✓ Go migration rollback tests passed successfully."
else
    echo "✗ Go migration rollback tests failed."
    exit 1
fi

# 2. If psql is available, execute a direct migration verification cycle
if command -v psql >/dev/null 2>&1; then
    echo ""
    echo "2. Direct CLI Migration Up/Down Integrity Verification..."
    echo "----------------------------------------------------------"

    if ! psql "$DB_URL" -c "SELECT 1;" >/dev/null 2>&1; then
        echo "Warning: Database not directly reachable with psql. Skipping direct CLI checks."
    else
        echo "Applying migrations up..."
        "$REPO_ROOT/scripts/migrate.sh" up

        echo "Checking migration version..."
        "$REPO_ROOT/scripts/migrate.sh" version

        echo "Testing rollback of the latest migration..."
        "$REPO_ROOT/scripts/migrate.sh" down 1

        echo "Re-applying migration up to ensure idempotency..."
        "$REPO_ROOT/scripts/migrate.sh" up

        echo "✓ CLI migration rollback cycle verified."
    fi
fi

echo ""
echo "=========================================================="
echo "All Migration Rollback Tests Completed Successfully!"
echo "=========================================================="
