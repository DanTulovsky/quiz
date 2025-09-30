package services

import (
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func newTestLearningService() *LearningService {
	cfg := &config.Config{
		LanguageLevels: map[string]config.LanguageLevelConfig{
			"test": {
				Levels: []string{"A1", "A2", "B1", "B1+", "B1++", "B2", "C1", "C2"},
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	return NewLearningServiceWithLogger(nil, cfg, logger)
}

func TestLearningService_suggestLevelAdjustment(t *testing.T) {
	learningService := newTestLearningService()

	tests := []struct {
		name     string
		progress *models.UserProgress
		expected string
	}{
		{
			name: "not enough data",
			progress: &models.UserProgress{
				TotalQuestions: 10,
				AccuracyRate:   85.0,
				CurrentLevel:   "A1",
			},
			expected: "",
		},
		{
			name: "high accuracy - suggest level up",
			progress: &models.UserProgress{
				TotalQuestions: 25,
				AccuracyRate:   90.0,
				CurrentLevel:   "A1",
			},
			expected: "A2",
		},
		{
			name: "low accuracy - suggest level down",
			progress: &models.UserProgress{
				TotalQuestions: 25,
				AccuracyRate:   40.0,
				CurrentLevel:   "B1",
			},
			expected: "A2",
		},
		{
			name: "good accuracy - no change",
			progress: &models.UserProgress{
				TotalQuestions: 25,
				AccuracyRate:   75.0,
				CurrentLevel:   "B1",
			},
			expected: "",
		},
		{
			name: "already at highest level",
			progress: &models.UserProgress{
				TotalQuestions: 25,
				AccuracyRate:   95.0,
				CurrentLevel:   "C2",
			},
			expected: "C2", // Can't go higher
		},
		{
			name: "already at lowest level",
			progress: &models.UserProgress{
				TotalQuestions: 25,
				AccuracyRate:   30.0,
				CurrentLevel:   "A1",
			},
			expected: "A1", // Can't go lower
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learningService.suggestLevelAdjustment(tt.progress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLearningService_getNextLevel(t *testing.T) {
	learningService := newTestLearningService()

	tests := []struct {
		name         string
		currentLevel string
		expected     string
	}{
		{"A1", "A1", "A2"},
		{"A2", "A2", "B1"},
		{"B1", "B1", "B1+"},
		{"B1+", "B1+", "B1++"},
		{"B1++", "B1++", "B2"},
		{"B2", "B2", "C1"},
		{"C1", "C1", "C2"},
		{"C2", "C2", "C2"}, // Can't go higher
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learningService.getNextLevel(tt.currentLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLearningService_getPreviousLevel(t *testing.T) {
	learningService := newTestLearningService()

	tests := []struct {
		name         string
		currentLevel string
		expected     string
	}{
		{"A1", "A1", "A1"}, // Can't go lower
		{"A2", "A2", "A1"},
		{"B1", "B1", "A2"},
		{"B1+", "B1+", "B1"},
		{"B1++", "B1++", "B1+"},
		{"B2", "B2", "B1++"},
		{"C1", "C1", "B2"},
		{"C2", "C2", "C1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learningService.getPreviousLevel(tt.currentLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLearningService_NewLearningServiceWithLogger tests the constructor
func TestLearningService_NewLearningServiceWithLogger(t *testing.T) {
	service := newTestLearningService()
	assert.NotNil(t, service)
}
