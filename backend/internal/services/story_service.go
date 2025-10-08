package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
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
	DeleteAllStoriesForUser(ctx context.Context, userID uint) error
	FixCurrentStoryConstraint(ctx context.Context) error
	GetStorySections(ctx context.Context, storyID uint) ([]models.StorySection, error)
	GetSection(ctx context.Context, sectionID, userID uint) (*models.StorySectionWithQuestions, error)
	CreateSection(ctx context.Context, storyID uint, content, level string, wordCount int, generatedBy models.GeneratorType) (*models.StorySection, error)
	GetLatestSection(ctx context.Context, storyID uint) (*models.StorySection, error)
	GetAllSectionsText(ctx context.Context, storyID uint) (string, error)
	GetSectionQuestions(ctx context.Context, sectionID uint) ([]models.StorySectionQuestion, error)
	CreateSectionQuestions(ctx context.Context, sectionID uint, questions []models.StorySectionQuestionData) error
	GetRandomQuestions(ctx context.Context, sectionID uint, count int) ([]models.StorySectionQuestion, error)
	UpdateLastGenerationTime(ctx context.Context, storyID uint, generatorType models.GeneratorType) error
	GetSectionLengthTarget(level string, lengthPref *models.SectionLength) int
	GetSectionLengthTargetWithLanguage(language, level string, lengthPref *models.SectionLength) int
	SanitizeInput(input string) string
	GenerateStorySection(ctx context.Context, storyID, userID uint, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.StorySectionWithQuestions, error)
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

	// Unset any existing current story in the same language first
	unsetQuery := "UPDATE stories SET is_current = $1, updated_at = NOW() WHERE user_id = $2 AND language = $3 AND is_current = $4"
	_, err = s.db.ExecContext(ctx, unsetQuery, false, userID, language, true)
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

