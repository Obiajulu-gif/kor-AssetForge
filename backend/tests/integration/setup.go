//go:build ignore
// +build ignore

package integration

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// CleanDatabase deletes or truncates all table rows in the database to prevent test pollution.
func CleanDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	// Detect database dialect
	dialName := db.Dialector.Name()

	// List of tables in dependency order for safe wiping
	tables := []string{
		"webhook_delivery_logs",
		"webhook_subscriptions",
		"audit_logs",
		"login_attempts",
		"sessions",
		"kyc_records",
		"beneficial_owners",
		"user_activities",
		"user_preferences",
		"disputes",
		"approvals",
		"transactions",
		"payments",
		"stakings",
		"listings",
		"assets",
		"users",
	}

	if dialName == "postgres" {
		// PostgreSQL: TRUNCATE CASCADE with restart identity for fast and clean wipe
		quoted := make([]string, len(tables))
		for i, t := range tables {
			quoted[i] = fmt.Sprintf(`"%s"`, t)
		}
		stmt := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", strings.Join(quoted, ", "))
		return db.Exec(stmt).Error
	}

	// SQLite / In-memory: disable foreign keys, delete all records, re-enable foreign keys
	if err := db.Exec("PRAGMA foreign_keys = OFF;").Error; err != nil {
		return err
	}
	defer db.Exec("PRAGMA foreign_keys = ON;")

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s;", table)).Error; err != nil {
			// Table might not exist in auto-migrated schema, ignore if missing
			continue
		}
	}

	return nil
}

// ResetDatabase wipes the database and re-seeds it with baseline test fixtures.
func ResetDatabase(db *gorm.DB, setup *TestSetup) error {
	if err := CleanDatabase(db); err != nil {
		return fmt.Errorf("failed to clean database: %w", err)
	}

	if setup != nil {
		setup.TestUsers = []TestUser{}
		setup.seedTestData(nil)
	}

	return nil
}
