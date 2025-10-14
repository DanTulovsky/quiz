//go:build integration
// +build integration

package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
)

// ConversationServiceIntegrationTestSuite tests the conversation service with a real database
type ConversationServiceIntegrationTestSuite struct {
	suite.Suite
	db              *sql.DB
	conversationSvc ConversationServiceInterface
	userSvc         UserServiceInterface
	cfg             *config.Config
	logger          *observability.Logger
	testUser        *models.User
}

// SetupSuite runs once before all tests in the suite
func (suite *ConversationServiceIntegrationTestSuite) SetupSuite() {
	// Initialize database using the same pattern as other integration tests
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	suite.db = db

	// Load the real config
	cfg, err := config.NewConfig()
	suite.Require().NoError(err)
	suite.cfg = cfg

	// Create services
	suite.userSvc = NewUserServiceWithLogger(db, cfg, logger)
	suite.conversationSvc = NewConversationService(db)

	// Create test user
	testCtx := context.Background()
	user, err := suite.userSvc.CreateUserWithPassword(testCtx, "conversation_test_user", "password123", "english", "A1")
	suite.Require().NoError(err)
	suite.testUser = user

	// Update user with required fields for validation middleware
	_, err = suite.db.Exec(`
		UPDATE users
		SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5
		WHERE id = $6
	`, "conversation_test@example.com", "UTC", "ollama", "llama3", time.Now(), user.ID)
	suite.Require().NoError(err)
}

// TearDownSuite runs once after all tests in the suite
func (suite *ConversationServiceIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		// Clean up test data
		ctx := context.Background()
		_, err := suite.db.ExecContext(ctx, "DELETE FROM ai_chat_messages WHERE conversation_id IN (SELECT id FROM ai_conversations WHERE user_id = $1)", suite.testUser.ID)
		if err != nil {
			suite.T().Logf("Warning: Failed to cleanup chat messages: %v", err)
		}
		_, err = suite.db.ExecContext(ctx, "DELETE FROM ai_conversations WHERE user_id = $1", suite.testUser.ID)
		if err != nil {
			suite.T().Logf("Warning: Failed to cleanup conversations: %v", err)
		}
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *ConversationServiceIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data for this user
	ctx := context.Background()
	_, err := suite.db.ExecContext(ctx, "DELETE FROM ai_chat_messages WHERE conversation_id IN (SELECT id FROM ai_conversations WHERE user_id = $1)", suite.testUser.ID)
	suite.Require().NoError(err)
	_, err = suite.db.ExecContext(ctx, "DELETE FROM ai_conversations WHERE user_id = $1", suite.testUser.ID)
	suite.Require().NoError(err)
}

// TestCreateConversation tests creating a new conversation
func (suite *ConversationServiceIntegrationTestSuite) TestCreateConversation() {
	ctx := context.Background()

	req := &api.CreateConversationRequest{
		Title: "Test Conversation",
	}

	conversation, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(conversation)

	// Verify conversation properties
	suite.Assert().NotEmpty(conversation.Id.String())
	suite.Assert().Equal(suite.testUser.ID, conversation.UserId)
	suite.Assert().Equal("Test Conversation", conversation.Title)
	suite.Assert().NotZero(conversation.CreatedAt)
	suite.Assert().NotZero(conversation.UpdatedAt)
}

// TestGetUserConversations tests retrieving user conversations with pagination
func (suite *ConversationServiceIntegrationTestSuite) TestGetUserConversations() {
	ctx := context.Background()

	// Create multiple conversations
	for i := 0; i < 3; i++ {
		req := &api.CreateConversationRequest{
			Title: fmt.Sprintf("Test Conversation %d", i+1),
		}
		_, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
		suite.Require().NoError(err)
	}

	// Test getting conversations with pagination
	conversations, total, err := suite.conversationSvc.GetUserConversations(ctx, uint(suite.testUser.ID), 2, 0)
	suite.Require().NoError(err)
	suite.Assert().Len(conversations, 2) // Should return 2 due to limit
	suite.Assert().Equal(3, total)       // But total should be 3

	// Test getting remaining conversations
	conversations2, total2, err := suite.conversationSvc.GetUserConversations(ctx, uint(suite.testUser.ID), 2, 2)
	suite.Require().NoError(err)
	suite.Assert().Len(conversations2, 1) // Should return 1 more
	suite.Assert().Equal(3, total2)       // Total should still be 3
}

