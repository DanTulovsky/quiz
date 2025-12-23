package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/lib/pq"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

// UserServiceInterface defines the interface for user-related operations.
// This allows for easier mocking in tests.
type UserServiceInterface interface {
	CreateUserWithPassword(ctx context.Context, username, password, language, level string) (*models.User, error)
	CreateUserWithEmailAndTimezone(ctx context.Context, username, email, timezone, language, level string) (*models.User, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	AuthenticateUser(ctx context.Context, username, password string) (*models.User, error)
	UpdateUserSettings(ctx context.Context, userID int, settings *models.UserSettings) error
	UpdateUserProfile(ctx context.Context, userID int, username, email, timezone string) error
	UpdateUserPassword(ctx context.Context, userID int, newPassword string) error
	UpdateLastActive(ctx context.Context, userID int) error
	GetAllUsers(ctx context.Context) ([]models.User, error)
	GetUsersPaginated(ctx context.Context, page, pageSize int, search, language, level, aiProvider, aiModel, aiEnabled, active string) ([]models.User, int, error)
	DeleteUser(ctx context.Context, userID int) error
	DeleteAllUsers(ctx context.Context) error
	EnsureAdminUserExists(ctx context.Context, adminUsername, adminPassword string) error
	ResetDatabase(ctx context.Context) error
	ClearUserData(ctx context.Context) error
	ClearUserDataForUser(ctx context.Context, userID int) error
	GetUserAPIKey(ctx context.Context, userID int, provider string) (string, error)
	GetUserAPIKeyWithID(ctx context.Context, userID int, provider string) (string, *int, error)
	SetUserAPIKey(ctx context.Context, userID int, provider, apiKey string) error
	HasUserAPIKey(ctx context.Context, userID int, provider string) (bool, error)
	// Role management methods
	GetUserRoles(ctx context.Context, userID int) ([]models.Role, error)
	GetAllRoles(ctx context.Context) ([]models.Role, error)
	AssignRole(ctx context.Context, userID, roleID int) error
	AssignRoleByName(ctx context.Context, userID int, roleName string) error
	RemoveRole(ctx context.Context, userID, roleID int) error
	HasRole(ctx context.Context, userID int, roleName string) (bool, error)
	IsAdmin(ctx context.Context, userID int) (bool, error)
	GetDB() *sql.DB
	UpdateWordOfDayEmailEnabled(ctx context.Context, userID int, enabled bool) error
	// Device token management methods
	RegisterDeviceToken(ctx context.Context, userID int, deviceToken string) error
	GetUserDeviceTokens(ctx context.Context, userID int) ([]string, error)
	RemoveDeviceToken(ctx context.Context, userID int, deviceToken string) error
}

// UserService provides methods for user management.
type UserService struct {
	db     *sql.DB
	cfg    *config.Config
	logger *observability.Logger
}

// Shared query constants to eliminate duplication
const (
	// userSelectFields contains all user fields for SELECT queries
	userSelectFields = `id, username, email, timezone, password_hash, last_active, preferred_language, current_level, ai_provider, ai_model, ai_enabled, ai_api_key, word_of_day_email_enabled, created_at, updated_at`

	// userSelectFieldsNoPassword contains user fields excluding password_hash for GetAllUsers
	userSelectFieldsNoPassword = `id, username, email, timezone, last_active, preferred_language, current_level, ai_provider, ai_model, ai_enabled, ai_api_key, word_of_day_email_enabled, created_at, updated_at`
)

// scanUserFromRow scans a database row into a models.User struct
func (s *UserService) scanUserFromRow(row *sql.Row) (result0 *models.User, err error) {
	user := &models.User{}
	err = row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Timezone, &user.PasswordHash, &user.LastActive,
		&user.PreferredLanguage, &user.CurrentLevel, &user.AIProvider,
		&user.AIModel, &user.AIEnabled, &user.AIAPIKey, &user.WordOfDayEmailEnabled, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// scanUserFromRowsNoPassword scans a database rows into a models.User struct (without password_hash)
func (s *UserService) scanUserFromRowsNoPassword(rows *sql.Rows) (result0 *models.User, err error) {
	user := &models.User{}
	err = rows.Scan(
		&user.ID, &user.Username, &user.Email, &user.Timezone, &user.LastActive,
		&user.PreferredLanguage, &user.CurrentLevel, &user.AIProvider,
		&user.AIModel, &user.AIEnabled, &user.AIAPIKey, &user.WordOfDayEmailEnabled, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// getUserByQuery is a shared method for getting a user by any query
func (s *UserService) getUserByQuery(ctx context.Context, query string, args ...interface{}) (result0 *models.User, err error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	var user *models.User
	user, err = s.scanUserFromRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found is not an error here
		}
		return nil, err
	}

	// Try to apply default settings, but don't fail if there's an issue
	s.applyDefaultSettings(ctx, user)
	return user, nil
}

// NewUserServiceWithLogger creates a new UserService instance with logger
func NewUserServiceWithLogger(db *sql.DB, cfg *config.Config, logger *observability.Logger) *UserService {
	return &UserService{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}
}

// CreateUser creates a new user with the specified username, language, and level
// Only used for testing purposes, should be moved to test utils if possible.
func (s *UserService) CreateUser(ctx context.Context, username, language, level string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "create_user", attribute.String("user.username", username))
	defer observability.FinishSpan(span, &err)

	// Validate username is not empty
	if username == "" || len(strings.TrimSpace(username)) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrInvalidInput, "username cannot be empty")
	}

	// default timezone to UTC for new users
	query := `INSERT INTO users (username, preferred_language, current_level, last_active, created_at, updated_at, timezone) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	now := time.Now()
	var id int
	err = s.db.QueryRowContext(ctx, query, username, language, level, now, now, now, "UTC").Scan(&id)
	if err != nil {
		return nil, err
	}
	var user *models.User
	user, err = s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, contextutils.WrapError(contextutils.ErrDatabaseQuery, "user was created but could not be retrieved from database")
	}
	return user, nil
}

// CreateUserWithEmailAndTimezone creates a new user with email and timezone
func (s *UserService) CreateUserWithEmailAndTimezone(ctx context.Context, username, email, timezone, language, level string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "create_user_with_email", attribute.String("user.username", username))
	defer observability.FinishSpan(span, &err)

	// Validate username is not empty
	if username == "" || len(strings.TrimSpace(username)) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrInvalidInput, "username cannot be empty")
	}

	query := `INSERT INTO users (username, email, timezone, preferred_language, current_level, ai_enabled, last_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	now := time.Now()
	var id int
	err = s.db.QueryRowContext(ctx, query, username, email, timezone, language, level, false, now, now, now).Scan(&id)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, contextutils.ErrRecordExists
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	var user *models.User
	user, err = s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, contextutils.WrapError(contextutils.ErrDatabaseQuery, "user was created but could not be retrieved from database")
	}

	// Assign default "user" role to new users
	err = s.AssignRoleByName(ctx, user.ID, "user")
	if err != nil {
		// Log the error but don't fail the user creation
		// The user role assignment can be done manually by admin if needed
		s.logger.Warn(ctx, "Failed to assign default user role", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	return user, nil
}

// CreateUserWithPassword creates a new user with password authentication
func (s *UserService) CreateUserWithPassword(ctx context.Context, username, password, language, level string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "create_user_with_password", attribute.String("user.username", username))
	defer observability.FinishSpan(span, &err)

	// Validate username is not empty
	if username == "" || len(strings.TrimSpace(username)) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrInvalidInput, "username cannot be empty")
	}

	// Hash the password using bcrypt
	var hashedPassword []byte
	hashedPassword, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// default timezone to UTC for new users created with password
	query := `INSERT INTO users (username, password_hash, preferred_language, current_level, ai_enabled, last_active, created_at, updated_at, timezone) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	now := time.Now()
	var id int
	err = s.db.QueryRowContext(ctx, query, username, string(hashedPassword), language, level, false, now, now, now, "UTC").Scan(&id)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, contextutils.ErrRecordExists
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, contextutils.WrapError(contextutils.ErrDatabaseQuery, "user was created but could not be retrieved from database")
	}

	// Assign default "user" role to new users
	err = s.AssignRoleByName(ctx, user.ID, "user")
	if err != nil {
		// Log the error but don't fail the user creation
		// The user role assignment can be done manually by admin if needed
		s.logger.Warn(ctx, "Failed to assign default user role", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	return user, nil
}

// AuthenticateUser verifies user credentials and returns the user if valid
func (s *UserService) AuthenticateUser(ctx context.Context, username, password string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "authenticate_user", attribute.String("user.username", username))
	defer observability.FinishSpan(span, &err)
	// Get user by username
	var user *models.User
	user, err = s.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Check if password hash exists
	if !user.PasswordHash.Valid {
		return nil, errors.New("user has no password set")
	}

	// Compare provided password with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(ctx context.Context, id int) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_by_id", attribute.Int("user.id", id))
	defer observability.FinishSpan(span, &err)
	query := fmt.Sprintf("SELECT %s FROM users WHERE id = $1", userSelectFields)
	var user *models.User
	user, err = s.getUserByQuery(ctx, query, id)
	if err != nil {
		s.logger.Error(ctx, "Database error retrieving user", err, map[string]interface{}{"user_id": id})
		return nil, err
	}
	if user == nil {
		s.logger.Debug(ctx, "User not found in database", map[string]interface{}{"user_id": id})
		return nil, nil
	}

	// Load user roles
	roles, err := s.GetUserRoles(ctx, id)
	if err != nil {
		s.logger.Warn(ctx, "Failed to load user roles", map[string]interface{}{"user_id": id, "error": err.Error()})
		// Don't fail the entire request if roles can't be loaded
		user.Roles = []models.Role{}
	} else {
		user.Roles = roles
	}

	return user, nil
}

// GetUserByUsername retrieves a user by their username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_by_username", attribute.String("user.username", username))
	defer observability.FinishSpan(span, &err)
	query := fmt.Sprintf("SELECT %s FROM users WHERE username = $1", userSelectFields)
	return s.getUserByQuery(ctx, query, username)
}

// UpdateUserSettings updates user settings including AI configuration
func (s *UserService) UpdateUserSettings(ctx context.Context, userID int, settings *models.UserSettings) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_user_settings", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)

	// Check if user exists before updating settings
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}

	// Start a transaction to update both user settings and API key
	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction for user settings update")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
		}
	}()

	// Handle AI enabled logic
	aiProvider := settings.AIProvider
	aiModel := settings.AIModel

	// If AI is disabled, clear the provider and model
	if !settings.AIEnabled {
		aiProvider = ""
		aiModel = ""
	}

	// Update user settings (excluding API key which is now stored separately)
	query := `UPDATE users SET preferred_language = $1, current_level = $2, ai_provider = $3, ai_model = $4, ai_enabled = $5, updated_at = $6 WHERE id = $7`
	var result sql.Result
	result, err = tx.ExecContext(ctx, query, settings.Language, settings.Level, aiProvider, aiModel, settings.AIEnabled, time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update user settings in transaction")
	}

	// Check if the user was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "user with ID %d not found", userID)
	}

	// If an API key is provided and AI is enabled, save it for the specific provider
	if settings.AIAPIKey != "" && settings.AIProvider != "" && settings.AIEnabled {
		err = s.setUserAPIKeyTx(ctx, tx, userID, settings.AIProvider, settings.AIAPIKey)
		if err != nil {
			return contextutils.WrapError(err, "failed to set user API key in transaction")
		}
	}

	return tx.Commit()
}

// UpdateWordOfDayEmailEnabled updates the user's preference for word-of-day emails
func (s *UserService) UpdateWordOfDayEmailEnabled(ctx context.Context, userID int, enabled bool) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_word_of_day_email_enabled",
		attribute.Int("user.id", userID),
		attribute.Bool("word_of_day_email_enabled", enabled),
	)
	defer observability.FinishSpan(span, &err)

	// Ensure user exists
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.ErrRecordNotFound
	}

	_, err = s.db.ExecContext(ctx, `UPDATE users SET word_of_day_email_enabled = $1, updated_at = NOW() WHERE id = $2`, enabled, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update word_of_day_email_enabled")
	}
	return nil
}

// GetUserAPIKey retrieves the API key for a specific provider for a user
func (s *UserService) GetUserAPIKey(ctx context.Context, userID int, provider string) (result0 string, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_api_key", attribute.Int("user.id", userID), attribute.String("user.provider", provider))
	defer observability.FinishSpan(span, &err)

	// Check if user exists before getting API key
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return "", contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}
	span.SetAttributes(attribute.String("user.username", user.Username))

	query := `SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2`
	var apiKey string
	err = s.db.QueryRowContext(ctx, query, userID, provider).Scan(&apiKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", contextutils.WrapError(contextutils.ErrRecordNotFound, "API key for provider not found")
		}
		return "", contextutils.WrapError(err, "failed to get user API key")
	}
	return apiKey, nil
}

// GetUserAPIKeyWithID retrieves the API key and its ID for a specific provider for a user
func (s *UserService) GetUserAPIKeyWithID(ctx context.Context, userID int, provider string) (apiKey string, apiKeyID *int, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_api_key_with_id", attribute.Int("user.id", userID), attribute.String("user.provider", provider))
	defer observability.FinishSpan(span, &err)

	// Check if user exists before getting API key
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return "", nil, contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return "", nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}
	span.SetAttributes(attribute.String("user.username", user.Username))

	query := `SELECT id, api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2`
	var id int
	var key string
	err = s.db.QueryRowContext(ctx, query, userID, provider).Scan(&id, &key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "API key for provider not found")
		}
		return "", nil, contextutils.WrapError(err, "failed to get user API key with ID")
	}
	return key, &id, nil
}

// SetUserAPIKey sets the API key for a specific provider for a user
func (s *UserService) SetUserAPIKey(ctx context.Context, userID int, provider, apiKey string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "set_user_api_key", attribute.Int("user.id", userID), attribute.String("user.provider", provider))
	defer observability.FinishSpan(span, &err)

	// Check if user exists before setting API key
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}
	span.SetAttributes(attribute.String("user.username", user.Username))

	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction for API key update")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
			}
		}
	}()

	err = s.setUserAPIKeyTx(ctx, tx, userID, provider, apiKey)
	if err != nil {
		return contextutils.WrapError(err, "failed to set user API key in transaction")
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		return contextutils.WrapError(commitErr, "failed to commit API key transaction")
	}

	// Clear the error so defer doesn't try to rollback
	err = nil
	return nil
}

// setUserAPIKeyTx sets the API key for a specific provider within a transaction
func (s *UserService) setUserAPIKeyTx(ctx context.Context, tx *sql.Tx, userID int, provider, apiKey string) error {
	query := `INSERT INTO user_api_keys (user_id, provider, api_key, updated_at)
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (user_id, provider)
			  DO UPDATE SET api_key = $3, updated_at = $4`
	_, err := tx.ExecContext(ctx, query, userID, provider, apiKey, time.Now())
	return contextutils.WrapError(err, "failed to execute API key transaction")
}

// HasUserAPIKey checks if a user has an API key for a specific provider
func (s *UserService) HasUserAPIKey(ctx context.Context, userID int, provider string) (result0 bool, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "has_user_api_key", attribute.Int("user.id", userID), attribute.String("user.provider", provider))
	defer observability.FinishSpan(span, &err)
	var apiKey string
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return false, contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return false, contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}
	span.SetAttributes(attribute.String("user.username", user.Username))
	apiKey, err = s.GetUserAPIKey(ctx, userID, provider)
	if err != nil {
		// If the error is "not found" and it's specifically about the API key not existing (not the user),
		// then it means no API key exists, which is not an error
		if errors.Is(err, contextutils.ErrRecordNotFound) {
			// Check if the error message indicates it's about the API key, not the user
			if strings.Contains(err.Error(), "API key for provider not found") {
				return false, nil
			}
			// If it's about the user not found, return the error
			return false, err
		}
		return false, contextutils.WrapError(err, "failed to check if user has API key")
	}
	return apiKey != "", nil
}

// UpdateLastActive updates the user's last activity timestamp
func (s *UserService) UpdateLastActive(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_last_active", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}
	span.SetAttributes(attribute.String("user.username", user.Username))

	span.SetAttributes(attribute.String("user.username", user.Username))
	query := `UPDATE users SET last_active = $1 WHERE id = $2`
	var result sql.Result
	result, err = s.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update user last active timestamp")
	}

	// Check if the user was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "user with ID %d not found", userID)
	}

	return nil
}

// GetAllUsers retrieves all users from the database
func (s *UserService) GetAllUsers(ctx context.Context) (result0 []models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_all_users")
	defer observability.FinishSpan(span, &err)
	query := fmt.Sprintf("SELECT %s FROM users", userSelectFieldsNoPassword)
	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query all users")
	}
	defer func() {
		if err = rows.Close(); err != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var users []models.User
	for rows.Next() {
		user, err := s.scanUserFromRowsNoPassword(rows)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan user from rows")
		}

		// Load user roles
		roles, err := s.GetUserRoles(ctx, user.ID)
		if err != nil {
			s.logger.Warn(ctx, "Failed to load user roles", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
			// Don't fail the entire request if roles can't be loaded
			user.Roles = []models.Role{}
		} else {
			user.Roles = roles
		}

		users = append(users, *user)
	}

	return users, nil
}

// GetUsersPaginated retrieves paginated users with filtering and search
func (s *UserService) GetUsersPaginated(ctx context.Context, page, pageSize int, search, language, level, aiProvider, aiModel, aiEnabled, active string) (result0 []models.User, result1 int, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_users_paginated")
	defer observability.FinishSpan(span, &err)

	// Build WHERE clause and args
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Search filter
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Language filter
	if language != "" {
		conditions = append(conditions, fmt.Sprintf("preferred_language = $%d", argIndex))
		args = append(args, language)
		argIndex++
	}

	// Level filter
	if level != "" {
		conditions = append(conditions, fmt.Sprintf("current_level = $%d", argIndex))
		args = append(args, level)
		argIndex++
	}

	// AI Provider filter
	if aiProvider != "" {
		conditions = append(conditions, fmt.Sprintf("ai_provider = $%d", argIndex))
		args = append(args, aiProvider)
		argIndex++
	}

	// AI Model filter
	if aiModel != "" {
		conditions = append(conditions, fmt.Sprintf("ai_model = $%d", argIndex))
		args = append(args, aiModel)
		argIndex++
	}

	// AI Enabled filter
	if aiEnabled != "" {
		enabled := aiEnabled == "true"
		conditions = append(conditions, fmt.Sprintf("ai_enabled = $%d", argIndex))
		args = append(args, enabled)
		argIndex++
	}

	// Active filter (based on last_active within 7 days)
	if active != "" {
		activeThreshold := time.Now().AddDate(0, 0, -7)
		switch active {
		case "true":
			conditions = append(conditions, fmt.Sprintf("last_active >= $%d", argIndex))
			args = append(args, activeThreshold)
		case "false":
			conditions = append(conditions, fmt.Sprintf("(last_active < $%d OR last_active IS NULL)", argIndex))
			args = append(args, activeThreshold)
		}
		argIndex++
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to count users")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	query := fmt.Sprintf("SELECT %s FROM users %s ORDER BY username LIMIT $%d OFFSET $%d",
		userSelectFieldsNoPassword, whereClause, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to query paginated users")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var users []models.User
	for rows.Next() {
		user, err := s.scanUserFromRowsNoPassword(rows)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to scan user from rows")
		}

		// Load user roles
		roles, err := s.GetUserRoles(ctx, user.ID)
		if err != nil {
			s.logger.Warn(ctx, "Failed to load user roles", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
			// Don't fail the entire request if roles can't be loaded
			user.Roles = []models.Role{}
		} else {
			user.Roles = roles
		}

		users = append(users, *user)
	}

	return users, total, nil
}

// GetUserByEmail retrieves a user by their email address
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (result0 *models.User, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_by_email", attribute.String("user.email", email))
	defer observability.FinishSpan(span, &err)
	query := fmt.Sprintf("SELECT %s FROM users WHERE email = $1", userSelectFields)
	return s.getUserByQuery(ctx, query, email)
}

// UpdateUserProfile updates user profile information (username, email, timezone)
func (s *UserService) UpdateUserProfile(ctx context.Context, userID int, username, email, timezone string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_user_profile", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)
	query := `UPDATE users SET username = $1, email = $2, timezone = $3, updated_at = $4 WHERE id = $5`
	var result sql.Result
	result, err = s.db.ExecContext(ctx, query, username, email, timezone, time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update user profile")
	}

	// Check if the user was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "user with ID %d not found", userID)
	}

	return nil
}

// UpdateUserPassword updates a user's password
func (s *UserService) UpdateUserPassword(ctx context.Context, userID int, newPassword string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_user_password", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)

	// Validate password is not empty
	if newPassword == "" {
		return contextutils.ErrorWithContextf("password cannot be empty")
	}

	// Check if user exists first
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}

	// Hash the new password using bcrypt
	var hashedPassword []byte
	hashedPassword, err = bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return contextutils.WrapError(err, "failed to hash password")
	}

	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	result, err := s.db.ExecContext(ctx, query, string(hashedPassword), time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update user password")
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}

	s.logger.Info(ctx, "Password updated successfully", map[string]interface{}{"user_id": userID, "username": user.Username})
	return nil
}

// DeleteUser removes a user and their associated data
func (s *UserService) DeleteUser(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "delete_user", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)

	// Check if user exists before deleting
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if user exists")
	}
	if user == nil {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}

	// Best-effort cleanup of dependent rows for tables that may not have ON DELETE CASCADE in some environments
	// This keeps tests deterministic and avoids orphaned data
	// TODO: This is a hack to make the tests deterministic. We should use ON DELETE CASCADE instead.
	cleanupQueries := []string{
		`DELETE FROM question_reports WHERE reported_by_user_id = $1`,
		`DELETE FROM user_api_keys WHERE user_id = $1`,
		`DELETE FROM user_roles WHERE user_id = $1`,
		`DELETE FROM user_learning_preferences WHERE user_id = $1`,
		`DELETE FROM question_priority_scores WHERE user_id = $1`,
		`DELETE FROM user_question_metadata WHERE user_id = $1`,
		`DELETE FROM user_responses WHERE user_id = $1`,
		`DELETE FROM user_questions WHERE user_id = $1`,
	}
	for _, q := range cleanupQueries {
		if _, err := s.db.ExecContext(ctx, q, userID); err != nil {
			s.logger.Warn(ctx, "Non-fatal cleanup failure during user delete", map[string]interface{}{"error": err.Error(), "query": q, "user_id": userID})
		}
	}

	// Delete the user
	query := `DELETE FROM users WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to delete user")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "user not found")
	}

	s.logger.Info(ctx, "User %d deleted successfully", map[string]interface{}{"user_id": userID})
	return nil
}

