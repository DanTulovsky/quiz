package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"

	"quizapp/internal/observability"
)

func TestAIService_ConcurrencyControl(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
		Providers: []config.ProviderConfig{
			{
				Name:              "Test Provider",
				Code:              "test",
				URL:               "http://test:11434/v1",
				SupportsGrammar:   true,
				QuestionBatchSize: 3,
				Models: []config.AIModel{
					{
						Name:      "Test Model",
						Code:      "test-model",
						MaxTokens: 4096,
					},
				},
			},
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	t.Run("GetConcurrencyStats", func(t *testing.T) {
		stats := service.GetConcurrencyStats()
		assert.Equal(t, 2, stats.MaxConcurrent)
		assert.Equal(t, 1, stats.MaxPerUser)
		assert.Equal(t, 0, stats.ActiveRequests)
		assert.Equal(t, int64(0), stats.TotalRequests)
		assert.Empty(t, stats.UserActiveCount)
	})

	t.Run("Global semaphore limits", func(t *testing.T) {
		ctx := context.Background()

		// Acquire max concurrent slots
		err1 := service.acquireGlobalSlot(ctx)
		assert.NoError(t, err1)

		err2 := service.acquireGlobalSlot(ctx)
		assert.NoError(t, err2)

		// Third acquisition should fail
		err3 := service.acquireGlobalSlot(ctx)
		assert.Error(t, err3)
		assert.Contains(t, err3.Error(), "AI service at capacity")

		// Release one slot
		service.releaseGlobalSlot(ctx)

		// Now should be able to acquire again
		err4 := service.acquireGlobalSlot(ctx)
		assert.NoError(t, err4)

		// Clean up
		service.releaseGlobalSlot(ctx)
		service.releaseGlobalSlot(ctx)
	})

	t.Run("Per-user limits", func(t *testing.T) {
		ctx := context.Background()
		// User can acquire their slot
		err1 := service.acquireUserSlot(ctx, "user1")
		assert.NoError(t, err1)

		// User cannot acquire second slot (limit is 1)
		err2 := service.acquireUserSlot(ctx, "user1")
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "user concurrency limit exceeded")

		// Different user can acquire their slot
		err3 := service.acquireUserSlot(ctx, "user2")
		assert.NoError(t, err3)

		// Release user1's slot
		service.releaseUserSlot(ctx, "user1")

		// Now user1 can acquire again
		err4 := service.acquireUserSlot(ctx, "user1")
		assert.NoError(t, err4)

		// Clean up
		service.releaseUserSlot(ctx, "user1")
		service.releaseUserSlot(ctx, "user2")
	})

	t.Run("Empty username handling", func(t *testing.T) {
		ctx := context.Background()
		err := service.acquireUserSlot(ctx, "")
		assert.NoError(t, err) // Empty username should be allowed now

		// Release with empty username should not panic
		service.releaseUserSlot(ctx, "")
	})

	t.Run("Stats tracking", func(t *testing.T) {
		// Initially no active requests
		stats := service.GetConcurrencyStats()
		assert.Equal(t, 0, stats.ActiveRequests)

		// Use withConcurrencyControl to properly track active requests
		ctx := context.Background()
		err := service.withConcurrencyControl(ctx, "testuser", func() error {
			// Stats should reflect active requests
			stats = service.GetConcurrencyStats()
			assert.Equal(t, 1, stats.ActiveRequests)
			assert.Equal(t, 1, stats.UserActiveCount["testuser"])
			return nil
		})
		assert.NoError(t, err)

		// Stats should be clean again after operation completes
		stats = service.GetConcurrencyStats()
		assert.Equal(t, 0, stats.ActiveRequests)
		assert.Empty(t, stats.UserActiveCount)
	})
}

func TestAIService_SupportsGrammarField(t *testing.T) {
	// Create a config with AI provider configurations that match config.yaml
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:            "Ollama",
				Code:            "ollama",
				URL:             "http://localhost:11434/v1",
				SupportsGrammar: true,
			},
			{
				Name:            "OpenAI",
				Code:            "openai",
				URL:             "https://api.openai.com/v1",
				SupportsGrammar: true,
			},
			{
				Name:            "Anthropic",
				Code:            "anthropic",
				URL:             "https://api.anthropic.com/v1",
				SupportsGrammar: true,
			},
			{
				Name:            "Google",
				Code:            "google",
				URL:             "https://generativelanguage.googleapis.com/v1beta/openai",
				SupportsGrammar: false,
			},
			{
				Name:            "Custom",
				Code:            "custom",
				SupportsGrammar: true,
			},
		},
	}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	tests := []struct {
		name     string
		provider string
		expected bool
	}{
		{"Ollama supports grammar", "ollama", true},
		{"OpenAI supports grammar", "openai", true},
		{"Anthropic supports grammar", "anthropic", true},
		{"Google does not support grammar", "google", false},
		{"Custom might support grammar", "custom", true},
		{"Unknown provider defaults to false", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.supportsGrammarField(tt.provider)
			assert.Equal(t, tt.expected, result, "Provider %s should have grammar support = %v", tt.provider, tt.expected)
		})
	}
}

