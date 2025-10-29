package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

// AuthAPIKeyServiceInterface defines the interface for auth API key operations
type AuthAPIKeyServiceInterface interface {
	CreateAPIKey(ctx context.Context, userID int, keyName string, permissionLevel string) (*models.AuthAPIKey, string, error)
	ListAPIKeys(ctx context.Context, userID int) ([]models.AuthAPIKey, error)
	GetAPIKeyByID(ctx context.Context, userID int, keyID int) (*models.AuthAPIKey, error)
	DeleteAPIKey(ctx context.Context, userID int, keyID int) error
	ValidateAPIKey(ctx context.Context, rawKey string) (*models.AuthAPIKey, error)
	UpdateLastUsed(ctx context.Context, keyID int) error
}

// AuthAPIKeyService implements AuthAPIKeyServiceInterface
type AuthAPIKeyService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewAuthAPIKeyService creates a new AuthAPIKeyService instance
func NewAuthAPIKeyService(db *sql.DB, logger *observability.Logger) *AuthAPIKeyService {
	return &AuthAPIKeyService{
		db:     db,
		logger: logger,
	}
}

const (
	// KeyPrefix is the prefix for all auth API keys
	KeyPrefix = "qapp_"
	// KeyLength is the length of the random part of the key (32 characters)
	KeyLength = 32
)

// generateAPIKey generates a new random API key
func generateAPIKey() (string, error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, KeyLength/2) // 16 bytes = 32 hex characters
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Convert to hex string
	randomStr := hex.EncodeToString(randomBytes)

	// Add prefix
	return KeyPrefix + randomStr, nil
}

// hashAPIKey hashes an API key using bcrypt
func hashAPIKey(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hash), nil
}

// CreateAPIKey creates a new API key for a user
func (s *AuthAPIKeyService) CreateAPIKey(ctx context.Context, userID int, keyName string, permissionLevel string) (*models.AuthAPIKey, string, error) {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.CreateAPIKey")
	defer observability.FinishSpan(span, nil)

	span.SetAttributes(
		attribute.Int("user_id", userID),
		attribute.String("key_name", keyName),
		attribute.String("permission_level", permissionLevel),
	)

	// Validate permission level
	if !models.IsValidPermissionLevel(permissionLevel) {
		err := contextutils.NewAppError(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid permission level",
			"Permission level must be 'readonly' or 'full'",
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, "", err
	}

	// Validate key name
	if keyName == "" {
		err := contextutils.NewAppError(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Key name is required",
			"",
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, "", err
	}

	// Generate new API key
	rawKey, err := generateAPIKey()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate API key")
		return nil, "", contextutils.WrapError(err, "failed to generate API key")
	}

	// Hash the key
	keyHash, err := hashAPIKey(rawKey)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to hash API key")
		return nil, "", contextutils.WrapError(err, "failed to hash API key")
	}

	// Extract key prefix (first 12 characters including "qapp_")
	keyPrefix := rawKey
	if len(rawKey) > 12 {
		keyPrefix = rawKey[:12]
	}

	// Insert into database
	query := `
		INSERT INTO auth_api_keys (user_id, key_name, key_hash, key_prefix, permission_level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var apiKey models.AuthAPIKey
	apiKey.UserID = userID
	apiKey.KeyName = keyName
	apiKey.KeyHash = keyHash
	apiKey.KeyPrefix = keyPrefix
	apiKey.PermissionLevel = permissionLevel

	err = s.db.QueryRowContext(ctx, query, userID, keyName, keyHash, keyPrefix, permissionLevel, now, now).
		Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)

	if err != nil {
		s.logger.Error(ctx, "Failed to create API key", err, map[string]interface{}{
			"user_id":          userID,
			"key_name":         keyName,
			"permission_level": permissionLevel,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to insert API key")
		return nil, "", contextutils.WrapError(err, "failed to create API key")
	}

	span.SetAttributes(attribute.Int("api_key_id", apiKey.ID))
	s.logger.Info(ctx, "Created new API key", map[string]interface{}{
		"user_id":          userID,
		"api_key_id":       apiKey.ID,
		"key_name":         keyName,
		"permission_level": permissionLevel,
	})

	// Return the API key object and the raw key (only time it's returned)
	return &apiKey, rawKey, nil
}

// ListAPIKeys returns all API keys for a user
func (s *AuthAPIKeyService) ListAPIKeys(ctx context.Context, userID int) ([]models.AuthAPIKey, error) {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.ListAPIKeys")
	defer observability.FinishSpan(span, nil)

	span.SetAttributes(attribute.Int("user_id", userID))

	query := `
		SELECT id, user_id, key_name, key_hash, key_prefix, permission_level, last_used_at, created_at, updated_at
		FROM auth_api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		s.logger.Error(ctx, "Failed to list API keys", err, map[string]interface{}{"user_id": userID})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query API keys")
		return nil, contextutils.WrapError(err, "failed to list API keys")
	}
	defer rows.Close()

	var apiKeys []models.AuthAPIKey
	for rows.Next() {
		var apiKey models.AuthAPIKey
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.KeyName,
			&apiKey.KeyHash,
			&apiKey.KeyPrefix,
			&apiKey.PermissionLevel,
			&apiKey.LastUsedAt,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan API key", err, map[string]interface{}{"user_id": userID})
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to scan API key")
			return nil, contextutils.WrapError(err, "failed to scan API key")
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating API keys", err, map[string]interface{}{"user_id": userID})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to iterate API keys")
		return nil, contextutils.WrapError(err, "failed to list API keys")
	}

	span.SetAttributes(attribute.Int("count", len(apiKeys)))
	return apiKeys, nil
}

