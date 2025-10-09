//go:build integration
// +build integration

package models

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_MarshalJSON_Integration(t *testing.T) {
	user := User{
		ID:                1,
		Username:          "testuser",
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "A1", Valid: true},
		Email:             sql.NullString{String: "test@example.com", Valid: true},
		Timezone:          sql.NullString{String: "UTC", Valid: true},
		AIEnabled:         sql.NullBool{Bool: true, Valid: true},
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		LastActive:        sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "testuser", result["username"])
	assert.Equal(t, "italian", result["preferred_language"])
	assert.Equal(t, "A1", result["current_level"])
	assert.Equal(t, "test@example.com", result["email"])
	assert.Equal(t, "UTC", result["timezone"])
	assert.Equal(t, true, result["ai_enabled"])
	assert.NotNil(t, result["created_at"])
	assert.NotNil(t, result["updated_at"])
	assert.NotNil(t, result["last_active"])
}

func TestUser_MarshalJSON_NullValues_Integration(t *testing.T) {
	user := User{
		ID:       1,
		Username: "testuser",
		// All other fields are null/invalid
		PreferredLanguage: sql.NullString{Valid: false},
		CurrentLevel:      sql.NullString{Valid: false},
		Email:             sql.NullString{Valid: false},
		Timezone:          sql.NullString{Valid: false},
		AIEnabled:         sql.NullBool{Valid: false},
		LastActive:        sql.NullTime{Valid: false},
		CreatedAt:         time.Now().UTC(),
		// UpdatedAt is not set - will be zero value
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "testuser", result["username"])
	assert.Nil(t, result["preferred_language"])
	assert.Nil(t, result["current_level"])
	assert.Nil(t, result["email"])
	assert.Nil(t, result["timezone"])
	assert.Nil(t, result["ai_enabled"])
	assert.NotNil(t, result["created_at"])
	assert.NotNil(t, result["updated_at"]) // UpdatedAt should have a value (zero time)
	assert.Nil(t, result["last_active"])
}

func TestNullStringToPointer_Integration(t *testing.T) {
	// Test valid null string
	validNS := sql.NullString{String: "test", Valid: true}
	result := nullStringToPointer(validNS)
	require.NotNil(t, result)
	assert.Equal(t, "test", *result)

	// Test invalid null string
	invalidNS := sql.NullString{Valid: false}
	result = nullStringToPointer(invalidNS)
	assert.Nil(t, result)
}

func TestNullTimeToPointer_Integration(t *testing.T) {
	now := time.Now().UTC()

	// Test valid null time
	validNT := sql.NullTime{Time: now, Valid: true}
	result := nullTimeToPointer(validNT)
	require.NotNil(t, result)
	assert.Equal(t, now, *result)

	// Test invalid null time
	invalidNT := sql.NullTime{Valid: false}
	result = nullTimeToPointer(invalidNT)
	assert.Nil(t, result)
}

func TestNullBoolToPointer_Integration(t *testing.T) {
	// Test valid null bool - true
	validNBTrue := sql.NullBool{Bool: true, Valid: true}
	result := nullBoolToPointer(validNBTrue)
	require.NotNil(t, result)
	assert.True(t, *result)

	// Test valid null bool - false
	validNBFalse := sql.NullBool{Bool: false, Valid: true}
	result = nullBoolToPointer(validNBFalse)
	require.NotNil(t, result)
	assert.False(t, *result)

	// Test invalid null bool
	invalidNB := sql.NullBool{Valid: false}
	result = nullBoolToPointer(invalidNB)
	assert.Nil(t, result)
}

func TestUserProgress_AccuracyRate_Integration(t *testing.T) {
	tests := []struct {
		name             string
		correctAnswers   int
		totalAttempts    int
		expectedAccuracy float64
	}{
		{
			name:             "Perfect accuracy",
			correctAnswers:   10,
			totalAttempts:    10,
			expectedAccuracy: 1.0,
		},
		{
			name:             "Half accuracy",
			correctAnswers:   5,
			totalAttempts:    10,
			expectedAccuracy: 0.5,
		},
		{
			name:             "No attempts",
			correctAnswers:   0,
			totalAttempts:    0,
			expectedAccuracy: 0.0,
		},
		{
			name:             "Zero correct",
			correctAnswers:   0,
			totalAttempts:    10,
			expectedAccuracy: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := UserProgress{
				CorrectAnswers: tt.correctAnswers,
				TotalQuestions: tt.totalAttempts,
			}

			var accuracy float64
			if progress.TotalQuestions == 0 {
				accuracy = 0.0
			} else {
				accuracy = float64(progress.CorrectAnswers) / float64(progress.TotalQuestions)
			}
			assert.Equal(t, tt.expectedAccuracy, accuracy)
		})
	}
}