func TestAIService_SupportsGrammarField_NoConfig(t *testing.T) {
	// Test backward compatibility when no AI provider configs are loaded
	cfg := &config.Config{
		Providers: nil, // No config loaded
	}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// All providers should default to true for backward compatibility
	tests := []struct {
		name     string
		provider string
		expected bool
	}{
		{"Ollama defaults to false", "ollama", false},
		{"OpenAI defaults to false", "openai", false},
		{"Google defaults to false", "google", false},
		{"Unknown provider defaults to false", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.supportsGrammarField(tt.provider)
			assert.Equal(t, tt.expected, result, "Provider %s should default to grammar support = %v", tt.provider, tt.expected)
		})
	}
}

func TestGetGrammarSchema(t *testing.T) {
	tests := []struct {
		name         string
		questionType models.QuestionType
		expected     string
	}{
		{
			name:         "Vocabulary questions",
			questionType: models.Vocabulary,
			expected:     BatchVocabularyQuestionSchema,
		},
		{
			name:         "Reading comprehension questions",
			questionType: models.ReadingComprehension,
			expected:     BatchReadingComprehensionSchema,
		},
		{
			name:         "Fill in blank questions",
			questionType: models.FillInBlank,
			expected:     BatchQuestionsSchema,
		},
		{
			name:         "Question answer questions",
			questionType: models.QuestionAnswer,
			expected:     BatchQuestionsSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGrammarSchema(tt.questionType)
			if got != tt.expected {
				t.Errorf("getGrammarSchema() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Add a test to ensure the schema is included in the prompt for non-grammar providers
func TestBatchPromptIncludesSchema(t *testing.T) {
	templateManager, err := NewAITemplateManager()
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}
	s := &AIService{
		templateManager: templateManager,
	}
	req := &models.AIQuestionGenRequest{
		Language:     "Italian",
		Level:        "A2",
		QuestionType: models.Vocabulary,
		Count:        5,
	}
	prompt := s.buildBatchQuestionPrompt(context.Background(), req, nil)
	if !strings.Contains(prompt, BatchVocabularyQuestionSchema) {
		t.Errorf("Prompt does not include schema for non-grammar providers")
	}
}

func TestAIService_CleanJSONResponse(t *testing.T) {
	// Create a config with AI provider configurations that match config.yaml
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:            "OpenAI",
				Code:            "openai",
				URL:             "https://api.openai.com/v1",
				SupportsGrammar: true,
			},
			{
				Name:            "Google",
				Code:            "google",
				URL:             "https://generativelanguage.googleapis.com/v1beta/openai",
				SupportsGrammar: false,
			},
		},
	}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	tests := []struct {
		name     string
		response string
		provider string
		expected string
	}{
		{
			name:     "clean JSON for grammar-supporting provider",
			response: `{"question": "test"}`,
			provider: "openai",
			expected: `{"question": "test"}`,
		},
		{
			name:     "markdown code block for non-grammar provider",
			response: "```json\n{\"question\": \"test\"}\n```",
			provider: "google",
			expected: `{"question": "test"}`,
		},
		{
			name:     "generic markdown code block for non-grammar provider",
			response: "```\n{\"question\": \"test\"}\n```",
			provider: "google",
			expected: `{"question": "test"}`,
		},
		{
			name:     "no markdown for non-grammar provider",
			response: `{"question": "test"}`,
			provider: "google",
			expected: `{"question": "test"}`,
		},
		{
			name:     "whitespace handling",
			response: "  ```json\n{\"question\": \"test\"}\n```  ",
			provider: "google",
			expected: `{"question": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.cleanJSONResponse(context.Background(), tt.response, tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAIService_BuildBatchQuestionPrompt_WithVariety(t *testing.T) {
	// Create a config with variety configuration
	cfg := &config.Config{
		Variety: &config.QuestionVarietyConfig{
			TopicCategories: []string{"daily_life", "travel", "work"},
			GrammarFocusByLevel: map[string][]string{
				"B1": {"present_perfect", "past_continuous", "conditionals_0_1"},
			},
			VocabularyDomains:   []string{"food_and_dining", "transportation"},
			Scenarios:           []string{"at_the_airport", "in_a_restaurant"},
			StyleModifiers:      []string{"conversational", "formal"},
			DifficultyModifiers: []string{"basic", "intermediate"},
			TimeContexts:        []string{"morning_routine", "workday"},
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	req := &models.AIQuestionGenRequest{
		Language:              "italian",
		Level:                 "B1",
		QuestionType:          models.Vocabulary,
		Count:                 3,
		RecentQuestionHistory: []string{},
	}

	variety := &VarietyElements{
		TopicCategory:      "travel",
		GrammarFocus:       "present_perfect",
		VocabularyDomain:   "food_and_dining",
		Scenario:           "at_the_airport",
		StyleModifier:      "conversational",
		DifficultyModifier: "basic",
		TimeContext:        "morning_routine",
	}

	prompt := service.buildBatchQuestionPrompt(context.Background(), req, variety)

	// Verify that the prompt contains variety elements
	assert.Contains(t, prompt, "Generate a batch of 3 italian language learning questions for level B1")
	assert.Contains(t, prompt, "Ensure variety in the questions - make each question as different as possible from the others")
	assert.Contains(t, prompt, "Style: Use a conversational tone and approach for these questions.")
	assert.Contains(t, prompt, "Complexity: Focus on basic aspects within the B1 level.")
	assert.Contains(t, prompt, "Grammar Focus: Emphasize present_perfect in the questions.")
	assert.Contains(t, prompt, "Vocabulary Domain: Include vocabulary related to food_and_dining.")
	assert.Contains(t, prompt, "Context: Frame questions around travel situations and scenarios.")
	assert.Contains(t, prompt, "Scenario: Set questions in the context of at_the_airport.")
	assert.Contains(t, prompt, "Time Context: Use morning_routine as the temporal setting for questions.")
}

func TestAIService_GetDifficultyScore_LanguageSpecific(t *testing.T) {
	// Create test configuration with language-specific levels
	cfg := &config.Config{
		LanguageLevels: map[string]config.LanguageLevelConfig{
			"japanese": {
				Levels: []string{"N5", "N4", "N3", "N2", "N1"},
				Descriptions: map[string]string{
					"N5": "Beginner (JLPT)",
					"N4": "Elementary (JLPT)",
					"N3": "Intermediate (JLPT)",
					"N2": "Upper-Intermediate (JLPT)",
					"N1": "Advanced (JLPT)",
				},
			},
			"chinese": {
				Levels: []string{"HSK1", "HSK2", "HSK3", "HSK4", "HSK5", "HSK6"},
				Descriptions: map[string]string{
					"HSK1": "Beginner (HSK)",
					"HSK2": "Elementary (HSK)",
					"HSK3": "Intermediate (HSK)",
					"HSK4": "Upper-Intermediate (HSK)",
					"HSK5": "Advanced (HSK)",
					"HSK6": "Mastery (HSK)",
				},
			},
			"italian": {
				Levels: []string{"A1", "A2", "B1", "B2", "C1", "C2"},
				Descriptions: map[string]string{
					"A1": "Beginner",
					"A2": "Elementary",
					"B1": "Intermediate",
					"B2": "Upper-Intermediate",
					"C1": "Advanced",
					"C2": "Proficient",
				},
			},
		},
	}

	aiService := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	tests := []struct {
		name     string
		level    string
		expected float64
	}{
		// Japanese JLPT levels (5 levels: 0.0, 0.25, 0.5, 0.75, 1.0)
		{"Japanese N5 (beginner)", "N5", 0.0},
		{"Japanese N4 (elementary)", "N4", 0.25},
		{"Japanese N3 (intermediate)", "N3", 0.5},
		{"Japanese N2 (upper-intermediate)", "N2", 0.75},
		{"Japanese N1 (advanced)", "N1", 1.0},

		// Chinese HSK levels (6 levels: 0.0, 0.2, 0.4, 0.6, 0.8, 1.0)
		{"Chinese HSK1 (beginner)", "HSK1", 0.0},
		{"Chinese HSK2 (elementary)", "HSK2", 0.2},
		{"Chinese HSK3 (intermediate)", "HSK3", 0.4},
		{"Chinese HSK4 (upper-intermediate)", "HSK4", 0.6},
		{"Chinese HSK5 (advanced)", "HSK5", 0.8},
		{"Chinese HSK6 (mastery)", "HSK6", 1.0},

		// European CEFR levels (6 levels: 0.0, 0.2, 0.4, 0.6, 0.8, 1.0)
		{"Italian A1 (beginner)", "A1", 0.0},
		{"Italian A2 (elementary)", "A2", 0.2},
		{"Italian B1 (intermediate)", "B1", 0.4},
		{"Italian B2 (upper-intermediate)", "B2", 0.6},
		{"Italian C1 (advanced)", "C1", 0.8},
		{"Italian C2 (proficient)", "C2", 1.0},

		// Unknown levels should return default
		{"Unknown level", "UNKNOWN", 0.5},
		{"Empty level", "", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aiService.getDifficultyScore(tt.level)
			assert.Equal(t, tt.expected, result, "Level: %s", tt.level)
		})
	}
}

func TestAIService_GetDifficultyScore_EmptyConfig(t *testing.T) {
	// Test with empty configuration
	cfg := &config.Config{
		LanguageLevels: map[string]config.LanguageLevelConfig{},
	}

	aiService := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	result := aiService.getDifficultyScore("A1")
	assert.Equal(t, 0.5, result, "Should return default score when no language levels configured")
}

func TestAIService_GetDifficultyScore_LevelOrdering(t *testing.T) {
	// Test that difficulty scores increase with level position
	cfg := &config.Config{
		LanguageLevels: map[string]config.LanguageLevelConfig{
			"test": {
				Levels: []string{"BEGINNER", "INTERMEDIATE", "ADVANCED", "EXPERT"},
			},
		},
	}

	aiService := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Verify that difficulty increases with level position
	assert.Equal(t, 0.0, aiService.getDifficultyScore("BEGINNER"))
	assert.InDelta(t, 0.33, aiService.getDifficultyScore("INTERMEDIATE"), 0.01)
	assert.InDelta(t, 0.67, aiService.getDifficultyScore("ADVANCED"), 0.01)
	assert.Equal(t, 1.0, aiService.getDifficultyScore("EXPERT"))
}

// TestExampleSchemaValidation validates that all example JSON files match their corresponding schemas
func TestExampleSchemaValidation(t *testing.T) {
	templateManager, err := NewAITemplateManager()
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	// Test cases for each question type
	testCases := []struct {
		name         string
		questionType models.QuestionType
		schema       string
		isBatch      bool
	}{
		{
			name:         "vocabulary",
			questionType: models.Vocabulary,
			schema:       BatchVocabularyQuestionSchema,
			isBatch:      true,
		},
		{
			name:         "question_answer",
			questionType: models.QuestionAnswer,
			schema:       BatchQuestionsSchema,
			isBatch:      true,
		},
		{
			name:         "fill_in_blank",
			questionType: models.FillInBlank,
			schema:       BatchQuestionsSchema,
			isBatch:      true,
		},
		{
			name:         "reading_comprehension",
			questionType: models.ReadingComprehension,
			schema:       BatchReadingComprehensionSchema,
			isBatch:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load the example JSON
			exampleContent, err := templateManager.LoadExample(tc.name)
			if err != nil {
				t.Fatalf("Failed to load example for %s: %v", tc.name, err)
			}

			// Parse the example as JSON to validate it's valid JSON
			var exampleData interface{}
			if err := json.Unmarshal([]byte(exampleContent), &exampleData); err != nil {
				t.Fatalf("Example for %s is not valid JSON: %v", tc.name, err)
			}

			// Get the appropriate schema for validation
			var schemaToUse string
			if tc.isBatch {
				// For batch schemas, extract the items schema
				schemaToUse, err = extractItemsSchema(tc.schema)
				if err != nil {
					t.Fatalf("Failed to extract items schema for %s: %v", tc.name, err)
				}
			} else {
				schemaToUse = tc.schema
			}

			// Create JSON schema loader
			schemaLoader := gojsonschema.NewStringLoader(schemaToUse)
			documentLoader := gojsonschema.NewStringLoader(exampleContent)

			// Validate the example against the schema
			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			if err != nil {
				t.Fatalf("Failed to validate schema for %s: %v", tc.name, err)
			}

			if !result.Valid() {
				t.Errorf("Example for %s does not match schema:", tc.name)
				for _, err := range result.Errors() {
					t.Errorf("  - %s", err)
				}
			}
		})
	}
}

func TestSchemaValidationFailureDetails(t *testing.T) {
	// Reset global state
	SchemaValidationMu.Lock()
	SchemaValidationFailures = make(map[models.QuestionType]int)
	SchemaValidationFailureDetails = make(map[models.QuestionType][]string)
	SchemaValidationMu.Unlock()

	qType := models.Vocabulary

	// Simulate 12 failures with different messages
	for i := 1; i <= 12; i++ {
		errMsg := fmt.Sprintf("error %d", i)
		SchemaValidationMu.Lock()
		SchemaValidationFailures[qType]++
		SchemaValidationFailureDetails[qType] = append(SchemaValidationFailureDetails[qType], errMsg)
		if len(SchemaValidationFailureDetails[qType]) > 10 {
			SchemaValidationFailureDetails[qType] = SchemaValidationFailureDetails[qType][len(SchemaValidationFailureDetails[qType])-10:]
		}
		SchemaValidationMu.Unlock()
	}

	SchemaValidationMu.Lock()
	assert.Equal(t, 12, SchemaValidationFailures[qType], "Failure count should be 12")
	assert.Equal(t, 10, len(SchemaValidationFailureDetails[qType]), "Should only keep last 10 error messages")
	for i, msg := range SchemaValidationFailureDetails[qType] {
		assert.Equal(t, fmt.Sprintf("error %d", i+3), msg, "Error message should be correct")
	}
	SchemaValidationMu.Unlock()
}

func TestAIService_ParseQuestionsResponse_ErrorHandling(t *testing.T) {
	cfg := &config.Config{}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	type testCase struct {
		name    string
		input   string
		wantErr string
	}

	tests := []testCase{
		{
			name:    "Empty response",
			input:   "",
			wantErr: "empty response",
		},
		{
			name:    "Malformed JSON",
			input:   "not a json",
			wantErr: "failed to parse ai response as json",
		},
		{
			name:    "Empty JSON array",
			input:   "[]",
			wantErr: "no questions in response",
		},
		{
			name:    "Nil question in array",
			input:   "[null]",
			wantErr: "only invalid or empty questions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			questions, err := service.parseQuestionsResponse(context.Background(), tc.input, "italian", "B1", models.Vocabulary, "openai")
			assert.Error(t, err)
			assert.Nil(t, questions)
			assert.Contains(t, strings.ToLower(err.Error()), tc.wantErr)
		})
	}
}

func TestAIService_ParseQuestionsResponse_NilService(t *testing.T) {
	var service *AIService

	questions, err := service.parseQuestionsResponse(context.Background(), "[]", "italian", "B1", models.Vocabulary, "openai")
	assert.Error(t, err)
	assert.Nil(t, questions)
	assert.Contains(t, err.Error(), "AIService instance is nil")
}

func TestAIService_CreateQuestionFromData_NilService(t *testing.T) {
	var service *AIService

	question, err := service.createQuestionFromData(context.Background(), map[string]interface{}{}, "italian", "B1", models.Vocabulary)
	assert.Error(t, err)
	assert.Nil(t, question)
	assert.Contains(t, err.Error(), "AIService instance is nil")
}

func TestAIService_CreateQuestionFromData_NilData(t *testing.T) {
	cfg := &config.Config{}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	question, err := service.createQuestionFromData(context.Background(), nil, "italian", "B1", models.Vocabulary)
	assert.Error(t, err)
	assert.Nil(t, question)
	assert.Contains(t, err.Error(), "question data is nil")
}

func TestAIService_ParseQuestionsResponse_RealWorldScenarios(t *testing.T) {
	cfg := &config.Config{}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	tests := []struct {
		name        string
		input       string
		description string
		shouldPanic bool
		qType       models.QuestionType
	}{
		{
			name: "Malformed JSON with null values",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato",
					"options": ["sleepy", "calm", "nervous", "hungry"],
					"correct_answer": 2
				},
				null,
				{
					"sentence": "Mentre cenavamo, la mia amica si sentiva nervosa.",
					"question": "nervosa",
					"options": ["happy", "sad", "nervous", "excited"],
					"correct_answer": 2
				}
			]`,
			description: "JSON with null question in array",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Incomplete question data",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato"
				}
			]`,
			description: "Question missing required fields",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name:        "Empty question object",
			input:       `[{}]`,
			description: "Empty question object",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Question with invalid correct_answer type",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato",
					"options": ["sleepy", "calm", "nervous", "hungry"],
					"correct_answer": "invalid"
				}
			]`,
			description: "Invalid correct_answer type",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Question with nil options",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato",
					"options": null,
					"correct_answer": 2
				}
			]`,
			description: "Nil options array",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Question with empty options",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato",
					"options": [],
					"correct_answer": 2
				}
			]`,
			description: "Empty options array",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Question with mixed valid and invalid data",
			input: `[
				{
					"sentence": "Il cameriere era molto lento.",
					"question": "frustrato",
					"options": ["sleepy", "calm", "nervous", "hungry"],
					"correct_answer": 2
				},
				{
					"sentence": "Mentre cenavamo, la mia amica si sentiva nervosa.",
					"question": "nervosa",
					"options": ["happy", "sad", "nervous", "excited"],
					"correct_answer": 2
				},
				null,
				{
					"sentence": "Invalid question",
					"question": "",
					"options": ["a", "b", "c"],
					"correct_answer": 1
				}
			]`,
			description: "Mixed valid and invalid questions",
			shouldPanic: false,
			qType:       models.Vocabulary,
		},
		{
			name: "Valid reading comprehension response",
			input: `[
				{
					"passage": "Mentre cenavamo al ristorante, la mia amica si sentiva un po' nervosa a causa di un leggero mal di stomaco. Il cameriere era molto lento e mi stavo sentendo frustrato perché avevo un forte dolore al dente.",
					"question": "Perché la persona si sentiva frustrata?",
					"options": [
						"Perché il cameriere era lento",
						"Perché aveva fame",
						"Perché il ristorante era rumoroso",
						"Perché la musica era troppo alta"
					],
					"correct_answer": 0,
					"explanation": "La persona si sentiva frustrata perché aveva un forte dolore al dente e il cameriere era molto lento.",
					"topic": "emotions_and_feelings"
				}
			]`,
			description: "Valid reading comprehension question",
			shouldPanic: false,
			qType:       models.ReadingComprehension,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			questions, err := service.parseQuestionsResponse(context.Background(), tc.input, "italian", "B1", tc.qType, "google")

			if tc.name == "Valid reading comprehension response" {
				if err != nil {
					t.Errorf("Unexpected error for valid reading comprehension response: %v\nInput: %s", err, tc.input)
					return
				}
				if len(questions) == 0 {
					t.Errorf("No questions returned for valid reading comprehension response. Input: %s", tc.input)
					return
				}
				for i, q := range questions {
					if q == nil {
						t.Errorf("Question %d is nil for valid reading comprehension response. Input: %s", i, tc.input)
						continue
					}
					if q.Content == nil {
						t.Errorf("Question %d content is nil for valid reading comprehension response. Input: %s", i, tc.input)
						continue
					}
				}
				return
			}

			if tc.shouldPanic {
				assert.Error(t, err)
				assert.Nil(t, questions)
			} else {
				if err != nil {
					t.Logf("Expected error for %s: %v", tc.description, err)
				}
				for i, q := range questions {
					assert.NotNil(t, q, "Question %d should not be nil", i)
					assert.NotEmpty(t, q.Content, "Question %d should have content", i)
				}
			}
		})
	}
}

