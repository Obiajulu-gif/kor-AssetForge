package migrations_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/kor-assetforge/migrations"
)

// getTestDB connects to the test database specified by SQL_TEST_DATABASE_URL or DATABASE_URL.
// If neither is set or the database is unreachable, the test is skipped.
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("SQL_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "host=localhost user=postgres password=password dbname=testdb port=5432 sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("cannot open test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("test database not reachable (%s): %v", dsn, err)
	}
	return db
}

// TestMigrationRollbackFullCycle tests that running all migrations Up, then Down,
// and Up again completes successfully and idempotently without errors (#309).
func TestMigrationRollbackFullCycle(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	m := migrations.New(db)

	// 1. Run all migrations up
	err := m.Up()
	require.NoError(t, err, "m.Up() should apply all migrations cleanly")

	vUp, dirty, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirty, "migration version should not be dirty after Up")
	assert.Greater(t, vUp, uint(0), "version should be > 0 after applying migrations")

	// 2. Verify foreign key constraints and schema elements exist
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'users'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "users table should exist after Up")

	// 3. Rollback the most recent migration (step -1)
	err = m.Down(1)
	require.NoError(t, err, "m.Down(1) should roll back the latest migration")

	vDown, dirtyDown, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirtyDown)
	assert.Less(t, vDown, vUp, "version should decrease after rollback")

	// 4. Re-apply up to verify idempotency
	err = m.Up()
	require.NoError(t, err, "m.Up() should re-apply rolled back migration cleanly")

	vFinal, dirtyFinal, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirtyFinal)
	assert.Equal(t, vUp, vFinal, "final version should match original up version")
}

// TestMigrationForeignKeyIntegrity verifies that foreign key constraints
// remain active and enforce referential integrity across migration cycles (#309).
func TestMigrationForeignKeyIntegrity(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	m := migrations.New(db)
	require.NoError(t, m.Up())

	// Attempt inserting an invalid foreign key into user_balances (non-existent user 99999999)
	_, err := db.Exec(`
		INSERT INTO user_balances (user_id, asset_id, balance, locked_balance, updated_at)
		VALUES (99999999, 1, 100, 0, NOW())
	`)
	assert.Error(t, err, "inserting invalid foreign key reference must fail")

	// Rollback migration 0040 (missing indexes)
	err = m.Down(1)
	require.NoError(t, err, "rolling back index migration should succeed")

	// Referential integrity should still be enforced even when specific index is rolled back
	_, err = db.Exec(`
		INSERT INTO user_balances (user_id, asset_id, balance, locked_balance, updated_at)
		VALUES (99999999, 1, 100, 0, NOW())
	`)
	assert.Error(t, err, "foreign key constraint must be maintained after index rollback")

	// Re-apply migration
	require.NoError(t, m.Up())
}

// TestStepByStepRollbackVerification executes a step-by-step migration rollback
// to ensure each up/down pair unwinds cleanly without dangling constraints (#309).
func TestStepByStepRollbackVerification(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	m := migrations.New(db)
	require.NoError(t, m.Up())

	currentVersion, dirty, err := m.Version()
	require.NoError(t, err)
	require.False(t, dirty)

	// Step back 2 migrations and step forward 2 migrations
	steps := 2
	if currentVersion >= uint(steps) {
		err = m.Down(steps)
		require.NoError(t, err, fmt.Sprintf("rolling back %d steps should succeed", steps))

		rolledBackVersion, dirty, err := m.Version()
		require.NoError(t, err)
		assert.False(t, dirty)
		assert.Equal(t, currentVersion-uint(steps), rolledBackVersion)

		// Re-apply the 2 steps
		err = m.Up()
		require.NoError(t, err, "re-applying migrations after multi-step rollback should succeed")

		restoredVersion, dirty, err := m.Version()
		require.NoError(t, err)
		assert.False(t, dirty)
		assert.Equal(t, currentVersion, restoredVersion)
	}
}
