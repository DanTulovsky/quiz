//go:build integration
// +build integration

package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAIServiceWithServer(t *testing.T, handler http.HandlerFunc) (*AIService, func()) {
	ts := httptest.NewServer(handler)
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             ts.URL,
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, observability.NewLogger(nil))
	return service, ts.Close
}

func TestAIService_GenerateQuestion_MalformedResponse_Integration(t *testing.T) {
	service, cleanup := newTestAIServiceWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"not_a_question": true}`))
	})
	defer cleanup()

	ctx := context.Background()
	userCfg := &UserAIConfig{Provider: "testai", Model: "test-model"}
	genReq := &models.AIQuestionGenRequest{QuestionType: models.Vocabulary, Language: "english", Level: "A1"}

	q, err := service.GenerateQuestion(ctx, userCfg, genReq)
	require.Error(t, err)
	require.Nil(t, q)
}

func TestAIService_GenerateQuestion_ProviderError_Integration(t *testing.T) {
	service, cleanup := newTestAIServiceWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	})
	defer cleanup()

	ctx := context.Background()
	userCfg := &UserAIConfig{Provider: "testai", Model: "test-model"}
	genReq := &models.AIQuestionGenRequest{QuestionType: models.Vocabulary, Language: "english", Level: "A1"}

	q, err := service.GenerateQuestion(ctx, userCfg, genReq)
	require.Error(t, err)
	require.Nil(t, q)
}

func TestAIService_GenerateQuestions_SchemaValidationFailure_Integration(t *testing.T) {
	service, cleanup := newTestAIServiceWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"question": "Q?", "options": ["A", "B", "C", "D"], "correct_answer": 0}]`)) // missing required fields
	})
	defer cleanup()

	ctx := context.Background()
	userCfg := &UserAIConfig{Provider: "testai", Model: "test-model"}
	genReq := &models.AIQuestionGenRequest{QuestionType: models.Vocabulary, Language: "english", Level: "A1"}

	qs, err := service.GenerateQuestions(ctx, userCfg, genReq)
	require.Error(t, err)
	require.Nil(t, qs)
}

func TestAIService_GenerateQuestion_ShutdownDuringRequest_Integration(t *testing.T) {
	block := make(chan struct{})
	service, cleanup := newTestAIServiceWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-block // block until shutdown
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"question": "Q?", "options": ["A", "B", "C", "D"], "correct_answer": 0, "explanation": "", "topic": ""}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	userCfg := &UserAIConfig{Provider: "testai", Model: "test-model"}
	genReq := &models.AIQuestionGenRequest{QuestionType: models.Vocabulary, Language: "english", Level: "A1"}

	done := make(chan error, 1)
	go func() {
		_, err := service.GenerateQuestion(ctx, userCfg, genReq)
		done <- err
	}()

	// Wait for context timeout
	err := <-done
	require.Error(t, err)
	close(block)
}

func TestAIService_AddJSONStructureGuidance_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test with different question types
	testCases := []struct {
		name             string
		questionType     models.QuestionType
		expectedContains string
	}{
		{"vocabulary", models.Vocabulary, "vocabulary"},
		{"qa", models.QuestionAnswer, "qa"},
		{"reading_comprehension", models.ReadingComprehension, "reading"},
		{"fill_blank", models.FillInBlank, "fill"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prompt := "Generate a question"
			result := service.addJSONStructureGuidance(prompt, tc.questionType)

			assert.Contains(t, result, prompt)
			assert.Contains(t, result, "JSON")
			assert.Contains(t, result, "structure")
		})
	}
}

func TestAIService_GenerateQuestionsStream_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	userConfig := &UserAIConfig{
		Provider: "openai",
		Model:    "gpt-35-turbo",
		APIKey:   "test-key",
		Username: "testuser",
	}

	req := &models.AIQuestionGenRequest{
		Language:     "italian",
		Level:        "A1",
		QuestionType: models.Vocabulary,
		Count:        2,
	}

	progress := make(chan *models.Question, 10)
	variety := &VarietyElements{
		TopicCategory: "food",
	}

	// Test with context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := service.GenerateQuestionsStream(ctx, userConfig, req, progress, variety)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Test with valid context but no AI service (should fail gracefully)
	progress = make(chan *models.Question, 10) // new channel
	ctx = context.Background()
	err = service.GenerateQuestionsStream(ctx, userConfig, req, progress, variety)
	assert.Error(t, err) // Should fail due to no real AI service
}