// GetUserStories retrieves all stories for a user in their preferred language
func (s *StoryService) GetUserStories(ctx context.Context, userID uint, includeArchived bool) ([]models.Story, error) {
	// Get user's preferred language
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get user")
	}

	if user == nil {
		return nil, contextutils.ErrorWithContextf("user not found: %d", userID)
	}

	language := "en" // default
	if user.PreferredLanguage.Valid {
		language = user.PreferredLanguage.String
	}

	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE user_id = $1 AND language = $2`

	args := []interface{}{userID, language}

	if !includeArchived {
		query += " AND status != $3"
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

// GetCurrentStory retrieves the current active story for a user in their current language
func (s *StoryService) GetCurrentStory(ctx context.Context, userID uint) (*models.StoryWithSections, error) {
	// Get user's current language preference
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get user")
	}

	if user == nil {
		return nil, contextutils.ErrorWithContextf("user not found: %d", userID)
	}

	language := "en" // default
	if user.PreferredLanguage.Valid {
		language = user.PreferredLanguage.String
	}

	query := `
		SELECT id, user_id, title, language, subject, author_style, time_period, genre, tone,
		       character_names, custom_instructions, section_length_override, status, is_current,
		       last_section_generated_at, created_at, updated_at
		FROM stories
		WHERE user_id = $1 AND language = $2 AND is_current = $3 AND status != $4`

	var story models.Story
	err = s.db.QueryRowContext(ctx, query, userID, language, true, models.StoryStatusArchived).Scan(
		&story.ID, &story.UserID, &story.Title, &story.Language, &story.Subject,
		&story.AuthorStyle, &story.TimePeriod, &story.Genre, &story.Tone,
		&story.CharacterNames, &story.CustomInstructions, &story.SectionLengthOverride,
		&story.Status, &story.IsCurrent, &story.LastSectionGeneratedAt,
		&story.CreatedAt, &story.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No current story in user's language
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

	// Use a transaction to ensure atomicity and handle is_current flag properly
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// First, check if the story is completed (completed stories cannot be archived)
	var status string
	var isCurrentlyCurrent bool
	checkQuery := "SELECT status, is_current FROM stories WHERE id = $1"
	err = tx.QueryRowContext(ctx, checkQuery, storyID).Scan(&status, &isCurrentlyCurrent)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to check story status")
	}

	// Prevent archiving completed stories
	if status == string(models.StoryStatusCompleted) {
		return contextutils.ErrorWithContextf("cannot archive completed stories")
	}

	// Archive the story and unset is_current flag
	query := "UPDATE stories SET status = $1, is_current = $2, updated_at = NOW() WHERE id = $3"
	_, err = tx.ExecContext(ctx, query, models.StoryStatusArchived, false, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to archive story")
	}

	// If the archived story was the current story, ensure exactly one story is current
	// This handles constraint violations by ensuring only one current story per user
	if isCurrentlyCurrent {
		// Find any other active story to promote as current
		// If none exists, that's fine - no story will be current
		var activeStoryID *uint
		selectQuery := `
			SELECT id FROM stories
			WHERE user_id = $1 AND status = 'active' AND id != $2
			ORDER BY updated_at DESC
			LIMIT 1`
		err = tx.QueryRowContext(ctx, selectQuery, userID, storyID).Scan(&activeStoryID)
		if err != nil && err != sql.ErrNoRows {
			return contextutils.WrapErrorf(err, "failed to find active story to set as current")
		}

		// If we found an active story, set it as current
		if activeStoryID != nil {
			setCurrentQuery := "UPDATE stories SET is_current = true, updated_at = NOW() WHERE id = $1"
			_, err = tx.ExecContext(ctx, setCurrentQuery, *activeStoryID)
			if err != nil {
				return contextutils.WrapErrorf(err, "failed to set new current story")
			}
		}
		// If no active story exists, that's fine - no story will be current
	}

	err = tx.Commit()
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to commit transaction")
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

	query := "UPDATE stories SET status = $1, is_current = false, updated_at = NOW() WHERE id = $2"
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

// SetCurrentStory sets a story as the current active story for the user in its language
func (s *StoryService) SetCurrentStory(ctx context.Context, storyID, userID uint) error {
	if err := s.validateStoryOwnership(ctx, storyID, userID); err != nil {
		return err
	}

	// Get the story's language and status
	query := "SELECT language, status FROM stories WHERE id = $1 AND user_id = $2"
	var language string
	var status models.StoryStatus
	err := s.db.QueryRowContext(ctx, query, storyID, userID).Scan(&language, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextutils.ErrorWithContextf("story not found or access denied")
		}
		return contextutils.WrapErrorf(err, "failed to get story language and status")
	}

	// Only allow restoring active stories (not completed ones)
	if status == models.StoryStatusCompleted {
		return contextutils.ErrorWithContextf("cannot restore completed stories")
	}

	// Get the user's preferred language
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to get user")
	}

	if user == nil {
		return contextutils.ErrorWithContextf("user not found")
	}

	userPreferredLanguage := "en" // default
	if user.PreferredLanguage.Valid {
		userPreferredLanguage = user.PreferredLanguage.String
	}

	// Check if the story's language matches the user's preferred language
	if language != userPreferredLanguage {
		return contextutils.ErrorWithContextf("cannot restore story in different language than preferred language")
	}

	// Use a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// First, unset current stories in the same language for this user to avoid constraint violations
	unsetQuery := "UPDATE stories SET is_current = false, updated_at = NOW() WHERE user_id = $1 AND language = $2 AND is_current = true"
	_, err = tx.ExecContext(ctx, unsetQuery, userID, language)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to unset current stories")
	}

	// Set the specified story as current and active
	setQuery := "UPDATE stories SET is_current = true, status = $1, updated_at = NOW() WHERE id = $2"
	_, err = tx.ExecContext(ctx, setQuery, models.StoryStatusActive, storyID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to set current story")
	}

	return tx.Commit()
}

// FixCurrentStoryConstraint fixes any constraint violations where multiple stories are marked as current for the same user in the same language
func (s *StoryService) FixCurrentStoryConstraint(ctx context.Context) error {
	// Find all users who have multiple current stories in the same language
	query := `
		SELECT user_id, language, COUNT(*) as current_count
		FROM stories
		WHERE is_current = true
		GROUP BY user_id, language
		HAVING COUNT(*) > 1`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to find users with multiple current stories in same language")
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID uint
		var language string
		var currentCount int

		if err := rows.Scan(&userID, &language, &currentCount); err != nil {
			return contextutils.WrapErrorf(err, "failed to scan user constraint violation")
		}

		// Fix constraint violation for this user and language
		if err := s.fixUserCurrentStoryConstraint(ctx, userID, language); err != nil {
			return contextutils.WrapErrorf(err, "failed to fix constraint for user %d in language %s", userID, language)
		}
	}

	return rows.Err()
}

// fixUserCurrentStoryConstraint fixes constraint violations for a specific user in a specific language
func (s *StoryService) fixUserCurrentStoryConstraint(ctx context.Context, userID uint, language string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// Find all current stories for this user in this language, ordered by most recently updated
	var currentStories []uint
	selectQuery := `
		SELECT id FROM stories
		WHERE user_id = $1 AND language = $2 AND is_current = true
		ORDER BY updated_at DESC`

	rows, err := tx.QueryContext(ctx, selectQuery, userID, language)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to find current stories for user in language")
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var storyID uint
		if err := rows.Scan(&storyID); err != nil {
			return contextutils.WrapErrorf(err, "failed to scan story ID")
		}
		currentStories = append(currentStories, storyID)
	}

	if len(currentStories) <= 1 {
		// No constraint violation for this user in this language
		return tx.Commit()
	}

	// Unset all current stories except the most recently updated one
	for i := 1; i < len(currentStories); i++ {
		unsetQuery := "UPDATE stories SET is_current = false, updated_at = NOW() WHERE id = $1"
		_, err = tx.ExecContext(ctx, unsetQuery, currentStories[i])
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to unset current story %d", currentStories[i])
		}
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

// DeleteAllStoriesForUser deletes all stories (and their sections/questions) for a given user
func (s *StoryService) DeleteAllStoriesForUser(ctx context.Context, userID uint) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// Delete questions for all sections belonging to stories of this user
	q1 := `DELETE FROM story_section_questions WHERE section_id IN (SELECT id FROM story_sections WHERE story_id IN (SELECT id FROM stories WHERE user_id = $1))`
	if _, err := tx.ExecContext(ctx, q1, userID); err != nil {
		return contextutils.WrapErrorf(err, "failed to delete story questions for user %d", userID)
	}

	// Delete sections for all stories belonging to this user
	q2 := `DELETE FROM story_sections WHERE story_id IN (SELECT id FROM stories WHERE user_id = $1)`
	if _, err := tx.ExecContext(ctx, q2, userID); err != nil {
		return contextutils.WrapErrorf(err, "failed to delete story sections for user %d", userID)
	}

	// Finally delete stories
	q3 := `DELETE FROM stories WHERE user_id = $1`
	if _, err := tx.ExecContext(ctx, q3, userID); err != nil {
		return contextutils.WrapErrorf(err, "failed to delete stories for user %d", userID)
	}

	if err := tx.Commit(); err != nil {
		return contextutils.WrapErrorf(err, "failed to commit delete all stories transaction for user %d", userID)
	}

	s.logger.Info(context.Background(), "Deleted all stories for user", map[string]interface{}{"user_id": userID})
	return nil
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
			&section.LanguageLevel, &section.WordCount, &section.GeneratedBy, &section.GeneratedAt, &section.GenerationDate,
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
func (s *StoryService) CreateSection(ctx context.Context, storyID uint, content, level string, wordCount int, generatedBy models.GeneratorType) (*models.StorySection, error) {
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
		GeneratedBy:    generatedBy,
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
		       generated_by, generated_at, generation_date
		FROM story_sections
		WHERE story_id = $1
		ORDER BY section_number DESC
		LIMIT 1`

	var section models.StorySection
	err := s.db.QueryRowContext(ctx, query, storyID).Scan(
		&section.ID, &section.StoryID, &section.SectionNumber, &section.Content,
		&section.LanguageLevel, &section.WordCount, &section.GeneratedBy, &section.GeneratedAt, &section.GenerationDate,
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
		var optionsJSON []byte

		err := rows.Scan(
			&question.ID, &question.SectionID, &question.QuestionText, &optionsJSON,
			&question.CorrectAnswerIndex, &question.Explanation, &question.CreatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan question")
		}

		// Unmarshal JSON options back to []string
		err = json.Unmarshal(optionsJSON, &question.Options)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to unmarshal options from JSON")
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

		// Convert []string options to JSON for PostgreSQL JSONB column
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to marshal options to JSON")
		}

		_, err = tx.ExecContext(ctx, query,
			sectionID, q.QuestionText, optionsJSON, q.CorrectAnswerIndex, q.Explanation, time.Now(),
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
		var optionsJSON []byte

		err := rows.Scan(
			&question.ID, &question.SectionID, &question.QuestionText, &optionsJSON,
			&question.CorrectAnswerIndex, &question.Explanation, &question.CreatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan question")
		}

		// Unmarshal JSON options back to []string
		err = json.Unmarshal(optionsJSON, &question.Options)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to unmarshal options from JSON")
		}

		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// canGenerateSection checks if a new section can be generated for a story today by a specific generator
