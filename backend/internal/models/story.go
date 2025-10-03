package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// StoryStatus represents the status of a story
type StoryStatus string

const (
	StoryStatusActive    StoryStatus = "active"
	StoryStatusArchived  StoryStatus = "archived"
	StoryStatusCompleted StoryStatus = "completed"
)

// SectionLength represents the preferred length of story sections
type SectionLength string

const (
	SectionLengthShort  SectionLength = "short"
	SectionLengthMedium SectionLength = "medium"
	SectionLengthLong   SectionLength = "long"
)

// Story represents a user-created story with metadata
type Story struct {
	ID                     uint           `json:"id" gorm:"primaryKey"`
	UserID                 uint           `json:"user_id" gorm:"not null;index"`
	Title                  string         `json:"title" gorm:"not null;size:200"`
	Language               string         `json:"language" gorm:"not null;size:10"`
	Subject                *string        `json:"subject" gorm:"type:text"`
	AuthorStyle            *string        `json:"author_style" gorm:"type:text"`
	TimePeriod             *string        `json:"time_period" gorm:"type:text"`
	Genre                  *string        `json:"genre" gorm:"type:text"`
	Tone                   *string        `json:"tone" gorm:"type:text"`
	CharacterNames         *string        `json:"character_names" gorm:"type:text"`
	CustomInstructions     *string        `json:"custom_instructions" gorm:"type:text"`
	SectionLengthOverride  *SectionLength `json:"section_length_override" gorm:"type:varchar(10);check:section_length_override IN ('short', 'medium', 'long')"`
	Status                 StoryStatus    `json:"status" gorm:"not null;default:active;check:status IN ('active', 'archived', 'completed')"`
	IsCurrent              bool           `json:"is_current" gorm:"not null;default:false"`
	LastSectionGeneratedAt *time.Time     `json:"last_section_generated_at"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`

	// Relationships
	User     User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Sections []StorySection `json:"sections,omitempty" gorm:"foreignKey:StoryID"`
}

// StorySection represents an individual section of a story
type StorySection struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	StoryID        uint      `json:"story_id" gorm:"not null;index"`
	SectionNumber  int       `json:"section_number" gorm:"not null"`
	Content        string    `json:"content" gorm:"not null;type:text"`
	LanguageLevel  string    `json:"language_level" gorm:"not null;size:5"`
	WordCount      int       `json:"word_count" gorm:"not null"`
	GeneratedAt    time.Time `json:"generated_at"`
	GenerationDate time.Time `json:"generation_date" gorm:"not null;type:date;default:CURRENT_DATE"`

	// Relationships
	Story     Story                  `json:"story,omitempty" gorm:"foreignKey:StoryID"`
	Questions []StorySectionQuestion `json:"questions,omitempty" gorm:"foreignKey:SectionID"`
}

// StorySectionQuestion represents a comprehension question for a story section
type StorySectionQuestion struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	SectionID          uint      `json:"section_id" gorm:"not null;index"`
	QuestionText       string    `json:"question_text" gorm:"not null;type:text"`
	Options            []string  `json:"options" gorm:"not null;type:jsonb"`
	CorrectAnswerIndex int       `json:"correct_answer_index" gorm:"not null;check:correct_answer_index >= 0 AND correct_answer_index <= 3"`
	Explanation        *string   `json:"explanation" gorm:"type:text"`
	CreatedAt          time.Time `json:"created_at"`

	// Relationships
	Section StorySection `json:"section,omitempty" gorm:"foreignKey:SectionID"`
}

// StoryWithSections represents a story with all its sections loaded
type StoryWithSections struct {
	Story
	Sections []StorySection `json:"sections"`
}

// StorySectionWithQuestions represents a section with all its questions loaded
type StorySectionWithQuestions struct {
	StorySection
	Questions []StorySectionQuestion `json:"questions"`
}

