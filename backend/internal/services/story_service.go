package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"
)

// StoryServiceInterface defines the interface for story operations
type StoryServiceInterface interface {
	CreateStory(ctx context.Context, userID uint, language string, req *models.CreateStoryRequest) (*models.Story, error)
	GetUserStories(ctx context.Context, userID uint, includeArchived bool) ([]models.Story, error)
	GetCurrentStory(ctx context.Context, userID uint) (*models.StoryWithSections, error)
	GetStory(ctx context.Context, storyID, userID uint) (*models.StoryWithSections, error)
	ArchiveStory(ctx context.Context, storyID, userID uint) error
	CompleteStory(ctx context.Context, storyID, userID uint) error
	SetCurrentStory(ctx context.Context, storyID, userID uint) error
	DeleteStory(ctx context.Context, storyID, userID uint) error
	GetStorySections(ctx context.Context, storyID uint) ([]models.StorySection, error)
	GetSection(ctx context.Context, sectionID, userID uint) (*models.StorySectionWithQuestions, error)
	CreateSection(ctx context.Context, storyID uint, content, level string, wordCount int) (*models.StorySection, error)
	GetLatestSection(ctx context.Context, storyID uint) (*models.StorySection, error)
	GetAllSectionsText(ctx context.Context, storyID uint) (string, error)
	GetSectionQuestions(ctx context.Context, sectionID uint) ([]models.StorySectionQuestion, error)
	CreateSectionQuestions(ctx context.Context, sectionID uint, questions []models.StorySectionQuestionData) error
	GetRandomQuestions(ctx context.Context, sectionID uint, count int) ([]models.StorySectionQuestion, error)
	CanGenerateSection(ctx context.Context, storyID uint) (bool, error)
	UpdateLastGenerationTime(ctx context.Context, storyID uint) error
	GetSectionLengthTarget(level string, lengthPref *models.SectionLength) int
	GetSectionLengthTargetWithLanguage(language, level string, lengthPref *models.SectionLength) int
	SanitizeInput(input string) string
}

// StoryService handles all story-related operations
type StoryService struct {
	db     *sql.DB
	config *config.Config
	logger *observability.Logger
}

// NewStoryService creates a new StoryService instance
func NewStoryService(db *sql.DB, config *config.Config, logger *observability.Logger) *StoryService {
	if db == nil {
		panic("StoryService requires a valid database connection")
	}
	return &StoryService{
		db:     db,
		config: config,
		logger: logger,
	}
}

// CreateStory creates a new story for the user
func (s *StoryService) CreateStory(ctx context.Context, userID uint, language string, req *models.CreateStoryRequest) (*models.Story, error) {
	if err := req.Validate(); err != nil {
		return nil, contextutils.WrapErrorf(err, "invalid story request")
	}

	// Check if user has reached the archived story limit
	archivedCount, err := s.getArchivedStoryCount(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to check archived story count")
	}

	if archivedCount >= s.config.Story.MaxArchivedPerUser {
		return nil, contextutils.ErrorWithContextf("maximum archived stories limit reached (%d)", s.config.Story.MaxArchivedPerUser)
	}

	// Get user's current language level (stored for potential future use)
	_, err = s.getUserCurrentLevel(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get user level")
	}

	// Unset any existing current story first
	unsetQuery := "UPDATE stories SET is_current = $1, updated_at = NOW() WHERE user_id = $2 AND is_current = $3"
	_, err = s.db.ExecContext(ctx, unsetQuery, false, userID, true)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to unset existing current story")
	}

	// Create the story
	story := &models.Story{
		UserID:                userID,
		Title:                 req.Title,
		Language:              language,
		Subject:               req.Subject,
		AuthorStyle:           req.AuthorStyle,
		TimePeriod:            req.TimePeriod,
		Genre:                 req.Genre,
		Tone:                  req.Tone,
		CharacterNames:        req.CharacterNames,
		CustomInstructions:    req.CustomInstructions,
		SectionLengthOverride: req.SectionLengthOverride,
		Status:                models.StoryStatusActive,
		IsCurrent:             true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.createStory(ctx, story); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to create story")
	}

	s.logger.Info(context.Background(), "Story created successfully",
		map[string]interface{}{
			"story_id": story.ID,
			"user_id":  userID,
			"title":    story.Title,
		})

	return story, nil
}

