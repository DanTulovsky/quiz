// Package main provides the main entry point for the quiz application admin CLI tool.
package main

import (
	"context"
	"fmt"
	"os"

	"quizapp/cmd/adm/commands"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/spf13/cobra"
)

// Global variables for shared resources
var (
	cfg         *config.Config
	logger      *observability.Logger
	userService *services.UserService
)

func main() {
	ctx := context.Background()

	// Set default config file if not already set
	if os.Getenv("QUIZ_CONFIG_FILE") == "" {
		// Try to find the config file in common locations
		defaultPaths := []string{
			"../merged.config.yaml",    // From backend/cmd/adm/
			"../../merged.config.yaml", // From backend/cmd/adm/ (alternative)
			"merged.config.yaml",       // Current directory
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				if err := os.Setenv("QUIZ_CONFIG_FILE", path); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to set QUIZ_CONFIG_FILE environment variable: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
	}

	// Load configuration
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log level for admin tool
	cfg.Server.LogLevel = "error"

	// Disable all OpenTelemetry features for admin CLI to avoid connection errors
	cfg.OpenTelemetry.EnableTracing = false
	cfg.OpenTelemetry.EnableMetrics = false
	cfg.OpenTelemetry.EnableLogging = false

	// Setup observability (tracing/metrics/logging)
	tp, mp, loggerInstance, err := observability.SetupObservability(&cfg.OpenTelemetry, "quiz-admin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	// Store logger globally
	logger = loggerInstance

	// Defer cleanup
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

	// Initialize database manager
	dbManager := database.NewManager(logger)

	// Initialize database connection with configuration (no migrations for admin tool)
	db, err := dbManager.InitDBWithoutMigrations(cfg.Database)
	if err != nil {
		logger.Error(ctx, "Failed to connect to database", err, map[string]interface{}{"db_url": cfg.Database.URL})
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close database connection", map[string]interface{}{"error": err.Error(), "db_url": cfg.Database.URL})
		}
	}()

	// Initialize services
	userService = services.NewUserServiceWithLogger(db, cfg, logger)

	// Create the root command
	rootCmd := &cobra.Command{
		Use:   "adm",
		Short: "Quiz Application Administration Tool",
		Long: `Quiz Application Administration Tool

A comprehensive CLI tool for administering the quiz application.
Provides commands for user management, database operations, and system administration.`,

		Run: func(cmd *cobra.Command, _ []string) {
			// Show help if no subcommand provided
			if err := cmd.Help(); err != nil {
				fmt.Printf("Error showing help: %v\n", err)
			}
		},
	}

	// Add subcommands with initialized services
	rootCmd.AddCommand(commands.UserCommands(userService, logger, cfg.Database.URL))
	rootCmd.AddCommand(commands.DatabaseCommands(userService, logger, db))
	rootCmd.AddCommand(commands.TranslationCommands(logger, db))

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
