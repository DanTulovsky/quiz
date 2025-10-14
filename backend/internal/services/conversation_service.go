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
	ToggleMessageBookmark(ctx context.Context, conversationID string, messageID string, userID uint) (bool, error)

	// Search operations
	SearchMessages(ctx context.Context, userID uint, query string, limit, offset int) ([]api.ChatMessage, int, error)
	SearchConversations(ctx context.Context, userID uint, query string, limit, offset int) ([]api.Conversation, int, error)

    // Utility operations
    // GetUserMessageCounts returns a map of conversation ID -> message count for the user's conversations
    GetUserMessageCounts(ctx context.Context, userID uint) (map[string]int, error)
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

	// Ensure messages is never nil - always point to a valid slice
	if messages == nil {
		messages = []api.ChatMessage{}
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

// GetUserMessageCounts returns message counts for all conversations for a user
func (s *ConversationService) GetUserMessageCounts(ctx context.Context, userID uint) (map[string]int, error) {
    query := `
        SELECT c.id::text AS id, COUNT(m.id) AS message_count
        FROM ai_conversations c
        LEFT JOIN ai_chat_messages m ON m.conversation_id = c.id
        WHERE c.user_id = $1
        GROUP BY c.id`

    rows, err := s.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, contextutils.WrapError(err, "failed to query message counts")
    }
    defer func() { _ = rows.Close() }()

    counts := make(map[string]int)
    for rows.Next() {
        var id string
        var count int
        if err := rows.Scan(&id, &count); err != nil {
            return nil, contextutils.WrapError(err, "failed to scan message count")
        }
        counts[id] = count
    }
    if err := rows.Err(); err != nil {
        return nil, contextutils.WrapError(err, "error iterating message counts")
    }
    return counts, nil
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

	var contentBytes []byte
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
		&contentBytes,
		&message.Bookmarked,
		&message.CreatedAt,
		&message.UpdatedAt,
	)
	if err != nil {
		return contextutils.WrapError(err, "failed to add message")
	}

	// Unmarshal the content from bytes
	var contentObj struct {
		Text *string `json:"text,omitempty"`
	}
	err = json.Unmarshal(contentBytes, &contentObj)
	if err != nil {
		return contextutils.WrapError(err, "failed to unmarshal message content")
	}
	message.Content = contentObj

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

		var answerBytes []byte
		err := rows.Scan(
			&msg.Id,
			&msg.ConversationId,
			&questionIDPtr,
			&msg.Role,
			&answerBytes,
			&msg.Bookmarked,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan message")
		}

		// Content is now stored as an object, unmarshal accordingly
		var contentObj struct {
			Text *string `json:"text,omitempty"`
		}
		err = json.Unmarshal(answerBytes, &contentObj)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to unmarshal message content")
		}
		msg.Content = contentObj
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

// ToggleMessageBookmark toggles the bookmark status of a message
func (s *ConversationService) ToggleMessageBookmark(ctx context.Context, conversationID string, messageID string, userID uint) (bool, error) {
	// First verify the conversation belongs to the user
	var ownerID uint
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM ai_conversations WHERE id = $1", conversationID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, contextutils.ErrorWithContextf("conversation not found")
		}
		return false, contextutils.WrapError(err, "failed to verify conversation ownership")
	}

	if ownerID != userID {
		return false, contextutils.ErrorWithContextf("conversation not found")
	}

	// Get current bookmark status and toggle it
	var currentBookmarked bool
	err = s.db.QueryRowContext(ctx,
		"SELECT bookmarked FROM ai_chat_messages WHERE id = $1 AND conversation_id = $2",
		messageID, conversationID).Scan(&currentBookmarked)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, contextutils.ErrorWithContextf("message not found")
		}
		return false, contextutils.WrapError(err, "failed to get message bookmark status")
	}

	newBookmarked := !currentBookmarked

	// Update the bookmark status
	query := `UPDATE ai_chat_messages SET bookmarked = $1, updated_at = $2 WHERE id = $3 AND conversation_id = $4`
	_, err = s.db.ExecContext(ctx, query, newBookmarked, time.Now(), messageID, conversationID)
	if err != nil {
		return false, contextutils.WrapError(err, "failed to update message bookmark status")
	}

	return newBookmarked, nil
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

		var answerBytes []byte
		err := rows.Scan(
			&msg.Id,
			&msg.ConversationId,
			&questionIDPtr,
			&msg.Role,
			&answerBytes,
			&msg.Bookmarked,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&conversationTitle,
		)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to scan search result")
		}

		// Content is now stored as an object, unmarshal accordingly
		var contentObj struct {
			Text *string `json:"text,omitempty"`
		}
		err = json.Unmarshal(answerBytes, &contentObj)
		if err != nil {
			return nil, 0, contextutils.WrapError(err, "failed to unmarshal message content")
		}
		msg.Content = contentObj

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

		// For search results, we need to create a minimal content object
		contentObj := struct {
			Text *string `json:"text,omitempty"`
		}{
			Text: &previewMessage,
		}

		// Add preview_message field for frontend compatibility
		conv.Messages = &[]api.ChatMessage{
			{
				Content: contentObj,
			},
		}

		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, contextutils.WrapError(err, "error iterating search results")
	}

	return conversations, total, nil
}
