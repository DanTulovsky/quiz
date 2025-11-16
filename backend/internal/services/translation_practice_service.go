package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
)

// TranslationPracticeServiceInterface defines the interface for translation practice operations
type TranslationPracticeServiceInterface interface {
	GenerateSentence(ctx context.Context, userID uint, req *models.GenerateSentenceRequest, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.TranslationPracticeSentence, error)
	GetSentenceFromExistingContent(ctx context.Context, userID uint, language, level string, direction models.TranslationDirection) (*models.TranslationPracticeSentence, error)
	SubmitTranslation(ctx context.Context, userID uint, req *models.SubmitTranslationRequest, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.TranslationPracticeSession, error)
	GetPracticeHistory(ctx context.Context, userID uint, limit int) ([]models.TranslationPracticeSession, error)
	GetPracticeStats(ctx context.Context, userID uint) (map[string]interface{}, error)
}

// TranslationPracticeService handles translation practice operations
type TranslationPracticeService struct {
	db             *sql.DB
	storyService   StoryServiceInterface
	questionService QuestionServiceInterface
	config         *config.Config
	logger         *observability.Logger
	templateManager *AITemplateManager
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

	return &TranslationPracticeService{
		db:              db,
		storyService:    storyService,
		questionService: questionService,
		config:          config,
		logger:          logger,
		templateManager: templateManager,
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
		Language: req.Language,
		Level:    req.Level,
		Topic:    stringPtrToString(req.Topic),
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
						"error": err.Error(),
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
			"error": err.Error(),
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
		Language:            lang,
		Level:               lvl,
		OriginalSentence:     req.OriginalSentence,
		UserTranslation:     req.UserTranslation,
		SourceLanguage:      sentence.SourceLanguage,
		TargetLanguage:      sentence.TargetLanguage,
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
		UserID:              userID,
		SentenceID:          req.SentenceID,
		OriginalSentence:    req.OriginalSentence,
		UserTranslation:     req.UserTranslation,
		TranslationDirection: req.TranslationDirection,
		AIFeedback:          cleanFeedback,
		AIScore:             score,
		CreatedAt:           time.Now(),
	}

	if err := s.saveSession(ctx, session); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to save session")
	}

	return session, nil
}

// GetPracticeHistory retrieves practice history for a user
func (s *TranslationPracticeService) GetPracticeHistory(
	ctx context.Context,
	userID uint,
	limit int,
) (result0 []models.TranslationPracticeSession, err error) {
	ctx, span := observability.TraceFunction(ctx, "translation_practice", "get_practice_history",
		attribute.Int("user_id", int(userID)),
		attribute.Int("limit", limit),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, user_id, sentence_id, original_sentence, user_translation,
		       translation_direction, ai_feedback, ai_score, created_at
		FROM translation_practice_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to query practice history")
	}
	defer rows.Close()

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
			return nil, contextutils.WrapErrorf(err, "failed to scan session")
		}

		if aiScore.Valid {
			score := aiScore.Float64
			session.AIScore = &score
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
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
		TotalSessions           int
		AverageScore            sql.NullFloat64
		MinScore                sql.NullFloat64
		MaxScore                sql.NullFloat64
		ExcellentCount          int
		GoodCount               int
		NeedsImprovementCount   int
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
				"total_sessions":           0,
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
		"total_sessions":           stats.TotalSessions,
		"excellent_count":          stats.ExcellentCount,
		"good_count":               stats.GoodCount,
		"needs_improvement_count":  stats.NeedsImprovementCount,
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
	// Remove quotes
	response = strings.Trim(response, "\"'`")
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
	sentences := s.extractSentences(section.Content)
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

	sentences := s.extractSentences(passageText)
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

func (s *TranslationPracticeService) extractSentences(text string) []string {
	// Simple sentence extraction - split by common sentence endings
	// This is a basic implementation; could be improved with proper NLP
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if len(sent) > 10 { // Filter out very short fragments
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

