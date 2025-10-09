// Package services provides business logic services for the quiz application.
package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/xeipuuv/gojsonschema"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// JSON Schema definitions for grammar field
// These schemas are used with the 'grammar' field in OpenAI-compatible API requests
// to enforce specific JSON structure validation. This ensures that AI models return
// exactly the expected format, eliminating parsing errors and improving reliability.
//
// The grammar field is conditionally included based on provider support (see supportsGrammarField).
// Providers that don't support grammar (like Google) will fall back to prompt-based structure guidance.
const (
	// Single-item schemas for ai-fix (single question objects)
	SingleQuestionSchema = `{
		"type": "object",
		"properties": {
			"question": {"type": "string"},
			"options": {"type": "array", "items": {"type": "string"}, "minItems": 4, "maxItems": 4},
			"correct_answer": {"type": "integer"},
			"explanation": {"type": "string"},
			"topic": {"type": "string"}
		},
		"required": ["question", "options", "correct_answer", "explanation"]
	}`

	SingleReadingComprehensionSchema = `{
		"type": "object",
		"properties": {
			"passage": {"type": "string"},
			"question": {"type": "string"},
			"options": {"type": "array", "items": {"type": "string"}, "minItems": 4, "maxItems": 4},
			"correct_answer": {"type": "integer"},
			"explanation": {"type": "string"},
			"topic": {"type": "string"}
		},
		"required": ["passage", "question", "options", "correct_answer", "explanation"]
	}`

	SingleVocabularyQuestionSchema = `{
		"type": "object",
		"properties": {
			"sentence": {"type": "string"},
			"question": {"type": "string"},
			"options": {"type": "array", "items": {"type": "string"}, "minItems": 4, "maxItems": 4},
			"correct_answer": {"type": "integer"},
			"explanation": {"type": "string"},
			"topic": {"type": "string"}
		},
		"required": ["sentence", "question", "options", "correct_answer", "explanation"]
	}`
)

var (
	// BatchQuestionsSchema is a batch wrapper around SingleQuestionSchema.
	BatchQuestionsSchema = fmt.Sprintf(`{"type":"array","items":%s}`, SingleQuestionSchema)

	// BatchReadingComprehensionSchema is a batch wrapper around SingleReadingComprehensionSchema.
	BatchReadingComprehensionSchema = fmt.Sprintf(`{"type":"array","items":%s}`, SingleReadingComprehensionSchema)

	// BatchVocabularyQuestionSchema is a batch wrapper around SingleVocabularyQuestionSchema.
	BatchVocabularyQuestionSchema = fmt.Sprintf(`{"type":"array","items":%s}`, SingleVocabularyQuestionSchema)
)

// UserAIConfig holds per-user AI configuration
type UserAIConfig struct {
	Provider string
	Model    string
	APIKey   string
	Username string // For logging purposes
}

// AIServiceInterface defines the interface for AI-powered question generation
type AIServiceInterface interface {
	GenerateQuestion(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (*models.Question, error)
	GenerateQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) ([]*models.Question, error)
	GenerateQuestionsStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *VarietyElements) error
	GenerateChatResponse(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest) (string, error)
	GenerateChatResponseStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest, chunks chan<- string) error
	GenerateStorySection(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryGenerationRequest) (string, error)
	GenerateStoryQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryQuestionsRequest) ([]*models.StorySectionQuestionData, error)
	TestConnection(ctx context.Context, provider, model, apiKey string) error
	GetConcurrencyStats() ConcurrencyStats
	GetQuestionBatchSize(provider string) int
	VarietyService() *VarietyService

	// TemplateManager exposes template rendering and example loading for prompts
	TemplateManager() *AITemplateManager

	// SupportsGrammarField reports whether the provider supports the grammar field
	SupportsGrammarField(provider string) bool

	// CallWithPrompt sends a raw prompt (and optional grammar) to the provider and returns the response
	CallWithPrompt(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string) (string, error)
	Shutdown(ctx context.Context) error
}

// ConcurrencyStats provides metrics about AI request concurrency
type ConcurrencyStats struct {
	ActiveRequests  int            `json:"active_requests"`
	MaxConcurrent   int            `json:"max_concurrent"`
	QueuedRequests  int            `json:"queued_requests"`
	TotalRequests   int64          `json:"total_requests"`
	UserActiveCount map[string]int `json:"user_active_count"`
	MaxPerUser      int            `json:"max_per_user"`
}

// AIService provides AI-powered question generation using OpenAI-compatible APIs
type AIService struct {
	httpClient *http.Client
	debug      bool
	cfg        *config.Config

	// Template management
	templateManager *AITemplateManager

	// Variety service for question diversity
	varietyService *VarietyService

	// Concurrency control
	globalSemaphore chan struct{} // Limits total concurrent requests
	maxConcurrent   int           // Maximum concurrent requests globally
	maxPerUser      int           // Maximum concurrent requests per user

	// Per-user concurrency tracking
	userRequestCount map[string]int // Username -> active request count
	concurrencyMu    sync.RWMutex   // Protects user maps

	// Metrics
	totalRequests  int64        // Total requests processed
	activeRequests int          // Current active requests
	statsMu        sync.RWMutex // Protects stats

	// Observability
	logger *observability.Logger

	// Shutdown control
	shutdownCtx context.Context
	shutdownMu  sync.RWMutex
}

// Schema validation counters
var (
	SchemaValidationFailures       = make(map[models.QuestionType]int)
	SchemaValidationFailureDetails = make(map[models.QuestionType][]string) // NEW: error details
	SchemaValidationMu             sync.Mutex
)

// extractItemsSchema extracts the items schema from a batch schema
func extractItemsSchema(batchSchema string) (result0 string, err error) {
	var schemaMap map[string]interface{}
	if err = json.Unmarshal([]byte(batchSchema), &schemaMap); err != nil {
		return "", err
	}
	// For batch schemas, extract the items schema
	if items, ok := schemaMap["items"]; ok {
		var itemsBytes []byte
		itemsBytes, err = json.Marshal(items)
		if err != nil {
			return "", err
		}
		return string(itemsBytes), nil
	}
	return "", contextutils.ErrorWithContextf("no items found in batch schema")
}

// ValidateQuestionSchema validates a question against the appropriate schema
func (s *AIService) ValidateQuestionSchema(ctx context.Context, qType models.QuestionType, question interface{}) (result0 bool, err error) {
	_, span := observability.TraceAIFunction(ctx, "validate_question_schema",
		observability.AttributeQuestionType(qType),
	)
	defer observability.FinishSpan(span, &err)

	// Validate input parameters
	if question == nil {
		span.SetAttributes(attribute.String("validation.result", "nil_question"))
		return false, contextutils.ErrorWithContextf("question cannot be nil")
	}

	var schema string
	switch qType {
	case models.Vocabulary:
		schema = BatchVocabularyQuestionSchema
	case models.ReadingComprehension:
		schema = BatchReadingComprehensionSchema
	case models.FillInBlank, models.QuestionAnswer:
		schema = BatchQuestionsSchema
	default:
		span.SetAttributes(attribute.String("validation.result", "unknown_type"))
		return false, contextutils.ErrorWithContextf("unknown question type: %v", qType)
	}

	// Extract the items schema for validation
	itemSchema, err := extractItemsSchema(schema)
	if err != nil {
		span.SetAttributes(attribute.String("validation.result", "schema_extract_error"), attribute.String("validation.error", err.Error()))
		return false, contextutils.WrapErrorf(err, "failed to extract schema for question type %v", qType)
	}

	// Marshal the question to JSON
	// If question is a *models.Question, validate only Content
	toValidate := question
	if q, ok := question.(*models.Question); ok {
		if q == nil {
			span.SetAttributes(attribute.String("validation.result", "nil_question_model"))
			return false, contextutils.ErrorWithContextf("question model is nil")
		}
		toValidate = q.Content
	}

	questionBytes, err := json.Marshal(toValidate)
	if err != nil {
		span.SetAttributes(attribute.String("validation.result", "marshal_error"), attribute.String("validation.error", err.Error()))
		return false, contextutils.WrapErrorf(err, "failed to marshal question for validation")
	}

	// Validate
	result, err := gojsonschema.Validate(
		gojsonschema.NewStringLoader(itemSchema),
		gojsonschema.NewBytesLoader(questionBytes),
	)
	if err != nil {
		span.SetAttributes(attribute.String("validation.result", "validate_error"), attribute.String("validation.error", err.Error()))
		return false, contextutils.WrapErrorf(err, "schema validation failed for question type %v", qType)
	}

	if !result.Valid() {
		errs := result.Errors()
		var errorMessages []string
		for _, e := range errs {
			errorMessages = append(errorMessages, e.String())
		}
		span.SetAttributes(attribute.String("validation.result", "invalid"))
		return false, contextutils.ErrorWithContextf("question failed schema validation: %s", strings.Join(errorMessages, "; "))
	}

	span.SetAttributes(attribute.String("validation.result", "valid"))
	return true, nil
}

