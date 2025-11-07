package commands

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"

	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/spf13/cobra"
)

// UserCommands returns the user management commands
func UserCommands(userService *services.UserService, logger *observability.Logger, databaseURL string) *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
		Long: `User management commands for the quiz application.

Available commands:
  list     - List all users
  reset-password - Reset password for a specific user`,
	}

	// Add subcommands
	userCmd.AddCommand(listCmd(userService, logger, databaseURL))
	userCmd.AddCommand(resetPasswordCmd(userService, logger))

	return userCmd
}

// listCmd returns the list command
func listCmd(userService *services.UserService, logger *observability.Logger, databaseURL string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Long:  `List all users in the database with their basic information.`,
		RunE:  runListUsers(userService, logger, databaseURL),
	}
}

// resetPasswordCmd returns the reset-password command
func resetPasswordCmd(userService *services.UserService, logger *observability.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password [username]",
		Short: "Reset password for a user",
		Long:  `Reset the password for a specific user. If username is not provided, you will be prompted for it.`,
		RunE:  runResetPassword(userService, logger),
	}
}

// runListUsers returns a function that lists all users
func runListUsers(userService *services.UserService, logger *observability.Logger, databaseURL string) func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()

		// Show diagnostic information
		logger.Info(ctx, "Admin command diagnostics", map[string]interface{}{"config_file": os.Getenv("QUIZ_CONFIG_FILE"), "database_url": maskDatabaseURL(databaseURL)})

		logger.Info(ctx, "Listing all users", map[string]interface{}{})

		users, err := userService.GetAllUsers(ctx)
		if err != nil {
			logger.Error(ctx, "Failed to get users", err, map[string]interface{}{})
			return contextutils.WrapError(err, "failed to get users")
		}

		if len(users) == 0 {
			logger.Info(ctx, "No users found in the database", nil)
			return nil
		}

		// Print header to stdout (user-facing table)
		fmt.Printf("%-5s %-20s %-30s %-15s %-10s %-10s %-10s\n", "ID", "Username", "Email", "Language", "Level", "AI Enabled", "Created")
		fmt.Println(string(make([]byte, 120))) // Print 120 dashes

		// Print each user
		for _, user := range users {
			aiEnabled := "No"
			if user.AIEnabled.Valid && user.AIEnabled.Bool {
				aiEnabled = "Yes"
			}

			email := "N/A"
			if user.Email.Valid {
				email = user.Email.String
			}

			language := "N/A"
			if user.PreferredLanguage.Valid {
				language = user.PreferredLanguage.String
			}

			level := "N/A"
			if user.CurrentLevel.Valid {
				level = user.CurrentLevel.String
			}

			fmt.Printf("%-5d %-20s %-30s %-15s %-10s %-10s %-10s\n",
				user.ID,
				user.Username,
				email,
				language,
				level,
				aiEnabled,
				user.CreatedAt.Format("2006-01-02"),
			)
		}

		logger.Info(ctx, "Listed users", map[string]interface{}{"total": len(users)})
		return nil
	}
}

// runResetPassword returns a function that resets a user's password
func runResetPassword(userService *services.UserService, logger *observability.Logger) func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, args []string) error {
		ctx := context.Background()

		var username string
		var newPassword string

		// Get username from args or prompt
		if len(args) > 0 {
			username = args[0]
		} else {
			fmt.Print("Enter username: ")
			if _, err := fmt.Scanln(&username); err != nil {
				return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to read username: %v", err)
			}
		}

		if username == "" {
			return contextutils.ErrorWithContextf("username is required")
		}

		// Prompt for password securely
		fmt.Print("Enter new password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to read password: %v", err)
		}
		newPassword = string(passwordBytes)
		fmt.Println() // New line after password input

		if newPassword == "" {
			return contextutils.ErrorWithContextf("password cannot be empty")
		}

		// Confirm password
		fmt.Print("Confirm new password: ")
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to read password confirmation: %v", err)
		}
		confirmPassword := string(confirmBytes)
		fmt.Println() // New line after password input

		if newPassword != confirmPassword {
			return contextutils.ErrorWithContextf("passwords do not match")
		}

		logger.Info(ctx, "Resetting password for user", map[string]interface{}{
			"username": username,
		})

		// Get user by username
		user, err := userService.GetUserByUsername(ctx, username)
		if err != nil {
			logger.Error(ctx, "Failed to get user", err, map[string]interface{}{"username": username})
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to get user '%s': %v", username, err)
		}

		if user == nil {
			logger.Error(ctx, "User not found", nil, map[string]interface{}{"username": username})
			return contextutils.ErrorWithContextf("user '%s' not found", username)
		}

		// Update the password
		err = userService.UpdateUserPassword(ctx, user.ID, newPassword)
		if err != nil {
			logger.Error(ctx, "Failed to update password", err, map[string]interface{}{
				"username": username,
				"user_id":  user.ID,
			})
			return contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to update password for user '%s': %v", username, err)
		}

		fmt.Printf("✅ Password successfully reset for user '%s' (ID: %d)\n", username, user.ID)
		logger.Info(ctx, "Password reset successful", map[string]interface{}{
			"username": username,
			"user_id":  user.ID,
		})

		return nil
	}
}
