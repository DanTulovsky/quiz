// Package models defines data structures used throughout the quiz application.
package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"quizapp/internal/api"
)

// User represents a user in the system
type User struct {
	ID                int            `json:"id" yaml:"id"`
	Username          string         `json:"username" yaml:"username"`
	Email             sql.NullString `json:"email" yaml:"email"`
	Timezone          sql.NullString `json:"timezone" yaml:"timezone"`
	PasswordHash      sql.NullString `json:"-" yaml:"-"` // Omit from JSON responses
	LastActive        sql.NullTime   `json:"last_active" yaml:"last_active"`
	PreferredLanguage sql.NullString `json:"preferred_language" yaml:"preferred_language"`
	CurrentLevel      sql.NullString `json:"current_level" yaml:"current_level"`
	AIProvider        sql.NullString `json:"ai_provider" yaml:"ai_provider"`
	AIModel           sql.NullString `json:"ai_model" yaml:"ai_model"`
	AIEnabled         sql.NullBool   `json:"ai_enabled" yaml:"ai_enabled"`
	AIAPIKey          sql.NullString `json:"-" yaml:"ai_api_key"` // Omit from JSON responses
	CreatedAt         time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" yaml:"updated_at"`
	Roles             []Role         `json:"roles,omitempty" yaml:"roles,omitempty"`
}

