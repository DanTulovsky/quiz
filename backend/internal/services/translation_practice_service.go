package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/neurosnap/sentences"
	sentencesdata "github.com/neurosnap/sentences/data"
	"go.opentelemetry.io/otel/attribute"
)

// TranslationPracticeServiceInterface defines the interface for translation practice operations
type TranslationPracticeServiceInterface interface {
	GenerateSentence(ctx context.Context, userID uint, req *models.GenerateSentenceRequest, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.TranslationPracticeSentence, error)
	GetSentenceFromExistingContent(ctx context.Context, userID uint, language, level string, direction models.TranslationDirection) (*models.TranslationPracticeSentence, error)
	SubmitTranslation(ctx context.Context, userID uint, req *models.SubmitTranslationRequest, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.TranslationPracticeSession, error)
	GetPracticeHistory(ctx context.Context, userID uint, limit, offset int, search string) ([]models.TranslationPracticeSession, int, error)
	GetPracticeStats(ctx context.Context, userID uint) (map[string]interface{}, error)
	DeleteAllPracticeHistoryForUser(ctx context.Context, userID uint) error
}

// TranslationPracticeService handles translation practice operations
type TranslationPracticeService struct {
	db              *sql.DB
	storyService    StoryServiceInterface
	questionService QuestionServiceInterface
	config          *config.Config
	logger          *observability.Logger
	templateManager *AITemplateManager
	punktModels     map[string]*sentences.DefaultSentenceTokenizer
	punktModelsMu   sync.RWMutex
	punktModelDir   string
}

// NewTranslationPracticeService creates a new TranslationPracticeService instance
func NewTranslationPracticeService(
	db *sql.DB,
	storyService StoryServiceInterface,
	questionService QuestionServiceInterface,
	config *config.Config,
	logger *observability.Logger,
) *TranslationPracticeService {
	// Create template manager
	templateManager, err := NewAITemplateManager()
	if err != nil {
		logger.Error(context.Background(), "Failed to create template manager", err, map[string]interface{}{})
		panic(err) // Use panic for fatal errors in initialization
	}

	// Determine punkt model directory (relative to repo root)
	pwd, err := os.Getwd()
	if err != nil {
		logger.Error(context.Background(), "Failed to get working directory for Punkt models", err, map[string]interface{}{})
	}
	// Find repo root by looking for go.mod
	repoRoot := pwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			// Reached filesystem root
			repoRoot = pwd
			break
		}
		repoRoot = parent
	}
	punktModelDir := filepath.Join(repoRoot, "backend", "internal", "resources", "punkt")

	return &TranslationPracticeService{
		db:              db,
		storyService:    storyService,
		questionService: questionService,
		config:          config,
		logger:          logger,
		templateManager: templateManager,
		punktModels:     make(map[string]*sentences.DefaultSentenceTokenizer),
		punktModelDir:   punktModelDir,
	}
}

// GenerateSentence generates a new sentence using AI
func (s *TranslationPracticeService) GenerateSentence(
	ctx context.Context,
	userID uint,
	req *models.GenerateSentenceRequest,
	aiService AIServiceInterface,
	userAIConfig *models.UserAIConfig,
) (result0 *models.TranslationPracticeSentence, err error) {
	ctx, span := observability.TraceAIFunction(ctx, "generate_translation_sentence",
		attribute.Int("user_id", int(userID)),
		attribute.String("language", req.Language),
		attribute.String("level", req.Level),
		attribute.String("direction", string(req.Direction)),
	)
	defer observability.FinishSpan(span, &err)

	// Determine source and target languages based on direction
	var sourceLang, targetLang string
	if req.Direction == models.TranslationDirectionEnToLearning {
		sourceLang = "en"
		targetLang = req.Language
	} else {
		sourceLang = req.Language
		targetLang = "en"
	}

	// Build prompt for sentence generation
	templateData := AITemplateData{
		Language:  req.Language,
		Level:     req.Level,
		Topic:     stringPtrToString(req.Topic),
		Direction: string(req.Direction),
	}

	// Get template manager from AI service (we'll need to expose this or create our own)
	// For now, we'll use the AI service's method to generate the sentence
	prompt := s.buildTranslationSentencePrompt(templateData)

	// Call AI service
	response, err := aiService.CallWithPrompt(ctx, userAIConfig, prompt, "")
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to generate sentence")
	}

	// Clean the response - remove any markdown, quotes, or extra whitespace
	sentenceText := s.cleanSentenceResponse(response)

	// Save the sentence to database
	sentence := &models.TranslationPracticeSentence{
		UserID:         userID,
		SentenceText:   sentenceText,
		SourceLanguage: sourceLang,
		TargetLanguage: targetLang,
		LanguageLevel:  req.Level,
		SourceType:     models.SentenceSourceTypeAIGenerated,
		Topic:          req.Topic,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.saveSentence(ctx, sentence); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to save sentence")
	}

	return sentence, nil
}