func TestAIService_ParseQuestionsResponse_EdgeCases(t *testing.T) {
	cfg := &config.Config{}
	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "Very large response",
			input:       `[` + strings.Repeat(`{"sentence":"test","question":"test","options":["a","b","c","d"],"correct_answer":0},`, 1000) + `{"sentence":"test","question":"test","options":["a","b","c","d"],"correct_answer":0}]`,
			description: "Large response with many questions",
		},
		{
			name:        "Response with special characters",
			input:       `[{"sentence":"Il cameriere era molto lento & mi stavo sentendo frustrato.","question":"frustrato","options":["sleepy","calm","nervous","hungry"],"correct_answer":2}]`,
			description: "Response with HTML entities and special chars",
		},
		{
			name:        "Response with unicode characters",
			input:       `[{"sentence":"Il cameriere era molto lento 🍕","question":"frustrato","options":["sleepy","calm","nervous","hungry"],"correct_answer":2}]`,
			description: "Response with unicode emoji",
		},
		{
			name:        "Response with deeply nested objects",
			input:       `[{"sentence":"test","question":"test","options":["a","b","c","d"],"correct_answer":0,"metadata":{"nested":{"deep":{"value":"test"}}}}]`,
			description: "Response with nested metadata",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			questions, err := service.parseQuestionsResponse(context.Background(), tc.input, "italian", "B1", models.Vocabulary, "google")
			if err != nil {
				t.Logf("Expected error for %s: %v", tc.description, err)
			}

			// If we get questions, they should be valid
			for i, q := range questions {
				assert.NotNil(t, q, "Question %d should not be nil", i)
			}
		})
	}
}