// NewAIService creates a new AI service instance
func NewAIService(cfg *config.Config, logger *observability.Logger) *AIService {
	// Create template manager
	templateManager, err := NewAITemplateManager()
	if err != nil {
		logger.Error(context.Background(), "Failed to create template manager", err, map[string]interface{}{})
		panic(err) // Use panic for fatal errors in initialization
	}

	// Create variety service
	varietyService := NewVarietyServiceWithLogger(cfg, logger)

	// Create instrumented HTTP client with reasonable timeouts and explicit span options
	// Use a timeout slightly less than AIRequestTimeout to allow context cancellation
	httpClient := &http.Client{
		Timeout: config.AIRequestTimeout - 5*time.Second, // Slightly less than AIRequestTimeout
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
		),
	}

	// Get concurrency limits from config
	maxConcurrent := cfg.Server.MaxAIConcurrent
	maxPerUser := cfg.Server.MaxAIPerUser

	// Create global semaphore for limiting concurrent requests
	globalSemaphore := make(chan struct{}, maxConcurrent)

	service := &AIService{
		httpClient:       httpClient,
		debug:            cfg.Server.Debug,
		cfg:              cfg,
		templateManager:  templateManager,
		varietyService:   varietyService,
		globalSemaphore:  globalSemaphore,
		maxConcurrent:    maxConcurrent,
		maxPerUser:       maxPerUser,
		userRequestCount: make(map[string]int),
		shutdownCtx:      context.Background(),
		logger:           logger,
	}

	return service
}

// Shutdown gracefully shuts down the AI service and cleans up resources
func (s *AIService) Shutdown(ctx context.Context) error {
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	// Create a new shutdown context
	shutdownCtx, cancel := context.WithCancel(ctx)
	s.shutdownCtx = shutdownCtx
	defer cancel()

	// Wait for all active requests to complete with timeout
	timeout := config.AIShutdownTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	// Wait for active requests to complete
	ticker := time.NewTicker(config.AIShutdownPollInterval)
	defer ticker.Stop()

	for i := 0; i < int(timeout/config.AIShutdownPollInterval); i++ {
		s.statsMu.RLock()
		active := s.activeRequests
		s.statsMu.RUnlock()

		if active == 0 {
			break
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Close the HTTP client
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}

	// Clean up user request counts
	s.concurrencyMu.Lock()
	s.userRequestCount = make(map[string]int)
	s.concurrencyMu.Unlock()

	s.logger.Info(ctx, "AI Service shutdown completed")
	return nil
}

// isShutdown checks if the service is shutting down
func (s *AIService) isShutdown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	select {
	case <-s.shutdownCtx.Done():
		return true
	default:
		return false
	}
}

// OpenAIRequest represents a request to the OpenAI-compatible API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Grammar     string    `json:"grammar,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a chat message in the API request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents a response from the OpenAI-compatible API
type OpenAIResponse struct {
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a choice in the API response
type Choice struct {
	Message Message `json:"message"`
}

// APIError represents an error response from the API
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// OpenAIStreamResponse represents a streaming response chunk from the OpenAI-compatible API
type OpenAIStreamResponse struct {
	Choices []StreamChoice `json:"choices"`
	Error   *APIError      `json:"error,omitempty"`
}

// StreamChoice represents a choice in the streaming API response
type StreamChoice struct {
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamDelta represents the delta content in a streaming response
type StreamDelta struct {
	Content string `json:"content"`
}

// getGrammarSchema returns the appropriate JSON schema for the given question type
func getGrammarSchema(questionType models.QuestionType) string {
	// Always return the batch schema for each type
	switch questionType {
	case models.ReadingComprehension:
		return BatchReadingComprehensionSchema
	case models.Vocabulary:
		return BatchVocabularyQuestionSchema
	case models.FillInBlank:
		return BatchQuestionsSchema
	case models.QuestionAnswer:
		return BatchQuestionsSchema
	}
	// Fallback for unknown types
	return BatchQuestionsSchema
}

// GetFixSchema returns the single-item JSON schema for ai-fix or an error if unsupported.
func GetFixSchema(questionType models.QuestionType) (string, error) {
	switch questionType {
	case models.ReadingComprehension:
		return SingleReadingComprehensionSchema, nil
	case models.Vocabulary:
		return SingleVocabularyQuestionSchema, nil
	case models.FillInBlank, models.QuestionAnswer:
		return SingleQuestionSchema, nil
	default:
		return "", contextutils.WrapErrorf(contextutils.ErrAIConfigInvalid, "no schema for question type: %v", questionType)
	}
}

// addJSONStructureGuidance appends JSON structure requirements to prompts for providers that don't support grammar
func (s *AIService) addJSONStructureGuidance(prompt string, questionType models.QuestionType) string {
	// Get the schema for this question type
	schema := getGrammarSchema(questionType)

	data := AITemplateData{
		SchemaForPrompt: schema,
	}

	guidance, err := s.templateManager.RenderTemplate(JSONStructureGuidanceTemplate, data)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to render JSON structure guidance template", err, map[string]interface{}{})
		panic(err)
	}

	return prompt + guidance
}

// GenerateQuestion generates a single question using AI
func (s *AIService) GenerateQuestion(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 *models.Question, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_question",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		observability.AttributeQuestionType(string(req.QuestionType)),
	)
	defer observability.FinishSpan(span, &err)
	// Check if the provider supports grammar field
	supportsGrammar := s.supportsGrammarField(userConfig.Provider)

	var prompt string
	var grammar string

	if supportsGrammar {
		// Use batch prompt with count=1 for single question
		prompt = s.buildBatchQuestionPrompt(ctx, req, nil)
		grammar = getGrammarSchema(req.QuestionType)
	} else {
		// Use batch prompt with JSON structure guidance embedded
		prompt = s.buildBatchQuestionPromptWithJSONStructure(ctx, req, nil)
		grammar = "" // No grammar field for providers that don't support it
	}

	response, err := s.callOpenAI(ctx, userConfig, prompt, grammar)
	if err != nil {
		return nil, err
	}

	question, err := s.parseQuestionResponse(ctx, response, req.Language, req.Level, req.QuestionType, userConfig.Provider)
	if err != nil {
		return nil, err
	}

	return question, nil
}

// GenerateQuestions generates multiple questions in a single batch request
func (s *AIService) GenerateQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 []*models.Question, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_questions",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		observability.AttributeQuestionType(string(req.QuestionType)),
		observability.AttributeLimit(req.Count),
	)
	defer observability.FinishSpan(span, &err)
	// Check if the provider supports grammar field
	supportsGrammar := s.supportsGrammarField(userConfig.Provider)

	var prompt string
	var grammar string

	if supportsGrammar {
		// Use regular prompt with grammar field
		prompt = s.buildBatchQuestionPrompt(ctx, req, nil)
		grammar = getGrammarSchema(req.QuestionType)
	} else {
		// Use prompt with JSON structure guidance embedded
		prompt = s.buildBatchQuestionPromptWithJSONStructure(ctx, req, nil)
		grammar = "" // No grammar field for providers that don't support it
	}

	response, err := s.callOpenAI(ctx, userConfig, prompt, grammar)
	if err != nil {
		return nil, err
	}

	questions, err := s.parseQuestionsResponse(ctx, response, req.Language, req.Level, req.QuestionType, userConfig.Provider)
	if err != nil {
		return nil, err
	}

	return questions, nil
}

// GenerateQuestionsStream generates questions and streams them via a channel, using the provided variety elements
func (s *AIService) GenerateQuestionsStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *VarietyElements) (err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_questions_stream",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		observability.AttributeQuestionType(string(req.QuestionType)),
		observability.AttributeLimit(req.Count),
	)
	defer observability.FinishSpan(span, &err)
	defer close(progress)

	return s.withConcurrencyControl(ctx, userConfig.Username, func() error {
		// Get the batch size for this provider
		batchSize := s.getQuestionBatchSize(userConfig.Provider)
		// Use batch generation for multiple questions
		return s.generateQuestionsInBatchesWithVariety(ctx, userConfig, req, progress, batchSize, variety)
	})
}

// generateQuestionsInBatchesWithVariety generates questions in batches for efficiency, using the provided variety elements
func (s *AIService) generateQuestionsInBatchesWithVariety(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, batchSize int, variety *VarietyElements) (err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_questions_in_batches_with_variety",
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		observability.AttributeQuestionType(req.QuestionType),
		observability.AttributeLanguage(req.Language),
		observability.AttributeLevel(req.Level),
		attribute.Int("batch_size", batchSize),
		attribute.Int("total_count", req.Count),
		attribute.Bool("variety.enabled", variety != nil),
	)
	defer observability.FinishSpan(span, &err)
	// Local copy of history to be updated as we generate questions
	localHistory := make([]string, len(req.RecentQuestionHistory))
	copy(localHistory, req.RecentQuestionHistory)

	remaining := req.Count
	generated := 0

	for remaining > 0 {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate how many questions to generate in this batch
		currentBatchSize := min(remaining, batchSize)

		// Create a batch request
		batchReq := &models.AIQuestionGenRequest{
			Language:              req.Language,
			Level:                 req.Level,
			QuestionType:          req.QuestionType,
			Count:                 currentBatchSize,
			RecentQuestionHistory: localHistory,
		}

		// Generate questions in batch using the provided variety elements
		questions, err := s.generateQuestionsWithVariety(ctx, userConfig, batchReq, variety)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to generate batch of %d questions for user %s", currentBatchSize, userConfig.Username)
		}

		// Stream the generated questions
		for _, question := range questions {
			// Add generated question content to history for next iterations
			if qContent, ok := question.Content["question"]; ok {
				if qStr, ok := qContent.(string); ok {
					localHistory = append(localHistory, qStr)
				}
			}

			progress <- question
			generated++
		}

		remaining -= len(questions)
	}

	return nil
}

