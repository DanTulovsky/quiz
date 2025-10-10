package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/api"
	contextutils "quizapp/internal/utils"

	"github.com/google/uuid"
)

// ConversationServiceInterface defines the interface for AI conversation operations
type ConversationServiceInterface interface {
	// Conversation CRUD operations
	CreateConversation(ctx context.Context, userID uint, req *api.CreateConversationRequest) (*api.Conversation, error)
	GetConversation(ctx context.Context, conversationID string, userID uint) (*api.Conversation, error)
	GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]api.Conversation, int, error)
	UpdateConversation(ctx context.Context, conversationID string, userID uint, req *api.UpdateConversationRequest) (*api.Conversation, error)
	DeleteConversation(ctx context.Context, conversationID string, userID uint) error

	// Message operations
	AddMessage(ctx context.Context, conversationID string, userID uint, req *api.CreateMessageRequest) error
	GetConversationMessages(ctx context.Context, conversationID string, userID uint) ([]api.ChatMessage, error)

	// Search operations
	SearchMessages(ctx context.Context, userID uint, query string, limit, offset int) ([]api.ChatMessage, int, error)
	SearchConversations(ctx context.Context, userID uint, query string, limit, offset int) ([]api.Conversation, int, error)
}

// ConversationService handles all AI conversation-related operations
type ConversationService struct {
	db *sql.DB
}

// NewConversationService creates a new ConversationService
func NewConversationService(db *sql.DB) *ConversationService {
	return &ConversationService{
		db: db,
	}
}

// CreateConversation creates a new AI conversation
func (s *ConversationService) CreateConversation(ctx context.Context, userID uint, req *api.CreateConversationRequest) (*api.Conversation, error) {
	conversationID := uuid.New()

	query := `
		INSERT INTO ai_conversations (id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, created_at, updated_at`

	var conversation api.Conversation
	err := s.db.QueryRowContext(ctx, query,
		conversationID,
		userID,
		req.Title,
		time.Now(),
		time.Now(),
	).Scan(
		&conversation.Id,
		&conversation.UserId,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to create conversation")
	}

	return &conversation, nil
}

// GetConversation retrieves a conversation with all its messages
func (s *ConversationService) GetConversation(ctx context.Context, conversationID string, userID uint) (*api.Conversation, error) {
	// First get the conversation
	query := `
		SELECT id, user_id, title, created_at, updated_at
		FROM ai_conversations
		WHERE id = $1 AND user_id = $2`

	var conversation api.Conversation
	err := s.db.QueryRowContext(ctx, query, conversationID, userID).Scan(
		&conversation.Id,
		&conversation.UserId,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.ErrorWithContextf("conversation not found")
		}
		return nil, contextutils.WrapError(err, "failed to get conversation")
	}

	// Get the messages for this conversation
	messages, err := s.GetConversationMessages(ctx, conversationID, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get conversation messages")
	}

	conversation.Messages = &messages

	return &conversation, nil
}

// GetUserConversations retrieves all conversations for a user with pagination
func (s *ConversationService) GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]api.Conversation, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM ai_conversations WHERE user_id = $1`
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to count conversations")
	}

	// Get conversations with pagination
	query := `
		SELECT id, user_id, title, created_at, updated_at
		FROM ai_conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to query conversations")
	}
	defer func() { _ = rows.Close() }()

	var conversations []api.Conversation
	for rows.Next() {
		var conv api.Conversation
		err := rows.Scan(
			&conv.Id,
			&conv.UserId,
			&conv.Title,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to scan conversation")
		}
		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, contextutils.WrapError(err, "error iterating conversations")
	}

	return conversations, total, nil
}

// UpdateConversation updates a conversation's title
func (s *ConversationService) UpdateConversation(ctx context.Context, conversationID string, userID uint, req *api.UpdateConversationRequest) (*api.Conversation, error) {
	query := `
		UPDATE ai_conversations
		SET title = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, title, created_at, updated_at`

	var conversation api.Conversation
	err := s.db.QueryRowContext(ctx, query,
		req.Title,
		time.Now(),
		conversationID,
		userID,
	).Scan(
		&conversation.Id,
		&conversation.UserId,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.ErrorWithContextf("conversation not found")
		}
		return nil, contextutils.WrapError(err, "failed to update conversation")
	}

	return &conversation, nil
}