func TestAIService_Shutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Initially not shutdown
	assert.False(t, service.isShutdown())

	// Shutdown the service
	err := service.Shutdown(context.Background())
	assert.NoError(t, err)

	// Should now be shutdown
	assert.True(t, service.isShutdown())

	// New operations should fail
	err = service.withConcurrencyControl(context.Background(), "testuser", func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI service is shutting down")
}

func TestAIService_Shutdown_WithActiveRequests(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Use a timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), config.AITestTimeout)
	defer cancel()

	// Shutdown should work even with timeout
	err := service.Shutdown(ctx)
	assert.NoError(t, err)

	// Should be shutdown
	assert.True(t, service.isShutdown())
}

func TestAIService_BuildChatPrompt_VocabularyQuestion(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:              "Test Provider",
				Code:              "test",
				URL:               "http://test:11434/v1",
				SupportsGrammar:   true,
				QuestionBatchSize: 3,
				Models: []config.AIModel{
					{
						Name:      "Test Model",
						Code:      "test-model",
						MaxTokens: 4096,
					},
				},
			},
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test vocabulary question with sentence
	req := &models.AIChatRequest{
		Language:     "italian",
		Level:        "intermediate",
		QuestionType: models.Vocabulary,
		Question:     "What does stazione mean in this context?",
		Options:      []string{"bank", "park", "shop", "station"},
		Passage:      "Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante.",
		UserMessage:  "Translate this question, text and options to English",
	}

	prompt := service.buildChatPrompt(req)

	// Verify the prompt contains the context sentence for vocabulary questions
	assert.Contains(t, prompt, "Context Sentence: Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante.")
	assert.Contains(t, prompt, "Question: What does stazione mean in this context?")
	assert.Contains(t, prompt, "Options:")
	assert.Contains(t, prompt, "- bank")
	assert.Contains(t, prompt, "- park")
	assert.Contains(t, prompt, "- shop")
	assert.Contains(t, prompt, "- station")
	assert.Contains(t, prompt, "Translate this question, text and options to English")
}

