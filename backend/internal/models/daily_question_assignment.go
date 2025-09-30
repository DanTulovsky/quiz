package models

import (
	"database/sql"
	"time"
)

// DailyQuestionAssignmentWithQuestion represents a daily question assignment with the full question details
type DailyQuestionAssignmentWithQuestion struct {
	ID             int          `json:"id" db:"id"`
	UserID         int          `json:"user_id" db:"user_id"`
	QuestionID     int          `json:"question_id" db:"question_id"`
	AssignmentDate time.Time    `json:"assignment_date" db:"assignment_date"`
	IsCompleted    bool         `json:"is_completed" db:"is_completed"`
	CompletedAt    sql.NullTime `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	// New fields for tracking user's answer
	UserAnswerIndex *int       `json:"user_answer_index" db:"user_answer_index"`
	SubmittedAt     *time.Time `json:"submitted_at" db:"submitted_at"`
	Question        *Question  `json:"question" db:"-"`
	// DailyShownCount represents how many times this question has been assigned in Daily view for this user (all dates)
	DailyShownCount int `json:"-" db:"-"`
	// Per-user aggregated stats from user_responses
	UserCorrectCount   int `json:"-" db:"-"`
	UserIncorrectCount int `json:"-" db:"-"`
	UserTotalResponses int `json:"-" db:"-"`
}

// DailyProgress represents the progress for a specific date
type DailyProgress struct {
	Date      time.Time `json:"date"`
	Completed int       `json:"completed"`
	Total     int       `json:"total"`
}

// DailyAssignmentRequest represents a request to assign daily questions
type DailyAssignmentRequest struct {
	Date time.Time `json:"date" binding:"required"`
}

// DailyCompletionRequest represents a request to mark a question as completed
type DailyCompletionRequest struct {
	QuestionID int `json:"question_id" binding:"required"`
}

// DailyQuestionHistory represents the history of a question being assigned to a user
type DailyQuestionHistory struct {
	AssignmentDate time.Time  `json:"assignment_date" db:"assignment_date"`
	IsCompleted    bool       `json:"is_completed" db:"is_completed"`
	SubmittedAt    *time.Time `json:"submitted_at" db:"submitted_at"`
	// IsCorrect indicates whether the user's answer for this assignment was correct.
	// Nil when the question was not attempted.
	IsCorrect *bool `json:"is_correct" db:"-"`
}