// Role represents a role in the system
type Role struct {
	ID          int       `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// UserRole represents the mapping between users and roles
type UserRole struct {
	ID        int       `json:"id" yaml:"id"`
	UserID    int       `json:"user_id" yaml:"user_id"`
	RoleID    int       `json:"role_id" yaml:"role_id"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// MarshalJSON customizes JSON marshaling for User to handle sql.NullString and sql.NullTime properly
func (u User) MarshalJSON() (result0 []byte, err error) { // Create a struct with the desired JSON structure
	return json.Marshal(&struct {
		ID                int        `json:"id"`
		Username          string     `json:"username"`
		Email             *string    `json:"email"`
		Timezone          *string    `json:"timezone"`
		LastActive        *time.Time `json:"last_active"`
		PreferredLanguage *string    `json:"preferred_language"`
		CurrentLevel      *string    `json:"current_level"`
		AIProvider        *string    `json:"ai_provider"`
		AIModel           *string    `json:"ai_model"`
		AIEnabled         *bool      `json:"ai_enabled"`
		CreatedAt         time.Time  `json:"created_at"`
		UpdatedAt         time.Time  `json:"updated_at"`
		Roles             []Role     `json:"roles,omitempty"`
	}{
		ID:                u.ID,
		Username:          u.Username,
		Email:             nullStringToPointer(u.Email),
		Timezone:          nullStringToPointer(u.Timezone),
		LastActive:        nullTimeToPointer(u.LastActive),
		PreferredLanguage: nullStringToPointer(u.PreferredLanguage),
		CurrentLevel:      nullStringToPointer(u.CurrentLevel),
		AIProvider:        nullStringToPointer(u.AIProvider),
		AIModel:           nullStringToPointer(u.AIModel),
		AIEnabled:         nullBoolToPointer(u.AIEnabled),
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		Roles:             u.Roles,
	})
}

// Helper functions for converting sql.Null types to pointers
func nullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullTimeToPointer(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func nullBoolToPointer(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

func nullInt32ToPointer(ni sql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}

// UserAPIKey represents an API key for a specific provider for a user
type UserAPIKey struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Provider  string    `json:"provider"`
	APIKey    string    `json:"-"` // Omit from JSON responses for security
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Question represents a quiz question
type Question struct {
	ID              int                    `json:"id" yaml:"id"`
	Type            QuestionType           `json:"type" yaml:"type"`
	Language        string                 `json:"language" yaml:"language"`
	Level           string                 `json:"level" yaml:"level"`
	DifficultyScore float64                `json:"difficulty_score" yaml:"difficulty_score"`
	Content         map[string]interface{} `json:"content" yaml:"content"`
	CorrectAnswer   int                    `json:"correct_answer" yaml:"correct_answer"`
	Explanation     string                 `json:"explanation,omitempty" yaml:"explanation"`
	CreatedAt       time.Time              `json:"created_at" yaml:"created_at"`
	Status          QuestionStatus         `json:"status" yaml:"status"`
	// Test data field for specifying which users should have this question
	Users []string `json:"users,omitempty" yaml:"users,omitempty"`
	// Variety elements for question generation diversity
	TopicCategory      string `json:"topic_category,omitempty" yaml:"topic_category"`
	GrammarFocus       string `json:"grammar_focus,omitempty" yaml:"grammar_focus"`
	VocabularyDomain   string `json:"vocabulary_domain,omitempty" yaml:"vocabulary_domain"`
	Scenario           string `json:"scenario,omitempty" yaml:"scenario"`
	StyleModifier      string `json:"style_modifier,omitempty" yaml:"style_modifier"`
	DifficultyModifier string `json:"difficulty_modifier,omitempty" yaml:"difficulty_modifier"`
	TimeContext        string `json:"time_context,omitempty" yaml:"time_context"`
}

// UserQuestion represents the mapping between users and questions
type UserQuestion struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	QuestionID int       `json:"question_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// QuestionReport represents a report of a question by a user
type QuestionReport struct {
	ID               int       `json:"id"`
	QuestionID       int       `json:"question_id"`
	ReportedByUserID int       `json:"reported_by_user_id"`
	ReportReason     string    `json:"report_reason"`
	CreatedAt        time.Time `json:"created_at"`
}

// QuestionType represents the type of question
type QuestionType string

// QuestionStatus represents the status of a question
type QuestionStatus string

const (
	// QuestionStatusActive is for questions that are in active use
	QuestionStatusActive QuestionStatus = "active"
	// QuestionStatusReported is for questions that have been reported as incorrect
	QuestionStatusReported QuestionStatus = "reported"
)

// Question types supported by the system
const (
	// Vocabulary represents vocabulary in context questions
	Vocabulary QuestionType = "vocabulary"
	// FillInBlank represents fill-in-the-blank questions
	FillInBlank QuestionType = "fill_blank"
	// QuestionAnswer represents simple Q&A questions
	QuestionAnswer QuestionType = "qa"
	// ReadingComprehension represents reading comprehension questions
	ReadingComprehension QuestionType = "reading_comprehension"
)

// UserResponse represents a user's answer to a question
type UserResponse struct {
	ID              int           `json:"id" yaml:"id"`
	UserID          int           `json:"user_id" yaml:"user_id"`
	QuestionID      int           `json:"question_id" yaml:"question_id"`
	UserAnswerIndex int           `json:"user_answer_index" yaml:"user_answer_index"`
	IsCorrect       bool          `json:"is_correct" yaml:"is_correct"`
	ResponseTimeMs  int           `json:"response_time_ms" yaml:"response_time_ms"`
	ConfidenceLevel sql.NullInt32 `json:"confidence_level" yaml:"confidence_level"`
	CreatedAt       time.Time     `json:"created_at" yaml:"created_at"`
}

// MarshalJSON customizes JSON marshaling for UserResponse to handle sql.NullInt32 properly
func (ur UserResponse) MarshalJSON() (result0 []byte, err error) {
	return json.Marshal(&struct {
		ID              int       `json:"id"`
		UserID          int       `json:"user_id"`
		QuestionID      int       `json:"question_id"`
		UserAnswerIndex int       `json:"user_answer_index"`
		IsCorrect       bool      `json:"is_correct"`
		ResponseTimeMs  int       `json:"response_time_ms"`
		ConfidenceLevel *int32    `json:"confidence_level"`
		CreatedAt       time.Time `json:"created_at"`
	}{
		ID:              ur.ID,
		UserID:          ur.UserID,
		QuestionID:      ur.QuestionID,
		UserAnswerIndex: ur.UserAnswerIndex,
		IsCorrect:       ur.IsCorrect,
		ResponseTimeMs:  ur.ResponseTimeMs,
		ConfidenceLevel: nullInt32ToPointer(ur.ConfidenceLevel),
		CreatedAt:       ur.CreatedAt,
	})
}

// PerformanceMetrics tracks user performance across different categories
type PerformanceMetrics struct {
	ID                    int       `json:"id"`
	UserID                int       `json:"user_id"`
	Topic                 string    `json:"topic"`
	Language              string    `json:"language"`
	Level                 string    `json:"level"`
	TotalAttempts         int       `json:"total_attempts"`
	CorrectAttempts       int       `json:"correct_attempts"`
	AverageResponseTimeMs float64   `json:"average_response_time_ms"`
	DifficultyAdjustment  float64   `json:"difficulty_adjustment"`
	LastUpdated           time.Time `json:"last_updated"`
}

// AccuracyRate calculates the accuracy percentage
func (pm *PerformanceMetrics) AccuracyRate() float64 {
	if pm.TotalAttempts == 0 {
		return 0.0
	}
	return float64(pm.CorrectAttempts) / float64(pm.TotalAttempts) * 100
}

// QuestionRequest represents a request for a new question
type QuestionRequest struct {
	UserID       int          `json:"user_id"`
	Language     string       `json:"language"`
	Level        string       `json:"level"`
	QuestionType QuestionType `json:"question_type,omitempty"`
}

// AnswerRequest represents a user's answer submission
type AnswerRequest struct {
	QuestionID     int    `json:"question_id"`
	UserAnswer     string `json:"user_answer"`
	ResponseTimeMs int    `json:"response_time_ms"`
}

// AnswerResponse represents the response to an answer submission
type AnswerResponse struct {
	IsCorrect      bool   `json:"is_correct"`
	CorrectAnswer  string `json:"correct_answer"`
	UserAnswer     string `json:"user_answer"`
	Explanation    string `json:"explanation"`
	NextDifficulty string `json:"next_difficulty,omitempty"`
}

// GetCorrectAnswerText returns the text of the correct answer from the question content
func (q *Question) GetCorrectAnswerText() string {
	if optionsRaw, ok := q.Content["options"]; ok {
		if options, ok := optionsRaw.([]interface{}); ok {
			if q.CorrectAnswer >= 0 && q.CorrectAnswer < len(options) {
				if optStr, ok := options[q.CorrectAnswer].(string); ok {
					return optStr
				}
			}
		}
	}
	return ""
}

// UserSettings represents user preference settings
type UserSettings struct {
	Language   string `json:"language" yaml:"language"`
	Level      string `json:"level" yaml:"level"`
	AIProvider string `json:"ai_provider" yaml:"ai_provider"`
	AIModel    string `json:"ai_model" yaml:"ai_model"`
	AIEnabled  bool   `json:"ai_enabled" yaml:"ai_enabled"`
	AIAPIKey   string `json:"api_key" yaml:"ai_api_key"`
}

// UserLearningPreferences represents user learning preferences and settings
type UserLearningPreferences struct {
	ID                        int      `json:"id" db:"id"`
	UserID                    int      `json:"user_id" db:"user_id"`
	PreferredLanguage         string   `json:"preferred_language" db:"preferred_language"`
	CurrentLevel              string   `json:"current_level" db:"current_level"`
	AIProvider                string   `json:"ai_provider" db:"ai_provider"`
	AIModel                   string   `json:"ai_model" db:"ai_model"`
	AIEnabled                 bool     `json:"ai_enabled" db:"ai_enabled"`
	AIAPIKey                  string   `json:"-" db:"ai_api_key"` // Omit from JSON for security
	DailyGoal                 int      `json:"daily_goal" db:"daily_goal"`
	WeeklyGoal                int      `json:"weekly_goal" db:"weekly_goal"`
	PreferredQuestionType     string   `json:"preferred_question_type" db:"preferred_question_type"`
	PreferredQuestionTypes    []string `json:"preferred_question_types" db:"preferred_question_types"`
	PreferredDifficultyLevel  string   `json:"preferred_difficulty_level" db:"preferred_difficulty_level"`
	PreferredTopics           []string `json:"preferred_topics" db:"preferred_topics"`
	PreferredQuestionCount    int      `json:"preferred_question_count" db:"preferred_question_count"`
	SpacedRepetitionEnabled   bool     `json:"spaced_repetition_enabled" db:"spaced_repetition_enabled"`
	AdaptiveDifficultyEnabled bool     `json:"adaptive_difficulty_enabled" db:"adaptive_difficulty_enabled"`
	FocusOnWeakAreas          bool     `json:"focus_on_weak_areas" db:"focus_on_weak_areas"`
	IncludeReviewQuestions    bool     `json:"include_review_questions" db:"include_review_questions"`
	FreshQuestionRatio        float64  `json:"fresh_question_ratio" db:"fresh_question_ratio"`
	KnownQuestionPenalty      float64  `json:"known_question_penalty" db:"known_question_penalty"`
	ReviewIntervalDays        int      `json:"review_interval_days" db:"review_interval_days"`
	WeakAreaBoost             float64  `json:"weak_area_boost" db:"weak_area_boost"`
	StudyTime                 string   `json:"study_time" db:"study_time"`
	DailyReminderEnabled      bool     `json:"daily_reminder_enabled" db:"daily_reminder_enabled"`
	// Preferred TTS voice (e.g., it-IT-IsabellaNeural)
	TTSVoice              string     `json:"tts_voice" db:"tts_voice"`
	LastDailyReminderSent *time.Time `json:"last_daily_reminder_sent" db:"last_daily_reminder_sent"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// UserProgress represents a user's overall progress
type UserProgress struct {
	CurrentLevel       string                         `json:"current_level"`
	TotalQuestions     int                            `json:"total_questions"`
	CorrectAnswers     int                            `json:"correct_answers"`
	AccuracyRate       float64                        `json:"accuracy_rate"`
	PerformanceByTopic map[string]*PerformanceMetrics `json:"performance_by_topic"`
	WeakAreas          []string                       `json:"weak_areas"`
	RecentActivity     []UserResponse                 `json:"recent_activity"`
	SuggestedLevel     string                         `json:"suggested_level,omitempty"`
}

// AIQuestionGenRequest represents a request to the AI service for question generation
type AIQuestionGenRequest struct {
	Language              string       `json:"language"`
	Level                 string       `json:"level"`
	QuestionType          QuestionType `json:"question_type"`
	Count                 int          `json:"count"`
	RecentQuestionHistory []string     `json:"-"` // Don't include in JSON, internal use
}

// AIChatRequest represents a request to the AI service for a new chat feature
type AIChatRequest struct {
	Language              string
	Level                 string
	QuestionType          QuestionType // Question type for context
	Question              string
	Options               []string
	Passage               string // For reading comprehension
	UserAnswer            string // Optional
	CorrectAnswer         string // Optional
	IsCorrect             *bool  // Optional
	UserMessage           string
	ConversationHistory   []ChatMessage `json:"conversation_history,omitempty"`
	RecentQuestionHistory []string      `json:"-"` // Don't include in JSON, internal use
}

// ChatMessage represents a single message in the chat conversation
type ChatMessage struct {
	Role    api.ChatMessageRole `json:"role"`    // "user" or "assistant"
	Content string              `json:"content"` // The message content
}

// AIExplanationRequest represents a request for an explanation of a wrong answer
type AIExplanationRequest struct {
	Question      string `json:"question"`
	UserAnswer    string `json:"user_answer"`
	CorrectAnswer string `json:"correct_answer"`
	Language      string `json:"language"`
	Level         string `json:"level"`
}

// MarshalContentToJSON serializes the question content to JSON string
func (q *Question) MarshalContentToJSON() (result0 string, err error) {
	// Clean up fields that should be at the top level, not in content
	// Remove fields that are not allowed in QuestionContent according to OpenAPI schema
	if q.Content != nil {
		// Always remove correct_answer from content as it should be at top level
		delete(q.Content, "correct_answer")
		// Always remove explanation from content as it should be at top level
		delete(q.Content, "explanation")
	}

	data, err := json.Marshal(q.Content)
	return string(data), err
}

// UnmarshalContentFromJSON deserializes JSON string into question content
func (q *Question) UnmarshalContentFromJSON(data string) error {
	err := json.Unmarshal([]byte(data), &q.Content)
	if err != nil {
		return err
	}

	// Clean up fields that should be at the top level, not in content
	// Remove fields that are not allowed in QuestionContent according to OpenAPI schema
	if q.Content != nil {
		// Always remove correct_answer from content as it should be at top level
		delete(q.Content, "correct_answer")
		// Always remove explanation from content as it should be at top level
		delete(q.Content, "explanation")
	}

	return nil
}

// WorkerSettings represents worker configuration settings stored in database
type WorkerSettings struct {
	ID           int       `json:"id" db:"id"`
	SettingKey   string    `json:"setting_key" db:"setting_key"`
	SettingValue string    `json:"setting_value" db:"setting_value"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// WorkerStatus represents worker health and activity status
type WorkerStatus struct {
	ID                      int            `json:"id" db:"id"`
	WorkerInstance          string         `json:"worker_instance" db:"worker_instance"`
	IsRunning               bool           `json:"is_running" db:"is_running"`
	IsPaused                bool           `json:"is_paused" db:"is_paused"`
	CurrentActivity         sql.NullString `json:"current_activity" db:"current_activity"`
	LastHeartbeat           sql.NullTime   `json:"last_heartbeat" db:"last_heartbeat"`
	LastRunStart            sql.NullTime   `json:"last_run_start" db:"last_run_start"`
	LastRunEnd              sql.NullTime   `json:"last_run_end" db:"last_run_end"`
	LastRunFinish           sql.NullTime   `json:"last_run_finish" db:"last_run_finish"`
	LastRunError            sql.NullString `json:"last_run_error" db:"last_run_error"`
	TotalQuestionsProcessed int            `json:"total_questions_processed" db:"total_questions_processed"`
	TotalQuestionsGenerated int            `json:"total_questions_generated" db:"total_questions_generated"`
	TotalRuns               int            `json:"total_runs" db:"total_runs"`
	CreatedAt               time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at" db:"updated_at"`
}

// MarshalJSON customizes JSON marshaling for WorkerStatus to handle sql.NullString and sql.NullTime properly
func (ws WorkerStatus) MarshalJSON() (result0 []byte, err error) {
	return json.Marshal(&struct {
		ID                      int        `json:"id"`
		WorkerInstance          string     `json:"worker_instance"`
		IsRunning               bool       `json:"is_running"`
		IsPaused                bool       `json:"is_paused"`
		CurrentActivity         *string    `json:"current_activity"`
		LastHeartbeat           *time.Time `json:"last_heartbeat"`
		LastRunStart            *time.Time `json:"last_run_start"`
		LastRunEnd              *time.Time `json:"last_run_end"`
		LastRunFinish           *time.Time `json:"last_run_finish"`
		LastRunError            *string    `json:"last_run_error"`
		TotalQuestionsProcessed int        `json:"total_questions_processed"`
		TotalQuestionsGenerated int        `json:"total_questions_generated"`
		TotalRuns               int        `json:"total_runs"`
		CreatedAt               time.Time  `json:"created_at"`
		UpdatedAt               time.Time  `json:"updated_at"`
	}{
		ID:                      ws.ID,
		WorkerInstance:          ws.WorkerInstance,
		IsRunning:               ws.IsRunning,
		IsPaused:                ws.IsPaused,
		CurrentActivity:         nullStringToPointer(ws.CurrentActivity),
		LastHeartbeat:           nullTimeToPointer(ws.LastHeartbeat),
		LastRunStart:            nullTimeToPointer(ws.LastRunStart),
		LastRunEnd:              nullTimeToPointer(ws.LastRunEnd),
		LastRunFinish:           nullTimeToPointer(ws.LastRunFinish),
		LastRunError:            nullStringToPointer(ws.LastRunError),
		TotalQuestionsProcessed: ws.TotalQuestionsProcessed,
		TotalQuestionsGenerated: ws.TotalQuestionsGenerated,
		TotalRuns:               ws.TotalRuns,
		CreatedAt:               ws.CreatedAt,
		UpdatedAt:               ws.UpdatedAt,
	})
}
