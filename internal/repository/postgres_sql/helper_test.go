package postgres_sql

import (
	"testing"
)

func TestMigrationFileExists(t *testing.T) {
	migrationPath := GetMigrationPath()

	if migrationPath == "" {
		t.Log("Migration file not found - will use embedded migrations")
	} else {
		t.Logf("Migration file found at: %s", migrationPath)
	}
}

func TestSplitSQLStatements(t *testing.T) {
	sql := `
		CREATE TABLE users (id INT);
		CREATE TABLE rooms (id INT);
		-- This is a comment;
		CREATE INDEX idx ON users(id);
	`

	statements := splitSQLStatements(sql)

	if len(statements) != 3 {
		t.Errorf("Expected 3 statements, got %d", len(statements))
	}

	expectedStatements := []string{
		"CREATE TABLE users (id INT)",
		"CREATE TABLE rooms (id INT)",
		"CREATE INDEX idx ON users(id)",
	}

	for i, stmt := range statements {
		if stmt != expectedStatements[i] {
			t.Errorf("Statement %d: expected '%s', got '%s'", i, expectedStatements[i], stmt)
		}
	}
}