func (s *StoryService) canGenerateSection(ctx context.Context, storyID uint, generatorType models.GeneratorType) (*models.StoryGenerationEligibilityResponse, error) {
	query := `
		SELECT status, is_current, last_section_generated_at, extra_generations_today
		FROM stories
		WHERE id = $1`

	var status string
	var isCurrent bool
	var lastGen *time.Time
	var extraGenerationsToday int

	err := s.db.QueryRowContext(ctx, query, storyID).Scan(&status, &isCurrent, &lastGen, &extraGenerationsToday)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.StoryGenerationEligibilityResponse{
				CanGenerate: false,
				Reason:      "story not found",
			}, nil
		}
		return nil, contextutils.WrapErrorf(err, "failed to get story")
	}

	// Check if story generation is enabled globally
	if !s.config.Story.GenerationEnabled {
		return &models.StoryGenerationEligibilityResponse{
			CanGenerate: false,
			Reason:      "story generation is disabled globally",
		}, nil
	}

	// Check if story is active and current
	if status != string(models.StoryStatusActive) || !isCurrent {
		return &models.StoryGenerationEligibilityResponse{
			CanGenerate: false,
			Reason:      "story is not active",
		}, nil
	}

	// Check generation count for today by generator type
	today := time.Now().Truncate(24 * time.Hour)
	var sectionCount int
	sectionQuery := `
		SELECT COUNT(*)
		FROM story_sections
		WHERE story_id = $1 AND generation_date = $2 AND generated_by = $3`

	err = s.db.QueryRowContext(ctx, sectionQuery, storyID, today, generatorType).Scan(&sectionCount)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to check existing sections today by generator type")
	}

	// Check limits based on generator type
	switch generatorType {
	case models.GeneratorTypeWorker:
		// Worker can generate exactly 1 section per day
		if sectionCount >= 1 {
			return &models.StoryGenerationEligibilityResponse{
				CanGenerate: false,
				Reason:      "worker has already generated a section today",
			}, nil
		}
	case models.GeneratorTypeUser:
		// User can generate MaxExtraGenerationsPerDay + 1 sections per day (includes 1 free generation)
		maxUserGenerations := s.config.Story.MaxExtraGenerationsPerDay + 1
		if sectionCount >= maxUserGenerations {
			return &models.StoryGenerationEligibilityResponse{
				CanGenerate: false,
				Reason:      "user has reached daily generation limit",
			}, nil
		}
	default:
		return &models.StoryGenerationEligibilityResponse{
			CanGenerate: false,
			Reason:      "invalid generator type",
		}, nil
	}

	// Allow generation if within limits
	return &models.StoryGenerationEligibilityResponse{
		CanGenerate: true,
	}, nil
}