func TestAIService_BuildChatPrompt_ReadingComprehensionQuestion(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:              "Test Provider",
				Code:              "test",
				URL:               "http://test:11434/v1",
				SupportsGrammar:   true,
				QuestionBatchSize: 3,
				Models: []config.AIModel{
					{
						Name:      "Test Model",
						Code:      "test-model",
						MaxTokens: 4096,
					},
				},
			},
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test reading comprehension question with passage
	req := &models.AIChatRequest{
		Language:     "italian",
		Level:        "intermediate",
		QuestionType: models.ReadingComprehension,
		Question:     "What is the main topic of this passage?",
		Options:      []string{"Travel", "Food", "Work", "Education"},
		Passage:      "Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale.",
		UserMessage:  "What is this passage about?",
	}

	prompt := service.buildChatPrompt(req)

	// Verify the prompt contains the passage for reading comprehension questions
	assert.Contains(t, prompt, "Passage: Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale.")
	assert.Contains(t, prompt, "Question: What is the main topic of this passage?")
	assert.Contains(t, prompt, "Options:")
	assert.Contains(t, prompt, "- Travel")
	assert.Contains(t, prompt, "- Food")
	assert.Contains(t, prompt, "- Work")
	assert.Contains(t, prompt, "- Education")
	assert.Contains(t, prompt, "What is this passage about?")
}