// GetUserStories retrieves all stories for a user
func (s *StoryService) GetUserStories(ctx context.Context, userID uint, includeArchived bool) ([]models.Story, error) {
	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE user_id = $1`

	args := []interface{}{userID}

	if !includeArchived {
		query += " AND status != $2"
		args = append(args, models.StoryStatusArchived)
	}

	query += " ORDER BY is_current DESC, created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var stories []models.Story
	for rows.Next() {
		var story models.Story
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Title, &story.Language, &story.Subject,
			&story.AuthorStyle, &story.TimePeriod, &story.Genre, &story.Tone,
			&story.CharacterNames, &story.CustomInstructions, &story.SectionLengthOverride,
			&story.Status, &story.IsCurrent, &story.LastSectionGeneratedAt,
			&story.CreatedAt, &story.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}

	return stories, rows.Err()
}

// GetCurrentStory retrieves the current active story for a user
func (s *StoryService) GetCurrentStory(ctx context.Context, userID uint) (*models.StoryWithSections, error) {
	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE user_id = $1 AND is_current = $2 AND status != $3`

	var story models.Story
	err := s.db.QueryRowContext(ctx, query, userID, true, models.StoryStatusArchived).Scan(
		&story.ID, &story.UserID, &story.Title, &story.Language, &story.Subject,
		&story.AuthorStyle, &story.TimePeriod, &story.Genre, &story.Tone,
		&story.CharacterNames, &story.CustomInstructions, &story.SectionLengthOverride,
		&story.Status, &story.IsCurrent, &story.LastSectionGeneratedAt,
		&story.CreatedAt, &story.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No current story
		}
		return nil, contextutils.WrapErrorf(err, "failed to get current story")
	}

	// Load sections
	sections, err := s.GetStorySections(ctx, story.ID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to load story sections")
	}

	return &models.StoryWithSections{
		Story:    story,
		Sections: sections,
	}, nil
}

// GetStory retrieves a specific story with ownership verification
func (s *StoryService) GetStory(ctx context.Context, storyID, userID uint) (*models.StoryWithSections, error) {
	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE id = $1 AND user_id = $2`

	var story models.Story
	err := s.db.QueryRowContext(ctx, query, storyID, userID).Scan(
		&story.ID, &story.UserID, &story.Title, &story.Language, &story.Subject,
		&story.AuthorStyle, &story.TimePeriod, &story.Genre, &story.Tone,
		&story.CharacterNames, &story.CustomInstructions, &story.SectionLengthOverride,
		&story.Status, &story.IsCurrent, &story.LastSectionGeneratedAt,
		&story.CreatedAt, &story.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contextutils.ErrorWithContextf("story not found or access denied")
		}
		return nil, contextutils.WrapErrorf(err, "failed to get story")
	}

	// Load sections
	sections, err := s.GetStorySections(ctx, story.ID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to load story sections")
	}

	return &models.StoryWithSections{
		Story:    story,
		Sections: sections,
	}, nil
}

// ArchiveStory changes a story's status to archived
func (s *StoryService) ArchiveStory(ctx context.Context, storyID, userID uint) error {
	if err := s.validateStoryOwnership(ctx, storyID, userID); err != nil {
		return err
	}

	// Archive the story and unset is_current flag in one transaction
	query := "UPDATE stories SET status = $1, is_current = $2, updated_at = NOW() WHERE id = $3"
	_, err := s.db.ExecContext(ctx, query, models.StoryStatusArchived, false, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to archive story")
	}

	s.logger.Info(context.Background(), "Story archived successfully",
		map[string]interface{}{
			"story_id": storyID,
			"user_id":  userID,
		})

	return nil
}

// CompleteStory changes a story's status to completed
func (s *StoryService) CompleteStory(ctx context.Context, storyID, userID uint) error {
	if err := s.validateStoryOwnership(ctx, storyID, userID); err != nil {
		return err
	}

	query := "UPDATE stories SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := s.db.ExecContext(ctx, query, models.StoryStatusCompleted, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to complete story")
	}

	s.logger.Info(context.Background(), "Story completed successfully",
		map[string]interface{}{
			"story_id": storyID,
			"user_id":  userID,
		})

	return nil
}

// SetCurrentStory sets a story as the current active story for the user
func (s *StoryService) SetCurrentStory(ctx context.Context, storyID, userID uint) error {
	if err := s.validateStoryOwnership(ctx, storyID, userID); err != nil {
		return err
	}

	// Use a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// First, unset all current stories for this user
	query1 := "UPDATE stories SET is_current = false, updated_at = NOW() WHERE user_id = $1 AND is_current = true"
	_, err = tx.ExecContext(ctx, query1, userID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to unset current stories")
	}

	// Then set the specified story as current
	query2 := "UPDATE stories SET is_current = true, updated_at = NOW() WHERE id = $1"
	_, err = tx.ExecContext(ctx, query2, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to set current story")
	}

	return tx.Commit()
}

// DeleteStory permanently deletes a story (only allowed for archived stories)
func (s *StoryService) DeleteStory(ctx context.Context, storyID, userID uint) error {
	// Verify story exists and user owns it
	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE id = $1 AND user_id = $2`

	var story models.Story
	err := s.db.QueryRowContext(ctx, query, storyID, userID).Scan(
		&story.ID, &story.UserID, &story.Title, &story.Language, &story.Subject,
		&story.AuthorStyle, &story.TimePeriod, &story.Genre, &story.Tone,
		&story.CharacterNames, &story.CustomInstructions, &story.SectionLengthOverride,
		&story.Status, &story.IsCurrent, &story.LastSectionGeneratedAt,
		&story.CreatedAt, &story.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextutils.ErrorWithContextf("story not found or access denied")
		}
		return contextutils.WrapErrorf(err, "failed to get story")
	}

	// Only allow deletion of archived or completed stories
	if story.Status != models.StoryStatusArchived && story.Status != models.StoryStatusCompleted {
		return contextutils.ErrorWithContextf("cannot delete active story")
	}

	// Use transaction for atomic deletion
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// Delete questions first (due to foreign key constraints)
	query1 := "DELETE FROM story_section_questions WHERE section_id IN (SELECT id FROM story_sections WHERE story_id = $1)"
	_, err = tx.ExecContext(ctx, query1, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to delete story questions")
	}

	// Delete sections
	query2 := "DELETE FROM story_sections WHERE story_id = $1"
	_, err = tx.ExecContext(ctx, query2, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to delete story sections")
	}

	// Delete story
	query3 := "DELETE FROM stories WHERE id = $1"
	_, err = tx.ExecContext(ctx, query3, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to delete story")
	}

	return tx.Commit()
}

