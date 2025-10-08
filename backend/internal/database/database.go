// Package database provides database connection and migration functionality.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	// Import PostgreSQL driver for database/sql
	_ "github.com/lib/pq"

	// Add golang-migrate imports
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for golang-migrate postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for golang-migrate file source

	// OpenTelemetry SQL instrumentation
	"go.nhat.io/otelsql"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Manager handles database operations with proper logging
type Manager struct {
	logger *observability.Logger
}

var (
	otelDriverNameCache string
	otelDriverOnce      sync.Once
	otelDriverErr       error
)

// NewManager creates a new database manager with the provided logger
func NewManager(logger *observability.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// ErrTableAlreadyExists is returned when trying to create a table that already exists
var ErrTableAlreadyExists = errors.New("table already exists")

// DefaultDatabaseConfig returns the default database configuration
func DefaultDatabaseConfig() config.DatabaseConfig {
	config := config.DatabaseConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: config.DatabaseConnMaxLifetime,
	}

	// Check for TEST_DATABASE_URL first (for tests)
	if testURL := os.Getenv("TEST_DATABASE_URL"); testURL != "" {
		config.URL = testURL
	}

	return config
}

// InitDB initializes and returns a database connection with migrations
func (dm *Manager) InitDB(databaseURL string) (result0 *sql.DB, err error) {
	dbName := extractDatabaseName(databaseURL)
	_, span := observability.TraceDatabaseFunction(context.Background(), "InitDB",
		attribute.String("db.url", databaseURL),
		attribute.String("db.name", dbName),
		attribute.String("db.system", "postgresql"),
		attribute.Bool("migrations.enabled", true),
	)
	defer observability.FinishSpan(span, &err)
	config := DefaultDatabaseConfig()
	config.URL = databaseURL
	return dm.InitDBWithConfig(config)
}

// InitDBWithConfig initializes and returns a database connection with migrations and custom config
func (dm *Manager) InitDBWithConfig(config config.DatabaseConfig) (result0 *sql.DB, err error) {
	dbName := extractDatabaseName(config.URL)
	_, span := observability.TraceDatabaseFunction(context.Background(), "InitDBWithConfig",
		attribute.String("db.url", config.URL),
		attribute.String("db.name", dbName),
		attribute.String("db.system", "postgresql"),
		attribute.Bool("migrations.enabled", true),
		attribute.Int("db.max_open_conns", config.MaxOpenConns),
		attribute.Int("db.max_idle_conns", config.MaxIdleConns),
		attribute.String("db.conn_max_lifetime", config.ConnMaxLifetime.String()),
	)
	defer observability.FinishSpan(span, &err)
	db, err := dm.InitDBWithoutMigrations(config)
	if err != nil {
		return nil, err
	}

	if err := dm.RunMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

// extractDatabaseName extracts the database name from a PostgreSQL connection string
func extractDatabaseName(databaseURL string) string {
	// Try to parse as URL first
	if u, err := url.Parse(databaseURL); err == nil && u.Path != "" {
		// Remove leading slash and return the database name
		dbName := strings.TrimPrefix(u.Path, "/")
		if dbName != "" {
			return dbName
		}
	}

	// Fallback: try to extract from connection string format
	// postgres://user:pass@host:port/dbname?sslmode=disable
	if strings.Contains(databaseURL, "/") {
		parts := strings.Split(databaseURL, "/")
		if len(parts) > 1 {
			// Get the last part and remove query parameters
			dbPart := parts[len(parts)-1]
			if idx := strings.Index(dbPart, "?"); idx != -1 {
				return dbPart[:idx]
			}
			return dbPart
		}
	}

	// Default fallback
	return "quiz_db"
}

// InitDBWithoutMigrations initializes and returns a database connection without running migrations
func (dm *Manager) InitDBWithoutMigrations(config config.DatabaseConfig) (result0 *sql.DB, err error) {
	// Extract database name for OpenTelemetry tracing
	ctx, span := observability.TraceDatabaseFunction(context.Background(), "InitDBWithoutMigrations",
		attribute.String("database.url", config.URL),
	)
	defer observability.FinishSpan(span, &err)

	// Register OpenTelemetry SQL driver once per process and reuse the name
	otelDriverOnce.Do(func() {
		otelDriverNameCache, otelDriverErr = otelsql.Register("postgres",
			otelsql.WithDatabaseName(extractDatabaseName(config.URL)),
			otelsql.TraceQueryWithArgs(),
			otelsql.WithSystem(semconv.DBSystemPostgreSQL),
			otelsql.TraceRowsAffected(),
		)
	})
	if otelDriverErr != nil {
		return nil, contextutils.WrapError(otelDriverErr, "failed to register otelsql driver")
	}

	// Connect to database using the instrumented driver
	db, err := sql.Open(otelDriverNameCache, config.URL)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to open database connection")
	}

	// Set connection pool settings
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test the connection
	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			dm.logger.Error(ctx, "Failed to close database connection after ping failure", closeErr)
		}
		return nil, contextutils.WrapError(err, "failed to ping database")
	}

	dm.logger.Info(ctx, "Database connection established without migrations", map[string]interface{}{
		"max_open_conns":    config.MaxOpenConns,
		"max_idle_conns":    config.MaxIdleConns,
		"conn_max_lifetime": config.ConnMaxLifetime,
	})

	return db, nil
}

