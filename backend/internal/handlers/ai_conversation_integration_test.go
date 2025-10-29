//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AIConversationIntegrationTestSuite tests the AI conversation handler with real database interactions
type AIConversationIntegrationTestSuite struct {
	suite.Suite
	Router          *gin.Engine
	UserService     *services.UserService
	LearningService *services.LearningService
	Config          *config.Config
	TestUserID      int
	DB              *sql.DB
}

func (suite *AIConversationIntegrationTestSuite) SetupSuite() {
	// Use shared test database setup
	db := services.SharedTestDBSetup(suite.T())

	// Load config
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Create services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	storyService := services.NewStoryService(db, cfg, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)
	generationHintService := services.NewGenerationHintService(db, logger)
	usageStatsService := services.NewUsageStatsService(cfg, db, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(db, logger)
	translationService := services.NewTranslationService(cfg, usageStatsService, translationCacheRepo, logger)

	// Create test user
	createdUser, err := userService.CreateUserWithPassword(context.Background(), "testuser_ai", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.TestUserID = createdUser.ID

	// Use the real application router
	snippetsService := services.NewSnippetsService(db, suite.Config, logger)
    authAPIKeyService := services.NewAuthAPIKeyService(db, logger)
	suite.Router = NewRouter(
		suite.Config,
		userService,
		questionService,
		learningService,
		aiService,
		workerService,
		dailyQuestionService,
		storyService,
		services.NewConversationService(db),
		oauthService,
		generationHintService,
		translationService,
		snippetsService,
		usageStatsService,
		services.NewWordOfTheDayService(db, logger),
		authAPIKeyService,
		logger,
	)

	suite.UserService = userService
	suite.LearningService = learningService
	suite.DB = db
}

func (suite *AIConversationIntegrationTestSuite) TearDownSuite() {
	// Cleanup test data
	suite.UserService.DeleteUser(context.Background(), suite.TestUserID)
	// Close database connection
	if suite.DB != nil {
		suite.DB.Close()
	}
}

func (suite *AIConversationIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	suite.cleanupDatabase()
}

func (suite *AIConversationIntegrationTestSuite) cleanupDatabase() {
	// Use the shared database cleanup function
	services.CleanupTestDatabase(suite.DB, suite.T())

	// Recreate test user
	createdUser, err := suite.UserService.CreateUserWithPassword(context.Background(), "testuser_ai", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.TestUserID = createdUser.ID
}

func (suite *AIConversationIntegrationTestSuite) login() string {
	loginReq := api.LoginRequest{
		Username: "testuser_ai",
		Password: "testpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Extract cookie from response
	cookies := w.Result().Cookies()
	var sessionCookie string
	for _, cookie := range cookies {
		if cookie.Name == config.SessionName {
			sessionCookie = cookie.String()
			break
		}
	}
	require.NotEmpty(suite.T(), sessionCookie)
	return sessionCookie
}

// TestCreateConversation_Success tests successful conversation creation
func (suite *AIConversationIntegrationTestSuite) TestCreateConversation_Success() {
	sessionCookie := suite.login()

	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}

	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Conversation", response.Title)
	assert.Equal(suite.T(), suite.TestUserID, response.UserId)
}

// TestCreateConversation_Unauthorized tests conversation creation without authentication
func (suite *AIConversationIntegrationTestSuite) TestCreateConversation_Unauthorized() {
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}

	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestCreateConversation_InvalidJSON tests conversation creation with invalid JSON
func (suite *AIConversationIntegrationTestSuite) TestCreateConversation_InvalidJSON() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBufferString("invalid json"))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestGetConversations_Success tests successful conversation listing
func (suite *AIConversationIntegrationTestSuite) TestGetConversations_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Now get conversations
	req, _ = http.NewRequest("GET", "/v1/ai/conversations", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	conversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), conversations, 1)
}