// generateQuestionsWithVariety generates a batch of questions using the provided variety elements
func (s *AIService) generateQuestionsWithVariety(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, variety *VarietyElements) (result0 []*models.Question, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_questions_with_variety",
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		observability.AttributeQuestionType(req.QuestionType),
		observability.AttributeLanguage(req.Language),
		observability.AttributeLevel(req.Level),
		attribute.Int("count", req.Count),
		attribute.Bool("variety.enabled", variety != nil),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	// Check if the provider supports grammar field
	supportsGrammar := s.supportsGrammarField(userConfig.Provider)

	var prompt string
	var grammar string

	if supportsGrammar {
		prompt = s.buildBatchQuestionPrompt(ctx, req, variety)
		grammar = getGrammarSchema(req.QuestionType)
	} else {
		prompt = s.buildBatchQuestionPromptWithJSONStructure(ctx, req, variety)
		grammar = ""
	}

	response, err := s.callOpenAI(ctx, userConfig, prompt, grammar)
	if err != nil {
		return nil, err
	}

	questions, err := s.parseQuestionsResponse(ctx, response, req.Language, req.Level, req.QuestionType, userConfig.Provider)
	if err != nil {
		return nil, err
	}

	return questions, nil
}

// GenerateChatResponse generates a chat response using AI
func (s *AIService) GenerateChatResponse(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest) (result0 string, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_chat_response",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
	)
	defer observability.FinishSpan(span, &err)
	var result string
	var resultErr error

	err = s.withConcurrencyControl(ctx, userConfig.Username, func() error {
		prompt := s.buildChatPrompt(req)
		// No grammar constraint for open-ended chat
		result, resultErr = s.callOpenAI(ctx, userConfig, prompt, "")
		return resultErr
	})
	if err != nil {
		return "", err
	}
	return result, resultErr
}

// GenerateChatResponseStream generates a streaming chat response using AI
func (s *AIService) GenerateChatResponseStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest, chunks chan<- string) (err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_chat_response_stream",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
	)
	defer observability.FinishSpan(span, &err)
	// Don't close the channel here - let the caller handle it to avoid race conditions

	return s.withConcurrencyControl(ctx, userConfig.Username, func() error {
		prompt := s.buildChatPrompt(req)
		// No grammar constraint for open-ended chat
		return s.callOpenAIStream(ctx, userConfig, prompt, "", chunks)
	})
}

// GenerateStorySection generates a story section using AI
func (s *AIService) GenerateStorySection(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryGenerationRequest) (result string, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_story_section",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		attribute.String("story.title", req.Title),
		attribute.String("story.language", req.Language),
		attribute.String("story.level", req.Level),
		attribute.Bool("story.is_first_section", req.IsFirstSection),
	)
	defer observability.FinishSpan(span, &err)

	var storyResult string
	var storyErr error

	err = s.withConcurrencyControl(ctx, userConfig.Username, func() error {
		prompt := s.buildStorySectionPrompt(req)
		storyResult, storyErr = s.callOpenAI(ctx, userConfig, prompt, "")
		return storyErr
	})
	if err != nil {
		return "", err
	}
	return storyResult, storyErr
}

