//go:build integration
// +build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
)

// MockAIService implements AIServiceInterface for testing
type MockAIService struct {
	realService *services.AIService
}

func NewMockAIService(cfg *config.Config, logger *observability.Logger) *MockAIService {
	return &MockAIService{
		realService: services.NewAIService(cfg, logger),
	}
}

// TestConnection returns a mock response for AI connection tests
func (m *MockAIService) TestConnection(ctx context.Context, provider, model, apiKey string) error {
	// For testing purposes, return success for valid-looking inputs
	if provider != "" && model != "" {
		// If it's a test API key, return an error to simulate failure
		if strings.Contains(apiKey, "test") || apiKey == "" {
			return fmt.Errorf("invalid API key")
		}
		return nil
	}
	return fmt.Errorf("missing provider or model")
}

// CallWithPrompt returns a mock response for AI fix requests, otherwise delegates to real service
func (m *MockAIService) CallWithPrompt(ctx context.Context, userConfig *services.UserAIConfig, prompt, grammar string) (string, error) {
	// Check if this is an AI fix request by looking for fix-related keywords in the prompt
	if strings.Contains(prompt, "fix") || strings.Contains(prompt, "Fix") ||
		strings.Contains(prompt, "problematic") || strings.Contains(prompt, "report") {
		// Return a mock AI fix response
		mockResponse := map[string]interface{}{
			"content": map[string]interface{}{
				"question":       "What is the capital of France?",
				"options":        []string{"Paris", "London", "Berlin", "Madrid"},
				"correct_answer": 0,
				"explanation":    "Paris is the capital and largest city of France.",
			},
			"correct_answer": 0,
			"explanation":    "Paris is the capital and largest city of France.",
			"change_reason":  "Fixed grammar and improved clarity of the question.",
		}

		responseJSON, err := json.Marshal(mockResponse)
		if err != nil {
			return "", err
		}
		return string(responseJSON), nil
	}

	// For non-fix requests, delegate to the real service
	if m.realService != nil {
		return m.realService.CallWithPrompt(ctx, userConfig, prompt, grammar)
	}

	// Fallback response
	return `{"response": "Mock AI response"}`, nil
}

// Implement other required methods by delegating to real service or returning defaults
func (m *MockAIService) GenerateQuestion(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIQuestionGenRequest) (*models.Question, error) {
	if m.realService != nil {
		return m.realService.GenerateQuestion(ctx, userConfig, req)
	}
	return nil, fmt.Errorf("GenerateQuestion not implemented in mock")
}

func (m *MockAIService) GenerateQuestions(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIQuestionGenRequest) ([]*models.Question, error) {
	if m.realService != nil {
		return m.realService.GenerateQuestions(ctx, userConfig, req)
	}
	return nil, fmt.Errorf("GenerateQuestions not implemented in mock")
}

func (m *MockAIService) GenerateQuestionsStream(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *services.VarietyElements) error {
	if m.realService != nil {
		return m.realService.GenerateQuestionsStream(ctx, userConfig, req, progress, variety)
	}
	return fmt.Errorf("GenerateQuestionsStream not implemented in mock")
}

func (m *MockAIService) GenerateChatResponse(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIChatRequest) (string, error) {
	if m.realService != nil {
		return m.realService.GenerateChatResponse(ctx, userConfig, req)
	}
	return "Mock chat response", nil
}

func (m *MockAIService) GenerateChatResponseStream(ctx context.Context, userConfig *services.UserAIConfig, req *models.AIChatRequest, chunks chan<- string) error {
	if m.realService != nil {
		return m.realService.GenerateChatResponseStream(ctx, userConfig, req, chunks)
	}
	select {
	case chunks <- "Mock streaming response":
	default:
	}
	return nil
}

func (m *MockAIService) GetConcurrencyStats() services.ConcurrencyStats {
	if m.realService != nil {
		return m.realService.GetConcurrencyStats()
	}
	return services.ConcurrencyStats{}
}

func (m *MockAIService) GetQuestionBatchSize(provider string) int {
	if m.realService != nil {
		return m.realService.GetQuestionBatchSize(provider)
	}
	return 1
}

func (m *MockAIService) VarietyService() *services.VarietyService {
	if m.realService != nil {
		return m.realService.VarietyService()
	}
	return nil
}

func (m *MockAIService) TemplateManager() *services.AITemplateManager {
	if m.realService != nil {
		return m.realService.TemplateManager()
	}
	return nil
}

func (m *MockAIService) SupportsGrammarField(provider string) bool {
	if m.realService != nil {
		return m.realService.SupportsGrammarField(provider)
	}
	return false
}

func (m *MockAIService) Shutdown(ctx context.Context) error {
	if m.realService != nil {
		return m.realService.Shutdown(ctx)
	}
	return nil
}
