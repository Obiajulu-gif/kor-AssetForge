#!/usr/bin/env bash
# scripts/analyze_indexes.sh
# Analyzes foreign keys, missing indexes, and index utilization in PostgreSQL (#308).

set -euo pipefail

DB_URL="${DATABASE_URL:-postgresql://postgres:password@localhost:5432/assetforge?sslmode=disable}"

echo "========================================================"
echo "kor-AssetForge Database Index Analyzer"
echo "========================================================"
echo "Target DB: ${DB_URL%%:*}://***@${DB_URL##*@}"
echo ""

# Query to find unindexed foreign keys
MISSING_FK_SQL="
WITH fk_columns AS (
    SELECT
        conrelid::regclass AS table_name,
        a.attname AS column_name,
        confrelid::regclass AS referenced_table,
        af.attname AS referenced_column
    FROM pg_constraint c
    JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS u(attnum, attorder) ON true
    JOIN LATERAL unnest(c.confkey) WITH ORDINALITY AS uf(attnum, attorder) ON u.attorder = uf.attorder
    JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = u.attnum
    JOIN pg_attribute af ON af.attrelid = c.confrelid AND af.attnum = uf.attnum
    WHERE c.contype = 'f'
),
indexed_columns AS (
    SELECT
        i.indrelid::regclass AS table_name,
        a.attname AS column_name
    FROM pg_index i
    JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
)
SELECT
    fk.table_name,
    fk.column_name,
    fk.referenced_table,
    fk.referenced_column
FROM fk_columns fk
LEFT JOIN indexed_columns idx
    ON fk.table_name = idx.table_name AND fk.column_name = idx.column_name
WHERE idx.column_name IS NULL
ORDER BY fk.table_name, fk.column_name;
"

echo "1. Scanning for Foreign Keys without Indexes..."
echo "--------------------------------------------------------"
if command -v psql >/dev/null 2>&1; then
    MISSING_COUNT=$(psql "$DB_URL" -t -A -c "
        SELECT count(*) FROM ($MISSING_FK_SQL) sub;
    " 2>/dev/null || echo "error")

    if [ "$MISSING_COUNT" = "error" ]; then
        echo "Warning: Unable to connect to database using psql. Please verify database connectivity."
    elif [ "$MISSING_COUNT" -eq 0 ]; then
        echo "SUCCESS: All foreign key columns have indexes! (0 unindexed foreign keys found)"
    else
        echo "WARNING: Found $MISSING_COUNT foreign key column(s) lacking an index:"
        psql "$DB_URL" -c "$MISSING_FK_SQL"
    fi

    echo ""
    echo "2. Index Usage and Size Summary (Top 20 Indexes):"
    echo "--------------------------------------------------------"
    psql "$DB_URL" -c "
        SELECT
            relname AS table_name,
            indexrelname AS index_name,
            idx_scan AS scan_count,
            pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
        FROM pg_stat_user_indexes
        ORDER BY pg_relation_size(indexrelid) DESC
        LIMIT 20;
    " 2>/dev/null || true

    echo ""
    echo "3. Sample Query EXPLAIN Plan (Foreign Key JOIN verification):"
    echo "--------------------------------------------------------"
    psql "$DB_URL" -c "
        EXPLAIN SELECT s.id, a.name, a.symbol
        FROM pool_swaps s
        JOIN assets a ON s.input_asset_id = a.id
        LIMIT 10;
    " 2>/dev/null || true
else
    echo "Note: 'psql' client not installed on host. Run this script inside the PostgreSQL container or where psql is available."
    echo "You can run the following SQL query to inspect missing foreign key indexes:"
    echo "$MISSING_FK_SQL"
fi

echo ""
echo "========================================================"
echo "Analysis Complete"
echo "========================================================"