// GetStorySections retrieves all sections for a story
func (s *StoryService) GetStorySections(ctx context.Context, storyID uint) ([]models.StorySection, error) {
	query := `
		SELECT id, story_id, section_number, content, language_level, word_count,
		       generated_at, generation_date
		FROM story_sections
		WHERE story_id = $1
		ORDER BY section_number ASC`

	rows, err := s.db.QueryContext(ctx, query, storyID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get story sections")
	}
	defer func() { _ = rows.Close() }()

	sections := make([]models.StorySection, 0)
	for rows.Next() {
		var section models.StorySection
		err := rows.Scan(
			&section.ID, &section.StoryID, &section.SectionNumber, &section.Content,
			&section.LanguageLevel, &section.WordCount, &section.GeneratedAt, &section.GenerationDate,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan story section")
		}
		sections = append(sections, section)
	}

	return sections, rows.Err()
}

// GetSection retrieves a specific section with ownership verification
func (s *StoryService) GetSection(ctx context.Context, sectionID, userID uint) (*models.StorySectionWithQuestions, error) {
	query := `
		SELECT ss.id, ss.story_id, ss.section_number, ss.content, ss.language_level, ss.word_count,
		       ss.generated_at, ss.generation_date
		FROM story_sections ss
		JOIN stories s ON ss.story_id = s.id
		WHERE ss.id = $1 AND s.user_id = $2`

	var section models.StorySection
	err := s.db.QueryRowContext(ctx, query, sectionID, userID).Scan(
		&section.ID, &section.StoryID, &section.SectionNumber, &section.Content,
		&section.LanguageLevel, &section.WordCount, &section.GeneratedAt, &section.GenerationDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contextutils.ErrorWithContextf("section not found or access denied")
		}
		return nil, contextutils.WrapErrorf(err, "failed to get section")
	}

	// Load questions
	questions, err := s.GetSectionQuestions(ctx, section.ID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to load section questions")
	}

	return &models.StorySectionWithQuestions{
		StorySection: section,
		Questions:    questions,
	}, nil
}

// CreateSection adds a new section to a story
func (s *StoryService) CreateSection(ctx context.Context, storyID uint, content, level string, wordCount int) (*models.StorySection, error) {
	// Get the next section number
	var maxSectionNumber int
	query := "SELECT COALESCE(MAX(section_number), 0) FROM story_sections WHERE story_id = $1"
	err := s.db.QueryRowContext(ctx, query, storyID).Scan(&maxSectionNumber)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get max section number")
	}

	section := &models.StorySection{
		StoryID:        storyID,
		SectionNumber:  maxSectionNumber + 1,
		Content:        content,
		LanguageLevel:  level,
		WordCount:      wordCount,
		GeneratedAt:    time.Now(),
		GenerationDate: time.Now().Truncate(24 * time.Hour),
	}

	if err := s.createSection(ctx, section); err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to create section")
	}

	return section, nil
}

