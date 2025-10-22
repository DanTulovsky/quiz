package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

//go:embed data/verb-conjugations/*.json
var verbConjugationFS embed.FS

// VerbConjugationHandler handles verb conjugation related HTTP requests
type VerbConjugationHandler struct {
	logger *observability.Logger
}

// NewVerbConjugationHandler creates a new VerbConjugationHandler instance
func NewVerbConjugationHandler(logger *observability.Logger) *VerbConjugationHandler {
	return &VerbConjugationHandler{
		logger: logger,
	}
}

// VerbConjugationData represents the complete verb conjugation data for a language
type VerbConjugationData struct {
	Language     string            `json:"language"`
	LanguageName string            `json:"languageName"`
	Verbs        []VerbConjugation `json:"verbs"`
}

// VerbConjugation represents a single verb with its conjugations across all tenses
type VerbConjugation struct {
	Infinitive   string  `json:"infinitive"`
	InfinitiveEn string  `json:"infinitiveEn"`
	Category     string  `json:"category"`
	Tenses       []Tense `json:"tenses"`
}

// Tense represents a grammatical tense with its conjugations and description
type Tense struct {
	TenseID      string        `json:"tenseId"`
	TenseName    string        `json:"tenseName"`
	TenseNameEn  string        `json:"tenseNameEn"`
	Description  string        `json:"description"`
	Conjugations []Conjugation `json:"conjugations"`
}

// Conjugation represents a single conjugated form with example sentence
type Conjugation struct {
	Pronoun           string `json:"pronoun"`
	Form              string `json:"form"`
	ExampleSentence   string `json:"exampleSentence"`
	ExampleSentenceEn string `json:"exampleSentenceEn"`
}

// VerbConjugationInfo represents metadata about the verb conjugation section
type VerbConjugationInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Emoji       string `json:"emoji"`
	Description string `json:"description"`
}

// GetVerbConjugationInfo returns metadata about verb conjugations
func (h *VerbConjugationHandler) GetVerbConjugationInfo(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_verb_conjugation_info")
	defer observability.FinishSpan(span, nil)

	data, err := verbConjugationFS.ReadFile("data/verb-conjugations/info.json")
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to read verb conjugation info", err)
		HandleAppError(c, contextutils.WrapError(err, "failed to read verb conjugation info"))
		return
	}

	var info VerbConjugationInfo
	if err := json.Unmarshal(data, &info); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to parse verb conjugation info", err)
		HandleAppError(c, contextutils.WrapError(err, "failed to parse verb conjugation info"))
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetVerbConjugations returns all verbs for a specific language
func (h *VerbConjugationHandler) GetVerbConjugations(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_verb_conjugations")
	defer observability.FinishSpan(span, nil)

	languageCode := c.Param("language")
	if languageCode == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	span.SetAttributes(attribute.String("language", languageCode))

	// Read the language-specific JSON file
	filename := fmt.Sprintf("data/verb-conjugations/verb-conjugations-%s.json", languageCode)
	data, err := verbConjugationFS.ReadFile(filename)
	if err != nil {
		// Check if it's a file not found error
		if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		h.logger.Error(c.Request.Context(), "Failed to read verb conjugations", err, map[string]interface{}{
			"language": languageCode,
			"filename": filename,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to read verb conjugations"))
		return
	}

	var verbData VerbConjugationData
	if err := json.Unmarshal(data, &verbData); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to parse verb conjugations", err, map[string]interface{}{
			"language": languageCode,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to parse verb conjugations"))
		return
	}

	c.JSON(http.StatusOK, verbData)
}

// GetVerbConjugation returns a specific verb's conjugations for a language
func (h *VerbConjugationHandler) GetVerbConjugation(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_verb_conjugation")
	defer observability.FinishSpan(span, nil)

	languageCode := c.Param("language")
	verbInfinitive := c.Param("verb")

	if languageCode == "" || verbInfinitive == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	span.SetAttributes(attribute.String("language", languageCode))
	span.SetAttributes(attribute.String("verb", verbInfinitive))

	// Read the language-specific JSON file
	filename := fmt.Sprintf("data/verb-conjugations/verb-conjugations-%s.json", languageCode)
	data, err := verbConjugationFS.ReadFile(filename)
	if err != nil {
		// Check if it's a file not found error
		if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		h.logger.Error(c.Request.Context(), "Failed to read verb conjugations", err, map[string]interface{}{
			"language": languageCode,
			"filename": filename,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to read verb conjugations"))
		return
	}

	var verbData VerbConjugationData
	if err := json.Unmarshal(data, &verbData); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to parse verb conjugations", err, map[string]interface{}{
			"language": languageCode,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to parse verb conjugations"))
		return
	}

	// Find the specific verb
	for _, verb := range verbData.Verbs {
		if strings.EqualFold(verb.Infinitive, verbInfinitive) {
			c.JSON(http.StatusOK, verb)
			return
		}
	}

	HandleAppError(c, contextutils.ErrRecordNotFound)
}

// GetAvailableLanguages returns the list of available languages for verb conjugations
func (h *VerbConjugationHandler) GetAvailableLanguages(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_available_languages")
	defer observability.FinishSpan(span, nil)

	// Read all files in the verb-conjugations directory
	entries, err := verbConjugationFS.ReadDir("data/verb-conjugations")
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to read verb conjugation directory", err)
		HandleAppError(c, contextutils.WrapError(err, "failed to read verb conjugation directory"))
		return
	}

	var languages []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "verb-conjugations-") && strings.HasSuffix(entry.Name(), ".json") {
			// Extract language code from filename like "verb-conjugations-it.json"
			languageCode := strings.TrimPrefix(entry.Name(), "verb-conjugations-")
			languageCode = strings.TrimSuffix(languageCode, ".json")
			languages = append(languages, languageCode)
		}
	}

	c.JSON(http.StatusOK, languages)
}
