package handlers

import (
	"testing"

	"quizapp/internal/models"
)

func TestConvertLearningPreferencesToAPIIncludesDailyGoal(t *testing.T) {
	prefs := &models.UserLearningPreferences{
		DailyGoal:            7,
		FocusOnWeakAreas:     true,
		FreshQuestionRatio:   0.3,
		KnownQuestionPenalty: 0.1,
		ReviewIntervalDays:   5,
		WeakAreaBoost:        1.5,
		DailyReminderEnabled: false,
		TTSVoice:             "",
	}

	out := convertLearningPreferencesToAPI(prefs)
	if out == nil || out.DailyGoal == nil {
		t.Fatalf("expected daily goal to be populated in API preferences")
	}
	if *out.DailyGoal != 7 {
		t.Fatalf("expected daily goal 7, got %d", *out.DailyGoal)
	}
}