// GetSentenceFromExistingContent retrieves a sentence from existing content
func (s *TranslationPracticeService) GetSentenceFromExistingContent(
	ctx context.Context,
	userID uint,
	language, level string,
	direction models.TranslationDirection,
) (result0 *models.TranslationPracticeSentence, err error) {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "get_sentence_from_existing_content",
		attribute.Int("user_id", int(userID)),
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("direction", string(direction)),
	)
	defer observability.FinishSpan(span, &err)

	// Determine source and target languages
	var sourceLang, targetLang string
	if direction == models.TranslationDirectionEnToLearning {
		sourceLang = "en"
		targetLang = language
	} else {
		sourceLang = language
		targetLang = "en"
	}

	// Try different sources in order of preference
	sources := []struct {
		sourceType models.SentenceSourceType
		fetcher    func() (*models.TranslationPracticeSentence, error)
	}{
		{
			sourceType: models.SentenceSourceTypeStorySection,
			fetcher: func() (*models.TranslationPracticeSentence, error) {
				return s.getSentenceFromStory(ctx, userID, language, level, sourceLang, targetLang)
			},
		},
		{
			sourceType: models.SentenceSourceTypeVocabularyQuestion,
			fetcher: func() (*models.TranslationPracticeSentence, error) {
				return s.getSentenceFromVocabulary(ctx, userID, language, level, sourceLang, targetLang)
			},
		},
		{
			sourceType: models.SentenceSourceTypeReadingComprehension,
			fetcher: func() (*models.TranslationPracticeSentence, error) {
				return s.getSentenceFromReadingComprehension(ctx, userID, language, level, sourceLang, targetLang)
			},
		},
	}

	// Try each source until we find one
	for _, source := range sources {
		sentence, err := source.fetcher()
		if err == nil && sentence != nil {
			// Save to database if not already saved
			if sentence.ID == 0 {
				if err := s.saveSentence(ctx, sentence); err != nil {
					s.logger.Warn(ctx, "Failed to save sentence from existing content", map[string]interface{}{
						"error":       err.Error(),
						"source_type": string(source.sourceType),
					})
					// Continue to next source
					continue
				}
			}
			return sentence, nil
		}
		s.logger.Debug(ctx, "Failed to get sentence from source", map[string]interface{}{
			"source_type": string(source.sourceType),
			"error":       err.Error(),
		})
	}

	return nil, contextutils.NewAppError(
		contextutils.ErrorCodeRecordNotFound,
		contextutils.SeverityWarn,
		"No suitable sentences found in existing content",
		"",
	)
}