// DeleteAllUsers removes all users from the database
func (s *UserService) DeleteAllUsers(ctx context.Context) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "delete_all_users")
	defer observability.FinishSpan(span, &err)
	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction for delete all users")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
			}
		}
	}()

	// Whitelist of valid table names to prevent SQL injection
	validTables := map[string]bool{
		"user_responses":      true,
		"performance_metrics": true,
		"users":               true,
	}

	// Delete all data in the correct order (to respect foreign key constraints)
	tables := []string{
		"user_responses",
		"performance_metrics",
		"users",
	}

	for _, table := range tables {
		// Validate table name against whitelist
		if !validTables[table] {
			return contextutils.ErrorWithContextf("invalid table name: %s", table)
		}

		// Use parameterized query with validated table name
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return contextutils.WrapErrorf(err, "failed to delete from table %s", table)
		}
		// Reset sequence for PostgreSQL
		sequenceQuery := fmt.Sprintf("ALTER SEQUENCE %s_id_seq RESTART WITH 1", table)
		if _, err := tx.ExecContext(ctx, sequenceQuery); err != nil {
			// This might fail if the table doesn't have a sequence, so we log but don't fail
			s.logger.Warn(ctx, "Note: Could not reset sequence for %s (this is normal for some tables)", map[string]interface{}{"table": table})
		}
	}

	return contextutils.WrapError(tx.Commit(), "failed to commit delete all users transaction")
}