// GetLatestSection retrieves the most recent section for a story
func (s *StoryService) GetLatestSection(ctx context.Context, storyID uint) (*models.StorySection, error) {
	query := `
		SELECT id, story_id, section_number, content, language_level, word_count,
		       generated_at, generation_date
		FROM story_sections
		WHERE story_id = $1
		ORDER BY section_number DESC
		LIMIT 1`

	var section models.StorySection
	err := s.db.QueryRowContext(ctx, query, storyID).Scan(
		&section.ID, &section.StoryID, &section.SectionNumber, &section.Content,
		&section.LanguageLevel, &section.WordCount, &section.GeneratedAt, &section.GenerationDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No sections yet
		}
		return nil, contextutils.WrapErrorf(err, "failed to get latest section")
	}

	return &section, nil
}

// GetAllSectionsText concatenates all section content for AI context
func (s *StoryService) GetAllSectionsText(ctx context.Context, storyID uint) (string, error) {
	sections, err := s.GetStorySections(ctx, storyID)
	if err != nil {
		return "", err
	}

	var sectionsText strings.Builder
	for i, section := range sections {
		if i > 0 {
			sectionsText.WriteString("\n\n")
		}
		sectionsText.WriteString(fmt.Sprintf("Section %d:\n%s", section.SectionNumber, section.Content))
	}

	return sectionsText.String(), nil
}

