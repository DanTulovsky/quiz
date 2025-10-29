package handlers

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestConvertAuthAPIKey_LastUsedAtSerialization(t *testing.T) {
	now := time.Date(2025, 8, 4, 12, 0, 0, 0, time.UTC)

	withLastUsed := models.AuthAPIKey{
		ID:              1,
		KeyName:         "Key A",
		KeyPrefix:       "qapp_xxx",
		PermissionLevel: models.PermissionLevelReadonly,
		LastUsedAt:      sql.NullTime{Valid: true, Time: now},
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now,
	}

	withoutLastUsed := models.AuthAPIKey{
		ID:              2,
		KeyName:         "Key B",
		KeyPrefix:       "qapp_yyy",
		PermissionLevel: models.PermissionLevelFull,
		// LastUsedAt invalid -> should serialize as null
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now,
	}

	apiList := []api.APIKeySummary{
		convertAuthAPIKeyToAPI(&withLastUsed),
		convertAuthAPIKeyToAPI(&withoutLastUsed),
	}
	count := len(apiList)
	resp := api.APIKeysListResponse{ApiKeys: &apiList, Count: &count}

	// Marshal and inspect
	b, err := json.Marshal(resp)
	assert.NoError(t, err)
	s := string(b)

	// Expect RFC3339 time string for first, and explicit null for second
	assert.Contains(t, s, "\"last_used_at\":\"2025-08-04T12:00:00Z\"")
	// Ensure there is at least one null occurrence for last_used_at
	assert.Contains(t, s, "\"last_used_at\":null")
}
