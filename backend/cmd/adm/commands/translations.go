// Package commands provides CLI commands for the admin tool
package commands

import (
	"context"
	"database/sql"
	"fmt"

	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/spf13/cobra"
)

// TranslationCommands returns the translation management commands
func TranslationCommands(logger *observability.Logger, db *sql.DB) *cobra.Command {
	translationCmd := &cobra.Command{
		Use:   "translation",
		Short: "Translation cache management commands",
		Long: `Translation cache management commands for the quiz application.

Available commands:
  cleanup   - Remove expired translation cache entries`,
	}

	// Add subcommands
	translationCmd.AddCommand(translationCleanupCmd(logger, db))

	return translationCmd
}

// translationCleanupCmd returns the cleanup command for translation cache
func translationCleanupCmd(logger *observability.Logger, db *sql.DB) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove expired translation cache entries",
		Long: `Remove expired translation cache entries from the database.

This command will:
- Delete all translation cache entries that have expired (older than 30 days)
- Report the number of entries deleted

Use --dry-run flag to see what would be cleaned up without actually performing the cleanup.`,
		RunE: runTranslationCleanup(logger, &dryRun, db),
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned up without actually performing the cleanup")

	return cmd
}

// runTranslationCleanup executes the translation cache cleanup
func runTranslationCleanup(logger *observability.Logger, dryRun *bool, db *sql.DB) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		cacheRepo := services.NewTranslationCacheRepository(db, logger)

		if *dryRun {
			// Count expired entries without deleting
			var count int64
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_cache WHERE expires_at < NOW()").Scan(&count)
			if err != nil {
				logger.Error(ctx, "Failed to count expired translation cache entries", err)
				return contextutils.WrapError(err, "failed to count expired entries")
			}

			fmt.Printf("Dry run: Would delete %d expired translation cache entries\n", count)
			return nil
		}

		// Perform actual cleanup
		count, err := cacheRepo.CleanupExpiredTranslations(ctx)
		if err != nil {
			logger.Error(ctx, "Failed to cleanup expired translation cache entries", err)
			return contextutils.WrapError(err, "failed to cleanup expired entries")
		}

		fmt.Printf("Successfully deleted %d expired translation cache entries\n", count)
		logger.Info(ctx, "Translation cache cleanup completed", map[string]interface{}{
			"deleted_count": count,
		})

		return nil
	}
}