func TestQuestion_GetCorrectAnswerText_Integration(t *testing.T) {
	tests := []struct {
		name         string
		questionType QuestionType
		content      map[string]interface{}
		correctIdx   int
		expected     string
	}{
		{
			name:         "Vocabulary question",
			questionType: Vocabulary,
			content: map[string]interface{}{
				"options": []interface{}{"ciao", "buongiorno", "grazie", "prego"},
			},
			correctIdx: 1,
			expected:   "buongiorno",
		},
		{
			name:         "Fill in blank question",
			questionType: FillInBlank,
			content: map[string]interface{}{
				"options": []interface{}{"capitale", "città", "paese", "stato"},
			},
			correctIdx: 0,
			expected:   "capitale",
		},
		{
			name:         "Question answer type",
			questionType: QuestionAnswer,
			content: map[string]interface{}{
				"options": []interface{}{"Roma è la capitale d'Italia", "Milano è la capitale d'Italia", "Napoli è la capitale d'Italia", "Torino è la capitale d'Italia"},
			},
			correctIdx: 0,
			expected:   "Roma è la capitale d'Italia",
		},
		{
			name:         "Reading comprehension",
			questionType: ReadingComprehension,
			content: map[string]interface{}{
				"options": []interface{}{"A", "B", "C", "D"},
			},
			correctIdx: 2,
			expected:   "C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			question := Question{
				Type:          tt.questionType,
				Content:       tt.content,
				CorrectAnswer: tt.correctIdx,
			}

			result := question.GetCorrectAnswerText()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuestion_GetCorrectAnswerText_EdgeCases_Integration(t *testing.T) {
	// Test with invalid options index
	question := Question{
		Type: Vocabulary,
		Content: map[string]interface{}{
			"options": []interface{}{"A", "B"},
		},
		CorrectAnswer: 5, // Out of bounds
	}
	result := question.GetCorrectAnswerText()
	assert.Equal(t, "", result)

	// Test with missing options for vocabulary
	question = Question{
		Type:          Vocabulary,
		Content:       map[string]interface{}{},
		CorrectAnswer: 0,
	}
	result = question.GetCorrectAnswerText()
	assert.Equal(t, "", result)

	// Test with missing answer for fill in blank
	question = Question{
		Type:          FillInBlank,
		Content:       map[string]interface{}{},
		CorrectAnswer: 0,
	}
	result = question.GetCorrectAnswerText()
	assert.Equal(t, "", result)

	// Test with nil content
	question = Question{
		Type:          Vocabulary,
		Content:       nil,
		CorrectAnswer: 0,
	}
	result = question.GetCorrectAnswerText()
	assert.Equal(t, "", result)
}

func TestQuestion_ContentMarshalUnmarshal_Integration(t *testing.T) {
	originalContent := map[string]interface{}{
		"question":    "What is the capital of Italy?",
		"options":     []interface{}{"Milan", "Rome", "Naples", "Turin"},
		"passage":     "Italy is a beautiful country...",
		"explanation": "Rome is the capital of Italy.",
	}

	question := Question{
		ID:      1,
		Type:    Vocabulary,
		Content: originalContent,
	}

	// Test MarshalContentToJSON
	jsonData, err := question.MarshalContentToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "What is the capital of Italy?", parsed["question"])

	// Test UnmarshalContentFromJSON
	var newQuestion Question
	err = newQuestion.UnmarshalContentFromJSON(jsonData)
	require.NoError(t, err)

	assert.Equal(t, originalContent["question"], newQuestion.Content["question"])
	assert.Equal(t, originalContent["passage"], newQuestion.Content["passage"])

	// Check options array
	originalOptions := originalContent["options"].([]interface{})
	newOptions := newQuestion.Content["options"].([]interface{})
	assert.Equal(t, len(originalOptions), len(newOptions))
	for i, opt := range originalOptions {
		assert.Equal(t, opt, newOptions[i])
	}
}

func TestQuestion_ContentMarshalUnmarshal_EmptyContent_Integration(t *testing.T) {
	question := Question{
		ID:      1,
		Type:    Vocabulary,
		Content: map[string]interface{}{"explanation": "No content."},
	}

	// Test marshaling empty content
	// Explanation should be removed from content according to OpenAPI schema
	jsonData, err := question.MarshalContentToJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", jsonData) // Should be empty after removing explanation

	// Test unmarshaling empty content
	var newQuestion Question
	err = newQuestion.UnmarshalContentFromJSON("{}")
	require.NoError(t, err)
	assert.NotNil(t, newQuestion.Content)
	assert.Empty(t, newQuestion.Content)
}

func TestQuestion_ContentMarshalUnmarshal_NilContent_Integration(t *testing.T) {
	question := Question{
		ID:      1,
		Type:    Vocabulary,
		Content: nil,
	}

	// Test marshaling nil content
	jsonData, err := question.MarshalContentToJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", jsonData)

	// Test unmarshaling null
	var newQuestion Question
	err = newQuestion.UnmarshalContentFromJSON("null")
	require.NoError(t, err)
	assert.Nil(t, newQuestion.Content)
}

func TestQuestionType_Validation_Integration(t *testing.T) {
	validTypes := []QuestionType{
		Vocabulary,
		FillInBlank,
		QuestionAnswer,
		ReadingComprehension,
	}

	for _, qtype := range validTypes {
		assert.NotEmpty(t, string(qtype), "Question type should not be empty")
	}

	// Test string conversion
	assert.Equal(t, "vocabulary", string(Vocabulary))
	assert.Equal(t, "fill_blank", string(FillInBlank))
	assert.Equal(t, "qa", string(QuestionAnswer))
	assert.Equal(t, "reading_comprehension", string(ReadingComprehension))
}

func TestQuestionStatus_Validation_Integration(t *testing.T) {
	validStatuses := []QuestionStatus{
		QuestionStatusActive,
		QuestionStatusReported,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, string(status), "Question status should not be empty")
	}

	// Test string conversion
	assert.Equal(t, "active", string(QuestionStatusActive))
	assert.Equal(t, "reported", string(QuestionStatusReported))
}

