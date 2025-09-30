package models

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name: "complete user with all fields",
			user: User{
				ID:                1,
				Username:          "testuser",
				Email:             sql.NullString{String: "test@example.com", Valid: true},
				Timezone:          sql.NullString{String: "UTC", Valid: true},
				LastActive:        sql.NullTime{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
				PreferredLanguage: sql.NullString{String: "english", Valid: true},
				CurrentLevel:      sql.NullString{String: "B1", Valid: true},
				AIProvider:        sql.NullString{String: "openai", Valid: true},
				AIModel:           sql.NullString{String: "gpt-4", Valid: true},
				AIEnabled:         sql.NullBool{Bool: true, Valid: true},
				CreatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:         time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			expected: `{"id":1,"username":"testuser","email":"test@example.com","timezone":"UTC","last_active":"2023-01-01T12:00:00Z","preferred_language":"english","current_level":"B1","ai_provider":"openai","ai_model":"gpt-4","ai_enabled":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-02T00:00:00Z"}`,
		},
		{
			name: "user with null fields",
			user: User{
				ID:                2,
				Username:          "nulluser",
				Email:             sql.NullString{Valid: false},
				Timezone:          sql.NullString{Valid: false},
				LastActive:        sql.NullTime{Valid: false},
				PreferredLanguage: sql.NullString{Valid: false},
				CurrentLevel:      sql.NullString{Valid: false},
				AIProvider:        sql.NullString{Valid: false},
				AIModel:           sql.NullString{Valid: false},
				AIEnabled:         sql.NullBool{Valid: false},
				CreatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: `{"id":2,"username":"nulluser","email":null,"timezone":null,"last_active":null,"preferred_language":null,"current_level":null,"ai_provider":null,"ai_model":null,"ai_enabled":null,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`,
		},
		{
			name: "user with mixed null and valid fields",
			user: User{
				ID:                3,
				Username:          "mixeduser",
				Email:             sql.NullString{String: "mixed@example.com", Valid: true},
				Timezone:          sql.NullString{Valid: false},
				LastActive:        sql.NullTime{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
				PreferredLanguage: sql.NullString{Valid: false},
				CurrentLevel:      sql.NullString{String: "A2", Valid: true},
				AIProvider:        sql.NullString{Valid: false},
				AIModel:           sql.NullString{String: "gpt-3.5", Valid: true},
				AIEnabled:         sql.NullBool{Bool: false, Valid: true},
				CreatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:         time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: `{"id":3,"username":"mixeduser","email":"mixed@example.com","timezone":null,"last_active":"2023-01-01T12:00:00Z","preferred_language":null,"current_level":"A2","ai_provider":null,"ai_model":"gpt-3.5","ai_enabled":false,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.user)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

// TestUserResponse_MarshalJSON verifies that our custom MarshalJSON logic for UserResponse
// correctly handles sql.NullInt32 fields: valid values become numbers, invalid become null in JSON.
// This is a regression test for our custom marshaling, not for Go's encoding/json package.
func TestUserResponse_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		response UserResponse
		expected string
	}{
		{
			name: "response with confidence level",
			response: UserResponse{
				ID:              1,
				UserID:          123,
				QuestionID:      456,
				UserAnswerIndex: 0,
				IsCorrect:       true,
				ResponseTimeMs:  1500,
				ConfidenceLevel: sql.NullInt32{Int32: 8, Valid: true},
				CreatedAt:       time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: `{"id":1,"user_id":123,"question_id":456,"user_answer_index":0,"is_correct":true,"response_time_ms":1500,"confidence_level":8,"created_at":"2023-01-01T12:00:00Z"}`,
		},
		{
			name: "response without confidence level",
			response: UserResponse{
				ID:              2,
				UserID:          123,
				QuestionID:      456,
				UserAnswerIndex: 1,
				IsCorrect:       false,
				ResponseTimeMs:  2000,
				ConfidenceLevel: sql.NullInt32{Int32: 0, Valid: false},
				CreatedAt:       time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: `{"id":2,"user_id":123,"question_id":456,"user_answer_index":1,"is_correct":false,"response_time_ms":2000,"confidence_level":null,"created_at":"2023-01-01T12:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestWorkerStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   WorkerStatus
		expected string
	}{
		{
			name: "active worker status",
			status: WorkerStatus{
				ID:                      1,
				WorkerInstance:          "worker-1",
				IsRunning:               true,
				IsPaused:                false,
				CurrentActivity:         sql.NullString{String: "Generating questions", Valid: true},
				LastHeartbeat:           sql.NullTime{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
				LastRunStart:            sql.NullTime{Time: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC), Valid: true},
				LastRunFinish:           sql.NullTime{Valid: false},
				LastRunError:            sql.NullString{Valid: false},
				TotalQuestionsGenerated: 150,
				TotalRuns:               10,
				CreatedAt:               time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:               time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: `{"id":1,"worker_instance":"worker-1","is_running":true,"is_paused":false,"current_activity":"Generating questions","last_heartbeat":"2023-01-01T12:00:00Z","last_run_start":"2023-01-01T11:00:00Z","last_run_end":null,"last_run_finish":null,"last_run_error":null,"total_questions_processed":0,"total_questions_generated":150,"total_runs":10,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T12:00:00Z"}`,
		},
		{
			name: "paused worker status",
			status: WorkerStatus{
				ID:                      2,
				WorkerInstance:          "worker-2",
				IsRunning:               false,
				IsPaused:                true,
				CurrentActivity:         sql.NullString{Valid: false},
				LastHeartbeat:           sql.NullTime{Time: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
				LastRunStart:            sql.NullTime{Valid: false},
				LastRunFinish:           sql.NullTime{Time: time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC), Valid: true},
				LastRunError:            sql.NullString{String: "Connection timeout", Valid: true},
				TotalQuestionsGenerated: 75,
				TotalRuns:               5,
				CreatedAt:               time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:               time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			expected: `{"id":2,"worker_instance":"worker-2","is_running":false,"is_paused":true,"current_activity":null,"last_heartbeat":"2023-01-01T10:00:00Z","last_run_start":null,"last_run_end":null,"last_run_finish":"2023-01-01T09:00:00Z","last_run_error":"Connection timeout","total_questions_processed":0,"total_questions_generated":75,"total_runs":5,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T10:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestQuestion_GetCorrectAnswerText(t *testing.T) {
	tests := []struct {
		name     string
		question Question
		expected string
	}{
		{
			name: "vocabulary question",
			question: Question{
				Type:          Vocabulary,
				CorrectAnswer: 1, // index 1 is "Hello"
				Content: map[string]interface{}{
					"question": "What does 'bonjour' mean?",
					"options":  []interface{}{"Goodbye", "Hello", "Thank you", "Please"},
				},
			},
			expected: "Hello",
		},
		{
			name: "fill in blank question",
			question: Question{
				Type:          FillInBlank,
				CorrectAnswer: 0,
				Content: map[string]interface{}{
					"question": "Complete: 'Je ___ français'",
					"options":  []interface{}{"suis", "es", "est", "sont"},
				},
			},
			expected: "suis",
		},
		{
			name: "question answer question",
			question: Question{
				Type:          QuestionAnswer,
				CorrectAnswer: 1,
				Content: map[string]interface{}{
					"question": "What is the capital of France?",
					"options":  []interface{}{"London", "Paris", "Berlin", "Madrid"},
				},
			},
			expected: "Paris",
		},
		{
			name: "reading comprehension question",
			question: Question{
				Type:          ReadingComprehension,
				CorrectAnswer: 3,
				Content: map[string]interface{}{
					"passage":  "Le chat dort sur le sofa.",
					"question": "Where is the cat sleeping?",
					"options":  []interface{}{"On the bed", "On the floor", "On the chair", "On the sofa"},
				},
			},
			expected: "On the sofa",
		},
		{
			name: "invalid content structure",
			question: Question{
				Type:          Vocabulary,
				CorrectAnswer: 0,
				Content: map[string]interface{}{
					"question": "Test question",
					// Missing options
				},
			},
			expected: "",
		},
		{
			name: "options not a slice",
			question: Question{
				Type:          Vocabulary,
				CorrectAnswer: 0,
				Content: map[string]interface{}{
					"question": "Test question",
					"options":  "not a slice",
				},
			},
			expected: "",
		},
		{
			name: "correct answer out of bounds",
			question: Question{
				Type:          Vocabulary,
				CorrectAnswer: 10, // Out of bounds
				Content: map[string]interface{}{
					"question": "Test question",
					"options":  []interface{}{"Option 1", "Option 2"},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.question.GetCorrectAnswerText()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuestion_MarshalContentToJSON(t *testing.T) {
	tests := []struct {
		name     string
		question Question
		expected string
		hasError bool
	}{
		{
			name: "valid content",
			question: Question{
				Content: map[string]interface{}{
					"question": "Test question?",
					"options":  []string{"A", "B", "C", "D"},
				},
			},
			expected: `{"options":["A","B","C","D"],"question":"Test question?"}`,
			hasError: false,
		},
		{
			name: "empty content",
			question: Question{
				Content: map[string]interface{}{},
			},
			expected: `{}`,
			hasError: false,
		},
		{
			name: "nil content",
			question: Question{
				Content: nil,
			},
			expected: `null`,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.question.MarshalContentToJSON()
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Parse both JSON strings and compare the parsed objects
				var expectedObj, actualObj map[string]interface{}
				err1 := json.Unmarshal([]byte(tt.expected), &expectedObj)
				err2 := json.Unmarshal([]byte(result), &actualObj)
				assert.NoError(t, err1)
				assert.NoError(t, err2)
				assert.Equal(t, expectedObj, actualObj)
			}
		})
	}
}

func TestQuestion_UnmarshalContentFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected map[string]interface{}
		hasError bool
	}{
		{
			name:     "valid JSON",
			jsonData: `{"question":"Test question?","options":["A","B","C","D"]}`,
			expected: map[string]interface{}{
				"question": "Test question?",
				"options":  []interface{}{"A", "B", "C", "D"},
			},
			hasError: false,
		},
		{
			name:     "empty JSON object",
			jsonData: `{}`,
			expected: map[string]interface{}{},
			hasError: false,
		},
		{
			name:     "null JSON",
			jsonData: `null`,
			expected: nil,
			hasError: false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{invalid json}`,
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			question := Question{}
			err := question.UnmarshalContentFromJSON(tt.jsonData)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, question.Content)
			}
		})
	}
}

func TestPerformanceMetrics_AccuracyRate(t *testing.T) {
	tests := []struct {
		name     string
		metrics  PerformanceMetrics
		expected float64
	}{
		{
			name: "perfect accuracy",
			metrics: PerformanceMetrics{
				TotalAttempts:   10,
				CorrectAttempts: 10,
			},
			expected: 100.0, // Returns percentage, not decimal
		},
		{
			name: "half accuracy",
			metrics: PerformanceMetrics{
				TotalAttempts:   10,
				CorrectAttempts: 5,
			},
			expected: 50.0, // Returns percentage, not decimal
		},
		{
			name: "zero accuracy",
			metrics: PerformanceMetrics{
				TotalAttempts:   10,
				CorrectAttempts: 0,
			},
			expected: 0.0,
		},
		{
			name: "no attempts",
			metrics: PerformanceMetrics{
				TotalAttempts:   0,
				CorrectAttempts: 0,
			},
			expected: 0.0,
		},
		{
			name: "partial accuracy",
			metrics: PerformanceMetrics{
				TotalAttempts:   7,
				CorrectAttempts: 3,
			},
			expected: 42.857142857142854, // 3/7 * 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metrics.AccuracyRate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNullValueHelpers(t *testing.T) {
	t.Run("nullStringToPointer", func(t *testing.T) {
		// Valid string
		ns := sql.NullString{String: "test", Valid: true}
		result := nullStringToPointer(ns)
		require.NotNil(t, result)
		assert.Equal(t, "test", *result)

		// Invalid string
		ns = sql.NullString{Valid: false}
		result = nullStringToPointer(ns)
		assert.Nil(t, result)
	})

	t.Run("nullTimeToPointer", func(t *testing.T) {
		testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

		// Valid time
		nt := sql.NullTime{Time: testTime, Valid: true}
		result := nullTimeToPointer(nt)
		require.NotNil(t, result)
		assert.Equal(t, testTime, *result)

		// Invalid time
		nt = sql.NullTime{Valid: false}
		result = nullTimeToPointer(nt)
		assert.Nil(t, result)
	})

	t.Run("nullBoolToPointer", func(t *testing.T) {
		// Valid bool
		nb := sql.NullBool{Bool: true, Valid: true}
		result := nullBoolToPointer(nb)
		require.NotNil(t, result)
		assert.True(t, *result)

		// Invalid bool
		nb = sql.NullBool{Valid: false}
		result = nullBoolToPointer(nb)
		assert.Nil(t, result)
	})

	t.Run("nullInt32ToPointer", func(t *testing.T) {
		// Valid int32
		ni := sql.NullInt32{Int32: 42, Valid: true}
		result := nullInt32ToPointer(ni)
		require.NotNil(t, result)
		assert.Equal(t, int32(42), *result)

		// Invalid int32
		ni = sql.NullInt32{Valid: false}
		result = nullInt32ToPointer(ni)
		assert.Nil(t, result)
	})
}

func TestQuestionTypeConstants(t *testing.T) {
	assert.Equal(t, QuestionType("vocabulary"), Vocabulary)
	assert.Equal(t, QuestionType("fill_blank"), FillInBlank)
	assert.Equal(t, QuestionType("qa"), QuestionAnswer)
	assert.Equal(t, QuestionType("reading_comprehension"), ReadingComprehension)
}

func TestQuestionStatusConstants(t *testing.T) {
	assert.Equal(t, QuestionStatus("active"), QuestionStatusActive)
	assert.Equal(t, QuestionStatus("reported"), QuestionStatusReported)
}

func TestUserAPIKey_JSONOmission(t *testing.T) {
	// Test that APIKey field is omitted from JSON for security
	userAPIKey := UserAPIKey{
		ID:        1,
		UserID:    123,
		Provider:  "openai",
		APIKey:    "secret-key-should-not-appear",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(userAPIKey)
	require.NoError(t, err)

	// Verify APIKey is not in the JSON
	assert.NotContains(t, string(data), "secret-key-should-not-appear")
	assert.NotContains(t, string(data), "api_key")

	// Verify other fields are present
	assert.Contains(t, string(data), "openai")
	assert.Contains(t, string(data), "123")
}

func TestUser_PasswordHashOmission(t *testing.T) {
	// Test that PasswordHash field is omitted from JSON for security
	user := User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: sql.NullString{String: "hashed-password-should-not-appear", Valid: true},
		CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	// Verify PasswordHash is not in the JSON
	assert.NotContains(t, string(data), "hashed-password-should-not-appear")
	assert.NotContains(t, string(data), "password_hash")

	// Verify other fields are present
	assert.Contains(t, string(data), "testuser")
	assert.Contains(t, string(data), "1")
}

func TestUser_MarshalJSON_WithRoles(t *testing.T) {
	user := User{
		ID:        1,
		Username:  "testuser",
		Email:     sql.NullString{String: "test@example.com", Valid: true},
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Roles: []Role{
			{
				ID:          1,
				Name:        "user",
				Description: "Normal site access",
				CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:          2,
				Name:        "admin",
				Description: "Administrative access to all features",
				CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	// Verify roles are included in JSON
	assert.Contains(t, string(data), `"roles":[`)
	assert.Contains(t, string(data), `"name":"user"`)
	assert.Contains(t, string(data), `"name":"admin"`)
	assert.Contains(t, string(data), `"description":"Normal site access"`)
	assert.Contains(t, string(data), `"description":"Administrative access to all features"`)
}

func TestUser_MarshalJSON_WithoutRoles(t *testing.T) {
	user := User{
		ID:        1,
		Username:  "testuser",
		Email:     sql.NullString{String: "test@example.com", Valid: true},
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Roles:     []Role{}, // Empty roles slice
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	// Verify roles field is omitted when empty
	assert.NotContains(t, string(data), `"roles"`)
}

func TestRole_JSONMarshaling(t *testing.T) {
	role := Role{
		ID:          1,
		Name:        "admin",
		Description: "Administrative access to all features",
		CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(role)
	require.NoError(t, err)

	expected := `{"id":1,"name":"admin","description":"Administrative access to all features","created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`
	assert.Equal(t, expected, string(data))
}

func TestUserRole_JSONMarshaling(t *testing.T) {
	userRole := UserRole{
		ID:        1,
		UserID:    123,
		RoleID:    456,
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(userRole)
	require.NoError(t, err)

	expected := `{"id":1,"user_id":123,"role_id":456,"created_at":"2023-01-01T00:00:00Z"}`
	assert.Equal(t, expected, string(data))
}