// SubmitTranslation submits a translation for AI evaluation
func (s *TranslationPracticeService) SubmitTranslation(
	ctx context.Context,
	userID uint,
	req *models.SubmitTranslationRequest,
	aiService AIServiceInterface,
	userAIConfig *models.UserAIConfig,
) (result0 *models.TranslationPracticeSession, err error) {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "submit_translation",
		attribute.Int("user_id", int(userID)),
		attribute.Int("sentence_id", int(req.SentenceID)),
	)
	defer observability.FinishSpan(span, &err)

	// Get the sentence
	sentence, err := s.getSentenceByID(ctx, req.SentenceID, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get sentence")
	}

	// Get user's language and level for context
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get user")
	}

	// Build evaluation prompt
	lang := ""
	if user.PreferredLanguage.Valid {
		lang = user.PreferredLanguage.String
	}
	lvl := ""
	if user.CurrentLevel.Valid {
		lvl = user.CurrentLevel.String
	}
	templateData := AITemplateData{
		Language:             lang,
		Level:                lvl,
		OriginalSentence:     req.OriginalSentence,
		UserTranslation:      req.UserTranslation,
		SourceLanguage:       sentence.SourceLanguage,
		TargetLanguage:       sentence.TargetLanguage,
		TranslationDirection: string(req.TranslationDirection),
	}

	prompt := s.buildTranslationEvaluationPrompt(templateData)

	// Call AI service for evaluation
	feedback, err := aiService.CallWithPrompt(ctx, userAIConfig, prompt, "")
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to evaluate translation")
	}

	// Extract score from feedback
	score := s.extractScoreFromFeedback(feedback)
	cleanFeedback := s.cleanFeedbackResponse(feedback)

	// Create session
	session := &models.TranslationPracticeSession{
		UserID:               userID,
		SentenceID:           req.SentenceID,
		OriginalSentence:     req.OriginalSentence,
		UserTranslation:      req.UserTranslation,
		TranslationDirection: req.TranslationDirection,
		AIFeedback:           cleanFeedback,
		AIScore:              score,
		CreatedAt:            time.Now(),
	}

	if err := s.saveSession(ctx, session); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to save session")
	}

	return session, nil
}

// GetPracticeHistory retrieves practice history for a user with pagination
func (s *TranslationPracticeService) GetPracticeHistory(
	ctx context.Context,
	userID uint,
	limit int,
	offset int,
	search string,
) (result0 []models.TranslationPracticeSession, total int, err error) {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "get_practice_history",
		attribute.Int("user_id", int(userID)),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
		attribute.String("search", search),
	)
	defer observability.FinishSpan(span, &err)

	// Build base WHERE clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		whereClause += fmt.Sprintf(`
			AND (
				LOWER(original_sentence) LIKE $%d OR
				LOWER(user_translation) LIKE $%d OR
				LOWER(ai_feedback) LIKE $%d OR
				LOWER(translation_direction) LIKE $%d
			)`, argIndex, argIndex, argIndex, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM translation_practice_sessions
		` + whereClause
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(err, "failed to count practice history")
	}

	// Get paginated results
	query := `
		SELECT id, user_id, sentence_id, original_sentence, user_translation,
		       translation_direction, ai_feedback, ai_score, created_at
		FROM translation_practice_sessions
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + `
		OFFSET $` + fmt.Sprintf("%d", argIndex+1) + `
	`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(err, "failed to query practice history")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var sessions []models.TranslationPracticeSession
	for rows.Next() {
		var session models.TranslationPracticeSession
		var aiScore sql.NullFloat64

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.SentenceID,
			&session.OriginalSentence,
			&session.UserTranslation,
			&session.TranslationDirection,
			&session.AIFeedback,
			&aiScore,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, 0, contextutils.WrapErrorf(err, "failed to scan session")
		}

		if aiScore.Valid {
			score := aiScore.Float64
			session.AIScore = &score
		}

		sessions = append(sessions, session)
	}

	return sessions, total, nil
}

// GetPracticeStats retrieves practice statistics for a user
func (s *TranslationPracticeService) GetPracticeStats(
	ctx context.Context,
	userID uint,
) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "get_practice_stats",
		attribute.Int("user_id", int(userID)),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT
			COUNT(*) as total_sessions,
			AVG(ai_score) as average_score,
			MIN(ai_score) as min_score,
			MAX(ai_score) as max_score,
			COUNT(CASE WHEN ai_score >= 4.0 THEN 1 END) as excellent_count,
			COUNT(CASE WHEN ai_score >= 3.0 AND ai_score < 4.0 THEN 1 END) as good_count,
			COUNT(CASE WHEN ai_score < 3.0 THEN 1 END) as needs_improvement_count
		FROM translation_practice_sessions
		WHERE user_id = $1 AND ai_score IS NOT NULL
	`

	var stats struct {
		TotalSessions         int
		AverageScore          sql.NullFloat64
		MinScore              sql.NullFloat64
		MaxScore              sql.NullFloat64
		ExcellentCount        int
		GoodCount             int
		NeedsImprovementCount int
	}

	err = s.db.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalSessions,
		&stats.AverageScore,
		&stats.MinScore,
		&stats.MaxScore,
		&stats.ExcellentCount,
		&stats.GoodCount,
		&stats.NeedsImprovementCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]interface{}{
				"total_sessions":          0,
				"average_score":           nil,
				"min_score":               nil,
				"max_score":               nil,
				"excellent_count":         0,
				"good_count":              0,
				"needs_improvement_count": 0,
			}, nil
		}
		return nil, contextutils.WrapErrorf(err, "failed to query stats")
	}

	result := map[string]interface{}{
		"total_sessions":          stats.TotalSessions,
		"excellent_count":         stats.ExcellentCount,
		"good_count":              stats.GoodCount,
		"needs_improvement_count": stats.NeedsImprovementCount,
	}

	if stats.AverageScore.Valid {
		result["average_score"] = stats.AverageScore.Float64
	} else {
		result["average_score"] = nil
	}

	if stats.MinScore.Valid {
		result["min_score"] = stats.MinScore.Float64
	} else {
		result["min_score"] = nil
	}

	if stats.MaxScore.Valid {
		result["max_score"] = stats.MaxScore.Float64
	} else {
		result["max_score"] = nil
	}

	return result, nil
}