// GetAPIKeyByID retrieves a specific API key by ID for a user
func (s *AuthAPIKeyService) GetAPIKeyByID(ctx context.Context, userID int, keyID int) (*models.AuthAPIKey, error) {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.GetAPIKeyByID")
	defer observability.FinishSpan(span, nil)

	span.SetAttributes(
		attribute.Int("user_id", userID),
		attribute.Int("key_id", keyID),
	)

	query := `
		SELECT id, user_id, key_name, key_hash, key_prefix, permission_level, last_used_at, created_at, updated_at
		FROM auth_api_keys
		WHERE id = $1 AND user_id = $2
	`

	var apiKey models.AuthAPIKey
	err := s.db.QueryRowContext(ctx, query, keyID, userID).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.KeyName,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.PermissionLevel,
		&apiKey.LastUsedAt,
		&apiKey.CreatedAt,
		&apiKey.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		s.logger.Error(ctx, "Failed to get API key", err, map[string]interface{}{
			"user_id": userID,
			"key_id":  keyID,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get API key")
		return nil, contextutils.WrapError(err, "failed to get API key")
	}

	return &apiKey, nil
}

// DeleteAPIKey deletes an API key
func (s *AuthAPIKeyService) DeleteAPIKey(ctx context.Context, userID int, keyID int) error {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.DeleteAPIKey")
	defer observability.FinishSpan(span, nil)

	span.SetAttributes(
		attribute.Int("user_id", userID),
		attribute.Int("key_id", keyID),
	)

	query := `DELETE FROM auth_api_keys WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, keyID, userID)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete API key", err, map[string]interface{}{
			"user_id": userID,
			"key_id":  keyID,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete API key")
		return contextutils.WrapError(err, "failed to delete API key")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error(ctx, "Failed to get rows affected", err, map[string]interface{}{
			"user_id": userID,
			"key_id":  keyID,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get rows affected")
		return contextutils.WrapError(err, "failed to check deletion")
	}

	if rowsAffected == 0 {
		err := contextutils.NewAppError(
			contextutils.ErrorCodeNotFound,
			contextutils.SeverityWarn,
			"API key not found",
			"",
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "API key not found")
		return err
	}

	s.logger.Info(ctx, "Deleted API key", map[string]interface{}{
		"user_id": userID,
		"key_id":  keyID,
	})

	return nil
}

// ValidateAPIKey validates a raw API key and returns the associated key info
func (s *AuthAPIKeyService) ValidateAPIKey(ctx context.Context, rawKey string) (*models.AuthAPIKey, error) {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.ValidateAPIKey")
	defer observability.FinishSpan(span, nil)

	// Basic validation
	if rawKey == "" {
		return nil, errors.New("API key is empty")
	}

	if len(rawKey) < len(KeyPrefix) || rawKey[:len(KeyPrefix)] != KeyPrefix {
		span.SetStatus(codes.Error, "invalid API key format")
		return nil, errors.New("invalid API key format")
	}

	// Query all API keys with matching prefix for this key
	// We need to check all because we hash the keys
	query := `
		SELECT id, user_id, key_name, key_hash, key_prefix, permission_level, last_used_at, created_at, updated_at
		FROM auth_api_keys
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error(ctx, "Failed to query API keys for validation", err, nil)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query API keys")
		return nil, contextutils.WrapError(err, "failed to validate API key")
	}
	defer rows.Close()

	// Check each key by comparing bcrypt hash
	for rows.Next() {
		var apiKey models.AuthAPIKey
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.KeyName,
			&apiKey.KeyHash,
			&apiKey.KeyPrefix,
			&apiKey.PermissionLevel,
			&apiKey.LastUsedAt,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan API key", err, nil)
			continue
		}

		// Compare hash
		err = bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(rawKey))
		if err == nil {
			// Found matching key
			span.SetAttributes(
				attribute.Int("api_key_id", apiKey.ID),
				attribute.Int("user_id", apiKey.UserID),
				attribute.String("permission_level", apiKey.PermissionLevel),
			)
			return &apiKey, nil
		}
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating API keys", err, nil)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to iterate API keys")
		return nil, contextutils.WrapError(err, "failed to validate API key")
	}

	// No matching key found
	span.SetStatus(codes.Error, "invalid API key")
	return nil, errors.New("invalid API key")
}

// UpdateLastUsed updates the last_used_at timestamp for an API key
// This should be called asynchronously to avoid blocking requests
func (s *AuthAPIKeyService) UpdateLastUsed(ctx context.Context, keyID int) error {
	ctx, span := observability.TraceFunction(ctx, "AuthAPIKeyService.UpdateLastUsed")
	defer observability.FinishSpan(span, nil)

	span.SetAttributes(attribute.Int("key_id", keyID))

	query := `UPDATE auth_api_keys SET last_used_at = $1, updated_at = $2 WHERE id = $3`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, now, now, keyID)
	if err != nil {
		s.logger.Error(ctx, "Failed to update last used timestamp", err, map[string]interface{}{
			"key_id": keyID,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update last used")
		// Don't return error - this is not critical
		return nil
	}

	return nil
}