// UpdateLastGenerationTime sets the last section generation time for a story
func (s *StoryService) UpdateLastGenerationTime(ctx context.Context, storyID uint, generatorType models.GeneratorType) error {
	// Check if this is an extra generation (second generation today)
	query := `
		SELECT last_section_generated_at, extra_generations_today
		FROM stories
		WHERE id = $1`

	var lastGen *time.Time
	var extraGenerationsToday int

	err := s.db.QueryRowContext(ctx, query, storyID).Scan(&lastGen, &extraGenerationsToday)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to get current generation info")
	}

	now := time.Now()

	// Check if we already generated today and update accordingly
	if lastGen != nil {
		lastGenTime := lastGen.Truncate(24 * time.Hour)
		today := now.Truncate(24 * time.Hour)
		if lastGenTime.Equal(today) {
			// Only increment counter for user generations
			if generatorType == models.GeneratorTypeUser {
				maxTotal := s.config.Story.MaxExtraGenerationsPerDay + 1
				if extraGenerationsToday < maxTotal {
					updateQuery := "UPDATE stories SET extra_generations_today = extra_generations_today + 1, last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
					_, err = s.db.ExecContext(ctx, updateQuery, now, storyID)
					if err != nil {
						return contextutils.WrapErrorf(err, "failed to update generation time")
					}
				} else {
					// Limit reached - just update timestamp
					updateQuery := "UPDATE stories SET last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
					_, err = s.db.ExecContext(ctx, updateQuery, now, storyID)
					if err != nil {
						return contextutils.WrapErrorf(err, "failed to update generation time")
					}
				}
			} else {
				// Worker generation - just update timestamp
				updateQuery := "UPDATE stories SET last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
				_, err = s.db.ExecContext(ctx, updateQuery, now, storyID)
				if err != nil {
					return contextutils.WrapErrorf(err, "failed to update generation time")
				}
			}
			return nil
		}
	}

	// First generation today - only increment counter for user generations
	if generatorType == models.GeneratorTypeUser {
		updateQuery := "UPDATE stories SET extra_generations_today = extra_generations_today + 1, last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
		_, err = s.db.ExecContext(ctx, updateQuery, now, storyID)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to update generation time for first generation")
		}
	} else {
		// Worker generation - just update timestamp
		updateQuery := "UPDATE stories SET last_section_generated_at = $1, updated_at = NOW() WHERE id = $2"
		_, err = s.db.ExecContext(ctx, updateQuery, now, storyID)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to update generation time for first generation")
		}
	}

	return nil
}