// GenerateStoryQuestions generates comprehension questions for a story section
func (s *AIService) GenerateStoryQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryQuestionsRequest) (result []*models.StorySectionQuestionData, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_story_questions",
		attribute.String("user.username", userConfig.Username),
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		attribute.String("story.language", req.Language),
		attribute.String("story.level", req.Level),
		attribute.Int("questions.count", req.QuestionCount),
	)
	defer observability.FinishSpan(span, &err)

	var questionsResult []*models.StorySectionQuestionData
	var questionsErr error

	err = s.withConcurrencyControl(ctx, userConfig.Username, func() error {
		prompt := s.buildStoryQuestionsPrompt(req)
		response, responseErr := s.callOpenAI(ctx, userConfig, prompt, "")
		if responseErr != nil {
			return responseErr
		}

		// Parse the JSON response into question data
		questionsResult, questionsErr = s.parseStoryQuestionsResponse(response)
		if questionsErr != nil {
			return contextutils.WrapErrorf(questionsErr, "failed to parse story questions response")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return questionsResult, questionsErr
}

// stringPtrToString converts a *string to string, returning empty string if nil
func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// buildStorySectionPrompt builds the prompt for story section generation
func (s *AIService) buildStorySectionPrompt(req *models.StoryGenerationRequest) string {
	// Create template data from the request
	templateData := AITemplateData{
		Language:           req.Language,
		Level:              req.Level,
		Title:              req.Title,
		Subject:            stringPtrToString(req.Subject),
		AuthorStyle:        stringPtrToString(req.AuthorStyle),
		TimePeriod:         stringPtrToString(req.TimePeriod),
		Genre:              stringPtrToString(req.Genre),
		Tone:               stringPtrToString(req.Tone),
		CharacterNames:     stringPtrToString(req.CharacterNames),
		CustomInstructions: stringPtrToString(req.CustomInstructions),
		TargetWords:        req.TargetWords,
		TargetSentences:    req.TargetSentences,
		IsFirstSection:     req.IsFirstSection,
		PreviousSections:   req.PreviousSections,
	}

	template, err := s.templateManager.RenderTemplate("story_section_prompt.tmpl", templateData)
	if err != nil {
		// No fallback - error out if template not found
		panic(contextutils.WrapErrorf(err, "failed to render story section template"))
	}

	return template
}

// buildStoryQuestionsPrompt builds the prompt for story questions generation
func (s *AIService) buildStoryQuestionsPrompt(req *models.StoryQuestionsRequest) string {
	// Create template data from the request
	templateData := AITemplateData{
		Language:    req.Language,
		Level:       req.Level,
		Count:       req.QuestionCount,
		SectionText: req.SectionText,
	}

	template, err := s.templateManager.RenderTemplate("story_questions_prompt.tmpl", templateData)
	if err != nil {
		// No fallback - error out if template not found
		panic(contextutils.WrapErrorf(err, "failed to render story questions template"))
	}

	return template
}

// parseStoryQuestionsResponse parses the AI response into question data
func (s *AIService) parseStoryQuestionsResponse(response string) ([]*models.StorySectionQuestionData, error) {
	// Clean the response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var questions []*models.StorySectionQuestionData
	if err := json.Unmarshal([]byte(response), &questions); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to unmarshal questions JSON")
	}

	// Validate the questions
	for i, q := range questions {
		if q.QuestionText == "" {
			return nil, contextutils.ErrorWithContextf("question %d: missing question text", i)
		}
		if len(q.Options) != 4 {
			return nil, contextutils.ErrorWithContextf("question %d: must have exactly 4 options, got %d", i, len(q.Options))
		}
		if q.CorrectAnswerIndex < 0 || q.CorrectAnswerIndex >= 4 {
			return nil, contextutils.ErrorWithContextf("question %d: correct_answer_index must be 0-3, got %d", i, q.CorrectAnswerIndex)
		}
	}

	return questions, nil
}

// TestConnection tests the connection to the AI service
func (s *AIService) TestConnection(ctx context.Context, provider, model, apiKey string) (err error) {
	_, span := observability.TraceAIFunction(ctx, "test_connection",
		attribute.String("ai.provider", provider),
		attribute.String("ai.model", model),
	)
	defer observability.FinishSpan(span, &err)

	// Validate input parameters
	if provider == "" {
		span.SetAttributes(attribute.String("test.result", "empty_provider"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "provider is required for testing connection")
	}

	if model == "" {
		span.SetAttributes(attribute.String("test.result", "empty_model"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "model is required for testing connection")
	}

	s.logger.Debug(ctx, "TestConnection called", map[string]interface{}{
		"provider": provider,
		"model":    model,
		"apiKey":   contextutils.MaskAPIKey(apiKey),
	})

	// Require API key for all providers that are not Ollama
	if provider != "ollama" && apiKey == "" {
		span.SetAttributes(attribute.String("test.result", "missing_api_key"), attribute.String("provider", provider))
		return contextutils.WrapErrorf(contextutils.ErrAIConfigInvalid, "API key is required for testing connection with provider '%s'", provider)
	}

	// Create a simple test configuration
	userConfig := &models.UserAIConfig{
		Provider: provider,
		Model:    model,
		APIKey:   apiKey,
		Username: "test-user",
	}

	s.logger.Debug(ctx, "Created userConfig", map[string]interface{}{
		"provider": userConfig.Provider,
		"model":    userConfig.Model,
	})

	// Create a simple test request
	testPrompt := "Respond with exactly the word 'SUCCESS' and nothing else."

	// Create a timeout context for the test
	testCtx, cancel := context.WithTimeout(ctx, config.AIRequestTimeout)
	defer cancel()

	// Test the actual AI service call
	response, err := s.callOpenAI(testCtx, userConfig, testPrompt, "")
	if err != nil {
		span.SetAttributes(attribute.String("test.result", "call_failed"), attribute.String("error", err.Error()))
		return contextutils.WrapErrorf(err, "connection test failed for provider '%s' with model '%s'", provider, model)
	}

	// Check if we got a reasonable response
	if response == "" {
		span.SetAttributes(attribute.String("test.result", "empty_response"))
		return contextutils.WrapError(contextutils.ErrAIResponseInvalid, "connection test failed: received empty response from AI service")
	}

	// Validate that the response contains something meaningful
	if len(response) < 3 {
		span.SetAttributes(attribute.String("test.result", "response_too_short"), attribute.Int("response_length", len(response)))
		return contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "connection test failed: response too short (%d characters)", len(response))
	}

	// The response should contain something meaningful
	s.logger.Info(ctx, "TestConnection successful", map[string]interface{}{
		"provider":        provider,
		"response_length": len(response),
	})
	span.SetAttributes(attribute.String("test.result", "success"), attribute.Int("response_length", len(response)))
	return nil
}

// buildBatchQuestionPromptWithJSONStructure now takes variety elements
func (s *AIService) buildBatchQuestionPromptWithJSONStructure(ctx context.Context, req *models.AIQuestionGenRequest, variety *VarietyElements) string {
	prompt := s.buildBatchQuestionPrompt(ctx, req, variety)
	return s.addJSONStructureGuidance(prompt, req.QuestionType)
}

// buildBatchQuestionPrompt now takes variety elements
func (s *AIService) buildBatchQuestionPrompt(ctx context.Context, req *models.AIQuestionGenRequest, variety *VarietyElements) string {
	_, span := observability.TraceAIFunction(ctx, "build_batch_question_prompt",
		observability.AttributeQuestionType(req.QuestionType),
		observability.AttributeLanguage(req.Language),
		observability.AttributeLevel(req.Level),
		attribute.Int("count", req.Count),
		attribute.Bool("variety.enabled", variety != nil),
	)
	defer span.End()
	tmplData := AITemplateData{
		SchemaForPrompt:       getGrammarSchema(req.QuestionType),
		Language:              req.Language,
		Level:                 req.Level,
		QuestionType:          string(req.QuestionType),
		Count:                 req.Count,
		RecentQuestionHistory: req.RecentQuestionHistory,
	}
	if variety != nil {
		tmplData.TopicCategory = variety.TopicCategory
		tmplData.GrammarFocus = variety.GrammarFocus
		tmplData.VocabularyDomain = variety.VocabularyDomain
		tmplData.Scenario = variety.Scenario
		tmplData.StyleModifier = variety.StyleModifier
		tmplData.DifficultyModifier = variety.DifficultyModifier
		tmplData.TimeContext = variety.TimeContext
	}

	// Priority data is handled by the worker, not passed to AI service

	// Load example for this question type
	if exampleContent, err := s.templateManager.LoadExample(string(req.QuestionType)); err == nil {
		tmplData.ExampleContent = exampleContent
	}

	prompt, err := s.templateManager.RenderTemplate(BatchQuestionPromptTemplate, tmplData)
	if err != nil {
		s.logger.Error(ctx, "Failed to render batch question prompt template", err, map[string]interface{}{})
		panic(err) // Use panic for fatal errors in template rendering
	}

	return prompt
}

func (s *AIService) buildChatPrompt(req *models.AIChatRequest) string {
	// Convert conversation history to template format
	var conversationHistory []ChatMessage
	for _, msg := range req.ConversationHistory {
		conversationHistory = append(conversationHistory, ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	data := AITemplateData{
		Language:            req.Language,
		Level:               req.Level,
		QuestionType:        string(req.QuestionType),
		Passage:             req.Passage,
		Question:            req.Question,
		Options:             req.Options,
		IsCorrect:           req.IsCorrect,
		ConversationHistory: conversationHistory,
		UserMessage:         req.UserMessage,
	}

	prompt, err := s.templateManager.RenderTemplate(ChatPromptTemplate, data)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to render chat prompt template", err, map[string]interface{}{})
		panic(err) // Use panic for fatal errors in template rendering
	}

	return prompt
}

// getMaxTokensForModel looks up the max_tokens for a specific provider and model
func (s *AIService) getMaxTokensForModel(provider, model string) int {
	// Look up the model in the provider configuration
	if s.cfg.Providers != nil {
		for _, providerConfig := range s.cfg.Providers {
			if providerConfig.Code == provider {
				for _, modelConfig := range providerConfig.Models {
					if modelConfig.Code == model {
						if modelConfig.MaxTokens > 0 {
							return modelConfig.MaxTokens
						}
						break
					}
				}
				break
			}
		}
	}

	// Default fallback
	return 4000
}

// callOpenAI makes a request to the OpenAI-compatible API
func (s *AIService) callOpenAI(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string) (result0 string, err error) {
	if userConfig == nil {
		return "", contextutils.WrapError(contextutils.ErrAIConfigInvalid, "userConfig is required")
	}
	_, span := observability.TraceAIFunction(ctx, "call_openai",
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		attribute.String("ai.username", userConfig.Username),
		attribute.Int("prompt.length", len(prompt)),
		attribute.Bool("grammar.enabled", grammar != ""),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Validate input parameters
	if userConfig.Provider == "" {
		span.SetAttributes(attribute.String("call.result", "empty_provider"))
		return "", contextutils.WrapError(contextutils.ErrAIConfigInvalid, "provider is required")
	}

	if userConfig.Model == "" {
		span.SetAttributes(attribute.String("call.result", "empty_model"))
		return "", contextutils.WrapError(contextutils.ErrAIConfigInvalid, "model is required")
	}

	if prompt == "" {
		span.SetAttributes(attribute.String("call.result", "empty_prompt"))
		return "", contextutils.WrapError(contextutils.ErrAIConfigInvalid, "prompt cannot be empty")
	}

	apiURL := ""
	model := userConfig.Model
	apiKey := userConfig.APIKey

	// Look up the default URL from provider config
	if s.cfg.Providers != nil {
		for _, providerConfig := range s.cfg.Providers {
			if providerConfig.Code == userConfig.Provider && providerConfig.URL != "" {
				apiURL = providerConfig.URL
				break
			}
		}
	}

	if apiURL == "" {
		span.SetAttributes(attribute.String("call.result", "no_url_configured"), attribute.String("provider", userConfig.Provider))
		return "", contextutils.WrapErrorf(contextutils.ErrAIConfigInvalid, "no base URL configured for provider '%s'", userConfig.Provider)
	}

	userPrefix := ""
	if userConfig.Username != "" {
		userPrefix = fmt.Sprintf("[user=%s] ", userConfig.Username)
	}

	s.logger.Debug(ctx, "Starting AI request", map[string]interface{}{
		"user_prefix": userPrefix,
		"url":         apiURL + "/chat/completions",
		"model":       model,
		"provider":    userConfig.Provider,
	})

	// Create messages with just the user prompt - grammar field will enforce JSON structure
	messages := []Message{{Role: "user", Content: prompt}}

	// Check if the provider supports grammar field
	supportsGrammar := s.supportsGrammarField(userConfig.Provider)

	reqBody := OpenAIRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   s.getMaxTokensForModel(userConfig.Provider, userConfig.Model),
	}

	// Only include grammar field if the provider supports it
	if supportsGrammar && grammar != "" {
		reqBody.Grammar = grammar
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal AI request", err, map[string]interface{}{
			"user_prefix": userPrefix,
		})
		span.SetAttributes(attribute.String("call.result", "marshal_failed"), attribute.String("error", err.Error()))
		return "", contextutils.WrapErrorf(err, "failed to marshal request body")
	}

	s.logger.Debug(ctx, "Making AI HTTP request", map[string]interface{}{
		"user_prefix": userPrefix,
		"url":         apiURL + "/chat/completions",
	})
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error(ctx, "Failed to create AI HTTP request", err, map[string]interface{}{
			"user_prefix": userPrefix,
		})
		span.SetAttributes(attribute.String("call.result", "request_creation_failed"), attribute.String("error", err.Error()))
		return "", contextutils.WrapErrorf(err, "failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "quizapp/1.0")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		s.logger.Debug(ctx, "Using API key authentication", map[string]interface{}{
			"user_prefix": userPrefix,
		})
	} else {
		s.logger.Debug(ctx, "No API key provided, using anonymous access", map[string]interface{}{
			"user_prefix": userPrefix,
		})
	}

	startTime := time.Now()
	resp, err := s.httpClient.Do(req.WithContext(ctx))
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error(ctx, "AI HTTP request failed", err, map[string]interface{}{
			"user_prefix": userPrefix,
			"duration":    duration.String(),
		})
		span.SetAttributes(attribute.String("call.result", "http_request_failed"), attribute.String("error", err.Error()), attribute.String("duration", duration.String()))
		return "", contextutils.WrapErrorf(err, "HTTP request failed after %v", duration)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	s.logger.Info(ctx, "AI Service HTTP request completed", map[string]interface{}{
		"user_prefix": userPrefix,
		"duration":    duration.String(),
		"status_code": resp.StatusCode,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetAttributes(attribute.String("call.result", "body_read_failed"), attribute.String("error", err.Error()))
		return "", contextutils.WrapErrorf(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.String("call.result", "http_error"), attribute.Int("status_code", resp.StatusCode), attribute.String("body", string(body)))
		return "", contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "API request failed with status %d to %s: %s", resp.StatusCode, apiURL+"/chat/completions", string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		span.SetAttributes(attribute.String("call.result", "json_unmarshal_failed"), attribute.String("error", err.Error()), attribute.String("body", string(body)))
		return "", contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "failed to parse AI response as JSON: %w. Raw Response: %s", err, string(body))
	}

	if openAIResp.Error != nil {
		span.SetAttributes(attribute.String("call.result", "api_error"), attribute.String("error_message", openAIResp.Error.Message), attribute.String("error_type", openAIResp.Error.Type))
		return "", contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		span.SetAttributes(attribute.String("call.result", "no_choices"))
		return "", contextutils.WrapError(contextutils.ErrAIResponseInvalid, "no response from OpenAI")
	}

	content := openAIResp.Choices[0].Message.Content
	if content == "" {
		span.SetAttributes(attribute.String("call.result", "empty_content"))
		return "", contextutils.WrapError(contextutils.ErrAIResponseInvalid, "AI returned empty content")
	}

	span.SetAttributes(attribute.String("call.result", "success"), attribute.Int("content_length", len(content)), attribute.String("duration", duration.String()))
	return content, nil
}