func TestAIService_GenerateChatResponse_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	userConfig := &UserAIConfig{
		Provider: "openai",
		Model:    "gpt-35-turbo",
		APIKey:   "test-key",
		Username: "testuser",
	}

	req := &models.AIChatRequest{
		UserMessage: "Hello, how are you?",
	}

	// Test with no real AI service (should fail gracefully)
	response, err := service.GenerateChatResponse(context.Background(), userConfig, req)
	assert.Error(t, err)
	assert.Empty(t, response)
}

func TestAIService_GenerateChatResponseStream_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}, {
			Name:            "OpenAI",
			Code:            "openai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	userConfig := &UserAIConfig{
		Provider: "openai",
		Model:    "gpt-35-turbo",
		APIKey:   "test-key",
		Username: "testuser",
	}

	req := &models.AIChatRequest{
		UserMessage: "Hello, how are you?",
	}

	chunks := make(chan string, 10)
	// Test with context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := service.GenerateChatResponseStream(ctx, userConfig, req, chunks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Test with valid context but no AI service (should fail gracefully)
	ctx = context.Background()
	err = service.GenerateChatResponseStream(ctx, userConfig, req, chunks)
	assert.Error(t, err) // Should fail due to no real AI service
}

func TestAIService_TestConnection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test with invalid API key
	err := service.TestConnection(context.Background(), "openai", "gpt-3.5-turbo", "invalid-key")
	assert.Error(t, err)

	// Test with empty API key
	err = service.TestConnection(context.Background(), "openai", "gpt-3.5-turbo", "")
	assert.Error(t, err)

	// Test with invalid provider
	err = service.TestConnection(context.Background(), "invalid-provider", "gpt-3.5-turbo", "test-key")
	assert.Error(t, err)
}

func TestAIService_BuildBatchQuestionPromptWithJSONStructure_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	req := &models.AIQuestionGenRequest{
		Language:     "italian",
		Level:        "A1",
		QuestionType: models.Vocabulary,
		Count:        2,
	}

	variety := &VarietyElements{
		TopicCategory: "food",
	}

	// Test with variety elements
	prompt := service.buildBatchQuestionPromptWithJSONStructure(context.Background(), req, variety)
	assert.Contains(t, prompt, "italian")
	assert.Contains(t, prompt, "A1")
	assert.Contains(t, prompt, "vocabulary")
	assert.Contains(t, prompt, "JSON")

	// Test without variety elements
	prompt = service.buildBatchQuestionPromptWithJSONStructure(context.Background(), req, nil)
	assert.Contains(t, prompt, "italian")
	assert.Contains(t, prompt, "A1")
	assert.Contains(t, prompt, "vocabulary")
	assert.Contains(t, prompt, "JSON")
}

func TestAIService_BuildChatPrompt_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	req := &models.AIChatRequest{
		UserMessage: "Hello, how are you?",
	}

	prompt := service.buildChatPrompt(req)
	assert.Contains(t, prompt, "Hello, how are you?")
}

func TestAIService_FilterThinkingContent_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test with thinking model
	content := "Let me think about this...\nThe answer is: Hello"
	result := service.filterThinkingContent(content, "gpt-4")
	assert.Equal(t, "Hello", result)

	// Test with non-thinking model
	content = "Let me think about this...\nThe answer is: Hello"
	result = service.filterThinkingContent(content, "gpt-30.5bo")
	assert.Equal(t, content, result) // Should not filter

	// Test with empty content
	result = service.filterThinkingContent("", "gpt-4")
	assert.Equal(t, "", result)
}

func TestAIService_IsThinkingModel_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test thinking models
	assert.True(t, service.isThinkingModel("gpt-4"))
	assert.True(t, service.isThinkingModel("gpt-4-turbo"))
	assert.True(t, service.isThinkingModel("claude-3"))

	// Test non-thinking models
	assert.False(t, service.isThinkingModel("gpt-3.5turbo"))
	assert.False(t, service.isThinkingModel("text-davinci-003"))
	assert.False(t, service.isThinkingModel("unknown-model"))
}