func TestAIService_BuildChatPrompt_VocabularyWithSentence(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test vocabulary question with sentence
	req := &models.AIChatRequest{
		Language:     "italian",
		Level:        "intermediate",
		QuestionType: models.Vocabulary,
		Question:     "What does stazione mean in this context?",
		Options:      []string{"bank", "park", "shop", "station"},
		Passage:      "Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante.",
		UserMessage:  "Translate this question, text and options to English",
	}

	prompt := service.buildChatPrompt(req)
	assert.Contains(t, prompt, "Context Sentence: Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante.")
	assert.Contains(t, prompt, "Question: What does stazione mean in this context?")
	assert.Contains(t, prompt, "Options:")
	assert.Contains(t, prompt, "bank")
	assert.Contains(t, prompt, "station")
}

func TestAIService_BuildChatPrompt_ReadingComprehensionWithPassage(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test reading comprehension question with passage
	req := &models.AIChatRequest{
		Language:     "italian",
		Level:        "intermediate",
		QuestionType: models.ReadingComprehension,
		Question:     "What is the main topic of this passage?",
		Options:      []string{"Travel", "Food", "Work", "Education"},
		Passage:      "Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale.",
		UserMessage:  "What is this passage about?",
	}

	prompt := service.buildChatPrompt(req)
	assert.Contains(t, prompt, "Passage: Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale.")
	assert.Contains(t, prompt, "Question: What is the main topic of this passage?")
	assert.Contains(t, prompt, "Options:")
	assert.Contains(t, prompt, "Travel")
	assert.Contains(t, prompt, "Education")
}

