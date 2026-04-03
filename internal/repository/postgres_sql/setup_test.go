package postgres_sql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func SetupTestDatabase(t *testing.T) *sql.DB {
	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Cannot connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("Cannot ping PostgreSQL: %v", err)
	}

	testDBName := "booking_service_test"

	_, err = db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()`, testDBName))
	if err != nil {
		t.Logf("Warning: Could not terminate connections: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if err != nil {
		t.Logf("Warning: Could not drop test database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Skipf("Cannot create test database: %v", err)
	}
	t.Logf("Created database: %s", testDBName)

	testConnStr := fmt.Sprintf("host=localhost port=5432 user=postgres password=postgres dbname=%s sslmode=disable", testDBName)
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		t.Skipf("Cannot ping test database: %v", err)
	}

	migrationPath := findMigrationPath()
	if migrationPath == "" {
		t.Fatal("Migration file not found! Please ensure 001_initial_schema.up.sql exists in migrations directory")
	}

	t.Logf("Running migrations from: %s", migrationPath)
	if err := RunMigrationsFromFile(t, testDB, migrationPath); err != nil {
		t.Fatalf("Cannot run migrations: %v", err)
	}
	t.Log("Migrations completed successfully")

	if err := verifyTablesExist(t, testDB); err != nil {
		t.Fatalf("Tables verification failed: %v", err)
	}

	return testDB
}

func findMigrationPath() string {
	paths := []string{
		"../../../migrations/001_initial_schema_test.sql",
		"../../migrations/001_initial_schema_test.sql",
		"../migrations/001_initial_schema_test.sql",
		"./migrations/001_initial_schema_test.sql",
		"migrations/001_initial_schema_test.sql",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}
	return ""
}

func RunMigrationsFromFile(t *testing.T, db *sql.DB, migrationFile string) error {
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migrationFile, err)
	}

	statements := splitSQLStatements(string(content))
	t.Logf("Found %d SQL statements in migration file", len(statements))

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if strings.HasPrefix(stmt, "--") {
			continue
		}

		preview := stmt
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		t.Logf("Executing statement %d: %s", i, preview)

		if _, err := db.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "already exists") {
				t.Logf("Statement %d already exists, skipping", i)
				continue
			}
			return fmt.Errorf("failed to execute statement %d: %s\nError: %w", i, preview, err)
		}
	}

	return nil
}

func splitSQLStatements(sql string) []string {
	var statements []string
	var currentStmt strings.Builder
	inString := false
	inDollarQuote := false
	escapeNext := false
	lineComment := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if escapeNext {
			currentStmt.WriteByte(ch)
			escapeNext = false
			continue
		}

		if ch == '\\' {
			currentStmt.WriteByte(ch)
			escapeNext = true
			continue
		}

		if !inString && !inDollarQuote && i+1 < len(sql) && sql[i:i+2] == "--" {
			lineComment = true
			i++
			continue
		}

		if lineComment && ch == '\n' {
			lineComment = false
			continue
		}

		if lineComment {
			continue
		}

		if !inString && !inDollarQuote && ch == '$' && i+1 < len(sql) && sql[i+1] == '$' {
			inDollarQuote = true
			currentStmt.WriteString("$$")
			i++
			continue
		}

		if inDollarQuote && ch == '$' && i+1 < len(sql) && sql[i+1] == '$' {
			inDollarQuote = false
			currentStmt.WriteString("$$")
			i++
			continue
		}

		if !inDollarQuote && ch == '\'' {
			currentStmt.WriteByte(ch)
			inString = !inString
			continue
		}

		if ch == ';' && !inString && !inDollarQuote {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" && !strings.HasPrefix(stmt, "--") {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
			continue
		}

		currentStmt.WriteByte(ch)
	}

	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		if stmt != "" && !strings.HasPrefix(stmt, "--") {
			statements = append(statements, stmt)
		}
	}

	return statements
}

func verifyTablesExist(t *testing.T, db *sql.DB) error {
	tables := []string{"users", "rooms", "schedules", "slots", "bookings"}

	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1
		)`
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if !exists {
			rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
			if err == nil {
				defer rows.Close()
				tables := []string{}
				for rows.Next() {
					var name string
					rows.Scan(&name)
					tables = append(tables, name)
				}
				t.Logf("Existing tables in database: %v", tables)
			}
			return fmt.Errorf("table %s does not exist after migrations", table)
		}
		t.Logf("✓ Table %s exists", table)
	}

	return nil
}

func TruncateTables(db *sql.DB) error {
	_, err := db.Exec("SET session_replication_role = 'replica'")
	if err != nil {
	}

	tables := []string{"bookings", "slots", "schedules", "rooms", "users"}

	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1
		)`
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if table %s exists: %w", table, err)
		}

		if exists {
			_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
			if err != nil {
				return fmt.Errorf("failed to truncate table %s: %w", table, err)
			}
		}
	}

	_, err = db.Exec("SET session_replication_role = 'origin'")

	return nil
}

func TeardownTestDatabase(t *testing.T, db *sql.DB, dbName string) {
	if db != nil {
		db.Close()
	}

	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
	adminDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("Cannot connect to admin DB for cleanup: %v", err)
		return
	}
	defer adminDB.Close()

	_, err = adminDB.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = $1
		AND pid <> pg_backend_pid() %s`, dbName))
	if err != nil {
		t.Logf("Warning: Could not terminate connections: %v", err)
	}

	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		t.Logf("Cannot drop test database: %v", err)
	}
}

func GetMigrationPath() string {
	return findMigrationPath()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