// EnsureAdminUserExists creates the admin user if it doesn't exist
func (s *UserService) EnsureAdminUserExists(ctx context.Context, adminUsername, adminPassword string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "ensure_admin_user_exists", attribute.String("admin.username", adminUsername))
	defer observability.FinishSpan(span, &err)

	// Validate input parameters
	if adminUsername == "" {
		return contextutils.ErrorWithContextf("admin username cannot be empty")
	}

	if adminPassword == "" {
		return contextutils.ErrorWithContextf("admin password cannot be empty")
	}
	// Check if admin user already exists
	var existingUser *models.User
	existingUser, err = s.GetUserByUsername(ctx, adminUsername)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if admin user exists")
	}

	if existingUser != nil {
		// User exists, check if password needs to be updated
		if existingUser.PasswordHash.Valid {
			// User has a password, test if it matches current admin password
			err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash.String), []byte(adminPassword))
			if err == nil {
				// Password matches, ensure AI settings are configured
				err = s.ensureAdminAISettings(ctx, existingUser.ID)
				if err != nil {
					s.logger.Warn(ctx, "Warning: Failed to set AI settings for existing admin user", map[string]interface{}{"error": err.Error()})
				}

				// Ensure admin user has email and timezone if not set
				if !existingUser.Email.Valid || !existingUser.Timezone.Valid {
					err = s.ensureAdminProfile(ctx, existingUser.ID)
					if err != nil {
						s.logger.Warn(ctx, "Warning: Failed to update admin profile", map[string]interface{}{"error": err.Error()})
					}
				}

				// Ensure admin user has admin role
				isAdmin, err := s.IsAdmin(ctx, existingUser.ID)
				if err != nil {
					s.logger.Warn(ctx, "Warning: Failed to check admin role for existing admin user", map[string]interface{}{"error": err.Error()})
				} else if !isAdmin {
					err = s.AssignRoleByName(ctx, existingUser.ID, "admin")
					if err != nil {
						s.logger.Warn(ctx, "Warning: Failed to assign admin role to existing admin user", map[string]interface{}{"error": err.Error()})
					}
				}

				s.logger.Info(ctx, "Admin user already exists with correct password", map[string]interface{}{"username": adminUsername})
				return nil
			}
		}

		// Update password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return contextutils.WrapError(err, "failed to hash admin password")
		}

		query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE username = $3`
		_, err = s.db.ExecContext(ctx, query, string(hashedPassword), time.Now(), adminUsername)
		if err != nil {
			return contextutils.WrapError(err, "failed to update admin user password")
		}

		// Ensure AI settings are configured
		err = s.ensureAdminAISettings(ctx, existingUser.ID)
		if err != nil {
			s.logger.Warn(ctx, "Warning: Failed to set AI settings for existing admin user", map[string]interface{}{"error": err.Error()})
		}

		// Ensure admin user has email and timezone if not set
		if !existingUser.Email.Valid || !existingUser.Timezone.Valid {
			err = s.ensureAdminProfile(ctx, existingUser.ID)
			if err != nil {
				s.logger.Warn(ctx, "Warning: Failed to update admin profile", map[string]interface{}{"error": err.Error()})
			}
		}

		// Ensure admin user has admin role
		isAdmin, err := s.IsAdmin(ctx, existingUser.ID)
		if err != nil {
			s.logger.Warn(ctx, "Warning: Failed to check admin role for existing admin user", map[string]interface{}{"error": err.Error()})
		} else if !isAdmin {
			err = s.AssignRoleByName(ctx, existingUser.ID, "admin")
			if err != nil {
				s.logger.Warn(ctx, "Warning: Failed to assign admin role to existing admin user", map[string]interface{}{"error": err.Error()})
			}
		}

		s.logger.Info(ctx, "Updated password for admin user", map[string]interface{}{"username": adminUsername})
		return nil
	}

	// Create new admin user with email and timezone
	user, err := s.CreateUserWithEmailAndTimezone(ctx, adminUsername, "admin@example.com", "America/New_York", "italian", "A1")
	if err != nil {
		return contextutils.WrapError(err, "failed to create admin user")
	}

	// Set password for the admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return contextutils.WrapError(err, "failed to hash new admin password")
	}

	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err = s.db.ExecContext(ctx, query, string(hashedPassword), time.Now(), user.ID)
	if err != nil {
		return contextutils.WrapError(err, "failed to set password for new admin user")
	}

	// Set up AI settings for the admin user
	err = s.ensureAdminAISettings(ctx, user.ID)
	if err != nil {
		s.logger.Warn(ctx, "Warning: Failed to set AI settings for new admin user", map[string]interface{}{"error": err.Error()})
	}

	// Assign admin role to the admin user
	err = s.AssignRoleByName(ctx, user.ID, "admin")
	if err != nil {
		s.logger.Warn(ctx, "Warning: Failed to assign admin role to new admin user", map[string]interface{}{"error": err.Error()})
	}

	s.logger.Info(ctx, "Created admin user", map[string]interface{}{"username": adminUsername})
	return nil
}

// ensureAdminAISettings ensures the admin user has AI settings configured
// Only sets default values if the user doesn't already have AI settings configured
func (s *UserService) ensureAdminAISettings(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "ensure_admin_ai_settings", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)
	var user *models.User
	user, err = s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("admin user not found")
	}

	// If user already has AI provider configured, don't override their settings
	if user.AIProvider.Valid && user.AIProvider.String != "" {
		s.logger.Info(ctx, "User ID already has AI settings configured, preserving existing settings", map[string]interface{}{"user_id": userID, "provider": user.AIProvider.String})
		return nil
	}

	// Set default AI settings with a default API key
	settings := &models.UserSettings{
		AIProvider: "ollama",
		AIModel:    "llama4:latest",
		AIAPIKey:   "not_needed", // Default API key
	}

	// Only update AI settings, preserve other user settings
	query := `UPDATE users SET ai_provider = $1, ai_model = $2, ai_api_key = $3, updated_at = $4 WHERE id = $5`
	_, err = s.db.ExecContext(ctx, query, settings.AIProvider, settings.AIModel, settings.AIAPIKey, time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update user AI settings")
	}

	// Save the API key to the user_api_keys table
	err = s.SetUserAPIKey(ctx, userID, settings.AIProvider, settings.AIAPIKey)
	if err != nil {
		s.logger.Warn(ctx, "Warning: Failed to save API key for user %d", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}

	s.logger.Info(ctx, "Set default AI settings for user", map[string]interface{}{"user_id": userID, "provider": settings.AIProvider, "model": settings.AIModel})
	return nil
}

// ensureAdminProfile ensures the admin user has email and timezone set
func (s *UserService) ensureAdminProfile(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "ensure_admin_profile", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)
	query := `UPDATE users SET email = $1, timezone = $2, updated_at = $3 WHERE id = $4 AND (email IS NULL OR timezone IS NULL)`
	_, err = s.db.ExecContext(ctx, query, "admin@example.com", "America/New_York", time.Now(), userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update admin profile")
	}

	s.logger.Info(ctx, "Updated admin user profile with default email and timezone", map[string]interface{}{"user_id": userID})
	return nil
}

// ResetDatabase completely resets the database to an empty state
func (s *UserService) ResetDatabase(ctx context.Context) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "reset_database")
	defer observability.FinishSpan(span, &err)
	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction for database reset")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
		}
	}()

	// Whitelist of valid table names to prevent SQL injection
	validTables := map[string]bool{
		"user_responses":      true,
		"performance_metrics": true,
		"questions":           true,
		"users":               true,
	}

	// Delete all data in the correct order (to respect foreign key constraints)
	tables := []string{
		"user_responses",
		"performance_metrics",
		"questions",
		"users",
	}

	for _, table := range tables {
		// Validate table name against whitelist
		if !validTables[table] {
			return contextutils.ErrorWithContextf("invalid table name: %s", table)
		}

		// Use parameterized query with validated table name
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return contextutils.WrapErrorf(err, "failed to delete from table %s during reset", table)
		}
		s.logger.Info(ctx, "Cleared table: %s", map[string]interface{}{"table": table})

		// Reset sequence for PostgreSQL
		sequenceQuery := fmt.Sprintf("ALTER SEQUENCE %s_id_seq RESTART WITH 1", table)
		if _, err := tx.ExecContext(ctx, sequenceQuery); err != nil {
			// This might fail if the table doesn't have a sequence, so we log but don't fail
			s.logger.Warn(ctx, "Note: Could not reset sequence for %s (this is normal for some tables)", map[string]interface{}{"table": table})
		}
	}

	err = tx.Commit()
	if err != nil {
		return contextutils.WrapError(err, "failed to commit database reset transaction")
	}

	s.logger.Info(ctx, "Database reset completed successfully")
	return nil
}

// ClearUserData removes all user activity data but keeps the users themselves
func (s *UserService) ClearUserData(ctx context.Context) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "clear_user_data")
	defer observability.FinishSpan(span, &err)
	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction for clear user data")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
		}
	}()

	// Whitelist of valid table names to prevent SQL injection
	validTables := map[string]bool{
		"user_responses":      true,
		"performance_metrics": true,
		"questions":           true,
	}

	// Delete user data but keep users (order matters due to foreign key constraints)
	tables := []string{
		"user_responses",
		"performance_metrics",
		"questions",
	}

	for _, table := range tables {
		// Validate table name against whitelist
		if !validTables[table] {
			return contextutils.ErrorWithContextf("invalid table name: %s", table)
		}

		// Use parameterized query with validated table name
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return contextutils.WrapErrorf(err, "failed to delete from table %s during clear user data", table)
		}
		s.logger.Info(ctx, "Cleared table: %s", map[string]interface{}{"table": table})

		// Reset sequence for PostgreSQL
		sequenceQuery := fmt.Sprintf("ALTER SEQUENCE %s_id_seq RESTART WITH 1", table)
		if _, err := tx.ExecContext(ctx, sequenceQuery); err != nil {
			// This might fail if the table doesn't have a sequence, so we log but don't fail
			s.logger.Warn(ctx, "Note: Could not reset sequence for %s (this is normal for some tables)", map[string]interface{}{"table": table})
		}
	}

	err = tx.Commit()
	if err != nil {
		return contextutils.WrapError(err, "failed to commit clear user data transaction")
	}

	s.logger.Info(ctx, "User data cleared successfully (users preserved)")
	return nil
}

// ClearUserDataForUser removes all user activity data for a specific user but keeps the user record
func (s *UserService) ClearUserDataForUser(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "clear_user_data_for_user", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)
	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		s.logger.Warn(ctx, "Failed to begin transaction", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to begin transaction for clear user data for specific user")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.logger.Warn(ctx, "Warning: failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
		}
	}()

	// Delete user_responses for this user's questions (via user_questions)
	query := `DELETE FROM user_responses WHERE question_id IN (SELECT question_id FROM user_questions WHERE user_id = $1)`
	result, err := tx.ExecContext(ctx, query, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to delete user_responses", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to delete user responses for specific user")
	}
	rows, _ := result.RowsAffected()
	s.logger.Info(ctx, "Deleted %d user_responses for user %d", map[string]interface{}{"count": rows, "user_id": userID})

	// Delete performance_metrics for this user (performance_metrics has user_id, not question_id)
	query = `DELETE FROM performance_metrics WHERE user_id = $1`
	result, err = tx.ExecContext(ctx, query, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to delete performance_metrics", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to delete performance metrics for specific user")
	}
	rows, _ = result.RowsAffected()
	s.logger.Info(ctx, "Deleted %d performance_metrics for user %d", map[string]interface{}{"count": rows, "user_id": userID})

	// Delete user_questions for this user
	query = `DELETE FROM user_questions WHERE user_id = $1`
	result, err = tx.ExecContext(ctx, query, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to delete user_questions", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to delete user questions for specific user")
	}
	rows, _ = result.RowsAffected()
	s.logger.Info(ctx, "Deleted %d user_questions for user %d", map[string]interface{}{"count": rows, "user_id": userID})

	// Optionally, delete orphaned questions (not assigned to any user)
	query = `DELETE FROM questions WHERE id NOT IN (SELECT question_id FROM user_questions)`
	result, err = tx.ExecContext(ctx, query)
	if err != nil {
		s.logger.Warn(ctx, "Failed to delete orphaned questions", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to delete orphaned questions")
	}
	rows, _ = result.RowsAffected()
	s.logger.Info(ctx, "Deleted %d orphaned questions", map[string]interface{}{"count": rows})

	if err := tx.Commit(); err != nil {
		s.logger.Warn(ctx, "Failed to commit transaction", map[string]interface{}{"error": err.Error()})
		return contextutils.WrapError(err, "failed to commit clear user data for specific user transaction")
	}
	s.logger.Info(ctx, "User data cleared successfully for user %d (users preserved)", map[string]interface{}{"user_id": userID})
	return nil
}

func (s *UserService) applyDefaultSettings(ctx context.Context, user *models.User) {
	if user == nil || s.cfg == nil {
		return
	}
	_, span := observability.TraceUserFunction(ctx, "apply_default_settings", attribute.Int("user.id", user.ID))
	defer span.End()
	if user.AIProvider.String == "" && len(s.cfg.Providers) > 0 {
		// Use the first available provider as default
		provider := s.cfg.Providers[0]
		user.AIProvider.String = provider.Code
		// Use first model in the list as default
		if len(provider.Models) > 0 {
			user.AIModel.String = provider.Models[0].Code
		}
	}
	if user.CurrentLevel.String == "" {
		// Set default level based on user's preferred language, or use first available language
		language := user.PreferredLanguage.String
		if language == "" {
			languages := s.cfg.GetLanguages()
			if len(languages) > 0 {
				language = languages[0]
			}
		}
		if language != "" {
			levels := s.cfg.GetLevelsForLanguage(language)
			if len(levels) > 0 {
				user.CurrentLevel.String = levels[0]
			}
		}
	}
	if user.PreferredLanguage.String == "" {
		user.PreferredLanguage.String = "english"
	}
}

// GetUserRoles retrieves all roles for a user
func (s *UserService) GetUserRoles(ctx context.Context, userID int) (result0 []models.Role, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_roles", attribute.Int("user.id", userID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user roles")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan user role")
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating user roles")
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (s *UserService) AssignRole(ctx context.Context, userID, roleID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "assign_role", attribute.Int("user.id", userID), attribute.Int("role.id", roleID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if user exists
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to get user for role assignment")
	}
	if user == nil {
		return contextutils.ErrorWithContextf("user with ID %d not found", userID)
	}

	// Check if role exists
	var roleName string
	err = s.db.QueryRowContext(ctx, "SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextutils.ErrorWithContextf("role with ID %d not found", roleID)
		}
		return contextutils.WrapError(err, "failed to check role existence")
	}

	// Assign role (using ON CONFLICT DO NOTHING to handle duplicate assignments gracefully)
	query := `INSERT INTO user_roles (user_id, role_id, created_at) VALUES ($1, $2, $3) ON CONFLICT (user_id, role_id) DO NOTHING`
	_, err = s.db.ExecContext(ctx, query, userID, roleID, time.Now())
	if err != nil {
		return contextutils.WrapError(err, "failed to assign role to user")
	}

	s.logger.Info(ctx, "Role assigned successfully", map[string]interface{}{
		"user_id":   userID,
		"role_id":   roleID,
		"role_name": roleName,
	})

	return nil
}

// AssignRoleByName assigns a role to a user by role name
func (s *UserService) AssignRoleByName(ctx context.Context, userID int, roleName string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "assign_role_by_name", attribute.Int("user.id", userID), attribute.String("role.name", roleName))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if user exists
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to get user for role assignment")
	}
	if user == nil {
		return contextutils.ErrorWithContextf("user with ID %d not found", userID)
	}

	// Get role ID by name
	var roleID int
	err = s.db.QueryRowContext(ctx, "SELECT id FROM roles WHERE name = $1", roleName).Scan(&roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextutils.ErrorWithContextf("role with name '%s' not found", roleName)
		}
		return contextutils.WrapError(err, "failed to get role ID by name")
	}

	// Assign role (using ON CONFLICT DO NOTHING to handle duplicate assignments gracefully)
	query := `INSERT INTO user_roles (user_id, role_id, created_at) VALUES ($1, $2, $3) ON CONFLICT (user_id, role_id) DO NOTHING`
	_, err = s.db.ExecContext(ctx, query, userID, roleID, time.Now())
	if err != nil {
		return contextutils.WrapError(err, "failed to assign role to user")
	}

	s.logger.Info(ctx, "Role assigned successfully", map[string]interface{}{
		"user_id":   userID,
		"role_id":   roleID,
		"role_name": roleName,
	})

	return nil
}

// RemoveRole removes a role from a user
func (s *UserService) RemoveRole(ctx context.Context, userID, roleID int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "remove_role", attribute.Int("user.id", userID), attribute.Int("role.id", roleID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if user exists
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to get user for role removal")
	}
	if user == nil {
		return contextutils.ErrorWithContextf("user with ID %d not found", userID)
	}

	// Check if role exists
	var roleName string
	err = s.db.QueryRowContext(ctx, "SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextutils.ErrorWithContextf("role with ID %d not found", roleID)
		}
		return contextutils.WrapError(err, "failed to check role existence")
	}

	// Remove role
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	result, err := s.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return contextutils.WrapError(err, "failed to remove role from user")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.ErrorWithContextf("user %d does not have role %d", userID, roleID)
	}

	s.logger.Info(ctx, "Role removed successfully", map[string]interface{}{
		"user_id":   userID,
		"role_id":   roleID,
		"role_name": roleName,
	})

	return nil
}

// HasRole checks if a user has a specific role by name
func (s *UserService) HasRole(ctx context.Context, userID int, roleName string) (result0 bool, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "has_role", attribute.Int("user.id", userID), attribute.String("role.name", roleName))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT COUNT(*) > 0
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.name = $2
	`
	var hasRole bool
	err = s.db.QueryRowContext(ctx, query, userID, roleName).Scan(&hasRole)
	if err != nil {
		return false, contextutils.WrapError(err, "failed to check if user has role")
	}

	return hasRole, nil
}

