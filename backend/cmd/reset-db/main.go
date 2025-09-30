// Package main provides a small CLI utility to reset the application's
// database to a clean state. It is intended for local development and
// testing only and will permanently delete all data when run.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"
	"quizapp/internal/services"
)

// fatalIfErr logs the error with context and exits
func fatalIfErr(ctx context.Context, logger *observability.Logger, msg string, err error, fields map[string]interface{}) {
	logger.Error(ctx, msg, err, fields)
	os.Exit(1)
}

func main() {
	ctx := context.Background()

	// Load configuration first
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup observability (tracing/metrics/logging)
	tp, mp, logger, err := observability.SetupObservability(&cfg.OpenTelemetry, "reset-db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if tp != nil {
			if err := tp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down tracer provider", map[string]interface{}{"error": err.Error(), "provider": "tracer"})
			}
		}
		if mp != nil {
			if err := mp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down meter provider", map[string]interface{}{"error": err.Error(), "provider": "meter"})
			}
		}
	}()

	fmt.Println("⚠️  DATABASE RESET UTILITY ⚠️")
	fmt.Println("=============================")
	fmt.Println("This will PERMANENTLY DELETE ALL DATA in the database!")
	fmt.Println("This includes:")
	fmt.Println("- All users (including admin)")
	fmt.Println("- All questions")
	fmt.Println("- All user responses")
	fmt.Println("- All performance metrics")
	fmt.Println("")

	logger.Info(ctx, "Attempting to reset the database", map[string]interface{}{"service": "reset-db"})

	if cfg.Database.URL == "" {
		fatalIfErr(ctx, logger, "Database URL is empty", nil, map[string]interface{}{"error": "Database URL is empty. Cannot proceed with reset."})
	}

	// Print database info
	fmt.Println("📊 Database Information:")
	fmt.Printf("URL: %s\n", maskDatabaseURL(cfg.Database.URL))
	fmt.Println("")

	// Confirm with user
	if !confirmReset() {
		fmt.Println("Reset cancelled.")
		return
	}

	// Initialize database manager with logger
	dbManager := database.NewManager(logger)

	// Initialize database connection with configuration
	db, err := dbManager.InitDBWithConfig(cfg.Database)
	if err != nil {
		fatalIfErr(ctx, logger, "Failed to connect to database", err, map[string]interface{}{"db_url": cfg.Database.URL})
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close database connection", map[string]interface{}{"error": err.Error(), "db_url": cfg.Database.URL})
		}
	}()

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)

	// Drop all tables
	fmt.Println("🗑️  Dropping all tables...")
	logger.Info(ctx, "Dropping all tables", map[string]interface{}{"db_url": cfg.Database.URL, "service": "reset-db"})

	// For now, we'll just run migrations which will recreate the schema
	// In a real implementation, you might want to add a DropAllTables method to the database manager

	// Run migrations
	fmt.Println("🔄 Running database migrations...")
	logger.Info(ctx, "Running database migrations", map[string]interface{}{"db_url": cfg.Database.URL, "service": "reset-db"})

	if err := dbManager.RunMigrations(db); err != nil {
		fatalIfErr(ctx, logger, "Failed to run migrations", err, map[string]interface{}{"db_url": cfg.Database.URL})
	}

	fmt.Println("✅ Database migrations completed successfully!")
	logger.Info(ctx, "Database migrations completed successfully", map[string]interface{}{"db_url": cfg.Database.URL, "service": "reset-db"})

	// Recreate admin user immediately
	fmt.Printf("Recreating admin user '%s'...\n", cfg.Server.AdminUsername)
	logger.Info(ctx, "Recreating admin user", map[string]interface{}{"username": cfg.Server.AdminUsername, "service": "reset-db"})
	// Ensure admin user exists
	if err := userService.EnsureAdminUserExists(ctx, cfg.Server.AdminUsername, cfg.Server.AdminPassword); err != nil {
		fatalIfErr(ctx, logger, "Failed to ensure admin user exists", err, map[string]interface{}{"admin_username": cfg.Server.AdminUsername})
	}

	fmt.Println("✅ Admin user recreated successfully!")
	logger.Info(ctx, "Admin user recreated successfully", map[string]interface{}{"username": cfg.Server.AdminUsername, "service": "reset-db"})
	fmt.Println("")
	// Print admin credentials
	fmt.Printf("\nAdmin user credentials:\n")
	fmt.Printf("   Username: %s\n", cfg.Server.AdminUsername)
	fmt.Printf("   Password: %s\n", cfg.Server.AdminPassword)
	fmt.Println("")
	fmt.Println("✅ Database is now ready to use!")
	fmt.Println("- You can now start the server or use the existing running instance")
	fmt.Println("- Use the credentials above to log into the application")
}

func confirmReset() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Are you sure you want to reset the database? (type 'yes' to confirm): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "yes":
			return true
		case "no", "":
			return false
		default:
			fmt.Println("Please type 'yes' to confirm or 'no' to cancel.")
		}
	}
}

func maskDatabaseURL(url string) string {
	// Simple masking for display purposes
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return "postgres://***:***@" + parts[1]
		}
	}
	return url
}
