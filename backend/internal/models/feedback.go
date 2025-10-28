package models

import (
	"database/sql"
	"time"
)

// FeedbackReport represents a generic feedback / issue report submitted by a user.
type FeedbackReport struct {
	ID               int                    `json:"id" db:"id"`
	UserID           int                    `json:"user_id" db:"user_id"`
	FeedbackText     string                 `json:"feedback_text" db:"feedback_text"`
	FeedbackType     string                 `json:"feedback_type" db:"feedback_type"`
	ContextData      map[string]interface{} `json:"context_data" db:"context_data"`
	ScreenshotData   sql.NullString         `json:"screenshot_data" db:"screenshot_data"`
	ScreenshotURL    sql.NullString         `json:"screenshot_url" db:"screenshot_url"`
	Status           string                 `json:"status" db:"status"`
	AdminNotes       sql.NullString         `json:"admin_notes" db:"admin_notes"`
	AssignedToUserID sql.NullInt32          `json:"assigned_to_user_id" db:"assigned_to_user_id"`
	ResolvedAt       sql.NullTime           `json:"resolved_at" db:"resolved_at"`
	ResolvedByUserID sql.NullInt32          `json:"resolved_by_user_id" db:"resolved_by_user_id"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}
