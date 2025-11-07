// Package commands provides CLI commands for the admin tool
package commands

import (
	"context"
	"database/sql"
	"os"

	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/spf13/cobra"
)

// DatabaseCommands returns the database management commands
func DatabaseCommands(userService *services.UserService, logger *observability.Logger, db *sql.DB) *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long: `Database management commands for the quiz application.

Available commands:
  stats     - Show database statistics
  cleanup   - Run database cleanup operations`,
	}

	// Add subcommands
	dbCmd.AddCommand(statsCmd(userService, logger, db))
	dbCmd.AddCommand(cleanupCmd(logger, db))

	return dbCmd
}

// statsCmd returns the stats command
func statsCmd(userService *services.UserService, logger *observability.Logger, db *sql.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show database statistics",
		Long:  `Show database statistics including user counts and other metrics.`,
		RunE:  runStats(userService, logger, db),
	}
}

// cleanupCmd returns the cleanup command
func cleanupCmd(logger *observability.Logger, db *sql.DB) *cobra.Command {
	var statsOnly bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Run database cleanup operations",
		Long: `Run database cleanup operations to remove old data.

This command will:
- Remove questions with legacy question types
- Remove orphaned user responses

Use --stats flag to see what would be cleaned up without actually performing the cleanup.`,
		RunE: runCleanup(logger, &statsOnly, db),
	}

	// Add flags
	cmd.Flags().BoolVar(&statsOnly, "stats", false, "Only show cleanup statistics, don't perform cleanup")

	return cmd
}

// runStats returns a function that shows database statistics
func runStats(userService *services.UserService, logger *observability.Logger, db *sql.DB) func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()

		// Log diagnostic information
		logger.Info(ctx, "Diagnostic info", map[string]interface{}{"config_file": os.Getenv("QUIZ_CONFIG_FILE"), "database": getDatabaseInfo(db)})

		logger.Info(ctx, "Showing database statistics", map[string]interface{}{})

		// Get user statistics
		users, err := userService.GetAllUsers(ctx)
		if err != nil {
			logger.Error(ctx, "Failed to get user statistics", err, map[string]interface{}{})
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to get user statistics: %v", err)
		}

		logger.Info(ctx, "Database statistics", map[string]interface{}{"total_users": len(users), "database": "PostgreSQL", "status": "Connected"})

		return nil
	}
}

// runCleanup returns a function that runs database cleanup
func runCleanup(logger *observability.Logger, statsOnly *bool, db *sql.DB) func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()

		// Log diagnostic information
		logger.Info(ctx, "Diagnostic info", map[string]interface{}{"config_file": os.Getenv("QUIZ_CONFIG_FILE"), "database": getDatabaseInfo(db)})

		logger.Info(ctx, "Running database cleanup", map[string]interface{}{"stats_only": *statsOnly})

		// Use the database connection passed as parameter
		if db == nil {
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "database connection not available")
		}

		// Initialize cleanup service
		cleanupService := services.NewCleanupServiceWithLogger(db, logger)

		if *statsOnly {
			// Show cleanup statistics only
			stats, err := cleanupService.GetCleanupStats(ctx)
			if err != nil {
				logger.Error(ctx, "Failed to get cleanup stats", err, map[string]interface{}{"stats_only": true})
				return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to get cleanup stats: %v", err)
			}

			logger.Info(ctx, "Database cleanup statistics", map[string]interface{}{"legacy_questions": stats["legacy_questions"], "orphaned_responses": stats["orphaned_responses"]})

			total := stats["legacy_questions"] + stats["orphaned_responses"]
			if total == 0 {
				logger.Info(ctx, "No cleanup needed - database is clean", map[string]interface{}{"total": total})
			} else {
				logger.Info(ctx, "Cleanup would remove items", map[string]interface{}{"total": total})
			}
			return nil
		}

		// Run full cleanup
		logger.Info(ctx, "Starting database cleanup", map[string]interface{}{"service": "cleanup"})

		if err := cleanupService.RunFullCleanup(ctx); err != nil {
			logger.Error(ctx, "Cleanup failed", err, map[string]interface{}{"service": "cleanup"})
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "cleanup failed: %v", err)
		}

		logger.Info(ctx, "Database cleanup completed successfully", map[string]interface{}{"service": "cleanup"})
		return nil
	}
}
