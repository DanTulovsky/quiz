package models

import (
	"errors"
	"strings"
	"time"
)

// StoryStatus represents the status of a story
type StoryStatus string

// Story status constants
const (
	StoryStatusActive    StoryStatus = "active"    // StoryStatusActive represents an active story
	StoryStatusArchived  StoryStatus = "archived"  // StoryStatusArchived represents an archived story
	StoryStatusCompleted StoryStatus = "completed" // StoryStatusCompleted represents a completed story
)

// SectionLength represents the preferred length of story sections
type SectionLength string

// Section length constants
const (
	SectionLengthShort  SectionLength = "short"  // SectionLengthShort represents a short section length
	SectionLengthMedium SectionLength = "medium" // SectionLengthMedium represents a medium section length
	SectionLengthLong   SectionLength = "long"   // SectionLengthLong represents a long section length
)

// GeneratorType represents who generated a story section
type GeneratorType string

// Generator type constants
const (
	GeneratorTypeWorker GeneratorType = "worker" // GeneratorTypeWorker represents worker-generated sections
	GeneratorTypeUser   GeneratorType = "user"   // GeneratorTypeUser represents user-generated sections
)

// Story represents a user-created story with metadata
type Story struct {
	ID                     uint           `json:"id"`
	UserID                 uint           `json:"user_id"`
	Title                  string         `json:"title"`
	Language               string         `json:"language"`
	Subject                *string        `json:"subject"`
	AuthorStyle            *string        `json:"author_style"`
	TimePeriod             *string        `json:"time_period"`
	Genre                  *string        `json:"genre"`
	Tone                   *string        `json:"tone"`
	CharacterNames         *string        `json:"character_names"`
	CustomInstructions     *string        `json:"custom_instructions"`
	SectionLengthOverride  *SectionLength `json:"section_length_override,omitempty"`
	Status                 StoryStatus    `json:"status"`
	LastSectionGeneratedAt *time.Time     `json:"last_section_generated_at"`
	ExtraGenerationsToday  int            `json:"extra_generations_today"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`

	// Relationships
	User     User           `json:"user,omitempty"`
	Sections []StorySection `json:"sections,omitempty"`
}

// GetSectionLengthOverride returns the section length override as a string, handling nil pointers
func (s *Story) GetSectionLengthOverride() string {
	if s.SectionLengthOverride == nil {
		return ""
	}
	return string(*s.SectionLengthOverride)
}

// StorySection represents an individual section of a story
type StorySection struct {
	ID             uint          `json:"id"`
	StoryID        uint          `json:"story_id"`
	SectionNumber  int           `json:"section_number"`
	Content        string        `json:"content"`
	LanguageLevel  string        `json:"language_level"`
	WordCount      int           `json:"word_count"`
	GeneratedBy    GeneratorType `json:"generated_by"`
	GeneratedAt    time.Time     `json:"generated_at"`
	GenerationDate time.Time     `json:"generation_date"`

	// Relationships
	Story     Story                  `json:"story,omitempty"`
	Questions []StorySectionQuestion `json:"questions,omitempty"`
}

// StorySectionQuestion represents a comprehension question for a story section
type StorySectionQuestion struct {
	ID                 uint      `json:"id"`
	SectionID          uint      `json:"section_id"`
	QuestionText       string    `json:"question_text"`
	Options            []string  `json:"options"`
	CorrectAnswerIndex int       `json:"correct_answer_index"`
	Explanation        *string   `json:"explanation"`
	CreatedAt          time.Time `json:"created_at"`

	// Relationships
	Section StorySection `json:"section,omitempty"`
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
	result := strings.TrimSpace(input)

	// Remove null bytes and control characters
	for i := 0; i < len(result); i++ {
		if result[i] < 32 && result[i] != 9 && result[i] != 10 && result[i] != 13 {
			result = result[:i] + result[i+1:]
			i--
		}
	}

	return result
}

// UserAIConfig holds per-user AI configuration
type UserAIConfig struct {
	Provider string
	Model    string
	APIKey   string
	Username string // For logging purposes
}

// StoryGenerationEligibilityResponse represents the result of checking if a story section can be generated
type StoryGenerationEligibilityResponse struct {
	CanGenerate bool   `json:"can_generate"`
	Reason      string `json:"reason,omitempty"`
	Story       *Story `json:"story,omitempty"` // Include story data when needed for additional checks
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