// callOpenAIStream makes a streaming request to the OpenAI-compatible API
func (s *AIService) callOpenAIStream(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string, chunks chan<- string) error {
	if userConfig == nil {
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "userConfig is required")
	}
	_, span := observability.TraceAIFunction(ctx, "call_openai_stream",
		attribute.String("ai.provider", userConfig.Provider),
		attribute.String("ai.model", userConfig.Model),
		attribute.String("ai.username", userConfig.Username),
		attribute.Int("prompt.length", len(prompt)),
		attribute.Bool("grammar.enabled", grammar != ""),
	)
	defer span.End()

	// Validate input parameters
	if userConfig.Provider == "" {
		span.SetAttributes(attribute.String("stream.result", "empty_provider"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "provider is required")
	}

	if userConfig.Model == "" {
		span.SetAttributes(attribute.String("stream.result", "empty_model"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "model is required")
	}

	if prompt == "" {
		span.SetAttributes(attribute.String("stream.result", "empty_prompt"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "prompt cannot be empty")
	}

	if chunks == nil {
		span.SetAttributes(attribute.String("stream.result", "nil_chunks_channel"))
		return contextutils.WrapError(contextutils.ErrAIConfigInvalid, "chunks channel is required")
	}

	apiURL := ""
	model := userConfig.Model
	apiKey := userConfig.APIKey

	// Look up the default URL from provider config
	if s.cfg.Providers != nil {
		for _, providerConfig := range s.cfg.Providers {
			if providerConfig.Code == userConfig.Provider && providerConfig.URL != "" {
				apiURL = providerConfig.URL
				break
			}
		}
	}

	if apiURL == "" {
		span.SetAttributes(attribute.String("stream.result", "no_url_configured"), attribute.String("provider", userConfig.Provider))
		return contextutils.WrapErrorf(contextutils.ErrAIConfigInvalid, "no base URL configured for provider '%s'", userConfig.Provider)
	}

	userPrefix := ""
	if userConfig.Username != "" {
		userPrefix = fmt.Sprintf("[user=%s] ", userConfig.Username)
	}

	s.logger.Info(ctx, "AI Service Starting streaming request", map[string]interface{}{
		"user_prefix": userPrefix,
		"api_url":     apiURL + "/chat/completions",
		"model":       model,
		"provider":    userConfig.Provider,
	})

	// Create messages with just the user prompt - grammar field will enforce JSON structure
	messages := []Message{{Role: "user", Content: prompt}}

	// Check if the provider supports grammar field
	supportsGrammar := s.supportsGrammarField(userConfig.Provider)

	reqBody := OpenAIRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   s.getMaxTokensForModel(userConfig.Provider, userConfig.Model),
		Stream:      true, // Enable streaming
	}

	// Only include grammar field if the provider supports it
	if supportsGrammar && grammar != "" {
		reqBody.Grammar = grammar
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal request", err, map[string]interface{}{
			"user_prefix": userPrefix,
		})
		span.SetAttributes(attribute.String("stream.result", "marshal_failed"), attribute.String("error", err.Error()))
		return contextutils.WrapErrorf(err, "failed to marshal streaming request body")
	}

	s.logger.Info(ctx, "AI Service Making streaming HTTP request", map[string]interface{}{
		"user_prefix": userPrefix,
		"api_url":     apiURL + "/chat/completions",
	})
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error(ctx, "Failed to create HTTP request", err, map[string]interface{}{
			"user_prefix": userPrefix,
		})
		span.SetAttributes(attribute.String("stream.result", "request_creation_failed"), attribute.String("error", err.Error()))
		return contextutils.WrapErrorf(err, "failed to create streaming HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "quizapp/1.0")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		s.logger.Info(ctx, "AI Service Using API key authentication", map[string]interface{}{
			"user_prefix": userPrefix,
		})
	} else {
		s.logger.Info(ctx, "AI Service No API key provided, using anonymous access", map[string]interface{}{
			"user_prefix": userPrefix,
		})
	}

	startTime := time.Now()
	resp, err := s.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		s.logger.Error(ctx, "HTTP request failed", err, map[string]interface{}{
			"user_prefix": userPrefix,
		})
		span.SetAttributes(attribute.String("stream.result", "http_request_failed"), attribute.String("error", err.Error()))
		return contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "http client error: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		span.SetAttributes(attribute.String("stream.result", "http_error"), attribute.Int("status_code", resp.StatusCode), attribute.String("body", string(body)))
		return contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "API request failed with status %d to %s: %s", resp.StatusCode, apiURL+"/chat/completions", string(body))
	}

	s.logger.Info(ctx, "AI Service Streaming response started", map[string]interface{}{
		"user_prefix": userPrefix,
		"duration":    time.Since(startTime).String(),
	})

	// Read the streaming response
	scanner := bufio.NewScanner(resp.Body)
	var chunkCount int
	var totalContentLength int

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse Server-Sent Events format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for end of stream
			if data == "[DONE]" {
				break
			}

			// Parse the JSON chunk
			var streamResp OpenAIStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				s.logger.Warn(ctx, "AI Service WARNING: Failed to parse streaming chunk", map[string]interface{}{
					"error": err.Error(),
					"data":  data,
				})
				span.SetAttributes(attribute.String("stream.result", "chunk_parse_failed"), attribute.String("error", err.Error()), attribute.String("data", data))
				continue
			}

			if streamResp.Error != nil {
				span.SetAttributes(attribute.String("stream.result", "api_streaming_error"), attribute.String("error_message", streamResp.Error.Message), attribute.String("error_type", streamResp.Error.Type))
				return contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "OpenAI API streaming error: %s", streamResp.Error.Message)
			}

			// Extract content from the chunk
			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				content := streamResp.Choices[0].Delta.Content
				totalContentLength += len(content)

				// Filter out thinking content for thinking models
				filteredContent := s.filterThinkingContent(content, model)

				if filteredContent != "" {
					select {
					case chunks <- filteredContent:
						chunkCount++
					case <-ctx.Done():
						span.SetAttributes(attribute.String("stream.result", "context_cancelled"))
						return ctx.Err()
					}
				}
			}

			// Check if streaming is finished
			if len(streamResp.Choices) > 0 && streamResp.Choices[0].FinishReason != nil {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		span.SetAttributes(attribute.String("stream.result", "scanner_error"), attribute.String("error", err.Error()))
		return contextutils.WrapErrorf(contextutils.ErrAIRequestFailed, "error reading streaming response: %w", err)
	}

	s.logger.Info(ctx, "AI Service Streaming response completed", map[string]interface{}{
		"user_prefix":          userPrefix,
		"duration":             time.Since(startTime).String(),
		"chunk_count":          chunkCount,
		"total_content_length": totalContentLength,
	})
	span.SetAttributes(attribute.String("stream.result", "success"), attribute.Int("chunk_count", chunkCount), attribute.Int("total_content_length", totalContentLength), attribute.String("duration", time.Since(startTime).String()))
	return nil
}

