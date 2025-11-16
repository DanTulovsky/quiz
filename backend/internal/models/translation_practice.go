package models

import "time"

// TranslationDirection represents the direction of translation
type TranslationDirection string

// Translation direction constants
const (
	TranslationDirectionEnToLearning TranslationDirection = "en_to_learning" // Translate from English to learning language
	TranslationDirectionLearningToEn TranslationDirection = "learning_to_en" // Translate from learning language to English
)

// SentenceSourceType represents where a sentence came from
type SentenceSourceType string

// Sentence source type constants
const (
	SentenceSourceTypeAIGenerated          SentenceSourceType = "ai_generated"
	SentenceSourceTypeStorySection         SentenceSourceType = "story_section"
	SentenceSourceTypeVocabularyQuestion   SentenceSourceType = "vocabulary_question"
	SentenceSourceTypeReadingComprehension SentenceSourceType = "reading_comprehension"
	SentenceSourceTypeSnippet              SentenceSourceType = "snippet"
	SentenceSourceTypePhrasebook           SentenceSourceType = "phrasebook"
)

// TranslationPracticeSentence represents a sentence available for translation practice
type TranslationPracticeSentence struct {
	ID             uint               `json:"id"`
	UserID         uint               `json:"user_id"`
	SentenceText   string             `json:"sentence_text"`
	SourceLanguage string             `json:"source_language"`
	TargetLanguage string             `json:"target_language"`
	LanguageLevel  string             `json:"language_level"`
	SourceType     SentenceSourceType `json:"source_type"`
	SourceID       *uint              `json:"source_id,omitempty"`
	Topic          *string            `json:"topic,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// TranslationPracticeSession represents a single translation practice attempt
type TranslationPracticeSession struct {
	ID                   uint                 `json:"id"`
	UserID               uint                 `json:"user_id"`
	SentenceID           uint                 `json:"sentence_id"`
	OriginalSentence     string               `json:"original_sentence"`
	UserTranslation      string               `json:"user_translation"`
	TranslationDirection TranslationDirection `json:"translation_direction"`
	AIFeedback           string               `json:"ai_feedback"`
	AIScore              *float64             `json:"ai_score,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`

	// Relationships
	Sentence *TranslationPracticeSentence `json:"sentence,omitempty"`
}

// GenerateSentenceRequest represents a request to generate a new sentence
type GenerateSentenceRequest struct {
	Language  string               `json:"language"`
	Level     string               `json:"level"`
	Direction TranslationDirection `json:"direction"`
	Topic     *string              `json:"topic,omitempty"` // Optional topic or keywords
}

// SubmitTranslationRequest represents a request to submit a translation for evaluation
type SubmitTranslationRequest struct {
	SentenceID           uint                 `json:"sentence_id"`
	OriginalSentence     string               `json:"original_sentence"`
	UserTranslation      string               `json:"user_translation"`
	TranslationDirection TranslationDirection `json:"translation_direction"`
}

// TranslationEvaluationResponse represents the AI's evaluation of a translation
type TranslationEvaluationResponse struct {
	Feedback string   `json:"feedback"`
	Score    *float64 `json:"score,omitempty"` // Score from 0 to 5
}