// TestGetConversationWithMessages tests retrieving a conversation with its messages
func (suite *ConversationServiceIntegrationTestSuite) TestGetConversationWithMessages() {
	ctx := context.Background()

	// Create a conversation
	req := &api.CreateConversationRequest{
		Title: "Conversation with Messages",
	}
	conversation, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
	suite.Require().NoError(err)

	// Add messages to the conversation
	for i := 0; i < 2; i++ {
		msgReq := &api.CreateMessageRequest{
			Role: api.CreateMessageRequestRoleUser,
			Content: struct {
				Text *string `json:"text,omitempty"`
			}{
				Text: stringPtr(fmt.Sprintf("Test message %d", i+1)),
			},
		}
		_, err := suite.conversationSvc.AddMessage(ctx, conversation.Id.String(), uint(suite.testUser.ID), msgReq)
		suite.Require().NoError(err)
	}

	// Retrieve conversation with messages
	retrieved, err := suite.conversationSvc.GetConversation(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Require().NoError(err)
	suite.Require().NotNil(retrieved)

	// Verify conversation details
	suite.Assert().Equal(conversation.Id.String(), retrieved.Id.String())
	suite.Assert().Equal(conversation.UserId, retrieved.UserId)
	suite.Assert().Equal(conversation.Title, retrieved.Title)

	// Verify messages
	suite.Require().NotNil(retrieved.Messages)
	suite.Assert().Len(*retrieved.Messages, 2)
	suite.Assert().Equal(api.ChatMessageRoleUser, (*retrieved.Messages)[0].Role)
	suite.Assert().Equal("Test message 1", *(*retrieved.Messages)[0].Content.Text)
}

// TestUpdateConversation tests updating a conversation
func (suite *ConversationServiceIntegrationTestSuite) TestUpdateConversation() {
	ctx := context.Background()

	// Create a conversation
	req := &api.CreateConversationRequest{
		Title: "Original Title",
	}
	conversation, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
	suite.Require().NoError(err)

	// Update the conversation
	updateReq := &api.UpdateConversationRequest{
		Title: "Updated Title",
	}
	updated, err := suite.conversationSvc.UpdateConversation(ctx, conversation.Id.String(), uint(suite.testUser.ID), updateReq)
	suite.Require().NoError(err)
	suite.Require().NotNil(updated)

	// Verify update
	suite.Assert().Equal("Updated Title", updated.Title)
	suite.Assert().True(updated.UpdatedAt.After(conversation.CreatedAt))
}

// TestDeleteConversation tests deleting a conversation
func (suite *ConversationServiceIntegrationTestSuite) TestDeleteConversation() {
	ctx := context.Background()

	// Create a conversation
	req := &api.CreateConversationRequest{
		Title: "To Be Deleted",
	}
	conversation, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
	suite.Require().NoError(err)

	// Verify conversation exists
	retrieved, err := suite.conversationSvc.GetConversation(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Require().NoError(err)
	suite.Require().NotNil(retrieved)

	// Delete the conversation
	err = suite.conversationSvc.DeleteConversation(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Require().NoError(err)

	// Verify conversation is deleted (should return error)
	_, err = suite.conversationSvc.GetConversation(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Assert().Error(err) // Should fail to find conversation
}

// TestAddMessage tests adding messages to a conversation
func (suite *ConversationServiceIntegrationTestSuite) TestAddMessage() {
	ctx := context.Background()

	// Create a conversation
	req := &api.CreateConversationRequest{
		Title: "Message Test Conversation",
	}
	conversation, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), req)
	suite.Require().NoError(err)

	// Add a user message
	msgReq := &api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Hello, AI!"),
		},
	}
	_, err = suite.conversationSvc.AddMessage(ctx, conversation.Id.String(), uint(suite.testUser.ID), msgReq)
	suite.Require().NoError(err)

	// Verify user message
	messages, err := suite.conversationSvc.GetConversationMessages(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)
	suite.Assert().Equal(api.ChatMessageRoleUser, messages[0].Role)
	suite.Assert().Equal("Hello, AI!", *messages[0].Content.Text)

	// Add an assistant message
	msg := "Hello! How can I help you today?"
	assistantMsgReq := &api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleAssistant,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: &msg,
		},
	}
	fmt.Println(assistantMsgReq.Content)
	_, err = suite.conversationSvc.AddMessage(ctx, conversation.Id.String(), uint(suite.testUser.ID), assistantMsgReq)
	suite.Require().NoError(err)

	// Retrieve messages for the conversation
	messages, err = suite.conversationSvc.GetConversationMessages(ctx, conversation.Id.String(), uint(suite.testUser.ID))
	suite.Require().NoError(err)
	suite.Require().Len(messages, 2)

	// Check that the assistant message is present
	found := false
	for _, m := range messages {
		if m.Role == api.ChatMessageRoleAssistant && m.Content.Text != nil && *m.Content.Text == msg {
			found = true
			break
		}
	}
	suite.Assert().True(found, "Assistant message should be present in conversation messages")
}