func TestAIService_ParseQuestionResponse_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test with valid JSON response
	validResponse := `{"sentence": "Ciao, come stai oggi?", "question": "Ciao", "options": ["Hello", "Goodbye", "Thank you", "Please"], "correct_answer": 0, "explanation": "Ciao means hello in Italian", "topic": "greetings"}`

	question, err := service.parseQuestionResponse(context.Background(), validResponse, "italian", "A1", models.Vocabulary, "openai")
	assert.NoError(t, err)
	assert.NotNil(t, question)
	assert.Equal(t, "italian", question.Language)
	assert.Equal(t, "A1", question.Level)
	assert.Equal(t, models.Vocabulary, question.Type)

	// Test with invalid JSON response
	invalidResponse := `{invalid json}`
	question, err = service.parseQuestionResponse(context.Background(), invalidResponse, "italian", "A1", models.Vocabulary, "openai")
	assert.Error(t, err)
	assert.Nil(t, question)

	// Test with empty response
	question, err = service.parseQuestionResponse(context.Background(), "", "italian", "A1", models.Vocabulary, "openai")
	assert.Error(t, err)
	assert.Nil(t, question)
}

func TestAIService_GetQuestionBatchSize_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:              "OpenAI",
			Code:              "openai",
			URL:               "http://localhost:8080",
			SupportsGrammar:   true,
			QuestionBatchSize: 5,
		}, {
			Name:              "Google",
			Code:              "google",
			URL:               "http://localhost:8081",
			SupportsGrammar:   false,
			QuestionBatchSize: 3,
		}, {
			Name:              "Anthropic",
			Code:              "anthropic",
			URL:               "http://localhost:8082",
			SupportsGrammar:   true,
			QuestionBatchSize: 2,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test different providers
	assert.Equal(t, 5, service.getQuestionBatchSize("openai"))
	assert.Equal(t, 3, service.getQuestionBatchSize("google"))
	assert.Equal(t, 2, service.getQuestionBatchSize("anthropic"))
	assert.Equal(t, 1, service.getQuestionBatchSize("unknown-provider"))
}

func TestAIService_SupportsGrammarField_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "OpenAI",
			Code:            "openai",
			URL:             "http://localhost:8080",
			SupportsGrammar: true,
		}, {
			Name:            "Anthropic",
			Code:            "anthropic",
			URL:             "http://localhost:8082",
			SupportsGrammar: true,
		}, {
			Name:            "Google",
			Code:            "google",
			URL:             "http://localhost:8081",
			SupportsGrammar: false,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test providers that support grammar
	assert.True(t, service.supportsGrammarField("openai"))
	assert.True(t, service.supportsGrammarField("anthropic"))

	// Test providers that dont support grammar
	assert.False(t, service.supportsGrammarField("google"))
	assert.False(t, service.supportsGrammarField("unknown-provider")) // default is false
}

func TestAIService_GetMaxTokensForModel_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test max tokens for different providers
	assert.Equal(t, 4000, service.getMaxTokensForModel("openai", "gpt-4"))
	assert.Equal(t, 4000, service.getMaxTokensForModel("google", "ini-pro"))
	assert.Equal(t, 4000, service.getMaxTokensForModel("anthropic", "claude-3"))
	assert.Equal(t, 4000, service.getMaxTokensForModel("unknown", "unknown-model"))
}

func TestAIService_GetConcurrencyStats_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	stats := service.GetConcurrencyStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.ActiveRequests)
	assert.Equal(t, 2, stats.MaxConcurrent) // From config MaxAIConcurrent: 2
	assert.Equal(t, 0, stats.QueuedRequests)
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.NotNil(t, stats.UserActiveCount)
	assert.Equal(t, 1, stats.MaxPerUser) // From config MaxAIPerUser: 1
}

func TestAIService_VarietyService_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	varietyService := service.VarietyService()
	assert.NotNil(t, varietyService)
}

func TestAIService_Shutdown_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{
			Name:            "TestAI",
			Code:            "testai",
			URL:             "http://localhost:8080", // Mock URL for testing
			SupportsGrammar: true,
		}},
		Server: config.ServerConfig{
			MaxAIConcurrent: 2,
			MaxAIPerUser:    1,
		},
	}
	service := NewAIService(cfg, logger)

	// Test shutdown
	err := service.Shutdown(context.Background())
	assert.NoError(t, err)

	// Test shutdown when already shut down
	err = service.Shutdown(context.Background())
	assert.NoError(t, err) // Should not error on repeated shutdown
}