func TestUserResponse_Validation_Integration(t *testing.T) {
	now := time.Now().UTC()
	response := UserResponse{
		ID:              1,
		UserID:          2,
		QuestionID:      3,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  5000,
		CreatedAt:       now,
	}

	// Verify all fields are set correctly
	assert.Equal(t, 1, response.ID)
	assert.Equal(t, 2, response.UserID)
	assert.Equal(t, 3, response.QuestionID)
	assert.Equal(t, 0, response.UserAnswerIndex)
	assert.True(t, response.IsCorrect)
	assert.Equal(t, 5000, response.ResponseTimeMs)
	assert.Equal(t, now, response.CreatedAt)
}

func TestPerformanceMetrics_Validation_Integration(t *testing.T) {
	now := time.Now().UTC()
	metrics := PerformanceMetrics{
		ID:                   1,
		UserID:               2,
		Language:             "italian",
		Level:                "A1",
		TotalAttempts:        10,
		CorrectAttempts:      7,
		DifficultyAdjustment: 0.8,
		LastUpdated:          now,
	}

	// Verify all fields are set correctly
	assert.Equal(t, 1, metrics.ID)
	assert.Equal(t, 2, metrics.UserID)
	assert.Equal(t, "italian", metrics.Language)
	assert.Equal(t, "A1", metrics.Level)
	assert.Equal(t, 10, metrics.TotalAttempts)
	assert.Equal(t, 7, metrics.CorrectAttempts)
	assert.Equal(t, 0.8, metrics.DifficultyAdjustment)
	assert.Equal(t, now, metrics.LastUpdated)
}