// DeleteAllPracticeHistoryForUser deletes all translation practice sessions for a given user
func (s *TranslationPracticeService) DeleteAllPracticeHistoryForUser(ctx context.Context, userID uint) error {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "delete_all_practice_history_for_user",
		attribute.Int("user_id", int(userID)),
	)
	defer observability.FinishSpan(span, nil)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// Delete all translation practice sessions for this user
	query := `DELETE FROM translation_practice_sessions WHERE user_id = $1`
	result, err := tx.ExecContext(ctx, query, userID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to delete translation practice sessions for user %d", userID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Warn(ctx, "Failed to get rows affected count", map[string]interface{}{"error": err.Error()})
	} else {
		s.logger.Info(ctx, "Deleted translation practice sessions for user", map[string]interface{}{
			"user_id":       userID,
			"rows_affected": rowsAffected,
		})
	}

	if err := tx.Commit(); err != nil {
		return contextutils.WrapErrorf(err, "failed to commit delete all translation practice history transaction for user %d", userID)
	}

	return nil
}

// Helper methods

func (s *TranslationPracticeService) buildTranslationSentencePrompt(data AITemplateData) string {
	prompt, err := s.templateManager.RenderTemplate(TranslationSentencePromptTemplate, data)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to render translation sentence template", err, map[string]interface{}{})
		// Fallback to simple prompt
		prompt = fmt.Sprintf("Generate a single sentence in %s at %s level for translation practice.", data.Language, data.Level)
		if data.Topic != "" {
			prompt += fmt.Sprintf(" Topic/Keywords: %s", data.Topic)
		}
		prompt += " Return ONLY the sentence text, nothing else."
	}
	return prompt
}

