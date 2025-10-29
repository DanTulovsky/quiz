package models

import (
	"encoding/json"
	"time"
)

// WordSourceType represents the type of source for the word of the day
type WordSourceType string

const (
	// WordSourceVocabularyQuestion represents a word from a vocabulary question
	WordSourceVocabularyQuestion WordSourceType = "vocabulary_question"
	// WordSourceSnippet represents a word from a user snippet
	WordSourceSnippet WordSourceType = "snippet"
)

// WordOfTheDay represents a daily word assignment for a user
type WordOfTheDay struct {
	ID             int            `json:"id" db:"id"`
	UserID         int            `json:"user_id" db:"user_id"`
	AssignmentDate time.Time      `json:"assignment_date" db:"assignment_date"`
	SourceType     WordSourceType `json:"source_type" db:"source_type"`
	SourceID       int            `json:"source_id" db:"source_id"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// WordOfTheDayWithContent represents a word of the day with full content details
type WordOfTheDayWithContent struct {
	WordOfTheDay
	// Question is populated when SourceType is WordSourceVocabularyQuestion
	Question *Question `json:"question,omitempty"`
	// Snippet is populated when SourceType is WordSourceSnippet
	Snippet *Snippet `json:"snippet,omitempty"`
}

// WordOfTheDayDisplay represents the simplified display format for word of the day
// This is used for API responses and contains the essential information
type WordOfTheDayDisplay struct {
	Date          time.Time      `json:"date"`
	Word          string         `json:"word"`
	Translation   string         `json:"translation"`
	Sentence      string         `json:"sentence"`
	SourceType    WordSourceType `json:"source_type"`
	SourceID      int            `json:"source_id"`
	Language      string         `json:"language"`
	Level         string         `json:"level,omitempty"`
	Context       string         `json:"context,omitempty"`
	Explanation   string         `json:"explanation,omitempty"`
	TopicCategory string         `json:"topic_category,omitempty"`
}

// MarshalJSON customizes JSON marshaling for WordOfTheDayDisplay to format the date field as YYYY-MM-DD
// This ensures compliance with OpenAPI date format (not date-time)
func (w WordOfTheDayDisplay) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Date          string         `json:"date"`
		Word          string         `json:"word"`
		Translation   string         `json:"translation"`
		Sentence      string         `json:"sentence"`
		SourceType    WordSourceType `json:"source_type"`
		SourceID      int            `json:"source_id"`
		Language      string         `json:"language"`
		Level         string         `json:"level,omitempty"`
		Context       string         `json:"context,omitempty"`
		Explanation   string         `json:"explanation,omitempty"`
		TopicCategory string         `json:"topic_category,omitempty"`
	}{
		Date:          w.Date.UTC().Format("2006-01-02"),
		Word:          w.Word,
		Translation:   w.Translation,
		Sentence:      w.Sentence,
		SourceType:    w.SourceType,
		SourceID:      w.SourceID,
		Language:      w.Language,
		Level:         w.Level,
		Context:       w.Context,
		Explanation:   w.Explanation,
		TopicCategory: w.TopicCategory,
	})
}