// CreateStoryRequest represents the request to create a new story
type CreateStoryRequest struct {
	Title                 string         `json:"title" validate:"required,min=1,max=200"`
	Subject               *string        `json:"subject" validate:"omitempty,max=500"`
	AuthorStyle           *string        `json:"author_style" validate:"omitempty,max=200"`
	TimePeriod            *string        `json:"time_period" validate:"omitempty,max=200"`
	Genre                 *string        `json:"genre" validate:"omitempty,max=100"`
	Tone                  *string        `json:"tone" validate:"omitempty,max=100"`
	CharacterNames        *string        `json:"character_names" validate:"omitempty,max=1000"`
	CustomInstructions    *string        `json:"custom_instructions" validate:"omitempty,max=2000"`
	SectionLengthOverride *SectionLength `json:"section_length_override" validate:"omitempty,oneof=short medium long"`
}

// StoryGenerationRequest represents the request for AI story generation
type StoryGenerationRequest struct {
	UserID             uint          `json:"-"`
	StoryID            uint          `json:"-"`
	Language           string        `json:"language"`
	Level              string        `json:"level"`
	Title              string        `json:"title"`
	Subject            *string       `json:"subject,omitempty"`
	AuthorStyle        *string       `json:"author_style,omitempty"`
	TimePeriod         *string       `json:"time_period,omitempty"`
	Genre              *string       `json:"genre,omitempty"`
	Tone               *string       `json:"tone,omitempty"`
	CharacterNames     *string       `json:"character_names,omitempty"`
	CustomInstructions *string       `json:"custom_instructions,omitempty"`
	SectionLength      SectionLength `json:"section_length"`
	PreviousSections   string        `json:"previous_sections"`
	IsFirstSection     bool          `json:"is_first_section"`
	TargetWords        int           `json:"target_words"`
	TargetSentences    int           `json:"target_sentences"`
}

// StoryQuestionsRequest represents the request for AI question generation
type StoryQuestionsRequest struct {
	UserID        uint   `json:"-"`
	SectionID     uint   `json:"-"`
	Language      string `json:"language"`
	Level         string `json:"level"`
	SectionText   string `json:"section_text"`
	QuestionCount int    `json:"question_count"`
}

// StorySectionQuestionData represents the structure returned by AI for questions
type StorySectionQuestionData struct {
	QuestionText       string   `json:"question_text"`
	Options            []string `json:"options"`
	CorrectAnswerIndex int      `json:"correct_answer_index"`
	Explanation        *string  `json:"explanation"`
}

// Validate validates the CreateStoryRequest
func (r *CreateStoryRequest) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}
	if len(r.Title) > 200 {
		return errors.New("title must be 200 characters or less")
	}
	if r.Subject != nil && len(*r.Subject) > 500 {
		return errors.New("subject must be 500 characters or less")
	}
	if r.AuthorStyle != nil && len(*r.AuthorStyle) > 200 {
		return errors.New("author style must be 200 characters or less")
	}
	if r.TimePeriod != nil && len(*r.TimePeriod) > 200 {
		return errors.New("time period must be 200 characters or less")
	}
	if r.Genre != nil && len(*r.Genre) > 100 {
		return errors.New("genre must be 100 characters or less")
	}
	if r.Tone != nil && len(*r.Tone) > 100 {
		return errors.New("tone must be 100 characters or less")
	}
	if r.CharacterNames != nil && len(*r.CharacterNames) > 1000 {
		return errors.New("character names must be 1000 characters or less")
	}
	if r.CustomInstructions != nil && len(*r.CustomInstructions) > 2000 {
		return errors.New("custom instructions must be 2000 characters or less")
	}
	if r.SectionLengthOverride != nil {
		switch *r.SectionLengthOverride {
		case SectionLengthShort, SectionLengthMedium, SectionLengthLong:
			// Valid
		default:
			return errors.New("section length override must be one of: short, medium, long")
		}
	}
	return nil
}