// filterThinkingContent filters out thinking sections for reasoning models
func (s *AIService) filterThinkingContent(content, model string) string {
	// Check if this is a thinking/reasoning model
	if !s.isThinkingModel(model) {
		return content
	}

	// For thinking models, filter out content between <thinking> tags
	if strings.Contains(content, "<thinking>") || strings.Contains(content, "</thinking>") {
		return ""
	}

	if idx := strings.Index(content, "The answer is:"); idx != -1 {
		answer := content[idx+len("The answer is:"):]
		lines := strings.Split(answer, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				return trimmed
			}
		}
		return ""
	}

	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "I need to") ||
		strings.HasPrefix(trimmed, "Let me think") ||
		strings.HasPrefix(trimmed, "First, I'll") {
		return ""
	}

	return content
}

// isThinkingModel checks if the model is a reasoning/thinking model
func (s *AIService) isThinkingModel(model string) bool {
	thinkingModels := []string{
		"o1-preview",
		"o1-mini",
		"o1",
		"qwen2.5-coder:32b",
		"deepseek-r1",
		"marco-o1",
		"gpt-4",
		"gpt-4-turbo",
		"claude-3",
	}

	modelLower := strings.ToLower(model)
	for _, thinkingModel := range thinkingModels {
		if strings.Contains(modelLower, strings.ToLower(thinkingModel)) {
			return true
		}
	}

	return false
}

// cleanJSONResponse extracts JSON from markdown code blocks or returns the original response
func (s *AIService) cleanJSONResponse(ctx context.Context, response, provider string) string {
	_, span := observability.TraceAIFunction(ctx, "clean_json_response",
		attribute.String("ai.provider", provider),
		attribute.Int("response.length", len(response)),
	)
	defer span.End()
	// If the provider supports grammar field, we expect clean JSON
	if s.supportsGrammarField(provider) {
		return response
	}

	// For providers that don't support grammar field, clean up markdown code blocks
	response = strings.TrimSpace(response)

	// Remove markdown code block markers
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

func (s *AIService) parseQuestionsResponse(ctx context.Context, response, language, level string, qType models.QuestionType, provider string) (result0 []*models.Question, err error) {
	if s == nil {
		return nil, contextutils.WrapError(contextutils.ErrInternalError, "AIService instance is nil")
	}
	_, span := observability.TraceAIFunction(ctx, "parse_questions_response",
		observability.AttributeQuestionType(qType),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.String("ai.provider", provider),
		attribute.Int("response.length", len(response)),
	)
	defer observability.FinishSpan(span, &err)
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "PANIC in parseQuestionsResponse", nil, map[string]interface{}{
				"panic":    fmt.Sprintf("%v", r),
				"response": response,
				"stack":    string(debug.Stack()),
			})
			span.SetAttributes(attribute.String("parse.result", "panic"), attribute.String("panic", fmt.Sprintf("%v", r)))
		}
	}()

	// Validate input parameters
	if response == "" {
		span.SetAttributes(attribute.String("parse.result", "empty_response"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "AI provider returned empty response")
	}
	if language == "" {
		span.SetAttributes(attribute.String("parse.result", "empty_language"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "language cannot be empty")
	}
	if level == "" {
		span.SetAttributes(attribute.String("parse.result", "empty_level"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "level cannot be empty")
	}

	// Clean the response to handle markdown code blocks for providers without grammar support
	cleanedResponse := s.cleanJSONResponse(ctx, response, provider)

	if cleanedResponse == "" {
		span.SetAttributes(attribute.String("parse.result", "empty_cleaned_response"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "AI provider returned empty response after cleaning")
	}

	// With grammar field enforcement, we should get clean JSON directly
	// No need for complex extraction - just parse the response directly
	var questions []map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &questions); err != nil {
		span.SetAttributes(attribute.String("parse.result", "json_unmarshal_failed"), attribute.String("error", err.Error()))
		return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "failed to parse AI response as JSON: %w", err)
	}

	if len(questions) == 0 {
		span.SetAttributes(attribute.String("parse.result", "no_questions_in_response"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "AI provider returned no questions in response")
	}

	var result []*models.Question
	var validationErrors []string
	var skippedCount int

	for i, qData := range questions {
		if qData == nil {
			skippedCount++
			span.SetAttributes(attribute.String("parse.result", "nil_question_data"), attribute.Int("question_index", i))
			continue
		}

		question, err := s.createQuestionFromData(ctx, qData, language, level, qType)
		if err != nil {
			// Try to extract more info about the failure
			var failedField, failedValue string
			for k, v := range qData {
				if v == nil || v == "" {
					failedField = k
					failedValue = fmt.Sprintf("%v", v)
					break
				}
			}
			validationErrors = append(validationErrors, fmt.Sprintf("question %d: %v (field: %s, value: %s)", i+1, err, failedField, failedValue))
			span.SetAttributes(attribute.String("parse.result", "question_creation_failed"), attribute.Int("question_index", i), attribute.String("error", err.Error()))
			continue
		}

		if question == nil {
			skippedCount++
			span.SetAttributes(attribute.String("parse.result", "nil_question_after_creation"), attribute.Int("question_index", i))
			continue
		}

		// Coerce correct_answer to int if it's a float64 (for schema validation)
		if m := question.Content; m != nil {
			if v, ok := m["correct_answer"]; ok {
				switch val := v.(type) {
				case float64:
					m["correct_answer"] = int(val)
				}
			}
		}

		valid, err := s.ValidateQuestionSchema(ctx, qType, question)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("question %d schema validation error: %v", i+1, err))
			span.SetAttributes(attribute.String("parse.result", "schema_validation_error"), attribute.Int("question_index", i), attribute.String("error", err.Error()))
		}

		if !valid {
			SchemaValidationMu.Lock()
			SchemaValidationFailures[qType]++
			if err != nil {
				SchemaValidationFailureDetails[qType] = append(SchemaValidationFailureDetails[qType], err.Error())
			} else {
				SchemaValidationFailureDetails[qType] = append(SchemaValidationFailureDetails[qType], "validation failed")
			}
			if len(SchemaValidationFailureDetails[qType]) > 10 {
				SchemaValidationFailureDetails[qType] = SchemaValidationFailureDetails[qType][len(SchemaValidationFailureDetails[qType])-10:]
			}
			SchemaValidationMu.Unlock()
			skippedCount++
			span.SetAttributes(attribute.String("parse.result", "schema_validation_failed"), attribute.Int("question_index", i))
			continue // skip invalid question
		}

		result = append(result, question)
	}

	// Log validation summary
	if len(validationErrors) > 0 {
		s.logger.Warn(ctx, "AI Service WARNING: validation errors in response", map[string]interface{}{
			"validation_errors_count": len(validationErrors),
			"validation_errors":       strings.Join(validationErrors, "; "),
		})
		span.SetAttributes(attribute.String("parse.result", "validation_errors"), attribute.String("errors", strings.Join(validationErrors, "; ")))
	}

	if len(result) == 0 {
		span.SetAttributes(attribute.String("parse.result", "no_valid_questions"), attribute.Int("total_questions", len(questions)), attribute.Int("skipped_count", skippedCount))
		return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "AI provider returned only invalid or empty questions (total: %d, skipped: %d)", len(questions), skippedCount)
	}

	span.SetAttributes(attribute.String("parse.result", "success"), attribute.Int("valid_questions", len(result)), attribute.Int("total_questions", len(questions)), attribute.Int("skipped_count", skippedCount))
	return result, nil
}

