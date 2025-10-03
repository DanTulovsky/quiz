// Package services provides embedded templates for AI service prompts
package services

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	contextutils "quizapp/internal/utils"
)

//go:embed templates/*.tmpl
var aiTemplatesFS embed.FS

//go:embed templates/examples/*.json
var exampleFilesFS embed.FS

// Template names as constants
const (
	BatchQuestionPromptTemplate   = "batch_question_prompt.tmpl"
	ChatPromptTemplate            = "chat_prompt.tmpl"
	JSONStructureGuidanceTemplate = "json_structure_guidance.tmpl"
	AIFixPromptTemplate           = "ai_fix_prompt.tmpl"
)

// AITemplateData holds data for rendering AI prompt templates
type AITemplateData struct {
	// Common fields
	Language              string
	Level                 string
	QuestionType          string
	Topic                 string
	RecentQuestionHistory []string
	ReportReasons         []string
	Count                 int // For batch generation

	// Variety fields for question generation
	TopicCategory      string
	GrammarFocus       string
	VocabularyDomain   string
	Scenario           string
	StyleModifier      string
	DifficultyModifier string
	TimeContext        string

	// Schema and formatting
	SchemaForPrompt     string // for direct inclusion in prompt for non-grammar providers
	ExampleContent      string // for including example in prompt
	CurrentQuestionJSON string // the actual question JSON to pass into ai-fix prompt
	AdditionalContext   string // optional freeform context provided by admin when requesting AI fix

	// Explanation specific
	Question      string
	UserAnswer    string
	CorrectAnswer string // The text of the correct answer for explanations

	// Chat specific
	Passage             string
	Options             []string
	IsCorrect           *bool
	ConversationHistory []ChatMessage
	UserMessage         string

	// Priority-aware generation fields (NEW)
	UserWeakAreas        []string
	HighPriorityTopics   []string
	GapAnalysis          map[string]int
	FocusOnWeakAreas     bool
	FreshQuestionRatio   float64
	PriorityDistribution map[string]int

	// Story generation fields
	Title              string
	Subject            string
	AuthorStyle        string
	TimePeriod         string
	Genre              string
	Tone               string
	CharacterNames     string
	CustomInstructions string
	TargetWords        int
	TargetSentences    int
	IsFirstSection     bool
	PreviousSections   string
}

// ChatMessage represents a chat message for templates
type ChatMessage struct {
	Role    string
	Content string
}

// AITemplateManager manages AI prompt templates
type AITemplateManager struct {
	templates *template.Template
}

// NewAITemplateManager creates a new template manager
func NewAITemplateManager() (result0 *AITemplateManager, err error) {
	templates, err := template.New("").ParseFS(aiTemplatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}

	return &AITemplateManager{
		templates: templates,
	}, nil
}

// RenderTemplate renders a template with the given data
func (tm *AITemplateManager) RenderTemplate(templateName string, data AITemplateData) (result0 string, err error) {
	var buf strings.Builder
	err = tm.templates.ExecuteTemplate(&buf, templateName, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// LoadExample loads the example JSON for a specific question type
func (tm *AITemplateManager) LoadExample(questionType string) (result0 string, err error) {
	examplePath := fmt.Sprintf("templates/examples/%s_example.json", questionType)
	content, err := exampleFilesFS.ReadFile(examplePath)
	if err != nil {
		return "", contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to load example for %s: %w", questionType, err)
	}
	return string(content), nil
}