// SanitizeInput sanitizes user input for safe use in AI prompts
func SanitizeInput(input string) string {
	// Basic sanitization - remove control characters and trim whitespace
	// In a production system, you might want more sophisticated sanitization
	result := input

	// Remove null bytes and control characters
	for i := 0; i < len(result); i++ {
		if result[i] < 32 && result[i] != 9 && result[i] != 10 && result[i] != 13 {
			result = result[:i] + result[i+1:]
			i--
		}
	}

	return result
}

// BeforeCreate hook to ensure only one current story per user
func (s *Story) BeforeCreate(tx *gorm.DB) error {
	if s.IsCurrent {
		// Unset any existing current story for this user
		return tx.Model(&Story{}).Where("user_id = ? AND is_current = ?", s.UserID, true).Update("is_current", false).Error
	}
	return nil
}

// BeforeUpdate hook to handle current story logic
func (s *Story) BeforeUpdate(tx *gorm.DB) error {
	if s.IsCurrent {
		// Unset any existing current story for this user (except this one)
		return tx.Model(&Story{}).Where("user_id = ? AND is_current = ? AND id != ?", s.UserID, true, s.ID).Update("is_current", false).Error
	}
	return nil
}

// GetSectionLengthTarget returns the target word count for a story section
func GetSectionLengthTarget(level string, lengthPref *SectionLength) int {
	// Map CEFR levels to generic proficiency levels for backward compatibility
	levelMapping := map[string]string{
		"A1": "beginner",
		"A2": "elementary",
		"B1": "intermediate",
		"B2": "upper_intermediate",
		"C1": "advanced",
		"C2": "proficient",
	}

	genericLevel := levelMapping[level]
	if genericLevel == "" {
		// If no mapping found, default to intermediate
		genericLevel = "intermediate"
	}

	// Default length targets by proficiency level (in words)
	lengthTargets := map[string]map[SectionLength]int{
		"beginner":           {SectionLengthShort: 50, SectionLengthMedium: 80, SectionLengthLong: 120},
		"elementary":         {SectionLengthShort: 80, SectionLengthMedium: 120, SectionLengthLong: 180},
		"intermediate":       {SectionLengthShort: 150, SectionLengthMedium: 220, SectionLengthLong: 300},
		"upper_intermediate": {SectionLengthShort: 250, SectionLengthMedium: 350, SectionLengthLong: 450},
		"advanced":           {SectionLengthShort: 350, SectionLengthMedium: 500, SectionLengthLong: 650},
		"proficient":         {SectionLengthShort: 500, SectionLengthMedium: 700, SectionLengthLong: 900},
	}

	levelTargets, exists := lengthTargets[genericLevel]
	if !exists {
		// Default to intermediate if level not found
		levelTargets = lengthTargets["intermediate"]
	}

	if lengthPref != nil {
		if target, exists := levelTargets[*lengthPref]; exists {
			return target
		}
	}

	// Default to medium length
	return levelTargets[SectionLengthMedium]
}

// GetSectionLengthTargetWithLanguage returns the target word count with language-specific overrides
func GetSectionLengthTargetWithLanguage(language string, level string, lengthPref *SectionLength) int {
	// TODO: This would need access to the config to check for language-specific overrides
	// For now, fall back to the basic implementation
	return GetSectionLengthTarget(level, lengthPref)
}

// ParseOptions parses the JSON options field into a string slice
func (q *StorySectionQuestion) ParseOptions() ([]string, error) {
	if q.Options == nil {
		return nil, errors.New("options field is nil")
	}

	options := make([]string, len(q.Options))
	for i, option := range q.Options {
		options[i] = option
	}
	return options, nil
}

// TableName returns the table name for GORM
func (Story) TableName() string {
	return "stories"
}

// TableName returns the table name for GORM
func (StorySection) TableName() string {
	return "story_sections"
}

// TableName returns the table name for GORM
func (StorySectionQuestion) TableName() string {
	return "story_section_questions"
}