// DeleteConversation deletes a conversation and all its messages
func (s *ConversationService) DeleteConversation(ctx context.Context, conversationID string, userID uint) error {
	// First verify the conversation belongs to the user
	var ownerID uint
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM ai_conversations WHERE id = $1", conversationID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return contextutils.ErrorWithContextf("conversation not found")
		}
		return contextutils.WrapError(err, "failed to verify conversation ownership")
	}

	if ownerID != userID {
		return contextutils.ErrorWithContextf("conversation not found")
	}

	// Delete the conversation (CASCADE will delete associated messages)
	query := `DELETE FROM ai_conversations WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, conversationID, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to delete conversation")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.ErrorWithContextf("conversation not found")
	}

	return nil
}

// AddMessage adds a new message to a conversation
func (s *ConversationService) AddMessage(ctx context.Context, conversationID string, userID uint, req *api.CreateMessageRequest) error {
	// First verify the conversation belongs to the user
	var ownerID uint
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM ai_conversations WHERE id = $1", conversationID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return contextutils.ErrorWithContextf("conversation not found")
		}
		return contextutils.WrapError(err, "failed to verify conversation ownership")
	}

	if ownerID != userID {
		return contextutils.ErrorWithContextf("conversation not found")
	}

	messageID := uuid.New()
	query := `
		INSERT INTO ai_chat_messages (id, conversation_id, question_id, role, answer_json, bookmarked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, conversation_id, question_id, role, answer_json, bookmarked, created_at, updated_at`

	var message api.ChatMessage
	var questionIDPtr *int
	if req.QuestionId != nil {
		questionIDPtr = req.QuestionId
	}

	// Store content directly as JSON string
	contentJSON, err := json.Marshal(req.Content)
	if err != nil {
		return contextutils.WrapError(err, "failed to marshal message content")
	}

	err = s.db.QueryRowContext(ctx, query,
		messageID,
		conversationID,
		questionIDPtr,
		string(req.Role),
		contentJSON, // Store as JSON string value
		false,       // bookmarked defaults to false
		time.Now(),
		time.Now(),
	).Scan(
		&message.Id,
		&message.ConversationId,
		&message.QuestionId,
		&message.Role,
		&message.Content,
		&message.Bookmarked,
		&message.CreatedAt,
		&message.UpdatedAt,
	)
	if err != nil {
		return contextutils.WrapError(err, "failed to add message")
	}

	return nil
}

// GetConversationMessages retrieves all messages for a conversation
func (s *ConversationService) GetConversationMessages(ctx context.Context, conversationID string, userID uint) ([]api.ChatMessage, error) {
	// First verify the conversation belongs to the user
	var ownerID uint
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM ai_conversations WHERE id = $1", conversationID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.ErrorWithContextf("conversation not found")
		}
		return nil, contextutils.WrapError(err, "failed to verify conversation ownership")
	}

	if ownerID != userID {
		return nil, contextutils.ErrorWithContextf("conversation not found")
	}

	query := `
		SELECT id, conversation_id, question_id, role, answer_json, bookmarked, created_at, updated_at
		FROM ai_chat_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query messages")
	}
	defer func() { _ = rows.Close() }()

	var messages []api.ChatMessage
	for rows.Next() {
		var msg api.ChatMessage
		var questionIDPtr *int

		var answerText string
		err := rows.Scan(
			&msg.Id,
			&msg.ConversationId,
			&questionIDPtr,
			&msg.Role,
			&answerText,
			&msg.Bookmarked,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan message")
		}

		err = json.Unmarshal([]byte(answerText), &msg.Content)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to unmarshal message content")
		}

		if questionIDPtr != nil {
			msg.QuestionId = questionIDPtr
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating messages")
	}

	return messages, nil
}