// TestSearchMessages tests searching across user messages
func (suite *ConversationServiceIntegrationTestSuite) TestSearchMessages() {
	ctx := context.Background()

	// Create two conversations with different messages
	conv1Req := &api.CreateConversationRequest{Title: "Search Test 1"}
	conv1, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), conv1Req)
	suite.Require().NoError(err)

	conv2Req := &api.CreateConversationRequest{Title: "Search Test 2"}
	conv2, err := suite.conversationSvc.CreateConversation(ctx, uint(suite.testUser.ID), conv2Req)
	suite.Require().NoError(err)

	// Add messages with searchable content
	msg1Req := &api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("I love learning Spanish"),
		},
	}
	_, err = suite.conversationSvc.AddMessage(ctx, conv1.Id.String(), uint(suite.testUser.ID), msg1Req)
	suite.Require().NoError(err)

	msg2Req := &api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleAssistant,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("Spanish grammar can be challenging"),
		},
	}
	_, err = suite.conversationSvc.AddMessage(ctx, conv1.Id.String(), uint(suite.testUser.ID), msg2Req)
	suite.Require().NoError(err)

	msg3Req := &api.CreateMessageRequest{
		Role: api.CreateMessageRequestRoleUser,
		Content: struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: stringPtr("I want to learn French instead"),
		},
	}
	_, err = suite.conversationSvc.AddMessage(ctx, conv2.Id.String(), uint(suite.testUser.ID), msg3Req)
	suite.Require().NoError(err)

	// Search for "Spanish"
	results, total, err := suite.conversationSvc.SearchMessages(ctx, uint(suite.testUser.ID), "Spanish", 10, 0)
	suite.Require().NoError(err)
	suite.Assert().Equal(2, total) // Should find 2 messages containing "Spanish"
	suite.Assert().Len(results, 2)

	// Verify search results include conversation context
	for _, result := range results {
		suite.Require().NotNil(result.ConversationTitle) // Should be populated in search results
		suite.Assert().NotEmpty(*result.ConversationTitle)
		suite.Assert().Contains(*result.Content.Text, "Spanish")
	}

	// Search for "French" (should find 1 message)
	frenchResults, frenchTotal, err := suite.conversationSvc.SearchMessages(ctx, uint(suite.testUser.ID), "French", 10, 0)
	suite.Require().NoError(err)
	suite.Assert().Equal(1, frenchTotal)
	suite.Assert().Len(frenchResults, 1)
	suite.Require().NotNil(frenchResults[0].ConversationTitle)
	suite.Assert().NotEmpty(*frenchResults[0].ConversationTitle)
	suite.Assert().Contains(*frenchResults[0].Content.Text, "French")
}

// TestMain runs the test suite
func TestConversationServiceIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(ConversationServiceIntegrationTestSuite))
}
