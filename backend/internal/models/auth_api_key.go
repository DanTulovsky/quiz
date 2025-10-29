package models

import (
	"database/sql"
	"time"
)

// AuthAPIKey represents an API key for programmatic authentication
// This is separate from user_api_keys which stores AI provider API keys
type AuthAPIKey struct {
	ID              int          `json:"id"`
	UserID          int          `json:"user_id"`
	KeyName         string       `json:"key_name"`
	KeyHash         string       `json:"-"` // Never expose the hash
	KeyPrefix       string       `json:"key_prefix"`
	PermissionLevel string       `json:"permission_level"` // "readonly" or "full"
	LastUsedAt      sql.NullTime `json:"last_used_at"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// PermissionLevel constants
const (
	PermissionLevelReadonly = "readonly"
	PermissionLevelFull     = "full"
)

// IsValidPermissionLevel checks if the permission level is valid
func IsValidPermissionLevel(level string) bool {
	return level == PermissionLevelReadonly || level == PermissionLevelFull
}

// CanPerformMethod checks if the permission level allows the given HTTP method
func (k *AuthAPIKey) CanPerformMethod(method string) bool {
	if k.PermissionLevel == PermissionLevelFull {
		return true
	}
	// Readonly keys can only perform GET and HEAD requests
	return method == "GET" || method == "HEAD"
}