// createQuestionFromData creates a Question from parsed JSON data
func (s *AIService) createQuestionFromData(ctx context.Context, data map[string]interface{}, language, level string, qType models.QuestionType) (result0 *models.Question, err error) {
	if s == nil {
		return nil, contextutils.WrapError(contextutils.ErrInternalError, "AIService instance is nil")
	}
	_, span := observability.TraceAIFunction(ctx, "create_question_from_data",
		observability.AttributeQuestionType(qType),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.Int("data.fields", len(data)),
	)
	defer observability.FinishSpan(span, &err)

	if data == nil {
		span.SetAttributes(attribute.String("creation.result", "nil_data"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "question data is nil")
	}

	// Validate required parameters
	if language == "" {
		span.SetAttributes(attribute.String("creation.result", "empty_language"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "language cannot be empty")
	}
	if level == "" {
		span.SetAttributes(attribute.String("creation.result", "empty_level"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "level cannot be empty")
	}

	if ok, errMsg := s.validateQuestionContent(ctx, qType, data); !ok {
		missingFields := []string{}
		for k, v := range data {
			if v == nil || v == "" {
				missingFields = append(missingFields, k)
			}
		}
		if len(missingFields) > 0 {
			span.SetAttributes(attribute.String("creation.result", "validation_failed_with_missing_fields"), attribute.String("missing_fields", strings.Join(missingFields, ",")))
			return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "invalid question content structure: %s. Missing or empty fields: %v", errMsg, missingFields)
		}
		span.SetAttributes(attribute.String("creation.result", "validation_failed"), attribute.String("error", errMsg))
		return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "invalid question content structure: %s", errMsg)
	}

	// Defensive: For reading comprehension, check passage, question, options, correct_answer
	if qType == models.ReadingComprehension {
		if _, ok := data["passage"].(string); !ok {
			span.SetAttributes(attribute.String("creation.result", "reading_missing_passage"))
			return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "reading comprehension question missing or invalid 'passage' field")
		}
		if _, ok := data["question"].(string); !ok {
			span.SetAttributes(attribute.String("creation.result", "reading_missing_question"))
			return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "reading comprehension question missing or invalid 'question' field")
		}
		options, ok := data["options"].([]interface{})
		if !ok || len(options) != 4 {
			span.SetAttributes(attribute.String("creation.result", "reading_invalid_options"))
			return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "reading comprehension question missing or invalid 'options' field (must be array of 4 strings)")
		}
		for i, opt := range options {
			if _, ok := opt.(string); !ok {
				span.SetAttributes(attribute.String("creation.result", "reading_invalid_option_type"), attribute.Int("option_index", i))
				return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "reading comprehension question 'options' must be array of strings, found invalid type at index %d", i)
			}
		}
		if _, ok := data["correct_answer"]; !ok {
			span.SetAttributes(attribute.String("creation.result", "reading_missing_correct_answer"))
			return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "reading comprehension question missing 'correct_answer' field")
		}
	}

	// Parse correct_answer as index (integer)
	var correctAnswerIndex int
	if correctAnswerRaw, exists := data["correct_answer"]; exists {
		switch v := correctAnswerRaw.(type) {
		case int:
			correctAnswerIndex = v
		case float64:
			correctAnswerIndex = int(v)
		case string:
			// Handle string indices like "0", "1", "2", "3"
			if idx, err := strconv.Atoi(v); err == nil {
				correctAnswerIndex = idx
			} else {
				// Handle answer text - find index in options
				if options, ok := data["options"].([]interface{}); ok {
					found := false
					for i, opt := range options {
						if optStr, ok := opt.(string); ok && optStr == v {
							correctAnswerIndex = i
							found = true
							break
						}
					}
					if !found {
						span.SetAttributes(attribute.String("creation.result", "correct_answer_not_found_in_options"))
						return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "correct_answer '%s' not found in options", v)
					}
				} else {
					span.SetAttributes(attribute.String("creation.result", "no_options_for_text_answer"))
					return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "correct_answer is text '%s' but no options available to match against", v)
				}
			}
		default:
			span.SetAttributes(attribute.String("creation.result", "invalid_correct_answer_type"), attribute.String("type", fmt.Sprintf("%T", v)))
			return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "invalid correct_answer type: %T", v)
		}
	} else {
		span.SetAttributes(attribute.String("creation.result", "missing_correct_answer"))
		return nil, contextutils.WrapError(contextutils.ErrAIResponseInvalid, "missing correct_answer field")
	}

	// Validate correct answer index
	if options, ok := data["options"].([]interface{}); ok {
		if correctAnswerIndex < 0 || correctAnswerIndex >= len(options) {
			span.SetAttributes(attribute.String("creation.result", "invalid_correct_answer_index"), attribute.Int("index", correctAnswerIndex), attribute.Int("options_count", len(options)))
			return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "correct_answer index %d is out of range (0-%d)", correctAnswerIndex, len(options)-1)
		}
	}

	// Note: Removed backend shuffling logic - frontend handles shuffling
	// This prevents mismatch between backend and frontend answer indices

	// Get explanation or provide default
	explanation, _ := data["explanation"].(string)
	if explanation == "" {
		// Provide a default explanation based on question type
		switch qType {
		case models.Vocabulary:
			explanation = "This vocabulary question tests your knowledge of words in context."
		case models.ReadingComprehension:
			explanation = "This reading comprehension question tests your understanding of the passage."
		case models.FillInBlank:
			explanation = "This fill-in-the-blank question tests your grammar and vocabulary knowledge."
		case models.QuestionAnswer:
			explanation = "This question tests your conversational and practical language skills."
		default:
			explanation = "This question tests your language skills."
		}
		// Add the explanation to the data for schema validation
		data["explanation"] = explanation
	}

	question := &models.Question{
		Type:            qType,
		Language:        language,
		Level:           level,
		DifficultyScore: s.getDifficultyScore(level),
		Content:         data,
		CorrectAnswer:   correctAnswerIndex,
		Explanation:     explanation,
		CreatedAt:       time.Now(),
	}

	span.SetAttributes(attribute.String("creation.result", "success"))
	return question, nil
}

func (s *AIService) parseQuestionResponse(ctx context.Context, response, language, level string, qType models.QuestionType, provider string) (result0 *models.Question, err error) {
	_, span := observability.TraceAIFunction(ctx, "parse_question_response",
		observability.AttributeQuestionType(qType),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.String("ai.provider", provider),
		attribute.Int("response.length", len(response)),
	)
	defer observability.FinishSpan(span, &err)
	// Clean the response to handle markdown code blocks for providers without grammar support
	cleanedResponse := s.cleanJSONResponse(ctx, response, provider)

	// With grammar field enforcement, we should get clean JSON directly
	// No need for complex extraction - just parse the response directly
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &data); err != nil {
		s.logger.Error(ctx, "Failed to parse JSON response", err, map[string]interface{}{
			"raw_response": response,
		})
		return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "failed to parse AI response as JSON: %w", err)
	}

	question, err := s.createQuestionFromData(ctx, data, language, level, qType)
	if err != nil {
		s.logger.Error(ctx, "Failed to create question from data", err, map[string]interface{}{
			"raw_question_data":   data,
			"full_model_response": response,
		})
		return nil, contextutils.WrapErrorf(contextutils.ErrAIResponseInvalid, "failed to create question: %w", err)
	}
	valid, err := s.ValidateQuestionSchema(ctx, qType, question)
	if err != nil {
		s.logger.Error(ctx, "Schema validation error for question", err, nil)
	}
	if !valid {
		SchemaValidationMu.Lock()
		SchemaValidationFailures[qType]++
		if err != nil {
			SchemaValidationFailureDetails[qType] = append(SchemaValidationFailureDetails[qType], err.Error())
		} else {
			SchemaValidationFailureDetails[qType] = append(SchemaValidationFailureDetails[qType], "validation failed")
		}
		if len(SchemaValidationFailureDetails[qType]) > 10 {
			SchemaValidationFailureDetails[qType] = SchemaValidationFailureDetails[qType][len(SchemaValidationFailureDetails[qType])-10:]
		}
		SchemaValidationMu.Unlock()
	}
	return question, nil
}

func (s *AIService) getDifficultyScore(level string) float64 {
	// Look up the level in the language levels configuration
	if s.cfg != nil && s.cfg.LanguageLevels != nil {
		for _, langConfig := range s.cfg.LanguageLevels {
			for i, lvl := range langConfig.Levels {
				if lvl == level {
					// Return a score based on the level's position (0.0 to 1.0)
					return float64(i) / float64(len(langConfig.Levels)-1)
				}
			}
		}
	}
	// Default to middle difficulty if level not found
	return 0.5
}

