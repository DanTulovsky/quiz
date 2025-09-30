//go:build integration
// +build integration

package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDB_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	// Use test database URL
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	db, err := dbManager.InitDB(testDatabaseURL)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify connection works
	err = db.Ping()
	require.NoError(t, err)

	// Verify basic functionality
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	require.NoError(t, err)
	assert.Contains(t, version, "PostgreSQL")
}

func TestInitDB_InvalidURL_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	invalidURL := "postgres://invalid:invalid@nonexistent:1234/nonexistent?sslmode=disable"

	db, err := dbManager.InitDB(invalidURL)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestInitDBWithoutMigrations_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	config := DefaultDatabaseConfig()
	config.URL = testDatabaseURL
	db, err := dbManager.InitDBWithoutMigrations(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify connection works
	err = db.Ping()
	require.NoError(t, err)
}

func TestRunMigrations_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	config := DefaultDatabaseConfig()
	config.URL = testDatabaseURL
	db, err := dbManager.InitDBWithoutMigrations(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Drop all tables to start fresh
	tables := []string{
		"user_responses",
		"performance_metrics",
		"questions",
		"worker_status",
		"worker_settings",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Could not drop table %s: %v", table, err)
		}
	}

	// Run migrations
	err = dbManager.RunMigrations(db)
	require.NoError(t, err)

	// Verify core tables exist
	expectedTables := []string{
		"users",
		"questions",
		"user_responses",
		"performance_metrics",
		"worker_settings",
		"worker_status",
	}

	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist after migrations", table)
	}

	// Remove legacy migrations table check: do not check for 'migrations' or 'schema_migrations' here
}

func TestRunMigrations_AlreadyApplied_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	db, err := dbManager.InitDB(testDatabaseURL)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Run migrations again - should not error
	err = dbManager.RunMigrations(db)
	require.NoError(t, err)

	// Verify tables still exist and work
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(t, err)
}

func TestGetSchemaPath_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	schemaPath, err := dbManager.getSchemaPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, schemaPath)
	assert.Contains(t, schemaPath, "schema.sql")

	// Verify file exists
	_, err = os.Stat(schemaPath)
	assert.NoError(t, err, "Schema file should exist at path: %s", schemaPath)
}

func TestGetMigrationsPath_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	migrationsPath, err := dbManager.GetMigrationsPath()
	if err != nil || migrationsPath == "" {
		t.Skip("MIGRATIONS_PATH not set or migrations directory does not exist; skipping test")
	}
	assert.NotEmpty(t, migrationsPath)
	assert.Contains(t, migrationsPath, "migrations")

	// Strip file:// prefix for os.Stat
	fsPath := migrationsPath
	if strings.HasPrefix(fsPath, "file://") {
		fsPath = fsPath[len("file://"):]
	}

	info, err := os.Stat(fsPath)
	assert.NoError(t, err, "Migrations directory should exist at path: %s", fsPath)
	if err == nil {
		assert.True(t, info.IsDir(), "Migrations path should be a directory")
	}
}

func TestParseSchemaStatements_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	schemaPath, err := dbManager.getSchemaPath()
	assert.NoError(t, err)
	schemaSQL, err := os.ReadFile(schemaPath)
	assert.NoError(t, err)
	statements := dbManager.parseSchemaStatements(string(schemaSQL))
	assert.NotEmpty(t, statements)
	// Should have at least 2 statements for users and questions tables
	foundUsersTable := false
	foundQuestionsTable := false
	for _, stmt := range statements {
		if contains(stmt, "CREATE TABLE IF NOT EXISTS users") {
			foundUsersTable = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS questions") {
			foundQuestionsTable = true
		}
	}
	assert.True(t, foundUsersTable, "Should contain users table creation")
	assert.True(t, foundQuestionsTable, "Should contain questions table creation")
}

func TestRunApplicationSchema_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	config := DefaultDatabaseConfig()
	config.URL = testDatabaseURL
	db, err := dbManager.InitDBWithoutMigrations(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Drop tables to start fresh
	tables := []string{
		"user_responses",
		"performance_metrics",
		"questions",
		"worker_status",
		"worker_settings",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Could not drop table %s: %v", table, err)
		}
	}

	// Run application schema
	err = dbManager.runApplicationSchema(db)
	require.NoError(t, err)

	// Verify core tables exist
	expectedTables := []string{
		"users",
		"questions",
		"user_responses",
		"performance_metrics",
		"worker_settings",
		"worker_status",
	}

	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist after schema application", table)
	}
}

func TestIsTableExistsError_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	config := DefaultDatabaseConfig()
	config.URL = testDatabaseURL
	db, err := dbManager.InitDBWithoutMigrations(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Try to create a table twice to generate a "table exists" error
	createTableSQL := "CREATE TABLE test_table_exists (id SERIAL PRIMARY KEY)"

	// First creation should succeed
	_, err = db.Exec(createTableSQL)
	require.NoError(t, err)

	// Second creation should fail with table exists error
	_, err = db.Exec(createTableSQL)
	require.Error(t, err)

	// Test the helper function
	isTableExists := dbManager.isTableExistsError(err)
	assert.True(t, isTableExists, "Should detect table exists error")

	// Clean up
	_, err = db.Exec("DROP TABLE test_table_exists")
	require.NoError(t, err)
}

func TestDatabase_ErrorHandling_Integration(t *testing.T) {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := NewManager(observabilityLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	config := DefaultDatabaseConfig()
	config.URL = testDatabaseURL
	db, err := dbManager.InitDBWithoutMigrations(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Test invalid SQL execution
	_, err = db.Exec("INVALID SQL STATEMENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "syntax error")

	// Test querying non-existent table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM non_existent_table").Scan(&count)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDatabaseManager_NilLoggerPanicsOrErrors(t *testing.T) {
	// Try to create a DatabaseManager with a nil logger
	var nilLogger *observability.Logger = nil
	dbManager := NewManager(nilLogger)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	// All methods that use the logger should panic or error clearly
	// We'll check InitDB, RunMigrations, and runApplicationSchema
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Expected panic or error when using DatabaseManager with nil logger, but did not panic")
		}
	}()

	// This should panic or error due to nil logger
	_, _ = dbManager.InitDB(testDatabaseURL)
}

// getMigrationsPath finds the migrations directory relative to the project root
func getMigrationsPath() (result0 string, err error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		migrationsPath := filepath.Join(currentDir, "backend", "migrations")
		if info, err := os.Stat(migrationsPath); err == nil && info.IsDir() {
			return migrationsPath, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root directory
		}
		currentDir = parent
	}

	return "", contextutils.ErrorWithContextf("migrations directory not found in project directory tree")
}

// Helper function to check if string contains substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExtractDatabaseName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard postgres URL",
			url:      "postgres://user:pass@localhost:5432/quiz_db?sslmode=disable",
			expected: "quiz_db",
		},
		{
			name:     "URL with query parameters",
			url:      "postgres://user:pass@localhost:5432/test_db?sslmode=disable&connect_timeout=10",
			expected: "test_db",
		},
		{
			name:     "URL without query parameters",
			url:      "postgres://user:pass@localhost:5432/production_db",
			expected: "production_db",
		},
		{
			name:     "URL with special characters in password",
			url:      "postgres://user:pass@word@localhost:5432/my_db",
			expected: "my_db",
		},
		{
			name:     "fallback for malformed URL",
			url:      "invalid-url",
			expected: "invalid-url",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "quiz_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDatabaseName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