// TestGetConversations_WithMessageCount tests that conversations include message_count field
func (suite *AIConversationIntegrationTestSuite) TestGetConversations_WithMessageCount() {
	sessionCookie := suite.login()

	// Create a conversation
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation with Messages",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add multiple messages to the conversation
	messageContents := []string{
		"Hello, AI!",
		"How are you today?",
		"Can you help me with grammar?",
	}

	for _, content := range messageContents {
		messageReq := api.CreateMessageRequest{
			Role: api.CreateMessageRequestRoleUser,
			Content: struct {
				Text *string `json:"text,omitempty"`
			}{
				Text: &content,
			},
		}
		msgBody, _ := json.Marshal(messageReq)
		req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		require.Equal(suite.T(), http.StatusCreated, w.Code)
	}

	// Get conversations and verify message_count is included and accurate
	req, _ = http.NewRequest("GET", "/v1/ai/conversations", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	conversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), conversations, 1)

	// Parse the conversation response which should include message_count
	conversationData, ok := conversations[0].(map[string]interface{})
	assert.True(suite.T(), ok)

	// Verify message_count is present and correct
	messageCount, ok := conversationData["message_count"].(float64)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), float64(len(messageContents)), messageCount)
}

// TestGetConversations_MessageCountAccuracy tests that message_count matches actual message count in database
func (suite *AIConversationIntegrationTestSuite) TestGetConversations_MessageCountAccuracy() {
	sessionCookie := suite.login()

	// Create multiple conversations with different numbers of messages
	testCases := []struct {
		title        string
		messageCount int
	}{
		{"Empty Conversation", 0},
		{"Single Message", 1},
		{"Multiple Messages", 3},
	}

	var conversationIDs []string

	for _, tc := range testCases {
		// Create conversation
		createReq := api.CreateConversationRequest{
			Title: tc.title,
		}
		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		require.Equal(suite.T(), http.StatusCreated, w.Code)

		var conversation api.Conversation
		err := json.Unmarshal(w.Body.Bytes(), &conversation)
		require.NoError(suite.T(), err)
		conversationIDs = append(conversationIDs, conversation.Id.String())

		// Add messages if needed
		for i := 0; i < tc.messageCount; i++ {
			messageReq := api.CreateMessageRequest{
				Role: api.CreateMessageRequestRoleUser,
				Content: struct {
					Text *string `json:"text,omitempty"`
				}{
					Text: stringPtr(fmt.Sprintf("Message %d", i+1)),
				},
			}
			msgBody, _ := json.Marshal(messageReq)
			req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
			req.Header.Set("Cookie", sessionCookie)
			req.Header.Set("Content-Type", "application/json")

			w = httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)
			require.Equal(suite.T(), http.StatusCreated, w.Code)
		}
	}

	// Get all conversations and verify message counts match expected values
	req, _ := http.NewRequest("GET", "/v1/ai/conversations", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	conversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), conversations, len(testCases))

	// Check each conversation's message count
	for i, conversationInterface := range conversations {
		conversationData, ok := conversationInterface.(map[string]interface{})
		assert.True(suite.T(), ok)

		messageCount, ok := conversationData["message_count"].(float64)
		assert.True(suite.T(), ok)

		expectedCount := testCases[i].messageCount
		assert.Equal(suite.T(), float64(expectedCount), messageCount,
			"Message count mismatch for conversation %s", testCases[i].title)
	}
}

// TestGetConversations_Unauthorized tests conversation listing without authentication
func (suite *AIConversationIntegrationTestSuite) TestGetConversations_Unauthorized() {
	req, _ := http.NewRequest("GET", "/v1/ai/conversations", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestGetConversation_Success tests successful conversation retrieval with messages
func (suite *AIConversationIntegrationTestSuite) TestGetConversation_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message to the conversation
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, AI!"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Now get the conversation with messages
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/ai/conversations/%s", conversation.Id.String()), nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response api.Conversation
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), conversation.Id, response.Id)
	require.NotNil(suite.T(), response.Messages)
	assert.Len(suite.T(), *response.Messages, 1)
	assert.Equal(suite.T(), "Hello, AI!", (*response.Messages)[0].Content)
}

