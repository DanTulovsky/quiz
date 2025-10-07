//go:build integration
// +build integration

package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoryService_CanGenerateSection_Integration tests the CanGenerateSection functionality
func TestStoryService_CanGenerateSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Story",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Test 1: Should be able to generate section initially
	canGenerate, err := storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	assert.True(t, canGenerate, "Should be able to generate section initially")

	// Test 2: Create a section for today
	sectionContent := "This is a test story section."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test 3: Debug - check what's actually in the database
	var sectionCount int
	today := time.Now().Truncate(24 * time.Hour)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)
	t.Logf("Found %d sections for story %d on date %v", sectionCount, story.ID, today)

	// Test 3b: Should not be able to generate another section today (no extra generations allowed)
	canGenerate, err = storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected false)", canGenerate)
	assert.False(t, canGenerate, "Should not be able to generate another section today")

	// Test 3c: Verify that trying to create another section fails with constraint violation
	_, err = storyService.CreateSection(ctx, story.ID, "This should fail due to constraint", "A1", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")

	// Test 4: Test with a different story (should be able to generate)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	canGenerate, err = storyService.CanGenerateSection(ctx, otherStory.ID)
	require.NoError(t, err)
	assert.True(t, canGenerate, "Should be able to generate section for different story")
}

// TestStoryService_CreateSection_Integration tests section creation
func TestStoryService_CreateSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)

	// Test creating a section
	sectionContent := "This is a test story section with enough content to count words properly."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Verify section properties
	assert.Equal(t, story.ID, section.StoryID)
	assert.Equal(t, 1, section.SectionNumber)
	assert.Equal(t, sectionContent, section.Content)
	assert.Equal(t, "A1", section.LanguageLevel)
	assert.Greater(t, section.WordCount, 0)
	assert.NotEmpty(t, section.GeneratedAt)
	assert.NotEmpty(t, section.GenerationDate)

	// Test that creating a second section on the same day fails due to constraint
	_, err = storyService.CreateSection(ctx, story.ID, "Second section content.", "A1", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")

	// Test creating a section for a different story (should work)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	// Creating a section for a different story should work
	otherSection, err := storyService.CreateSection(ctx, otherStory.ID, "Other story section.", "A1", 25)
	require.NoError(t, err)
	require.NotNil(t, otherSection)

	assert.Equal(t, otherStory.ID, otherSection.StoryID)
	assert.Equal(t, 1, otherSection.SectionNumber)
	assert.Equal(t, "Other story section.", otherSection.Content)
}

// TestStoryService_GetLatestSection_Integration tests retrieving the latest section
func TestStoryService_GetLatestSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)

	// Initially no sections
	latestSection, err := storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	assert.Nil(t, latestSection)

	// Create first section
	section1, err := storyService.CreateSection(ctx, story.ID, "First section.", "A1", 20)
	require.NoError(t, err)

	// Should return the first section
	latestSection, err = storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSection)
	assert.Equal(t, section1.ID, latestSection.ID)
	assert.Equal(t, 1, latestSection.SectionNumber)

	// Test that trying to create a second section on the same day fails
	_, err = storyService.CreateSection(ctx, story.ID, "Second section.", "A1", 25)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")

	// Latest section should still be the first section
	latestSection, err = storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSection)
	assert.Equal(t, section1.ID, latestSection.ID)
	assert.Equal(t, 1, latestSection.SectionNumber)
}