// RunMigrations executes the application SQL schema and any pending migrations
func (dm *Manager) RunMigrations(db *sql.DB) (err error) {
	_, span := observability.TraceDatabaseFunction(context.Background(), "RunMigrations",
		attribute.String("db.system", "postgresql"),
		attribute.String("migration.type", "application_schema"),
	)
	defer observability.FinishSpan(span, &err)
	dm.logger.Info(context.Background(), "Starting database migrations...")

	// Run the main application schema first
	if err := dm.runApplicationSchema(db); err != nil {
		return contextutils.WrapError(err, "failed to run application schema")
	}
	dm.logger.Info(context.Background(), "Application schema applied successfully")

	// Run golang-migrate migrations if directory exists
	if err := dm.runGolangMigrate(); err != nil {
		return contextutils.WrapError(err, "failed to run golang-migrate migrations")
	}

	dm.logger.Info(context.Background(), "Database migrations completed successfully")
	return nil
}

// runGolangMigrate runs migrations using golang-migrate from migrations
func (dm *Manager) runGolangMigrate() (err error) {
	migrationsPath, err := dm.GetMigrationsPath()
	if err != nil {
		dm.logger.Error(context.Background(), "Could not find migrations path", err)
		return err // HARD FAIL if migrations path is not set
	}

	_, span := observability.TraceDatabaseFunction(context.Background(), "runGolangMigrate",
		attribute.String("db.system", "postgresql"),
		attribute.String("migration.type", "golang_migrate"),
		attribute.String("migration.path", migrationsPath),
	)
	defer observability.FinishSpan(span, &err)

	if migrationsPath == "" {
		err = errors.New("no golang-migrate migrations directory found")
		dm.logger.Error(context.Background(), "No golang-migrate migrations directory found, hard fail!", err)
		return err // HARD FAIL
	}

	// Check if migrations directory exists and has migration files
	if _, statErr := os.Stat(migrationsPath); os.IsNotExist(statErr) {
		dm.logger.Error(context.Background(), "Migrations directory does not exist", statErr)
		err = statErr // HARD FAIL if directory does not exist
		return err
	}

	// Check if there are any migration files in the directory
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		dm.logger.Error(context.Background(), "Could not read migrations directory", err)
		return err // HARD FAIL
	}

	// Check if there are any .up.sql files
	hasMigrationFiles := false
	migrationFileCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			hasMigrationFiles = true
			migrationFileCount++
		}
	}

	span.SetAttributes(attribute.Int("migration.files.count", migrationFileCount))

	if !hasMigrationFiles {
		dm.logger.Info(context.Background(), fmt.Sprintf("No migration files found in %s. Skipping golang-migrate.", migrationsPath))
		return nil
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		err = errors.New("database_url or test_database_url must be set for migrations")
		return err
	}

	// Use file:// scheme with absolute path for golang-migrate
	// Convert to file:// URL format - use absolute path
	migrationSourceURL := "file://" + filepath.ToSlash(migrationsPath)

	// Debug logging
	dm.logger.Info(context.Background(), "Migration paths", map[string]interface{}{
		"migrations_path": migrationsPath,
		"source_url":      migrationSourceURL,
		"db_url":          dbURL,
	})

	m, err := migrate.New(
		migrationSourceURL,
		dbURL,
	)
	if err != nil {
		err = contextutils.WrapError(err, "failed to initialize golang-migrate")
		return err
	}
	defer func() {
		if _, closeErr := m.Close(); closeErr != nil {
			dm.logger.Error(context.Background(), "Error closing migration", closeErr)
		}
	}()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		err = contextutils.WrapError(err, "golang-migrate up failed")
		return err
	}
	if err == migrate.ErrNoChange {
		dm.logger.Info(context.Background(), "No new golang-migrate migrations to apply.")
	} else {
		dm.logger.Info(context.Background(), "golang-migrate migrations applied successfully.")
	}
	return nil
}