func (s *TranslationPracticeService) buildTranslationEvaluationPrompt(data AITemplateData) string {
	prompt, err := s.templateManager.RenderTemplate(TranslationEvaluationPromptTemplate, data)
	if err != nil {
		s.logger.Error(context.Background(), "Failed to render translation evaluation template", err, map[string]interface{}{})
		// Fallback to simple prompt
		prompt = fmt.Sprintf(`You are an expert language teacher evaluating a translation.

A user is learning %s at the %s level.

Original sentence (%s): "%s"
User's translation (%s): "%s"
Translation direction: %s

Evaluate the translation and provide detailed, educational feedback. Focus on accuracy, grammar, naturalness, word choice, and cultural context. At the end, provide a score from 0 to 5 in this format: SCORE: [number]

`, data.Language, data.Level, data.SourceLanguage, data.OriginalSentence, data.TargetLanguage, data.UserTranslation, data.TranslationDirection)
	}
	return prompt
}

func (s *TranslationPracticeService) cleanSentenceResponse(response string) string {
	// Remove markdown code blocks
	response = regexp.MustCompile("(?s)```[^`]*```").ReplaceAllString(response, "")
	// Strip leading and trailing quotes/brackets consistently
	response = s.stripQuotes(response)
	// Trim whitespace
	response = strings.TrimSpace(response)
	return response
}