func TestWorkerStatus_Validation_Integration(t *testing.T) {
	now := time.Now().UTC()
	status := WorkerStatus{
		ID:                      1,
		WorkerInstance:          "worker-1",
		IsRunning:               true,
		IsPaused:                false,
		CurrentActivity:         sql.NullString{String: "generating", Valid: true},
		LastHeartbeat:           sql.NullTime{Time: now, Valid: true},
		LastRunStart:            sql.NullTime{Time: now, Valid: true},
		LastRunFinish:           sql.NullTime{Time: now, Valid: true},
		LastRunError:            sql.NullString{String: "", Valid: false},
		TotalQuestionsGenerated: 10,
		TotalRuns:               5,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	// Verify all fields are set correctly
	assert.Equal(t, 1, status.ID)
	assert.Equal(t, "worker-1", status.WorkerInstance)
	assert.True(t, status.IsRunning)
	assert.False(t, status.IsPaused)
	assert.True(t, status.CurrentActivity.Valid)
	assert.Equal(t, "generating", status.CurrentActivity.String)
	assert.True(t, status.LastHeartbeat.Valid)
	assert.Equal(t, now, status.LastHeartbeat.Time)
	assert.True(t, status.LastRunStart.Valid)
	assert.Equal(t, now, status.LastRunStart.Time)
	assert.True(t, status.LastRunFinish.Valid)
	assert.Equal(t, now, status.LastRunFinish.Time)
	assert.False(t, status.LastRunError.Valid)
	assert.Equal(t, 10, status.TotalQuestionsGenerated)
	assert.Equal(t, 5, status.TotalRuns)
	assert.Equal(t, now, status.CreatedAt)
	assert.Equal(t, now, status.UpdatedAt)
}

func TestUserSettings_Validation_Integration(t *testing.T) {
	settings := UserSettings{
		Language: "spanish",
		Level:    "B2",
	}

	assert.Equal(t, "spanish", settings.Language)
	assert.Equal(t, "B2", settings.Level)
}

func TestComplexQuestionContent_Integration(t *testing.T) {
	// Test complex nested content
	complexContent := map[string]interface{}{
		"passage":  "Il clima mediterraneo è caratterizzato da estati calde e secche e inverni miti e piovosi.",
		"question": "Qual è la caratteristica principale del clima mediterraneo?",
		"options": []interface{}{
			"Inverni freddi e nevosi",
			"Estati calde e secche",
			"Piogge tutto l'anno",
			"Temperature costanti",
		},
		"explanation": "Il clima mediterraneo è noto per le sue estati calde e secche.",
		"metadata": map[string]interface{}{
			"difficulty": "intermediate",
			"topic":      "geografia",
			"skills":     []interface{}{"reading", "comprehension"},
		},
	}

	question := Question{
		ID:      1,
		Type:    ReadingComprehension,
		Content: complexContent,
	}

	// Test marshaling
	jsonData, err := question.MarshalContentToJSON()
	require.NoError(t, err)

	// Test unmarshaling
	var newQuestion Question
	err = newQuestion.UnmarshalContentFromJSON(jsonData)
	require.NoError(t, err)

	// Verify complex nested structures
	assert.Equal(t, complexContent["passage"], newQuestion.Content["passage"])
	assert.Equal(t, complexContent["question"], newQuestion.Content["question"])
	assert.Equal(t, complexContent["explanation"], newQuestion.Content["explanation"])

	// Check nested metadata
	metadata := newQuestion.Content["metadata"].(map[string]interface{})
	originalMetadata := complexContent["metadata"].(map[string]interface{})
	assert.Equal(t, originalMetadata["difficulty"], metadata["difficulty"])
	assert.Equal(t, originalMetadata["topic"], metadata["topic"])

	// Check options array
	options := newQuestion.Content["options"].([]interface{})
	originalOptions := complexContent["options"].([]interface{})
	assert.Equal(t, len(originalOptions), len(options))
	for i, opt := range originalOptions {
		assert.Equal(t, opt, options[i])
	}

	// Check skills array in metadata
	skills := metadata["skills"].([]interface{})
	originalSkills := originalMetadata["skills"].([]interface{})
	assert.Equal(t, len(originalSkills), len(skills))
	for i, skill := range originalSkills {
		assert.Equal(t, skill, skills[i])
	}
}