// runApplicationSchema executes the main application schema.sql
func (dm *Manager) runApplicationSchema(db *sql.DB) (err error) {
	schemaPath, err := dm.getSchemaPath()
	if err != nil {
		err = contextutils.WrapError(err, "failed to find schema file")
		return err
	}

	_, span := observability.TraceDatabaseFunction(context.Background(), "runApplicationSchema",
		attribute.String("db.system", "postgresql"),
		attribute.String("migration.type", "application_schema"),
		attribute.String("schema.path", schemaPath),
	)
	defer observability.FinishSpan(span, &err)
	// Get the schema file path relative to the project root
	schemaPath, err = dm.getSchemaPath()
	if err != nil {
		err = contextutils.WrapError(err, "failed to find schema file")
		return err
	}

	// Read the schema file
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		err = contextutils.WrapError(err, "failed to read schema file")
		return err
	}

	span.SetAttributes(attribute.Int("schema.file.size", len(schemaSQL)))

	// Parse SQL statements more carefully to handle comments and multi-line statements
	statements := dm.parseSchemaStatements(string(schemaSQL))

	span.SetAttributes(attribute.Int("schema.statements.count", len(statements)))

	// Execute table creation statements first
	var indexStatements []string
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		// Separate index creation from table creation
		if strings.HasPrefix(strings.ToUpper(statement), "CREATE INDEX") {
			indexStatements = append(indexStatements, statement)
			continue
		}

		_, execErr := db.Exec(statement)
		if execErr != nil {
			// For backwards compatibility, ignore table exists errors
			if !dm.isTableExistsError(execErr) {
				err = contextutils.WrapErrorf(execErr, "failed to execute schema statement: %s", statement)
				return err
			}
		}
	}

	span.SetAttributes(attribute.Int("schema.index_statements.count", len(indexStatements)))

	// Now execute index creation statements
	for _, statement := range indexStatements {
		_, execErr := db.Exec(statement)
		if execErr != nil {
			// For backwards compatibility, ignore index exists and column exists errors
			if !dm.isTableExistsError(execErr) && !dm.isColumnExistsError(execErr) {
				err = contextutils.WrapErrorf(execErr, "failed to execute index statement: %s", statement)
				return err
			}
		}
	}

	return nil
}

// getSchemaPath finds the schema.sql file relative to the project root
func (dm *Manager) getSchemaPath() (result0 string, err error) {
	_, span := observability.TraceDatabaseFunction(context.Background(), "getSchemaPath",
		attribute.String("file.name", "schema.sql"),
	)
	defer observability.FinishSpan(span, &err)
	// Start from the current directory and work up to find schema.sql
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	span.SetAttributes(attribute.String("search.start_dir", currentDir))

	for {
		schemaPath := filepath.Join(currentDir, "schema.sql")
		if _, statErr := os.Stat(schemaPath); statErr == nil {
			span.SetAttributes(attribute.String("schema.found_path", schemaPath))
			return schemaPath, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We've reached the root directory
			span.SetAttributes(attribute.String("search.result", "not_found"))
			err = contextutils.ErrorWithContextf("schema.sql not found in any parent directory")
			return "", err
		}
		currentDir = parentDir
	}
}

// parseSchemaStatements parses SQL statements from a schema file
func (dm *Manager) parseSchemaStatements(schemaSQL string) []string {
	_, span := observability.TraceDatabaseFunction(context.Background(), "parseSchemaStatements",
		attribute.Int("input.length", len(schemaSQL)),
	)
	defer span.End()

	// Remove comments and normalize whitespace
	lines := strings.Split(schemaSQL, "\n")
	var cleanedLines []string
	inComment := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Handle multi-line comments
		if strings.HasPrefix(line, "/*") {
			inComment = true
			continue
		}
		if strings.HasSuffix(line, "*/") {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		// Skip single-line comments
		if strings.HasPrefix(line, "--") {
			continue
		}

		// Remove inline comments (comments that appear after SQL code)
		if commentIndex := strings.Index(line, "--"); commentIndex != -1 {
			line = strings.TrimSpace(line[:commentIndex])
		}

		cleanedLines = append(cleanedLines, line)
	}

	// Join lines and split by semicolon
	cleanedSQL := strings.Join(cleanedLines, " ")
	statements := strings.Split(cleanedSQL, ";")

	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	span.SetAttributes(attribute.Int("statements.parsed", len(result)))
	return result
}

// isTableExistsError checks if the error is due to a table already existing
func (dm *Manager) isTableExistsError(err error) bool {
	_, span := observability.TraceDatabaseFunction(context.Background(), "isTableExistsError")
	defer span.End()
	// Check for the sentinel error first
	if errors.Is(err, ErrTableAlreadyExists) {
		return true
	}
	// Fallback to string matching for backwards compatibility
	return strings.Contains(err.Error(), "already exists")
}

// isColumnExistsError checks if the error is due to a column not existing (for index creation)
func (dm *Manager) isColumnExistsError(err error) bool {
	_, span := observability.TraceDatabaseFunction(context.Background(), "isColumnExistsError")
	defer span.End()
	return strings.Contains(err.Error(), "column") && strings.Contains(err.Error(), "does not exist")
}

// GetMigrationsPath returns the path to the migrations directory
func (dm *Manager) GetMigrationsPath() (result0 string, err error) {
	_, span := observability.TraceDatabaseFunction(context.Background(), "GetMigrationsPath",
		attribute.String("migration.dir.name", "migrations"),
	)
	defer observability.FinishSpan(span, &err)
	// Start from the current directory and work up to find migrations directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	span.SetAttributes(attribute.String("search.start_dir", currentDir))

	for {
		migrationsPath := filepath.Join(currentDir, "migrations")
		if _, statErr := os.Stat(migrationsPath); statErr == nil {
			span.SetAttributes(attribute.String("migration.found_path", migrationsPath))
			return migrationsPath, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We've reached the root directory
			span.SetAttributes(attribute.String("search.result", "not_found"))
			err = contextutils.ErrorWithContextf("migrations directory not found in any parent directory")
			return "", err
		}
		currentDir = parentDir
	}
}