func TestAIService_BuildChatPrompt_QuestionWithoutPassage(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test question without passage (like fill-in-blank)
	req := &models.AIChatRequest{
		Language:     "italian",
		Level:        "intermediate",
		QuestionType: models.FillInBlank,
		Question:     "Complete the sentence: Io ___ studente.",
		Options:      []string{"sono", "sei", "è", "siamo"},
		UserMessage:  "Help me understand this grammar question",
	}

	prompt := service.buildChatPrompt(req)
	assert.NotContains(t, prompt, "Passage:")
	assert.NotContains(t, prompt, "Context Sentence:")
	assert.Contains(t, prompt, "Question: Complete the sentence: Io ___ studente.")
	assert.Contains(t, prompt, "Options:")
	assert.Contains(t, prompt, "sono")
	assert.Contains(t, prompt, "siamo")
}

func TestAIService_BuildStorySectionPrompt(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxAIConcurrent: 5,
			MaxAIPerUser:    2,
		},
	}

	service := NewAIService(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	t.Run("StorySectionPrompt_WithAllFields", func(t *testing.T) {
		// Test with all optional fields populated
		subject := "A day in the life"
		authorStyle := "Hemingway-style"
		timePeriod := "Modern day"
		genre := "Drama"
		tone := "Reflective"
		characterNames := "Maria, Carlos"
		customInstructions := "Focus on daily routines"

		req := &models.StoryGenerationRequest{
			Language:           "spanish",
			Level:              "intermediate",
			Title:              "Mi Día Perfecto",
			Subject:            &subject,
			AuthorStyle:        &authorStyle,
			TimePeriod:         &timePeriod,
			Genre:              &genre,
			Tone:               &tone,
			CharacterNames:     &characterNames,
			CustomInstructions: &customInstructions,
			SectionLength:      models.SectionLengthMedium,
			PreviousSections:   "Previously: Maria woke up early...",
			IsFirstSection:     false,
			TargetWords:        150,
			TargetSentences:    10,
		}

		prompt := service.buildStorySectionPrompt(req)

		// Verify that the prompt doesn't panic and contains expected content
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "You are an expert Hemingway-style specializing in spanish")
		assert.Contains(t, prompt, "spanish language learning content")
		assert.Contains(t, prompt, "spanish at intermediate proficiency level")
		assert.Contains(t, prompt, "Title: Mi Día Perfecto")
		assert.Contains(t, prompt, "Subject: A day in the life")
		assert.Contains(t, prompt, "Time Period: Modern day")
		assert.Contains(t, prompt, "Genre: Drama")
		assert.Contains(t, prompt, "Tone: Reflective")
		assert.Contains(t, prompt, "Main Characters: Maria, Carlos")
		assert.Contains(t, prompt, "Custom Instructions: Focus on daily routines")
		assert.Contains(t, prompt, "Target approximately 10 sentences (150 words)")
		assert.Contains(t, prompt, "Continue the story naturally from previous sections")
		assert.Contains(t, prompt, "Previously: Maria woke up early...")
	})

	t.Run("StorySectionPrompt_MinimalFields", func(t *testing.T) {
		// Test with minimal fields (similar to what was causing the panic)
		req := &models.StoryGenerationRequest{
			Language:         "french",
			Level:            "beginner",
			Title:            "Simple Story",
			SectionLength:    models.SectionLengthShort,
			PreviousSections: "",
			IsFirstSection:   true,
			TargetWords:      100,
			TargetSentences:  8,
		}

		prompt := service.buildStorySectionPrompt(req)

		// Verify that the prompt doesn't panic and contains expected content
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "You are an expert creative writer specializing in french")
		assert.Contains(t, prompt, "french at beginner proficiency level")
		assert.Contains(t, prompt, "Title: Simple Story")
		assert.Contains(t, prompt, "This is the beginning of a new story.")
		assert.Contains(t, prompt, "Target approximately 8 sentences (100 words)")
		assert.Contains(t, prompt, "Introduce the main characters and setting")
	})
}