func (s *AIService) validateQuestionContent(ctx context.Context, qType models.QuestionType, content map[string]interface{}) (bool, string) {
	_, span := observability.TraceAIFunction(ctx, "validate_question_content",
		observability.AttributeQuestionType(qType),
		attribute.Int("content.fields", len(content)),
	)
	defer span.End()

	// Validate input parameters
	if content == nil {
		span.SetAttributes(attribute.String("validation.result", "nil_content"))
		return false, "question content cannot be nil"
	}

	requiredFields := make(map[string]func(interface{}) bool)
	isString := func(v interface{}) bool {
		if v == nil {
			return false
		}
		_, ok := v.(string)
		return ok && v.(string) != ""
	}
	isStringSlice := func(v interface{}) bool {
		if v == nil {
			return false
		}
		if slice, ok := v.([]interface{}); ok {
			if len(slice) < 4 {
				return false
			}
			for _, item := range slice {
				if item == nil {
					return false
				}
				if _, ok := item.(string); !ok {
					return false
				}
				if item.(string) == "" {
					return false
				}
			}
			return true
		}
		return false
	}
	isCorrectAnswer := func(v interface{}) bool {
		if v == nil {
			return false
		}
		switch val := v.(type) {
		case int:
			return val >= 0
		case float64:
			return val >= 0 && val == float64(int(val)) // Must be whole number
		case string:
			// Accept string indices like "0", "1", "2", "3" or answer text
			if _, err := strconv.Atoi(val); err == nil {
				return true
			}
			// Or accept answer text that matches one of the options
			if options, ok := content["options"].([]interface{}); ok {
				for _, opt := range options {
					if optStr, ok := opt.(string); ok && optStr == val {
						return true
					}
				}
			}
			return false
		default:
			return false
		}
	}

	switch qType {
	case models.Vocabulary:
		requiredFields["sentence"] = isString
		requiredFields["question"] = isString
		requiredFields["options"] = isStringSlice
		for field, validator := range requiredFields {
			if !validator(content[field]) {
				span.SetAttributes(attribute.String("validation.result", "field_validation_failed"), attribute.String("field", field))
				return false, fmt.Sprintf("[Vocabulary] Validation failed for field '%s': %v", field, content[field])
			}
		}
		sentence, _ := content["sentence"].(string)
		targetWord, _ := content["question"].(string)
		options, _ := content["options"].([]interface{})
		if sentence == "" || targetWord == "" || len(options) != 4 {
			span.SetAttributes(attribute.String("validation.result", "vocabulary_structure_failed"))
			return false, "[Vocabulary] Validation failed: missing or invalid sentence/question/options"
		}
		if !strings.Contains(sentence, targetWord) {
			span.SetAttributes(attribute.String("validation.result", "vocabulary_word_not_found"))
			return false, fmt.Sprintf("[Vocabulary] Validation failed: question '%s' not found in sentence '%s'", targetWord, sentence)
		}
		span.SetAttributes(attribute.String("validation.result", "valid"))
		return true, ""

	case models.ReadingComprehension:
		requiredFields["passage"] = isString
		requiredFields["question"] = isString
		requiredFields["options"] = isStringSlice
		requiredFields["correct_answer"] = isCorrectAnswer
		for field, validator := range requiredFields {
			if !validator(content[field]) {
				span.SetAttributes(attribute.String("validation.result", "field_validation_failed"), attribute.String("field", field))
				return false, fmt.Sprintf("[ReadingComprehension] Validation failed for field '%s': %v", field, content[field])
			}
		}
		passage, _ := content["passage"].(string)
		if passage == "" {
			span.SetAttributes(attribute.String("validation.result", "reading_passage_empty"))
			return false, "[ReadingComprehension] Validation failed: passage cannot be empty"
		}
		span.SetAttributes(attribute.String("validation.result", "valid"))
		return true, ""

	case models.FillInBlank:
		// Fill-in-blank questions now use multiple choice format like all other types
		requiredFields["question"] = isString
		requiredFields["options"] = isStringSlice
		requiredFields["correct_answer"] = isCorrectAnswer
		for field, validator := range requiredFields {
			if !validator(content[field]) {
				span.SetAttributes(attribute.String("validation.result", "field_validation_failed"), attribute.String("field", field))
				return false, fmt.Sprintf("[FillInBlank] Validation failed for field '%s': %v", field, content[field])
			}
		}
		span.SetAttributes(attribute.String("validation.result", "valid"))
		return true, ""

	case models.QuestionAnswer:
		// Question-answer questions now use multiple choice format like all other types
		requiredFields["question"] = isString
		requiredFields["options"] = isStringSlice
		requiredFields["correct_answer"] = isCorrectAnswer
		for field, validator := range requiredFields {
			if !validator(content[field]) {
				span.SetAttributes(attribute.String("validation.result", "field_validation_failed"), attribute.String("field", field))
				return false, fmt.Sprintf("[QuestionAnswer] Validation failed for field '%s': %v", field, content[field])
			}
		}
		span.SetAttributes(attribute.String("validation.result", "valid"))
		return true, ""
	}

	// If we reach here, it's an unknown question type
	span.SetAttributes(attribute.String("validation.result", "unknown_type"))
	return false, fmt.Sprintf("unknown question type: %v", qType)
}

// GetConcurrencyStats returns current concurrency metrics
func (s *AIService) GetConcurrencyStats() ConcurrencyStats {
	s.statsMu.RLock()
	s.concurrencyMu.RLock()
	defer s.statsMu.RUnlock()
	defer s.concurrencyMu.RUnlock()

	// Count active requests globally and per user
	queuedRequests := 0 // Currently we don't queue, we fail fast

	userActiveCount := make(map[string]int)
	for username, count := range s.userRequestCount {
		if count > 0 {
			userActiveCount[username] = count
		}
	}

	return ConcurrencyStats{
		ActiveRequests:  s.activeRequests,
		MaxConcurrent:   s.maxConcurrent,
		QueuedRequests:  queuedRequests,
		TotalRequests:   s.totalRequests,
		UserActiveCount: userActiveCount,
		MaxPerUser:      s.maxPerUser,
	}
}

// acquireGlobalSlot attempts to acquire a global concurrency slot
func (s *AIService) acquireGlobalSlot(ctx context.Context) error {
	select {
	case s.globalSemaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return contextutils.WrapErrorf(contextutils.ErrTimeout, "request cancelled while waiting for global AI slot: %w", ctx.Err())
	default:
		return contextutils.WrapErrorf(contextutils.ErrServiceUnavailable, "AI service at capacity (%d concurrent requests), please try again", s.maxConcurrent)
	}
}

// releaseGlobalSlot releases a global concurrency slot
func (s *AIService) releaseGlobalSlot(ctx context.Context) {
	s.concurrencyMu.Lock()
	defer s.concurrencyMu.Unlock()

	select {
	case <-s.globalSemaphore:
		// Successfully released a slot
		s.statsMu.Lock()
		if s.activeRequests > 0 {
			s.activeRequests--
		}
		s.statsMu.Unlock()
	default:
		// No slot was acquired
		s.logger.Warn(ctx, "WARNING: Attempted to release global AI slot but none were acquired", nil)
	}
}

// acquireUserSlot acquires a user-specific concurrency slot
func (s *AIService) acquireUserSlot(_ context.Context, username string) error {
	s.concurrencyMu.Lock()
	defer s.concurrencyMu.Unlock()

	currentCount := s.userRequestCount[username]
	if currentCount >= s.maxPerUser {
		return contextutils.WrapErrorf(contextutils.ErrServiceUnavailable, "user concurrency limit exceeded for %s: %d/%d", username, currentCount, s.maxPerUser)
	}

	s.userRequestCount[username] = currentCount + 1
	return nil
}

// releaseUserSlot releases a user-specific concurrency slot
func (s *AIService) releaseUserSlot(ctx context.Context, username string) {
	s.concurrencyMu.Lock()
	defer s.concurrencyMu.Unlock()

	currentCount := s.userRequestCount[username]
	if currentCount > 0 {
		s.userRequestCount[username] = currentCount - 1
	} else {
		s.logger.Warn(ctx, "WARNING: Attempted to release user AI slot but none were acquired", map[string]interface{}{
			"username": username,
		})
	}
}

// incrementTotalRequests increments the total request counter
func (s *AIService) incrementTotalRequests() {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.totalRequests++
}

// withConcurrencyControl wraps an AI operation with concurrency limits
func (s *AIService) withConcurrencyControl(ctx context.Context, username string, operation func() error) error {
	// Check if service is shutting down
	if s.isShutdown() {
		return contextutils.WrapError(contextutils.ErrServiceUnavailable, "AI service is shutting down")
	}

	// Increment total request counter
	s.incrementTotalRequests()

	// Acquire global slot
	if err := s.acquireGlobalSlot(ctx); err != nil {
		return err
	}

	// Track active request
	s.statsMu.Lock()
	s.activeRequests++
	s.statsMu.Unlock()

	defer func() {
		s.releaseGlobalSlot(ctx)
	}()

	// Acquire per-user slot
	if err := s.acquireUserSlot(ctx, username); err != nil {
		return err
	}
	defer s.releaseUserSlot(ctx, username)

	// Execute the actual operation
	return operation()
}

// supportsGrammarField checks if the provider supports the grammar field
func (s *AIService) supportsGrammarField(provider string) bool {
	// Check if the provider supports grammar field
	if s.cfg.Providers == nil {
		return false
	}

	for _, providerConfig := range s.cfg.Providers {
		if providerConfig.Code == provider {
			return providerConfig.SupportsGrammar
		}
	}
	return false
}

// getQuestionBatchSize returns the maximum number of questions that can be generated in a single request for the given provider
func (s *AIService) getQuestionBatchSize(provider string) int {
	// Get the batch size for the provider
	if s.cfg.Providers == nil {
		return 1 // Default batch size
	}

	for _, p := range s.cfg.Providers {
		if p.Code == provider {
			if p.QuestionBatchSize > 0 {
				return p.QuestionBatchSize
			}
			break
		}
	}
	return 1 // Default batch size
}

// GetQuestionBatchSize returns the maximum number of questions that can be generated in a single request for the given provider
func (s *AIService) GetQuestionBatchSize(provider string) int {
	return s.getQuestionBatchSize(provider)
}

// VarietyService returns the variety service used by the AI service
func (s *AIService) VarietyService() *VarietyService {
	return s.varietyService
}

// TemplateManager exposes template rendering and example loading for prompts
func (s *AIService) TemplateManager() *AITemplateManager {
	return s.templateManager
}

// SupportsGrammarField reports whether the provider supports the grammar field
func (s *AIService) SupportsGrammarField(provider string) bool {
	return s.supportsGrammarField(provider)
}

// CallWithPrompt sends a raw prompt (and optional grammar) to the provider and returns the response
func (s *AIService) CallWithPrompt(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string) (string, error) {
	return s.callOpenAI(ctx, userConfig, prompt, grammar)
}
