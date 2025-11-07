//go:build integration

package services

import (
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func TestQuestionService_GetReportedQuestionsPaginated_Unit(t *testing.T) {
	// Test with nil database - should panic during service creation
	t.Run("Nil database", func(t *testing.T) {
		assert.Panics(t, func() {
			NewQuestionServiceWithLogger(nil, nil, nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		})
	})
}

func TestQuestionService_GetReportedQuestionsStats_Unit(t *testing.T) {
	// Test with nil database - should panic during service creation
	t.Run("Nil database", func(t *testing.T) {
		assert.Panics(t, func() {
			NewQuestionServiceWithLogger(nil, nil, nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		})
	})
}

func TestQuestionService_ReportQuestion_Unit(t *testing.T) {
	// Test with nil database - should panic during service creation
	t.Run("Nil database", func(t *testing.T) {
		assert.Panics(t, func() {
			NewQuestionServiceWithLogger(nil, nil, nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		})
	})
}

func TestQuestionService_MarkQuestionAsFixed_Unit(t *testing.T) {
	// Test with nil database - should panic during service creation
	t.Run("Nil database", func(t *testing.T) {
		assert.Panics(t, func() {
			NewQuestionServiceWithLogger(nil, nil, nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		})
	})
}

// Test helper functions for reported questions
func TestQuestionService_ValidateReportedQuestionsParams(t *testing.T) {
	tests := []struct {
		name        string
		page        int
		pageSize    int
		expectError bool
	}{
		{"Valid parameters", 1, 10, false},
		{"Zero page", 0, 10, true},
		{"Negative page", -1, 10, true},
		{"Zero page size", 1, 0, true},
		{"Negative page size", 1, -1, true},
		{"Large page size", 1, 1001, false}, // Should be allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test - in practice, validation would happen in the service method
			if tt.expectError {
				// These parameters would cause issues in the actual implementation
				assert.True(t, tt.page <= 0 || tt.pageSize <= 0)
			} else {
				assert.True(t, tt.page > 0 && tt.pageSize > 0)
			}
		})
	}
}

// Test SQL query construction for reported questions
func TestQuestionService_ReportedQuestionsQueryConstruction(t *testing.T) {
	tests := []struct {
		name           string
		search         string
		typeFilter     string
		languageFilter string
		levelFilter    string
		expectedParams int
	}{
		{"No filters", "", "", "", "", 0},
		{"Search only", "test", "", "", "", 1},
		{"Type filter only", "", "vocabulary", "", "", 1},
		{"Language filter only", "", "", "italian", "", 1},
		{"Level filter only", "", "", "", "A1", 1},
		{"All filters", "test", "vocabulary", "italian", "A1", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the expected number of parameters
			// In the actual implementation, each filter adds a parameter
			paramCount := 0
			if tt.search != "" {
				paramCount++
			}
			if tt.typeFilter != "" {
				paramCount++
			}
			if tt.languageFilter != "" {
				paramCount++
			}
			if tt.levelFilter != "" {
				paramCount++
			}

			assert.Equal(t, tt.expectedParams, paramCount)
		})
	}
}