// TestGetConversation_NotFound tests getting a non-existent conversation
func (suite *AIConversationIntegrationTestSuite) TestGetConversation_NotFound() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("GET", "/v1/ai/conversations/550e8400-e29b-41d4-a716-446655440000", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestUpdateConversation_Success tests successful conversation update
func (suite *AIConversationIntegrationTestSuite) TestUpdateConversation_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Original Title",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Update the conversation
	updateReq := api.UpdateConversationRequest{
		Title: "Updated Title",
	}
	updateBody, _ := json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/v1/ai/conversations/%s", conversation.Id.String()), bytes.NewBuffer(updateBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response api.Conversation
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Title", response.Title)
}

// TestDeleteConversation_Success tests successful conversation deletion
func (suite *AIConversationIntegrationTestSuite) TestDeleteConversation_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Delete the conversation
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/ai/conversations/%s", conversation.Id.String()), nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify conversation is deleted by trying to get it
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/ai/conversations/%s", conversation.Id.String()), nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestAddMessage_Success tests successful message addition
func (suite *AIConversationIntegrationTestSuite) TestAddMessage_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, AI!"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
}

// TestSearchConversations_Success tests successful conversation search
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Spanish Learning Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message containing "Spanish"
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, I love learning Spanish!"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Search for conversations containing "Spanish"
	req, _ = http.NewRequest("GET", "/v1/ai/search?q=Spanish", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	conversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), conversations, 1)
}

// TestSearchConversations_NoQuery tests search without query parameter
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_NoQuery() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("GET", "/v1/ai/search", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestSearchConversations_Unauthorized tests search without authentication
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_Unauthorized() {
	req, _ := http.NewRequest("GET", "/v1/ai/search?q=test", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestSearchConversations_ByTitle tests searching conversations by title
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_ByTitle() {
	sessionCookie := suite.login()

	// Create conversations with different titles
	conversations := []string{
		"French Grammar Help",
		"Spanish Vocabulary Practice",
		"German Pronunciation Tips",
	}

	var createdConversations []api.Conversation
	for _, title := range conversations {
		createReq := api.CreateConversationRequest{
			Title: title,
		}
		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		require.Equal(suite.T(), http.StatusCreated, w.Code)

		var conversation api.Conversation
		err := json.Unmarshal(w.Body.Bytes(), &conversation)
		require.NoError(suite.T(), err)
		createdConversations = append(createdConversations, conversation)
	}

	// Search for conversations containing "Vocabulary"
	req, _ := http.NewRequest("GET", "/v1/ai/search?q=Vocabulary", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	foundConversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), foundConversations, 1)
}

// TestSearchConversations_ByMessageContent tests searching conversations by message content
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_ByMessageContent() {
	sessionCookie := suite.login()

	// Create a conversation
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add messages with different content
	messages := []string{
		"I need help with irregular verbs",
		"Can you explain past participles?",
		"What about future tense?",
	}

	for _, content := range messages {
		messageReq := api.CreateMessageRequest{
			Role: api.CreateMessageRequestRoleUser,
			Content: struct {
				Text *string `json:"text,omitempty"`
			}{
				Text: &content,
			},
		}
		msgBody, _ := json.Marshal(messageReq)
		req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		require.Equal(suite.T(), http.StatusCreated, w.Code)
	}

	// Search for conversations containing "participles"
	req, _ = http.NewRequest("GET", "/v1/ai/search?q=participles", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	foundConversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), foundConversations, 1)
}

// TestSearchConversations_NoResults tests search with no matching results
func (suite *AIConversationIntegrationTestSuite) TestSearchConversations_NoResults() {
	sessionCookie := suite.login()

	// Create a conversation with some content
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, I love learning!"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Search for something that won't match
	req, _ = http.NewRequest("GET", "/v1/ai/search?q=nonexistentterm", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	foundConversations, ok := response["conversations"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), foundConversations, 0)
}

// TestToggleMessageBookmark_Success tests successful message bookmarking and unbookmarking
func (suite *AIConversationIntegrationTestSuite) TestToggleMessageBookmark_Success() {
	sessionCookie := suite.login()

	// Create a conversation first
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message to the conversation
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, AI!"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Get the message details to get the message ID
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/ai/conversations/%s", conversation.Id.String()), nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code)

	var conversationWithMessages api.Conversation
	err = json.Unmarshal(w.Body.Bytes(), &conversationWithMessages)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), conversationWithMessages.Messages)
	require.Len(suite.T(), *conversationWithMessages.Messages, 1)

	messageID := (*conversationWithMessages.Messages)[0].Id

	// Test bookmarking the message (should start as false and become true)
	bookmarkReq := struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}{
		ConversationID: conversation.Id.String(),
		MessageID:      messageID.String(),
	}
	bookmarkBody, _ := json.Marshal(bookmarkReq)
	req, _ = http.NewRequest("PUT", "/v1/ai/conversations/bookmark", bytes.NewBuffer(bookmarkBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var bookmarkResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &bookmarkResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, bookmarkResponse["bookmarked"])

	// Test unbookmarking the message (should become false)
	req, _ = http.NewRequest("PUT", "/v1/ai/conversations/bookmark", bytes.NewBuffer(bookmarkBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &bookmarkResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), false, bookmarkResponse["bookmarked"])
}