// Helper methods

// getUserByID retrieves a user by their ID
func (s *StoryService) getUserByID(ctx context.Context, userID uint) (*models.User, error) {
	query := "SELECT id, username, email, preferred_language, current_level, ai_provider, ai_model, ai_api_key, created_at, updated_at FROM users WHERE id = $1"

	var user models.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PreferredLanguage,
		&user.CurrentLevel, &user.AIProvider, &user.AIModel, &user.AIAPIKey,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, contextutils.WrapErrorf(err, "failed to get user")
	}

	return &user, nil
}

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
			story_id, section_number, content, language_level, word_count, generated_by,
			generated_at, generation_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := s.db.QueryRowContext(ctx, query,
		section.StoryID, section.SectionNumber, section.Content, section.LanguageLevel,
		section.WordCount, section.GeneratedBy, section.GeneratedAt, section.GenerationDate,
	).Scan(&section.ID)

	return err
}

// GenerateStorySection generates a new section for a story using AI
func (s *StoryService) GenerateStorySection(ctx context.Context, storyID, userID uint, aiService AIServiceInterface, userAIConfig *models.UserAIConfig) (*models.StorySectionWithQuestions, error) {
	ctx, span := observability.TraceFunction(ctx, "story_service", "generate_section",
		attribute.Int("story.id", int(storyID)),
		observability.AttributeUserID(int(userID)),
	)
	defer observability.FinishSpan(span, nil)

	// Get the story to verify ownership and get details
	story, err := s.GetStory(ctx, storyID, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get story for generation")
	}

	// Determine generator type based on context
	// For now, we'll assume worker if userAIConfig is nil or if it's a worker context
	generatorType := models.GeneratorTypeUser
	if userAIConfig == nil {
		generatorType = models.GeneratorTypeWorker
	}

	// Check if generation is allowed today by this generator type
	eligibility, err := s.canGenerateSection(ctx, storyID, generatorType)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to check generation eligibility")
	}
	if !eligibility.CanGenerate {
		return nil, contextutils.WrapError(contextutils.ErrGenerationLimitReached, eligibility.Reason)
	}

	// Get user for AI configuration and language preferences
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get user")
	}
	if user == nil {
		return nil, contextutils.ErrorWithContextf("user not found")
	}

	// Get all previous sections for context
	previousSections, err := s.GetAllSectionsText(ctx, storyID)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get previous sections")
	}

	// Get the user's current language level (handle sql.NullString)
	if !user.CurrentLevel.Valid {
		return nil, contextutils.ErrorWithContextf("user level not found")
	}

	// Determine target length for this user's level
	targetWords := s.GetSectionLengthTarget(user.CurrentLevel.String, story.SectionLengthOverride)

	// Build the generation request
	genReq := &models.StoryGenerationRequest{
		UserID:             userID,
		StoryID:            storyID,
		Language:           story.Language,
		Level:              user.CurrentLevel.String,
		Title:              story.Title,
		Subject:            story.Subject,
		AuthorStyle:        story.AuthorStyle,
		TimePeriod:         story.TimePeriod,
		Genre:              story.Genre,
		Tone:               story.Tone,
		CharacterNames:     story.CharacterNames,
		CustomInstructions: story.CustomInstructions,
		SectionLength:      models.SectionLengthMedium, // Use medium as default
		PreviousSections:   previousSections,
		IsFirstSection:     len(story.Sections) == 0,
		TargetWords:        targetWords,
		TargetSentences:    targetWords / 15, // Rough estimate
	}

	// Generate the story section using AI
	sectionContent, err := aiService.GenerateStorySection(ctx, userAIConfig, genReq)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to generate story section")
	}

	// Count words in the generated content
	wordCount := len(strings.Fields(sectionContent))

	// Create the section in the database
	section, err := s.CreateSection(ctx, storyID, sectionContent, user.CurrentLevel.String, wordCount, generatorType)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to create story section")
	}

	// Generate comprehension questions for the section
	questionsReq := &models.StoryQuestionsRequest{
		UserID:        userID,
		SectionID:     section.ID,
		Language:      story.Language,
		Level:         user.CurrentLevel.String,
		SectionText:   sectionContent,
		QuestionCount: s.config.Story.QuestionsPerSection,
	}

	questions, err := aiService.GenerateStoryQuestions(ctx, userAIConfig, questionsReq)
	if err != nil {
		s.logger.Warn(ctx, "Failed to generate questions for story section",
			map[string]interface{}{
				"section_id": section.ID,
				"story_id":   storyID,
				"user_id":    userID,
				"error":      err.Error(),
			})
		// Continue anyway - questions are nice to have but not critical
	} else {
		// Convert to database model slice
		dbQuestions := make([]models.StorySectionQuestionData, len(questions))
		for i, q := range questions {
			dbQuestions[i] = *q
		}

		// Save the questions to the database
		if err := s.CreateSectionQuestions(ctx, section.ID, dbQuestions); err != nil {
			s.logger.Warn(ctx, "Failed to save story questions",
				map[string]interface{}{
					"section_id": section.ID,
					"story_id":   storyID,
					"user_id":    userID,
					"error":      err.Error(),
				})
		}
	}

	// Update the story's last generation time
	if err := s.UpdateLastGenerationTime(ctx, storyID, generatorType); err != nil {
		s.logger.Warn(ctx, "Failed to update story generation time",
			map[string]interface{}{
				"story_id": storyID,
				"user_id":  userID,
				"error":    err.Error(),
			})
	}

	s.logger.Info(ctx, "Story section generated successfully",
		map[string]interface{}{
			"story_id":       storyID,
			"section_id":     section.ID,
			"section_number": section.SectionNumber,
			"user_id":        userID,
			"word_count":     wordCount,
			"question_count": len(questions),
		})

	// Load questions for the section
	sectionQuestions, err := s.GetSectionQuestions(ctx, section.ID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to load section questions after generation",
			map[string]interface{}{
				"section_id": section.ID,
				"story_id":   storyID,
				"user_id":    userID,
				"error":      err.Error(),
			})
		// Return section without questions rather than failing
		sectionQuestions = []models.StorySectionQuestion{}
	}

	return &models.StorySectionWithQuestions{
		StorySection: *section,
		Questions:    sectionQuestions,
	}, nil
}