// SearchMessages searches across all messages for a user
func (s *ConversationService) SearchMessages(ctx context.Context, userID uint, query string, limit, offset int) ([]api.ChatMessage, int, error) {
	// Clean and prepare the search query
	searchQuery := strings.TrimSpace(query)
	if searchQuery == "" {
		return nil, 0, contextutils.ErrorWithContextf("search query cannot be empty")
	}

	// Search in the answer_json column (which contains the message content as JSON string)
	// We need to search within the JSON string value, so we search for the pattern within quotes
	searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(searchQuery))

	// Get total count of matching messages
	countQuery := `
		SELECT COUNT(*)
		FROM ai_chat_messages m
		JOIN ai_conversations c ON m.conversation_id = c.id
		WHERE c.user_id = $1 AND LOWER(m.answer_json::text) LIKE $2`

	var total int
	err := s.db.QueryRowContext(ctx, countQuery, userID, searchTerm).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to count search results")
	}

	// Get messages with conversation titles
	querySQL := `
		SELECT m.id, m.conversation_id, m.question_id, m.role, m.answer_json::text, m.bookmarked, m.created_at, m.updated_at, c.title
		FROM ai_chat_messages m
		JOIN ai_conversations c ON m.conversation_id = c.id
		WHERE c.user_id = $1 AND LOWER(m.answer_json::text) LIKE $2
		ORDER BY m.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := s.db.QueryContext(ctx, querySQL, userID, searchTerm, limit, offset)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to search messages")
	}
	defer func() { _ = rows.Close() }()

	var messages []api.ChatMessage
	for rows.Next() {
		var msg api.ChatMessage
		var questionIDPtr *int
		var conversationTitle string

		var answerText string
		err := rows.Scan(
			&msg.Id,
			&msg.ConversationId,
			&questionIDPtr,
			&msg.Role,
			&answerText,
			&msg.Bookmarked,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&conversationTitle,
		)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to scan search result")
		}

		// Since we're storing as plain text in JSONB, just use it directly
		msg.Content = answerText

		if questionIDPtr != nil {
			msg.QuestionId = questionIDPtr
		}

		// Content is retrieved directly as text using ->> operator

		// Set conversation title for search results
		msg.ConversationTitle = &conversationTitle

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, contextutils.WrapError(err, "error iterating search results")
	}

	return messages, total, nil
}

// SearchConversations searches across all conversations for a user
func (s *ConversationService) SearchConversations(ctx context.Context, userID uint, query string, limit, offset int) ([]api.Conversation, int, error) {
	// Clean and prepare the search query
	searchQuery := strings.TrimSpace(query)
	if searchQuery == "" {
		return nil, 0, contextutils.ErrorWithContextf("search query cannot be empty")
	}

	// Search in both conversation titles and message content
	searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(searchQuery))

	// Get total count of matching conversations
	countQuery := `
		SELECT COUNT(DISTINCT c.id)
		FROM ai_conversations c
		LEFT JOIN ai_chat_messages m ON c.id = m.conversation_id
		WHERE c.user_id = $1
		AND (LOWER(c.title) LIKE $2 OR LOWER(m.answer_json::text) LIKE $2)`

	var total int
	err := s.db.QueryRowContext(ctx, countQuery, userID, searchTerm).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to count search results")
	}

	// Get conversations with their latest message info
	querySQL := `
		SELECT DISTINCT c.id, c.title, c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM ai_chat_messages WHERE conversation_id = c.id) as message_count,
		       (SELECT answer_json::text FROM ai_chat_messages WHERE conversation_id = c.id ORDER BY created_at ASC LIMIT 1) as first_message,
		       (SELECT answer_json::text FROM ai_chat_messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1) as last_message
		FROM ai_conversations c
		LEFT JOIN ai_chat_messages m ON c.id = m.conversation_id
		WHERE c.user_id = $1
		AND (LOWER(c.title) LIKE $2 OR LOWER(m.answer_json::text) LIKE $2)
		ORDER BY c.updated_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := s.db.QueryContext(ctx, querySQL, userID, searchTerm, limit, offset)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to search conversations")
	}
	defer func() { _ = rows.Close() }()

	var conversations []api.Conversation
	for rows.Next() {
		var conv api.Conversation
		var firstMessagePtr, lastMessagePtr *string
		var messageCount int

		err := rows.Scan(
			&conv.Id,
			&conv.Title,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&messageCount,
			&firstMessagePtr,
			&lastMessagePtr,
		)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to scan search result")
		}

		// Set the preview message to the last message if available, otherwise the first message
		previewMessage := ""
		if lastMessagePtr != nil {
			previewMessage = *lastMessagePtr
		} else if firstMessagePtr != nil {
			previewMessage = *firstMessagePtr
		}

		// Add preview_message field for frontend compatibility
		conv.Messages = &[]api.ChatMessage{
			{
				Content: previewMessage,
			},
		}

		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, contextutils.WrapError(err, "error iterating search results")
	}

	return conversations, total, nil
}