// TestToggleMessageBookmark_Unauthorized tests bookmarking without authentication
func (suite *AIConversationIntegrationTestSuite) TestToggleMessageBookmark_Unauthorized() {
	bookmarkReq := struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}{
		ConversationID: "550e8400-e29b-41d4-a716-446655440000",
		MessageID:      "550e8400-e29b-41d4-a716-446655440001",
	}
	bookmarkBody, _ := json.Marshal(bookmarkReq)
	req, _ := http.NewRequest("PUT", "/v1/ai/conversations/bookmark", bytes.NewBuffer(bookmarkBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestToggleMessageBookmark_NotFound tests bookmarking a non-existent message
func (suite *AIConversationIntegrationTestSuite) TestToggleMessageBookmark_NotFound() {
	sessionCookie := suite.login()

	// Try to bookmark a non-existent message
	bookmarkReq := struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}{
		ConversationID: "550e8400-e29b-41d4-a716-446655440000",
		MessageID:      "550e8400-e29b-41d4-a716-446655440001",
	}
	bookmarkBody, _ := json.Marshal(bookmarkReq)
	req, _ := http.NewRequest("PUT", "/v1/ai/conversations/bookmark", bytes.NewBuffer(bookmarkBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestGetBookmarkedMessages_Success tests successful retrieval of bookmarked messages
func (suite *AIConversationIntegrationTestSuite) TestGetBookmarkedMessages_Success() {
	sessionCookie := suite.login()

	// Create a conversation
	createReq := api.CreateConversationRequest{
		Title: "Test Conversation for Bookmarks",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/ai/conversations", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var conversation api.Conversation
	err := json.Unmarshal(w.Body.Bytes(), &conversation)
	require.NoError(suite.T(), err)

	// Add a message
	messageReq := api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleAssistant,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("This is a helpful AI response about learning Italian"),
		},
	}
	msgBody, _ := json.Marshal(messageReq)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/ai/conversations/%s/messages", conversation.Id.String()), bytes.NewBuffer(msgBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var message api.ChatMessage
	err = json.Unmarshal(w.Body.Bytes(), &message)
	require.NoError(suite.T(), err)

	// Bookmark the message
	bookmarkReq := struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}{
		ConversationID: conversation.Id.String(),
		MessageID:      message.Id.String(),
	}
	bookmarkBody, _ := json.Marshal(bookmarkReq)
	req, _ = http.NewRequest("PUT", "/v1/ai/conversations/bookmark", bytes.NewBuffer(bookmarkBody))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code)

	// Get bookmarked messages
	req, _ = http.NewRequest("GET", "/v1/ai/bookmarks", nil)
	req.Header.Set("Cookie", sessionCookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	messages, ok := response["messages"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(messages), 1)

	// Verify the message content
	firstMessage := messages[0].(map[string]interface{})
	assert.NotEmpty(suite.T(), firstMessage["id"])
}

// TestGetBookmarkedMessages_Unauthorized tests accessing bookmarks without authentication
func (suite *AIConversationIntegrationTestSuite) TestGetBookmarkedMessages_Unauthorized() {
	req, _ := http.NewRequest("GET", "/v1/ai/bookmarks", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}