func (s *TranslationPracticeService) extractScoreFromFeedback(feedback string) *float64 {
	// Look for "SCORE: X" pattern
	re := regexp.MustCompile(`(?i)SCORE:\s*([0-9]+\.?[0-9]*)`)
	matches := re.FindStringSubmatch(feedback)
	if len(matches) > 1 {
		if score, err := parseFloat(matches[1]); err == nil {
			// Clamp score between 0 and 5
			if score < 0 {
				score = 0
			}
			if score > 5 {
				score = 5
			}
			return &score
		}
	}
	return nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func (s *TranslationPracticeService) cleanFeedbackResponse(feedback string) string {
	// Remove the score line if present
	feedback = regexp.MustCompile(`(?i)SCORE:\s*[0-9]+\.?[0-9]*\s*`).ReplaceAllString(feedback, "")
	return strings.TrimSpace(feedback)
}

func (s *TranslationPracticeService) getSentenceFromStory(
	ctx context.Context,
	userID uint,
	language, level string,
	sourceLang, targetLang string,
) (*models.TranslationPracticeSentence, error) {
	// Get user's current story
	story, err := s.storyService.GetCurrentStory(ctx, userID)
	if err != nil {
		return nil, err
	}

	if story == nil || len(story.Sections) == 0 {
		return nil, errors.New("no story sections available")
	}

	// Filter sections by language and level
	var suitableSections []models.StorySection
	for _, section := range story.Sections {
		if section.LanguageLevel == level {
			suitableSections = append(suitableSections, section)
		}
	}

	if len(suitableSections) == 0 {
		return nil, errors.New("no suitable story sections found")
	}

	// Pick a random section
	section := suitableSections[rand.Intn(len(suitableSections))]

	// Extract a sentence from the section content
	sentences := s.extractSentences(section.Content, language)
	if len(sentences) == 0 {
		return nil, errors.New("no sentences found in story section")
	}

	// Pick a random sentence
	sentenceText := sentences[rand.Intn(len(sentences))]

	return &models.TranslationPracticeSentence{
		UserID:         userID,
		SentenceText:   sentenceText,
		SourceLanguage: sourceLang,
		TargetLanguage: targetLang,
		LanguageLevel:  level,
		SourceType:     models.SentenceSourceTypeStorySection,
		SourceID:       &section.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (s *TranslationPracticeService) getSentenceFromVocabulary(
	ctx context.Context,
	userID uint,
	language, level string,
	sourceLang, targetLang string,
) (*models.TranslationPracticeSentence, error) {
	// Get vocabulary questions for the user
	questions, err := s.questionService.GetQuestionsByFilter(
		ctx,
		int(userID),
		language,
		level,
		models.Vocabulary,
		10, // Get 10 questions to choose from
	)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, errors.New("no vocabulary questions available")
	}

	// Pick a random question
	question := questions[rand.Intn(len(questions))]

	// Extract sentence from question content
	var sentenceText string
	if content, ok := question.Content["sentence"].(string); ok {
		sentenceText = content
	} else {
		return nil, errors.New("no sentence found in vocabulary question")
	}

	questionID := uint(question.ID)
	return &models.TranslationPracticeSentence{
		UserID:         userID,
		SentenceText:   sentenceText,
		SourceLanguage: sourceLang,
		TargetLanguage: targetLang,
		LanguageLevel:  level,
		SourceType:     models.SentenceSourceTypeVocabularyQuestion,
		SourceID:       &questionID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (s *TranslationPracticeService) getSentenceFromReadingComprehension(
	ctx context.Context,
	userID uint,
	language, level string,
	sourceLang, targetLang string,
) (*models.TranslationPracticeSentence, error) {
	// Get reading comprehension questions
	questions, err := s.questionService.GetQuestionsByFilter(
		ctx,
		int(userID),
		language,
		level,
		models.ReadingComprehension,
		10,
	)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, errors.New("no reading comprehension questions available")
	}

	// Pick a random question
	question := questions[rand.Intn(len(questions))]

	// Extract a sentence from the passage
	var passageText string
	if content, ok := question.Content["passage"].(string); ok {
		passageText = content
	} else {
		return nil, errors.New("no passage found in reading comprehension question")
	}

	sentences := s.extractSentences(passageText, language)
	if len(sentences) == 0 {
		return nil, errors.New("no sentences found in passage")
	}

	// Pick a random sentence
	sentenceText := sentences[rand.Intn(len(sentences))]

	questionID := uint(question.ID)
	return &models.TranslationPracticeSentence{
		UserID:         userID,
		SentenceText:   sentenceText,
		SourceLanguage: sourceLang,
		TargetLanguage: targetLang,
		LanguageLevel:  level,
		SourceType:     models.SentenceSourceTypeReadingComprehension,
		SourceID:       &questionID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// getPunktModelName maps language codes to Punkt model file names
// Handles both language codes (e.g., "it") and full names (e.g., "italian")
func (s *TranslationPracticeService) getPunktModelName(code string) string {
	switch code {
	case "en", "english":
		return "english"
	case "it", "italian":
		return "italian"
	case "fr", "french":
		return "french"
	case "de", "german":
		return "german"
	case "es", "spanish":
		return "spanish"
	case "ru", "russian":
		return "russian"
	case "hi", "hindi":
		return "hindi"
	case "ja", "japanese":
		return "japanese"
	case "zh", "chinese":
		return "chinese"
	default:
		return ""
	}
}

// getPunktModel loads and caches a Punkt model for the given language code.
// Returns nil if no model is available (caller should use regex fallback).
func (s *TranslationPracticeService) getPunktModel(languageCode string) *sentences.DefaultSentenceTokenizer {
	modelName := s.getPunktModelName(languageCode)
	if modelName == "" {
		return nil
	}

	// Check cache first (read lock)
	s.punktModelsMu.RLock()
	if model, exists := s.punktModels[languageCode]; exists {
		s.punktModelsMu.RUnlock()
		return model
	}
	s.punktModelsMu.RUnlock()

	// Try to load from file (write lock)
	s.punktModelsMu.Lock()
	defer s.punktModelsMu.Unlock()

	// Double-check after acquiring write lock
	if model, exists := s.punktModels[languageCode]; exists {
		return model
	}

	// Try built-in English model first
	var trainingData []byte
	if languageCode == "en" {
		// Use built-in embedded English model
		var err error
		trainingData, err = sentencesdata.Asset("english.json")
		if err != nil {
			// Fallback: try loading from file
			modelPath := filepath.Join(s.punktModelDir, "english.json")
			if data, err := os.ReadFile(modelPath); err == nil {
				trainingData = data
			}
		}
	} else {
		// Try loading from JSON file for other languages
		modelPath := filepath.Join(s.punktModelDir, modelName+".json")
		if data, err := os.ReadFile(modelPath); err == nil {
			trainingData = data
		}
	}

	if len(trainingData) == 0 {
		// No model available, don't cache nil (will try again next time)
		return nil
	}

	// Load training data into Storage
	storage, err := sentences.LoadTraining(trainingData)
	if err != nil {
		s.logger.Warn(context.Background(), "Failed to load Punkt model", map[string]interface{}{
			"language": languageCode,
			"error":    err.Error(),
		})
		return nil
	}

	// Create tokenizer with storage
	tokenizer := sentences.NewSentenceTokenizer(storage)

	// Cache it
	s.punktModels[languageCode] = tokenizer
	return tokenizer
}

// stripQuotes removes leading and trailing quote marks and brackets from a sentence
func (s *TranslationPracticeService) stripQuotes(sentence string) string {
	// Common quote and bracket characters (ASCII and Unicode) - both opening and closing
	quoteChars := []string{
		`"`, `'`, `«`, `»`, `"`, `'`, `'`, `"`, `"`, // ASCII and Unicode quotes
		`(`, `)`, `[`, `]`, `{`, `}`, // ASCII brackets
		`（`, `）`, `［`, `］`, `｛`, `｝`, // Full-width brackets
		`„`, `‚`, `‹`, `›`, // Other quote marks
		`"`, `"`, `'`, `'`, // Typographic quotes
		`'`, `'`, // Apostrophes/quotes
		`"`, `"`, // Double quotes
		`«`, `»`, // Guillemets
		`'`, `'`, // Single quotes
	}
	trimmed := strings.TrimSpace(sentence)
	// Keep stripping until no more quotes/brackets at either end
	changed := true
	for changed {
		changed = false
		for _, char := range quoteChars {
			if strings.HasPrefix(trimmed, char) {
				trimmed = strings.TrimPrefix(trimmed, char)
				trimmed = strings.TrimSpace(trimmed)
				changed = true
				break // Restart check after removal
			}
			if strings.HasSuffix(trimmed, char) {
				trimmed = strings.TrimSuffix(trimmed, char)
				trimmed = strings.TrimSpace(trimmed)
				changed = true
				break // Restart check after removal
			}
		}
	}
	return trimmed
}

// extractSentences uses Punkt tokenizer if available, otherwise falls back to regex.
// Preserves terminal punctuation but strips leading quotes/brackets.
func (s *TranslationPracticeService) extractSentences(text, language string) []string {
	// Try Punkt first if we have a model for this language
	if punktModel := s.getPunktModel(language); punktModel != nil {
		tokenized := punktModel.Tokenize(text)
		var sentences []string
		for _, sent := range tokenized {
			sentText := strings.TrimSpace(sent.Text)
			// Strip leading and trailing quotes/brackets that are part of context, not the sentence
			sentText = s.stripQuotes(sentText)
			if sentText != "" {
				sentences = append(sentences, sentText)
			}
		}
		if len(sentences) > 0 {
			return sentences
		}
		// If Punkt returned nothing, fall through to regex
	}

	// Regex fallback: Extract sentences while PRESERVING terminal punctuation.
	// Handles common ASCII and Unicode punctuation and optional trailing quotes/brackets.
	//
	// Examples matched:
	//  - "Hello world." / "Что это?" / "Да!.." / "— Правда?", '«Привет!»', "(Да?)."
	//  - "你好。" / "こんにちは。" (full-width punctuation)
	//
	// Strategy: find all occurrences of:
	//   any run of non-terminators (lazy), followed by one or more terminators, optionally followed by closing quote/bracket.
	//   Terminators: . ! ? … (ellipsis) plus full-width variants 。！？ plus combinations like ".." or "?!"
	// Use raw string with Unicode characters directly for ellipsis and guillemets
	// Note: Build pattern carefully - terminators in negated class needs escaping
	terminatorsCharClass := `\.\!\?…。！？` // characters for terminators (no brackets): ASCII + Unicode ellipsis + full-width
	terminatorsGroup := `[\.\!\?…。！？]+`  // includes ellipsis … (Unicode U+2026) + full-width punctuation
	closers := `["'\)\]»""'）］」』"]+?`     // quotes, brackets, guillemets (» Unicode U+00BB) + full-width brackets
	pattern := fmt.Sprintf(`(?s)([^%s]+?%s(?:%s)?)`, terminatorsCharClass, terminatorsGroup, closers)
	re := regexp.MustCompile(pattern)

	matches := re.FindAllString(text, -1)

	// Fallback: if nothing matched (no terminators), return the whole text trimmed once
	if len(matches) == 0 {
		trimmed := strings.TrimSpace(text)
		// Strip leading and trailing quotes/brackets that are part of context, not the sentence
		trimmed = s.stripQuotes(trimmed)
		if trimmed != "" {
			return []string{trimmed}
		}
		return nil
	}

	var result []string
	for _, m := range matches {
		sent := strings.TrimSpace(m)
		// Strip leading and trailing quotes/brackets that are part of context, not the sentence
		sent = s.stripQuotes(sent)
		// Filter obvious fragments but keep meaningful short sentences (e.g., "Да.")
		if len([]rune(sent)) >= 2 {
			result = append(result, sent)
		}
	}
	return result
}

func (s *TranslationPracticeService) saveSentence(ctx context.Context, sentence *models.TranslationPracticeSentence) error {
	query := `
		INSERT INTO translation_practice_sentences
		(user_id, sentence_text, source_language, target_language, language_level, source_type, source_id, topic, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err := s.db.QueryRowContext(
		ctx,
		query,
		sentence.UserID,
		sentence.SentenceText,
		sentence.SourceLanguage,
		sentence.TargetLanguage,
		sentence.LanguageLevel,
		sentence.SourceType,
		sentence.SourceID,
		sentence.Topic,
		sentence.CreatedAt,
		sentence.UpdatedAt,
	).Scan(&sentence.ID)

	return err
}

func (s *TranslationPracticeService) getSentenceByID(ctx context.Context, sentenceID, userID uint) (*models.TranslationPracticeSentence, error) {
	query := `
		SELECT id, user_id, sentence_text, source_language, target_language,
		       language_level, source_type, source_id, topic, created_at, updated_at
		FROM translation_practice_sentences
		WHERE id = $1 AND user_id = $2
	`

	var sentence models.TranslationPracticeSentence
	var sourceID sql.NullInt64
	var topic sql.NullString

	err := s.db.QueryRowContext(ctx, query, sentenceID, userID).Scan(
		&sentence.ID,
		&sentence.UserID,
		&sentence.SentenceText,
		&sentence.SourceLanguage,
		&sentence.TargetLanguage,
		&sentence.LanguageLevel,
		&sentence.SourceType,
		&sourceID,
		&topic,
		&sentence.CreatedAt,
		&sentence.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contextutils.NewAppError(
				contextutils.ErrorCodeRecordNotFound,
				contextutils.SeverityWarn,
				"Sentence not found",
				"",
			)
		}
		return nil, err
	}

	if sourceID.Valid {
		id := uint(sourceID.Int64)
		sentence.SourceID = &id
	}

	if topic.Valid {
		sentence.Topic = &topic.String
	}

	return &sentence, nil
}

func (s *TranslationPracticeService) saveSession(ctx context.Context, session *models.TranslationPracticeSession) error {
	query := `
		INSERT INTO translation_practice_sessions
		(user_id, sentence_id, original_sentence, user_translation, translation_direction, ai_feedback, ai_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := s.db.QueryRowContext(
		ctx,
		query,
		session.UserID,
		session.SentenceID,
		session.OriginalSentence,
		session.UserTranslation,
		session.TranslationDirection,
		session.AIFeedback,
		session.AIScore,
		session.CreatedAt,
	).Scan(&session.ID)

	return err
}

func (s *TranslationPracticeService) getUserByID(ctx context.Context, userID uint) (*models.User, error) {
	query := `
		SELECT id, username, email, preferred_language, current_level
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PreferredLanguage,
		&user.CurrentLevel,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