// GetSectionQuestions retrieves all questions for a section
func (s *StoryService) GetSectionQuestions(ctx context.Context, sectionID uint) ([]models.StorySectionQuestion, error) {
	query := `
		SELECT id, section_id, question_text, options, correct_answer_index, explanation, created_at
		FROM story_section_questions
		WHERE section_id = $1`

	rows, err := s.db.QueryContext(ctx, query, sectionID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get section questions")
	}
	defer func() { _ = rows.Close() }()

	var questions []models.StorySectionQuestion
	for rows.Next() {
		var question models.StorySectionQuestion
		err := rows.Scan(
			&question.ID, &question.SectionID, &question.QuestionText, &question.Options,
			&question.CorrectAnswerIndex, &question.Explanation, &question.CreatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan question")
		}
		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// CreateSectionQuestions bulk inserts questions for a section
func (s *StoryService) CreateSectionQuestions(ctx context.Context, sectionID uint, questions []models.StorySectionQuestionData) error {
	if len(questions) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	for _, q := range questions {
		query := `
			INSERT INTO story_section_questions (
				section_id, question_text, options, correct_answer_index, explanation, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)`

		_, err := tx.ExecContext(ctx, query,
			sectionID, q.QuestionText, q.Options, q.CorrectAnswerIndex, q.Explanation, time.Now(),
		)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to insert question")
		}
	}

	return tx.Commit()
}

// GetRandomQuestions retrieves N random questions for a section
func (s *StoryService) GetRandomQuestions(ctx context.Context, sectionID uint, count int) ([]models.StorySectionQuestion, error) {
	query := `
		SELECT id, section_id, question_text, options, correct_answer_index, explanation, created_at
		FROM story_section_questions
		WHERE section_id = $1
		ORDER BY RANDOM()
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, sectionID, count)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get random questions")
	}
	defer func() { _ = rows.Close() }()

	var questions []models.StorySectionQuestion
	for rows.Next() {
		var question models.StorySectionQuestion
		err := rows.Scan(
			&question.ID, &question.SectionID, &question.QuestionText, &question.Options,
			&question.CorrectAnswerIndex, &question.Explanation, &question.CreatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan question")
		}
		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// CanGenerateSection checks if a new section can be generated for a story today
func (s *StoryService) CanGenerateSection(ctx context.Context, storyID uint) (bool, error) {
	query := `
		SELECT status, is_current, last_section_generated_at
		FROM stories
		WHERE id = $1`

	var status string
	var isCurrent bool
	var lastGen *time.Time

	err := s.db.QueryRowContext(ctx, query, storyID).Scan(&status, &isCurrent, &lastGen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, contextutils.ErrorWithContextf("story not found")
		}
		return false, contextutils.WrapErrorf(err, "failed to get story")
	}

	// Check if story generation is enabled globally
	if !s.config.Story.GenerationEnabled {
		return false, nil
	}

	// Check if story is active and current
	if status != string(models.StoryStatusActive) || !isCurrent {
		return false, nil
	}

	// Check if already generated today
	if lastGen != nil {
		today := time.Now().Truncate(24 * time.Hour)
		lastGenTime := lastGen.Truncate(24 * time.Hour)
		if lastGenTime.Equal(today) {
			return false, nil
		}
	}

	return true, nil
}

// UpdateLastGenerationTime sets the last section generation time for a story
func (s *StoryService) UpdateLastGenerationTime(ctx context.Context, storyID uint) error {
	query := "UPDATE stories SET last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
	_, err := s.db.ExecContext(ctx, query, time.Now(), storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to update generation time")
	}

	return nil
}

// Helper methods

// getArchivedStoryCount counts archived stories for a user
func (s *StoryService) getArchivedStoryCount(ctx context.Context, userID uint) (int, error) {
	query := "SELECT COUNT(*) FROM stories WHERE user_id = $1 AND status = $2"
	var count int
	err := s.db.QueryRowContext(ctx, query, userID, models.StoryStatusArchived).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// getUserCurrentLevel retrieves the user's current language level
func (s *StoryService) getUserCurrentLevel(ctx context.Context, userID uint) (string, error) {
	query := "SELECT current_level FROM users WHERE id = $1"
	var level sql.NullString
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&level)
	if err != nil {
		return "", contextutils.WrapErrorf(err, "failed to get user")
	}

	if !level.Valid {
		return "", contextutils.ErrorWithContextf("user has no current level set")
	}

	return level.String, nil
}

// validateStoryOwnership verifies that a user owns a story
func (s *StoryService) validateStoryOwnership(ctx context.Context, storyID, userID uint) error {
	query := "SELECT COUNT(*) FROM stories WHERE id = $1 AND user_id = $2"
	var count int
	err := s.db.QueryRowContext(ctx, query, storyID, userID).Scan(&count)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to validate story ownership")
	}

	if count == 0 {
		return contextutils.ErrorWithContextf("story not found or access denied")
	}

	return nil
}

// GetSectionLengthTarget returns the target word count for a story section
func (s *StoryService) GetSectionLengthTarget(level string, lengthPref *models.SectionLength) int {
	return models.GetSectionLengthTarget(level, lengthPref)
}

// GetSectionLengthTargetWithLanguage returns the target word count with language-specific overrides
func (s *StoryService) GetSectionLengthTargetWithLanguage(language, level string, lengthPref *models.SectionLength) int {
	// Check for language-specific overrides in config
	if languageOverrides, exists := s.config.Story.SectionLengths.Overrides[language]; exists {
		if levelTargets, exists := languageOverrides[level]; exists {
			// Use the override if it exists for this level
			if lengthPref != nil {
				if target, exists := levelTargets[string(*lengthPref)]; exists {
					return target
				}
			}
			// Default to medium if no specific length preference
			if target, exists := levelTargets["medium"]; exists {
				return target
			}
		}
	}

	// Fall back to the default implementation
	return models.GetSectionLengthTarget(level, lengthPref)
}

// SanitizeInput sanitizes user input for safe use in AI prompts
func (s *StoryService) SanitizeInput(input string) string {
	return models.SanitizeInput(input)
}

// Database helper methods using sql.DB

// createStory inserts a new story into the database
func (s *StoryService) createStory(ctx context.Context, story *models.Story) error {
	query := `
		INSERT INTO stories (
			user_id, title, language, subject, author_style, time_period, genre, tone,
			character_names, custom_instructions, section_length_override, status, is_current,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`

	err := s.db.QueryRowContext(ctx, query,
		story.UserID, story.Title, story.Language, story.Subject, story.AuthorStyle,
		story.TimePeriod, story.Genre, story.Tone, story.CharacterNames,
		story.CustomInstructions, story.SectionLengthOverride, story.Status,
		story.IsCurrent, story.CreatedAt, story.UpdatedAt,
	).Scan(&story.ID)

	return err
}

// createSection inserts a new section into the database
func (s *StoryService) createSection(ctx context.Context, section *models.StorySection) error {
	query := `
		INSERT INTO story_sections (
			story_id, section_number, content, language_level, word_count,
			generated_at, generation_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := s.db.QueryRowContext(ctx, query,
		section.StoryID, section.SectionNumber, section.Content, section.LanguageLevel,
		section.WordCount, section.GeneratedAt, section.GenerationDate,
	).Scan(&section.ID)

	return err
}