// IsAdmin checks if a user has admin role
func (s *UserService) IsAdmin(ctx context.Context, userID int) (result0 bool, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "is_admin", attribute.Int("user.id", userID))
	defer observability.FinishSpan(span, &err)

	return s.HasRole(ctx, userID, "admin")
}

// GetAllRoles returns all available roles in the system
func (s *UserService) GetAllRoles(ctx context.Context) (result0 []models.Role, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_all_roles")
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get all roles")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan role")
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating roles")
	}

	return roles, nil
}

// GetDB returns the database connection
func (s *UserService) GetDB() *sql.DB {
	return s.db
}

// isDuplicateKeyError checks if the error is a duplicate key constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL unique constraint violation error code
	if pqErr, ok := err.(*pq.Error); ok {
		// PostgreSQL error code 23505 is for unique constraint violations
		if pqErr.Code == "23505" {
			return true
		}
	}

	return false
}

// RegisterDeviceToken registers or updates a device token for a user
func (s *UserService) RegisterDeviceToken(ctx context.Context, userID int, deviceToken string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "register_device_token",
		attribute.Int("user.id", userID),
	)
	defer observability.FinishSpan(span, &err)

	if deviceToken == "" {
		return contextutils.WrapError(contextutils.ErrInvalidInput, "device token cannot be empty")
	}

	query := `
		INSERT INTO ios_device_tokens (user_id, device_token, device_type, created_at, updated_at)
		VALUES ($1, $2, 'ios', NOW(), NOW())
		ON CONFLICT (user_id, device_token)
		DO UPDATE SET updated_at = NOW()
	`

	result, err := s.db.ExecContext(ctx, query, userID, deviceToken)
	if err != nil {
		return contextutils.WrapError(err, "failed to register device token")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Warn(ctx, "Failed to get rows affected for device token registration", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	} else {
		s.logger.Info(ctx, "Device token registered successfully", map[string]interface{}{
			"user_id":       userID,
			"device_token":  deviceToken[:20] + "...",
			"rows_affected": rowsAffected,
		})
	}

	return nil
}

// GetUserDeviceTokens returns all device tokens for a user
func (s *UserService) GetUserDeviceTokens(ctx context.Context, userID int) (result0 []string, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_user_device_tokens",
		attribute.Int("user.id", userID),
	)
	defer observability.FinishSpan(span, &err)

	query := `SELECT device_token FROM ios_device_tokens WHERE user_id = $1`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get device tokens")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, contextutils.WrapError(err, "failed to scan device token")
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating device tokens")
	}

	s.logger.Info(ctx, "Retrieved device tokens for user", map[string]interface{}{
		"user_id":     userID,
		"token_count": len(tokens),
	})

	return tokens, nil
}

// RemoveDeviceToken removes a device token for a user
func (s *UserService) RemoveDeviceToken(ctx context.Context, userID int, deviceToken string) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "remove_device_token",
		attribute.Int("user.id", userID),
	)
	defer observability.FinishSpan(span, &err)

	query := `DELETE FROM ios_device_tokens WHERE user_id = $1 AND device_token = $2`

	result, err := s.db.ExecContext(ctx, query, userID, deviceToken)
	if err != nil {
		return contextutils.WrapError(err, "failed to remove device token")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "device token not found")
	}

	return nil
}
