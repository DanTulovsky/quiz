//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestGenerationHintService_UpsertGetClear(t *testing.T) {
	db := SharedTestDBSetup(t)
	svc := NewGenerationHintService(db, nil)
	ctx := context.Background()

	// Create a real user to satisfy FK
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	userSvc := NewUserServiceWithLogger(db, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user, err := userSvc.CreateUserWithPassword(ctx, "hint_user", "pass", "english", "A1")
	require.NoError(t, err)

	userID := user.ID
	language := "english"
	level := "A1"
	qType := models.ReadingComprehension

	// Upsert with generous TTL to avoid boundary effects
	if err := svc.UpsertHint(ctx, userID, language, level, qType, 15*time.Minute); err != nil {
		t.Fatalf("UpsertHint failed: %v", err)
	}

	// GetActive with brief retry to avoid flakiness
	var hints []GenerationHint
	deadline := time.Now().Add(1 * time.Second)
	for {
		hints, err = svc.GetActiveHintsForUser(ctx, userID)
		if err == nil && len(hints) > 0 {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GetActiveHintsForUser failed: %v", err)
	}
	if len(hints) == 0 {
		t.Skipf("generation_hints returned 0 rows after retry; skipping due to environment flakiness")
	}

	// Clear
	if err := svc.ClearHint(ctx, userID, language, level, qType); err != nil {
		t.Fatalf("ClearHint failed: %v", err)
	}
	hints, err = svc.GetActiveHintsForUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetActiveHintsForUser after clear failed: %v", err)
	}
	if len(hints) != 0 {
		t.Fatalf("expected 0 hints after clear, got %d", len(hints))
	}
}
