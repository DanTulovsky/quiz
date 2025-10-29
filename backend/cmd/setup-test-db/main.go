// Package main provides a utility to set up the test database with initial data.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// TestUser represents a user in the test data files
type TestUser struct {
	Username          string   `yaml:"username"`
	Email             string   `yaml:"email"`
	Password          string   `yaml:"password"` // Special field for password creation
	PreferredLanguage string   `yaml:"preferred_language"`
	CurrentLevel      string   `yaml:"current_level"`
	AIProvider        string   `yaml:"ai_provider"`
	AIModel           string   `yaml:"ai_model"`
	AIAPIKey          string   `yaml:"ai_api_key"`
	Roles             []string `yaml:"roles"`
}

// TestUsers represents a collection of test users
type TestUsers struct {
	Users []TestUser `yaml:"users"`
}

// TestQuestions represents a collection of test questions
type TestQuestions struct {
	Questions []models.Question `yaml:"questions"`
}

// TestResponses represents a collection of test user responses
type TestResponses struct {
	UserResponses []struct {
		Username       string `yaml:"username"`
		QuestionIndex  int    `yaml:"question_index"`
		UserAnswer     string `yaml:"user_answer"`
		IsCorrect      bool   `yaml:"is_correct"`
		ResponseTimeMs int    `yaml:"response_time_ms"`
	} `yaml:"user_responses"`

	QuestionReports []struct {
		Username      string  `yaml:"username"`
		QuestionIndex int     `yaml:"question_index"`
		ReportReason  string  `yaml:"report_reason"`
		CreatedAt     *string `yaml:"created_at"`
	} `yaml:"question_reports"`
}

// TestAnalytics represents analytics test data
type TestAnalytics struct {
	PriorityScores []struct {
		Username         string  `yaml:"username"`
		QuestionIndex    int     `yaml:"question_index"`
		PriorityScore    float64 `yaml:"priority_score"`
		LastCalculatedAt string  `yaml:"last_calculated_at"`
	} `yaml:"priority_scores"`

	LearningPreferences []struct {
		Username             string  `yaml:"username"`
		FocusOnWeakAreas     bool    `yaml:"focus_on_weak_areas"`
		FreshQuestionRatio   float64 `yaml:"fresh_question_ratio"`
		WeakAreaBoost        float64 `yaml:"weak_area_boost"`
		KnownQuestionPenalty float64 `yaml:"known_question_penalty"`
		ReviewIntervalDays   int     `yaml:"review_interval_days"`
		DailyReminderEnabled bool    `yaml:"daily_reminder_enabled"`
	} `yaml:"learning_preferences"`

	PerformanceMetrics []struct {
		Username              string  `yaml:"username"`
		Topic                 string  `yaml:"topic"`
		Language              string  `yaml:"language"`
		Level                 string  `yaml:"level"`
		TotalAttempts         int     `yaml:"total_attempts"`
		CorrectAttempts       int     `yaml:"correct_attempts"`
		AverageResponseTimeMs float64 `yaml:"average_response_time_ms"`
	} `yaml:"performance_metrics"`

	UserQuestionMetadata []struct {
		Username        string  `yaml:"username"`
		QuestionIndex   int     `yaml:"question_index"`
		MarkedAsKnown   bool    `yaml:"marked_as_known"`
		MarkedAsKnownAt *string `yaml:"marked_as_known_at"`
	} `yaml:"user_question_metadata"`
}

// TestDailyAssignments represents the structure for daily question assignments in test data
type TestDailyAssignments struct {
	DailyAssignments []struct {
		Username           string `yaml:"username"`
		Date               string `yaml:"date"`
		QuestionIDs        []int  `yaml:"question_ids"`
		CompletedQuestions []int  `yaml:"completed_questions"`
	} `yaml:"daily_assignments"`
}

// TestMessageData represents message data for E2E tests
type TestMessageData struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Bookmarked     bool   `json:"bookmarked"`
	QuestionID     *int   `json:"question_id,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// TestConversationData represents conversation data for E2E tests
type TestConversationData struct {
	ID       string            `json:"id"`
	Username string            `json:"username"`
	Title    string            `json:"title"`
	Messages []TestMessageData `json:"messages"`
}

// TestConversations represents a collection of test conversations
type TestConversations struct {
	Conversations []struct {
		Username string `yaml:"username"`
		Title    string `yaml:"title"`
		Messages []struct {
			Role       string `yaml:"role"`
			Content    string `yaml:"content"`
			QuestionID *int   `yaml:"question_id"`
		} `yaml:"messages"`
	} `yaml:"conversations"`
}

// TestStorySectionData represents section data for E2E tests
type TestStorySectionData struct {
	ID            int    `json:"id"`
	StoryID       int    `json:"story_id"`
	SectionNumber int    `json:"section_number"`
	Content       string `json:"content"`
	LanguageLevel string `json:"language_level"`
	WordCount     int    `json:"word_count"`
	GeneratedBy   string `json:"generated_by"`
}

// TestStoryData represents story data for E2E tests
type TestStoryData struct {
	ID       int                    `json:"id"`
	Username string                 `json:"username"`
	Title    string                 `json:"title"`
	Status   string                 `json:"status"`
	Sections []TestStorySectionData `json:"sections"`
}

// TestStories represents a collection of test stories
type TestStories struct {
	Stories []struct {
		Username              string  `yaml:"username"`
		Title                 string  `yaml:"title"`
		Language              string  `yaml:"language"`
		Subject               *string `yaml:"subject"`
		AuthorStyle           *string `yaml:"author_style"`
		TimePeriod            *string `yaml:"time_period"`
		Genre                 *string `yaml:"genre"`
		Tone                  *string `yaml:"tone"`
		CharacterNames        *string `yaml:"character_names"`
		CustomInstructions    *string `yaml:"custom_instructions"`
		SectionLengthOverride *string `yaml:"section_length_override"`
		Status                string  `yaml:"status"`
		IsCurrent             bool    `yaml:"is_current"`
		Sections              []struct {
			SectionNumber int    `yaml:"section_number"`
			Content       string `yaml:"content"`
			LanguageLevel string `yaml:"language_level"`
			WordCount     int    `yaml:"word_count"`
			GeneratedBy   string `yaml:"generated_by"`
			Questions     []struct {
				QuestionText       string   `yaml:"question_text"`
				Options            []string `yaml:"options"`
				CorrectAnswerIndex int      `yaml:"correct_answer_index"`
				Explanation        *string  `yaml:"explanation"`
			} `yaml:"questions"`
		} `yaml:"sections"`
	} `yaml:"stories"`
}

// TestSnippetData represents snippet data for E2E tests
type TestSnippetData struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	OriginalText   string `json:"original_text"`
	TranslatedText string `json:"translated_text"`
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
}

// TestSnippets represents a collection of test snippets
type TestSnippets struct {
	Snippets []struct {
		Username        string  `yaml:"username"`
		OriginalText    string  `yaml:"original_text"`
		TranslatedText  string  `yaml:"translated_text"`
		SourceLanguage  string  `yaml:"source_language"`
		TargetLanguage  string  `yaml:"target_language"`
		Context         *string `yaml:"context"`
		DifficultyLevel string  `yaml:"difficulty_level"`
	} `yaml:"snippets"`
}

// TestFeedbackData represents feedback data for E2E tests
type TestFeedbackData struct {
	ID           int                    `json:"id"`
	Username     string                 `json:"username"`
	FeedbackText string                 `json:"feedback_text"`
	FeedbackType string                 `json:"feedback_type"`
	Status       string                 `json:"status"`
	ContextData  map[string]interface{} `json:"context_data"`
}

// TestFeedback represents a collection of test feedback
type TestFeedback struct {
	FeedbackReports []struct {
		Username     string                 `yaml:"username"`
		FeedbackText string                 `yaml:"feedback_text"`
		FeedbackType string                 `yaml:"feedback_type"`
		Status       string                 `yaml:"status"`
		ContextData  map[string]interface{} `yaml:"context_data"`
	} `yaml:"feedback_reports"`
}

func resetTestDatabase(databaseURL, testDB string, logger *observability.Logger) error {
	ctx := context.Background()

	// Create admin connection string by replacing the database name with 'postgres'
	// This connects to the admin database to drop/create the test database
	adminConnStr := strings.Replace(databaseURL, "/"+testDB+"?", "/postgres?", 1)
	if !strings.Contains(adminConnStr, "/postgres?") {
		// Handle case where there's no query string
		adminConnStr = strings.Replace(databaseURL, "/"+testDB, "/postgres", 1)
	}

	logger.Info(ctx, "Connecting to admin database", map[string]interface{}{"connection_string": adminConnStr})
	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return contextutils.WrapErrorf(contextutils.ErrDatabaseConnection, "failed to connect to postgres database for drop/create: %v", err)
	}
	defer func() {
		if err := adminDB.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close adminDB", map[string]interface{}{"error": err.Error()})
		}
	}()

	logger.Info(ctx, "Terminating connections to test DB", map[string]interface{}{"database": testDB})
	_, err = adminDB.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = '%s' AND pid <> pg_backend_pid();
	`, testDB))
	if err != nil {
		logger.Warn(ctx, "Warning: failed to terminate connections", map[string]interface{}{"error": err.Error()})
	}

	logger.Info(ctx, "Dropping test database", map[string]interface{}{"database": testDB})
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", testDB))
	if err != nil {
		return contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to drop test database: %v", err)
	}
	logger.Info(ctx, "Successfully dropped test database", map[string]interface{}{"database": testDB})

	logger.Info(ctx, "Creating test database", map[string]interface{}{"database": testDB})
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDB))
	if err != nil {
		return contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to create test database: %v", err)
	}
	logger.Info(ctx, "Successfully created test database", map[string]interface{}{"database": testDB})

	logger.Info(ctx, "Test database reset complete")
	return nil
}

func main() {
	ctx := context.Background()

	// CLI flags
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	// Load configuration first
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup observability (tracing/metrics). Suppress logger creation here to avoid startup noise.
	originalLogging := cfg.OpenTelemetry.EnableLogging
	cfg.OpenTelemetry.EnableLogging = false
	tp, mp, _, err := observability.SetupObservability(&cfg.OpenTelemetry, "setup-test-db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	// Create logger with level based on --verbose flag
	logLevel := zapcore.WarnLevel
	if *verbose {
		logLevel = zapcore.InfoLevel
	}
	// Restore config flag for logger construction (to allow OTLP exporter if enabled)
	cfg.OpenTelemetry.EnableLogging = originalLogging
	logger := observability.NewLoggerWithLevel(&cfg.OpenTelemetry, logLevel)
	defer func() {
		if tp != nil {
			if err := tp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down tracer provider", map[string]interface{}{"error": err.Error()})
			}
		}
		if mp != nil {
			if err := mp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down meter provider", map[string]interface{}{"error": err.Error()})
			}
		}
	}()

	// Get DB connection info from env or use defaults
	dbUser := "quiz_user"
	dbPassword := "quiz_password"
	dbHost := "localhost"
	dbPort := "5433"
	testDB := "quiz_test_db"

	// Allow override from DATABASE_URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, testDB)
	}

	// Debug: Print the DATABASE_URL we're using
	logger.Info(ctx, "DATABASE_URL from environment", map[string]interface{}{"database_url": os.Getenv("DATABASE_URL")})
	logger.Info(ctx, "Using database URL", map[string]interface{}{"database_url": databaseURL})

	// --- Drop and recreate the test database ---
	if err := resetTestDatabase(databaseURL, testDB, logger); err != nil {
		logger.Error(ctx, "Failed to reset test database", err)
		os.Exit(1)
	}

	// Now connect to the new test database
	logger.Info(ctx, "Connecting to database", map[string]interface{}{"database_url": databaseURL})

	// Initialize database manager with logger
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(databaseURL)
	if err != nil {
		logger.Error(ctx, "Failed to initialize database", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close database", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Get the root directory (backend is the working directory)
	rootDir, err := os.Getwd()
	if err != nil {
		logger.Error(ctx, "Failed to get working directory", err)
		os.Exit(1)
	}

	// Apply schema from schema.sql
	schemaPath := filepath.Join(rootDir, "..", "schema.sql")
	if err := applySchema(db, schemaPath, rootDir, logger); err != nil {
		logger.Error(ctx, "Failed to apply schema", err)
		os.Exit(1)
	}

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	// Create question service
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	// Ensure admin user exists
	if err := userService.EnsureAdminUserExists(ctx, "admin", "password"); err != nil {
		logger.Error(ctx, "Failed to ensure admin user exists", err)
		os.Exit(1)
	}

	// Load and insert test data
	users, err := setupTestData(ctx, rootDir, userService, questionService, learningService, db, logger)
	if err != nil {
		logger.Error(ctx, "Failed to setup test data", err)
		os.Exit(1)
	}

	// Output user data to JSON file for E2E tests
	if err := outputUserDataForTests(users, rootDir, logger); err != nil {
		logger.Error(ctx, "Failed to output user data for tests", err)
		os.Exit(1)
	}

	// Output roles data to JSON file for E2E tests
	if err := outputRolesDataForTests(db, rootDir, logger); err != nil {
		logger.Error(ctx, "Failed to output roles data for tests", err)
		os.Exit(1)
	}

	logger.Info(ctx, "Test database created successfully")
}

func applySchema(db *sql.DB, schemaPath, _ string, logger *observability.Logger) error {
	ctx := context.Background()

	// Apply the schema (database is already empty after resetTestDatabase)
	logger.Info(ctx, "Applying schema")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to execute schema: %w", err)
	}

	// Priority system tables are already included in the main schema.sql
	// No additional migration needed
	logger.Info(ctx, "Priority system tables already included in main schema")

	return nil
}

func setupTestData(ctx context.Context, rootDir string, userService *services.UserService, questionService *services.QuestionService, learningService *services.LearningService, db *sql.DB, logger *observability.Logger) (map[string]*models.User, error) {
	dataDir := filepath.Join(rootDir, "data")

	// 1. Load and create users
	users, err := loadAndCreateUsers(ctx, filepath.Join(dataDir, "test_users.yaml"), userService, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup users: %w", err)
	}

	// 2. Load and create questions
	questions, err := loadAndCreateQuestions(ctx, filepath.Join(dataDir, "test_questions.yaml"), questionService, users, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup questions: %w", err)
	}

	// 3. Load and create user responses
	if err := loadAndCreateResponses(ctx, filepath.Join(dataDir, "test_responses.yaml"), users, questions, learningService, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup responses: %w", err)
	}

	// 4. Load and create question reports
	if err := loadAndCreateQuestionReports(ctx, filepath.Join(dataDir, "test_responses.yaml"), users, questions, db, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup question reports: %w", err)
	}

	// 5. Load and create analytics data
	if err := loadAndCreateAnalytics(ctx, filepath.Join(dataDir, "test_analytics.yaml"), users, questions, learningService, db, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup analytics: %w", err)
	}

	// 6. Load and create daily assignments
	if err := loadAndCreateDailyAssignments(ctx, filepath.Join(dataDir, "test_daily_assignments.yaml"), users, questions, db, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup daily assignments: %w", err)
	}

	// 7. Load and create stories
	stories, err := loadAndCreateStories(ctx, filepath.Join(dataDir, "test_stories.yaml"), users, db, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup stories: %w", err)
	}

	// Output story data for E2E tests
	if err := outputStoryDataForTests(stories, rootDir, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to output story data: %w", err)
	}

	// 8. Load and create snippets
	snippets, err := loadAndCreateSnippets(ctx, filepath.Join(dataDir, "test_snippets.yaml"), users, db, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup snippets: %w", err)
	}

	// Output snippet data for E2E tests
	if err := outputSnippetDataForTests(snippets, rootDir, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to output snippet data: %w", err)
	}

	// 9. Load and create conversations
	conversations, err := loadAndCreateConversations(ctx, filepath.Join(dataDir, "test_conversations.yaml"), users, db, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup conversations: %w", err)
	}

	// Output conversation data for E2E tests
	if err := outputConversationDataForTests(conversations, rootDir, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to output conversation data: %w", err)
	}

	// 10. Load and create feedback reports
	feedback, err := loadAndCreateFeedback(ctx, filepath.Join(dataDir, "test_feedback.yaml"), users, db, logger)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup feedback: %w", err)
	}

	// Output feedback data for E2E tests
	if err := outputFeedbackDataForTests(feedback, rootDir, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to output feedback data: %w", err)
	}

	// 11. Create API Keys for test users
	if err := createAndOutputAPIKeysForTests(ctx, users, db, rootDir, logger); err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to setup api keys: %w", err)
	}

	return users, nil
}

// TestAPIKeyData represents API key data for E2E tests (non-sensitive)
type TestAPIKeyData struct {
	ID              int       `json:"id"`
	Username        string    `json:"username"`
	KeyName         string    `json:"key_name"`
	KeyPrefix       string    `json:"key_prefix"`
	PermissionLevel string    `json:"permission_level"`
	CreatedAt       time.Time `json:"created_at"`
}

// createAndOutputAPIKeysForTests creates API keys for selected users and writes a JSON artifact for tests
func createAndOutputAPIKeysForTests(ctx context.Context, users map[string]*models.User, db *sql.DB, rootDir string, logger *observability.Logger) error {
	// Initialize service
	apiKeyService := services.NewAuthAPIKeyService(db, logger)

	// Strategy:
	// - apitestuser: 2 keys (readonly, full)
	// - apitestadmin: 2 keys (readonly, full)
	// - others: 1 readonly key

	// Helper to create a key and capture minimal info
	create := func(username string, userID int, keyName, perm string) (*TestAPIKeyData, error) {
		key, _, err := apiKeyService.CreateAPIKey(ctx, userID, keyName, perm)
		if err != nil {
			return nil, err
		}
		return &TestAPIKeyData{
			ID:              key.ID,
			Username:        username,
			KeyName:         key.KeyName,
			KeyPrefix:       key.KeyPrefix,
			PermissionLevel: key.PermissionLevel,
			CreatedAt:       key.CreatedAt,
		}, nil
	}

	apiKeys := make(map[string]TestAPIKeyData)

	for username, user := range users {
		if username == "apitestuser" || username == "apitestadmin" {
			if d, err := create(username, user.ID, "test_key_readonly", string(models.PermissionLevelReadonly)); err == nil {
				apiKeys[fmt.Sprintf("%s_ro", username)] = *d
			} else {
				return contextutils.WrapErrorf(err, "failed creating readonly api key for %s", username)
			}
			if d, err := create(username, user.ID, "test_key_full", string(models.PermissionLevelFull)); err == nil {
				apiKeys[fmt.Sprintf("%s_full", username)] = *d
			} else {
				return contextutils.WrapErrorf(err, "failed creating full api key for %s", username)
			}
		} else {
			if d, err := create(username, user.ID, "test_key_readonly", string(models.PermissionLevelReadonly)); err == nil {
				apiKeys[fmt.Sprintf("%s_ro", username)] = *d
			} else {
				return contextutils.WrapErrorf(err, "failed creating readonly api key for %s", username)
			}
		}
	}

	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-api-keys.json")
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}
	jsonData, err := json.MarshalIndent(apiKeys, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal api keys data to JSON")
	}
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write api keys data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output API keys data for E2E tests", map[string]interface{}{
		"file_path":  outputPath,
		"keys_count": len(apiKeys),
	})

	return nil
}

func loadAndCreateUsers(ctx context.Context, filePath string, userService *services.UserService, logger *observability.Logger) (result0 map[string]*models.User, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var testUsers TestUsers
	if err := yaml.Unmarshal(data, &testUsers); err != nil {
		return nil, err
	}

	users := make(map[string]*models.User)
	for _, testUser := range testUsers.Users {
		// Create user with email and timezone
		user, err := userService.CreateUserWithEmailAndTimezone(
			ctx,
			testUser.Username,
			testUser.Email,
			"UTC", // Default timezone for test users
			testUser.PreferredLanguage,
			testUser.CurrentLevel,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to create user %s", testUser.Username)
		}

		// Set password separately since CreateUserWithEmailAndTimezone doesn't set password
		if err := userService.UpdateUserPassword(ctx, user.ID, testUser.Password); err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to set password for user %s", testUser.Username)
		}

		// Update additional settings
		settings := &models.UserSettings{
			Language:   testUser.PreferredLanguage,
			Level:      testUser.CurrentLevel,
			AIProvider: testUser.AIProvider,
			AIModel:    testUser.AIModel,
			AIAPIKey:   testUser.AIAPIKey,
			AIEnabled:  testUser.AIProvider != "", // Enable AI if provider is set
		}

		if err := userService.UpdateUserSettings(ctx, user.ID, settings); err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to update settings for user %s", testUser.Username)
		}

		// Assign roles from YAML configuration
		for _, roleName := range testUser.Roles {
			err = userService.AssignRoleByName(ctx, user.ID, roleName)
			if err != nil {
				logger.Warn(ctx, "Failed to assign role to user", map[string]interface{}{
					"username": testUser.Username,
					"role":     roleName,
					"error":    err.Error(),
				})
			} else {
				logger.Info(ctx, "Assigned role to user", map[string]interface{}{
					"username": testUser.Username,
					"role":     roleName,
					"user_id":  user.ID,
				})
			}
		}

		users[testUser.Username] = user
	}

	return users, nil
}

func loadAndCreateQuestions(ctx context.Context, filePath string, questionService *services.QuestionService, users map[string]*models.User, _ *observability.Logger) (result0 []*models.Question, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var testQuestions TestQuestions
	if err := yaml.Unmarshal(data, &testQuestions); err != nil {
		return nil, err
	}

	var questions []*models.Question
	for i, question := range testQuestions.Questions {
		// Set the created time since it's not in YAML
		question.CreatedAt = time.Now()

		// Get the users this question should be assigned to
		questionUsers := question.Users
		var assignedUserIDs []int
		if len(questionUsers) == 0 {
			// Fallback to round-robin if no users specified
			for _, user := range users {
				assignedUserIDs = append(assignedUserIDs, user.ID)
			}
			if len(assignedUserIDs) == 0 {
				return nil, contextutils.ErrorWithContextf("no users available to assign questions to")
			}
			// Assign to one user in round-robin
			assignedUserIDs = []int{assignedUserIDs[i%len(assignedUserIDs)]}
		} else {
			for _, username := range questionUsers {
				user, exists := users[username]
				if !exists {
					return nil, contextutils.ErrorWithContextf("user not found: %s", username)
				}
				assignedUserIDs = append(assignedUserIDs, user.ID)
			}
		}

		if err := questionService.SaveQuestion(ctx, &question); err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to save question %d", i)
		}

		for _, userID := range assignedUserIDs {
			if err := questionService.AssignQuestionToUser(ctx, question.ID, userID); err != nil {
				return nil, contextutils.WrapErrorf(err, "failed to assign question %d to user %d", question.ID, userID)
			}
		}

		questions = append(questions, &question)
	}

	return questions, nil
}

func loadAndCreateResponses(_ context.Context, filePath string, users map[string]*models.User, questions []*models.Question, learningService *services.LearningService, _ *observability.Logger) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var testResponses TestResponses
	if err := yaml.Unmarshal(data, &testResponses); err != nil {
		return err
	}

	for i, responseData := range testResponses.UserResponses {
		user, exists := users[responseData.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found: %s", responseData.Username)
		}

		if responseData.QuestionIndex >= len(questions) {
			return contextutils.ErrorWithContextf("question index out of range: %d", responseData.QuestionIndex)
		}

		question := questions[responseData.QuestionIndex]

		// Use RecordAnswerWithPriority to ensure priority scores are calculated
		if err := learningService.RecordAnswerWithPriority(
			context.Background(),
			user.ID,
			question.ID,
			0, // Use index 0 for test data
			responseData.IsCorrect,
			responseData.ResponseTimeMs,
		); err != nil {
			return contextutils.WrapErrorf(err, "failed to record response %d", i)
		}

	}

	return nil
}

func loadAndCreateQuestionReports(_ context.Context, filePath string, users map[string]*models.User, questions []*models.Question, db *sql.DB, _ *observability.Logger) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return contextutils.WrapError(err, "failed to read responses file")
	}

	var testResponses TestResponses
	if err := yaml.Unmarshal(data, &testResponses); err != nil {
		return contextutils.WrapError(err, "failed to parse responses data")
	}

	// Load question reports
	for i, reportData := range testResponses.QuestionReports {
		user, exists := users[reportData.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found for question report: %s", reportData.Username)
		}

		if reportData.QuestionIndex >= len(questions) {
			return contextutils.ErrorWithContextf("question index out of range for question report: %d", reportData.QuestionIndex)
		}

		question := questions[reportData.QuestionIndex]

		// Parse the timestamp if provided, otherwise use current time
		var createdAt time.Time
		if reportData.CreatedAt != nil {
			var err error
			createdAt, err = time.Parse(time.RFC3339, *reportData.CreatedAt)
			if err != nil {
				return contextutils.ErrorWithContextf("invalid timestamp format for question report: %s", *reportData.CreatedAt)
			}
		} else {
			createdAt = time.Now()
		}

		// Insert question report directly into database
		_, err := db.Exec(`
			INSERT INTO question_reports (question_id, reported_by_user_id, report_reason, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (question_id, reported_by_user_id) DO NOTHING
		`, question.ID, user.ID, reportData.ReportReason, createdAt)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to insert question report %d", i)
		}
	}

	return nil
}

func loadAndCreateAnalytics(ctx context.Context, filePath string, users map[string]*models.User, questions []*models.Question, learningService *services.LearningService, db *sql.DB, logger *observability.Logger) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Analytics file is optional, so just return if it doesn't exist
		logger.Warn(ctx, "Analytics file not found", map[string]interface{}{"file_path": filePath})
		return nil
	}

	var testAnalytics TestAnalytics
	if err := yaml.Unmarshal(data, &testAnalytics); err != nil {
		return contextutils.WrapError(err, "failed to parse analytics data")
	}

	// Load priority scores
	for _, priorityData := range testAnalytics.PriorityScores {
		user, exists := users[priorityData.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found for priority score: %s", priorityData.Username)
		}

		if priorityData.QuestionIndex >= len(questions) {
			return contextutils.ErrorWithContextf("question index out of range for priority score: %d", priorityData.QuestionIndex)
		}

		question := questions[priorityData.QuestionIndex]

		// Parse the timestamp
		lastCalculatedAt, err := time.Parse(time.RFC3339, priorityData.LastCalculatedAt)
		if err != nil {
			return contextutils.ErrorWithContextf("invalid timestamp format for priority score: %s", priorityData.LastCalculatedAt)
		}

		// Insert priority score directly into database
		_, err = db.Exec(`
			INSERT INTO question_priority_scores (user_id, question_id, priority_score, last_calculated_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (user_id, question_id) DO UPDATE SET
				priority_score = EXCLUDED.priority_score,
				last_calculated_at = EXCLUDED.last_calculated_at,
				updated_at = NOW()
		`, user.ID, question.ID, priorityData.PriorityScore, lastCalculatedAt)
		if err != nil {
			return contextutils.WrapError(err, "failed to insert priority score")
		}

	}

	// Load learning preferences
	for _, prefData := range testAnalytics.LearningPreferences {
		user, exists := users[prefData.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found for learning preferences: %s", prefData.Username)
		}

		// Ensure daily_goal is present and valid. The schema enforces daily_goal > 0
		// so default to the service's default if not provided or invalid.
		dailyGoal := 0
		// Try to parse a daily_goal field if it exists in the YAML by checking for a map
		// fallback: the YAML struct doesn't include daily_goal currently; use default
		// from the LearningService defaults.
		// We'll fetch defaults from service to avoid duplicating magic numbers.
		defaultPrefs := learningService.GetDefaultLearningPreferences()
		if dailyGoal <= 0 {
			dailyGoal = defaultPrefs.DailyGoal
		}

		prefs := &models.UserLearningPreferences{
			UserID:               user.ID,
			FocusOnWeakAreas:     prefData.FocusOnWeakAreas,
			FreshQuestionRatio:   prefData.FreshQuestionRatio,
			WeakAreaBoost:        prefData.WeakAreaBoost,
			KnownQuestionPenalty: prefData.KnownQuestionPenalty,
			ReviewIntervalDays:   prefData.ReviewIntervalDays,
			DailyReminderEnabled: prefData.DailyReminderEnabled,
			DailyGoal:            dailyGoal,
		}

		if _, err := learningService.UpdateUserLearningPreferences(ctx, user.ID, prefs); err != nil {
			return contextutils.WrapErrorf(err, "failed to update learning preferences for user %s", prefData.Username)
		}

	}

	// Load performance metrics
	for _, metricData := range testAnalytics.PerformanceMetrics {
		user, exists := users[metricData.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found for performance metrics: %s", metricData.Username)
		}

		// Insert performance metric directly into database
		_, err := db.Exec(`
			INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms, last_updated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			ON CONFLICT (user_id, topic, language, level) DO UPDATE SET
				total_attempts = EXCLUDED.total_attempts,
				correct_attempts = EXCLUDED.correct_attempts,
				average_response_time_ms = EXCLUDED.average_response_time_ms,
				last_updated = NOW()
		`, user.ID, metricData.Topic, metricData.Language, metricData.Level,
			metricData.TotalAttempts, metricData.CorrectAttempts, metricData.AverageResponseTimeMs)
		if err != nil {
			return contextutils.WrapError(err, "failed to insert performance metric")
		}

	}

	// Load user question metadata (marked as known)
	for _, metadata := range testAnalytics.UserQuestionMetadata {
		user, exists := users[metadata.Username]
		if !exists {
			return contextutils.ErrorWithContextf("user not found for question metadata: %s", metadata.Username)
		}

		if metadata.QuestionIndex >= len(questions) {
			return contextutils.ErrorWithContextf("question index out of range for metadata: %d", metadata.QuestionIndex)
		}

		question := questions[metadata.QuestionIndex]

		if metadata.MarkedAsKnown {
			var markedAt time.Time
			if metadata.MarkedAsKnownAt != nil {
				var err error
				markedAt, err = time.Parse(time.RFC3339, *metadata.MarkedAsKnownAt)
				if err != nil {
					return contextutils.ErrorWithContextf("invalid timestamp format for marked as known: %s", *metadata.MarkedAsKnownAt)
				}
			} else {
				markedAt = time.Now()
			}

			// Insert into user_question_metadata table
			_, err := db.Exec(`
				INSERT INTO user_question_metadata (user_id, question_id, marked_as_known, marked_as_known_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
				ON CONFLICT (user_id, question_id) DO UPDATE SET
					marked_as_known = EXCLUDED.marked_as_known,
					marked_as_known_at = EXCLUDED.marked_as_known_at,
					updated_at = NOW()
			`, user.ID, question.ID, metadata.MarkedAsKnown, markedAt)
			if err != nil {
				return contextutils.WrapError(err, "failed to insert question metadata")
			}

		}
	}

	return nil
}

func loadAndCreateDailyAssignments(ctx context.Context, filePath string, users map[string]*models.User, questions []*models.Question, db *sql.DB, logger *observability.Logger) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist, skip daily assignments
		logger.Info(ctx, "Daily assignments file not found, skipping", map[string]interface{}{
			"file_path": filePath,
		})
		return nil
	}

	var testDailyAssignments TestDailyAssignments
	if err := yaml.Unmarshal(data, &testDailyAssignments); err != nil {
		return err
	}

	for _, assignmentData := range testDailyAssignments.DailyAssignments {
		user, exists := users[assignmentData.Username]
		if !exists {
			logger.Warn(ctx, "User not found for daily assignment", map[string]interface{}{
				"username": assignmentData.Username,
			})
			continue
		}

		// Parse the date
		date, err := time.Parse("2006-01-02", assignmentData.Date)
		if err != nil {
			logger.Warn(ctx, "Invalid date format for daily assignment", map[string]interface{}{
				"username": assignmentData.Username,
				"date":     assignmentData.Date,
			})
			continue
		}

		// Create a map of completed questions for quick lookup
		completedQuestions := make(map[int]bool)
		for _, qID := range assignmentData.CompletedQuestions {
			completedQuestions[qID] = true
		}

		// Assign questions to the user for the specific date
		for _, questionID := range assignmentData.QuestionIDs {
			// Check if question exists
			if questionID <= 0 || questionID > len(questions) {
				logger.Warn(ctx, "Question ID out of range for daily assignment", map[string]interface{}{
					"username":    assignmentData.Username,
					"date":        assignmentData.Date,
					"question_id": questionID,
				})
				continue
			}

			question := questions[questionID-1] // Convert to 0-based index

			// Ensure we don't violate unique constraint by removing any existing assignment for the same
			// (user_id, question_id, assignment_date) tuple before inserting. This avoids relying on
			// ON CONFLICT which requires the constraint to be present in some test DB states.
			deleteQuery := `DELETE FROM daily_question_assignments WHERE user_id = $1 AND question_id = $2 AND assignment_date = $3`
			if _, err := db.ExecContext(ctx, deleteQuery, user.ID, question.ID, date); err != nil {
				logger.Error(ctx, "Failed to delete existing daily assignment", err, map[string]interface{}{
					"username":    assignmentData.Username,
					"date":        assignmentData.Date,
					"question_id": questionID,
				})
				return contextutils.WrapErrorf(err, "failed to delete existing daily assignment for user %s, question %d", assignmentData.Username, questionID)
			}

			// Insert the assignment directly into the database
			query := `
				INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, is_completed, completed_at)
				VALUES ($1, $2, $3, $4, $5)
			`

			isCompleted := completedQuestions[questionID]
			var completedAt *time.Time
			if isCompleted {
				now := time.Now()
				completedAt = &now
			}

			if _, err := db.ExecContext(ctx, query, user.ID, question.ID, date, isCompleted, completedAt); err != nil {
				logger.Error(ctx, "Failed to create daily assignment", err, map[string]interface{}{
					"username":    assignmentData.Username,
					"date":        assignmentData.Date,
					"question_id": questionID,
				})
				return contextutils.WrapErrorf(err, "failed to create daily assignment for user %s, question %d", assignmentData.Username, questionID)
			}
		}

		logger.Info(ctx, "Created daily assignments", map[string]interface{}{
			"username": assignmentData.Username,
			"date":     assignmentData.Date,
			"count":    len(assignmentData.QuestionIDs),
		})
	}

	return nil
}

func loadAndCreateStories(ctx context.Context, filePath string, users map[string]*models.User, db *sql.DB, logger *observability.Logger) (map[string]TestStoryData, error) {
	stories := make(map[string]TestStoryData)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Stories file is optional, so just return if it doesn't exist
		logger.Info(ctx, "Stories file not found, skipping", map[string]interface{}{
			"file_path": filePath,
		})
		return stories, nil
	}

	var testStories TestStories
	if err := yaml.Unmarshal(data, &testStories); err != nil {
		return stories, contextutils.WrapError(err, "failed to parse stories data")
	}

	for i, storyData := range testStories.Stories {
		user, exists := users[storyData.Username]
		if !exists {
			return stories, contextutils.ErrorWithContextf("user not found for story: %s", storyData.Username)
		}

		// Parse section length override if provided
		var sectionLengthOverride *models.SectionLength
		if storyData.SectionLengthOverride != nil {
			switch *storyData.SectionLengthOverride {
			case "short":
				sl := models.SectionLengthShort
				sectionLengthOverride = &sl
			case "medium":
				sl := models.SectionLengthMedium
				sectionLengthOverride = &sl
			case "long":
				sl := models.SectionLengthLong
				sectionLengthOverride = &sl
			}
		}

		// Create story
		story := &models.Story{
			UserID:                uint(user.ID),
			Title:                 storyData.Title,
			Language:              storyData.Language,
			Subject:               storyData.Subject,
			AuthorStyle:           storyData.AuthorStyle,
			TimePeriod:            storyData.TimePeriod,
			Genre:                 storyData.Genre,
			Tone:                  storyData.Tone,
			CharacterNames:        storyData.CharacterNames,
			CustomInstructions:    storyData.CustomInstructions,
			SectionLengthOverride: sectionLengthOverride,
			Status:                models.StoryStatus(storyData.Status),
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}

		// Insert story directly into database
		_, err := db.Exec(`
			INSERT INTO stories (user_id, title, language, subject, author_style, time_period, genre, tone,
			                     character_names, custom_instructions, section_length_override, status,
			                     created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, story.UserID, story.Title, story.Language, story.Subject, story.AuthorStyle, story.TimePeriod,
			story.Genre, story.Tone, story.CharacterNames, story.CustomInstructions, story.SectionLengthOverride,
			string(story.Status), story.CreatedAt, story.UpdatedAt)
		if err != nil {
			return stories, contextutils.WrapErrorf(err, "failed to insert story %d", i)
		}

		// Get the story ID (we need to query it back since we don't have RETURNING)
		var storyID int
		err = db.QueryRow(`
			SELECT id FROM stories WHERE user_id = $1 AND title = $2 ORDER BY created_at DESC LIMIT 1
		`, story.UserID, story.Title).Scan(&storyID)
		if err != nil {
			return stories, contextutils.WrapErrorf(err, "failed to get story ID for story %d", i)
		}

		// Initialize story data for test output
		storyKey := fmt.Sprintf("%s_%s", storyData.Username, storyData.Title)
		storyDataForOutput := TestStoryData{
			ID:       storyID,
			Username: storyData.Username,
			Title:    storyData.Title,
			Status:   storyData.Status,
			Sections: []TestStorySectionData{},
		}

		// Create sections for this story
		for j, sectionData := range storyData.Sections {
			section := &models.StorySection{
				StoryID:        uint(storyID),
				SectionNumber:  sectionData.SectionNumber,
				Content:        sectionData.Content,
				LanguageLevel:  sectionData.LanguageLevel,
				WordCount:      sectionData.WordCount,
				GeneratedBy:    models.GeneratorType(sectionData.GeneratedBy),
				GeneratedAt:    time.Now(),
				GenerationDate: time.Now(),
			}

			// Insert section
			_, err := db.Exec(`
				INSERT INTO story_sections (story_id, section_number, content, language_level, word_count,
				                           generated_by, generated_at, generation_date)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, section.StoryID, section.SectionNumber, section.Content, section.LanguageLevel,
				section.WordCount, string(section.GeneratedBy), section.GeneratedAt, section.GenerationDate)
			if err != nil {
				return stories, contextutils.WrapErrorf(err, "failed to insert section %d for story %d", j, i)
			}

			// Get the section ID
			var sectionID int
			err = db.QueryRow(`
				SELECT id FROM story_sections WHERE story_id = $1 AND section_number = $2
			`, section.StoryID, section.SectionNumber).Scan(&sectionID)
			if err != nil {
				return stories, contextutils.WrapErrorf(err, "failed to get section ID for section %d of story %d", j, i)
			}

			// Add section data to story data for test output
			sectionDataForOutput := TestStorySectionData{
				ID:            sectionID,
				StoryID:       storyID,
				SectionNumber: section.SectionNumber,
				Content:       section.Content,
				LanguageLevel: section.LanguageLevel,
				WordCount:     section.WordCount,
				GeneratedBy:   string(section.GeneratedBy),
			}
			storyDataForOutput.Sections = append(storyDataForOutput.Sections, sectionDataForOutput)

			// Create questions for this section
			for k, questionData := range sectionData.Questions {
				question := &models.StorySectionQuestion{
					SectionID:          uint(sectionID),
					QuestionText:       questionData.QuestionText,
					Options:            questionData.Options,
					CorrectAnswerIndex: questionData.CorrectAnswerIndex,
					Explanation:        questionData.Explanation,
					CreatedAt:          time.Now(),
				}

				// Convert options to JSON for database storage
				optionsJSON, err := json.Marshal(question.Options)
				if err != nil {
					return stories, contextutils.WrapErrorf(err, "failed to marshal options for question %d for section %d of story %d", k, j, i)
				}

				// Insert question
				_, err = db.Exec(`
					INSERT INTO story_section_questions (section_id, question_text, options, correct_answer_index, explanation, created_at)
					VALUES ($1, $2, $3, $4, $5, $6)
				`, question.SectionID, question.QuestionText, optionsJSON, question.CorrectAnswerIndex,
					question.Explanation, question.CreatedAt)
				if err != nil {
					return stories, contextutils.WrapErrorf(err, "failed to insert question %d for section %d of story %d", k, j, i)
				}
			}
		}

		// Store story data for test output after all sections are created
		stories[storyKey] = storyDataForOutput

		logger.Info(ctx, "Created test story", map[string]interface{}{
			"username": storyData.Username,
			"title":    storyData.Title,
			"story_id": storyID,
		})
	}

	return stories, nil
}

// loadAndCreateSnippets loads and creates snippets from test data
func loadAndCreateSnippets(ctx context.Context, filePath string, users map[string]*models.User, db *sql.DB, logger *observability.Logger) (map[string]TestSnippetData, error) {
	snippets := make(map[string]TestSnippetData)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Snippets file is optional, so just return if it doesn't exist
		logger.Info(ctx, "Snippets file not found, skipping", map[string]interface{}{
			"file_path": filePath,
		})
		return snippets, nil
	}

	var testSnippets TestSnippets
	if err := yaml.Unmarshal(data, &testSnippets); err != nil {
		return snippets, contextutils.WrapError(err, "failed to parse snippets data")
	}

	// Create snippets service
	snippetsService := services.NewSnippetsService(db, nil, logger)

	for i, snippetData := range testSnippets.Snippets {
		user, exists := users[snippetData.Username]
		if !exists {
			return snippets, contextutils.ErrorWithContextf("user not found for snippet: %s", snippetData.Username)
		}

		// Create snippet request
		createReq := api.CreateSnippetRequest{
			OriginalText:   snippetData.OriginalText,
			TranslatedText: snippetData.TranslatedText,
			SourceLanguage: snippetData.SourceLanguage,
			TargetLanguage: snippetData.TargetLanguage,
			Context:        snippetData.Context,
		}

		// Create snippet using the service
		snippet, err := snippetsService.CreateSnippet(ctx, int64(user.ID), createReq)
		if err != nil {
			return snippets, contextutils.WrapErrorf(err, "failed to create snippet %d", i)
		}

		// Initialize snippet data for test output
		snippetKey := fmt.Sprintf("%s_%s_%s", snippetData.Username, snippetData.OriginalText, snippetData.SourceLanguage)
		snippets[snippetKey] = TestSnippetData{
			ID:             int(snippet.ID),
			Username:       snippetData.Username,
			OriginalText:   snippet.OriginalText,
			TranslatedText: snippet.TranslatedText,
			SourceLanguage: snippet.SourceLanguage,
			TargetLanguage: snippet.TargetLanguage,
		}

		logger.Info(ctx, "Created test snippet", map[string]interface{}{
			"username":      snippetData.Username,
			"original_text": snippetData.OriginalText,
			"snippet_id":    snippet.ID,
		})
	}

	return snippets, nil
}

// outputUserDataForTests outputs the created user data to a JSON file for E2E tests to read
func outputUserDataForTests(users map[string]*models.User, rootDir string, logger *observability.Logger) error {
	// Create a simplified structure for the E2E test
	type TestUserData struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	userData := make(map[string]TestUserData)
	for username, user := range users {
		userData[username] = TestUserData{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email.String,
		}
	}

	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-users.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(userData, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal user data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write user data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output user data for E2E tests", map[string]interface{}{
		"file_path":  outputPath,
		"user_count": len(userData),
	})

	return nil
}

// outputStoryDataForTests outputs the created story data to a JSON file for E2E tests to read
func outputStoryDataForTests(stories map[string]TestStoryData, rootDir string, logger *observability.Logger) error {
	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-stories.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(stories, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal stories data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write stories data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output stories data for E2E tests", map[string]interface{}{
		"file_path":     outputPath,
		"stories_count": len(stories),
	})

	return nil
}

// outputSnippetDataForTests outputs the created snippet data to a JSON file for E2E tests to read
func outputSnippetDataForTests(snippets map[string]TestSnippetData, rootDir string, logger *observability.Logger) error {
	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-snippets.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(snippets, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal snippets data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write snippets data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output snippets data for E2E tests", map[string]interface{}{
		"file_path":      outputPath,
		"snippets_count": len(snippets),
	})

	return nil
}

// outputRolesDataForTests outputs the created roles data to a JSON file for E2E tests to read
func outputRolesDataForTests(db *sql.DB, rootDir string, logger *observability.Logger) error {
	// Query all roles from the database
	rows, err := db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY id
	`)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to query roles from database")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Warn(context.Background(), "Warning: failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Create a simplified structure for the E2E test
	type TestRoleData struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	roleData := make(map[string]TestRoleData)
	for rows.Next() {
		var role models.Role
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to scan role data")
		}
		roleData[role.Name] = TestRoleData{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		}
	}

	if err := rows.Err(); err != nil {
		return contextutils.WrapErrorf(err, "error iterating over roles")
	}

	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-roles.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(roleData, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal roles data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write roles data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output roles data for E2E tests", map[string]interface{}{
		"file_path":   outputPath,
		"roles_count": len(roleData),
	})

	return nil
}

func loadAndCreateConversations(ctx context.Context, filePath string, users map[string]*models.User, db *sql.DB, logger *observability.Logger) (map[string]TestConversationData, error) {
	conversations := make(map[string]TestConversationData)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Conversations file is optional, so just return if it doesn't exist
		logger.Info(ctx, "Conversations file not found, skipping", map[string]interface{}{
			"file_path": filePath,
		})
		return conversations, nil
	}

	var testConversations TestConversations
	if err := yaml.Unmarshal(data, &testConversations); err != nil {
		return conversations, contextutils.WrapError(err, "failed to parse conversations data")
	}

	// Create conversation service
	conversationService := services.NewConversationService(db)

	for i, convData := range testConversations.Conversations {
		user, exists := users[convData.Username]
		if !exists {
			return conversations, contextutils.ErrorWithContextf("user not found for conversation: %s", convData.Username)
		}

		// Create conversation
		createReq := &api.CreateConversationRequest{
			Title: convData.Title,
		}

		conversation, err := conversationService.CreateConversation(ctx, uint(user.ID), createReq)
		if err != nil {
			return conversations, contextutils.WrapErrorf(err, "failed to create conversation %d", i)
		}

		// Store conversation data for test output (messages will be added below)
		convKey := fmt.Sprintf("%s_%s", convData.Username, convData.Title)
		conversations[convKey] = TestConversationData{
			ID:       conversation.Id.String(),
			Username: convData.Username,
			Title:    convData.Title,
			Messages: []TestMessageData{},
		}

		// Create messages for this conversation
		for j, msgData := range convData.Messages {
			content := struct {
				Text *string `json:"text,omitempty"`
			}{
				Text: &msgData.Content,
			}

			createMsgReq := &api.CreateMessageRequest{
				Content:    content,
				Role:       api.CreateMessageRequestRole(msgData.Role),
				QuestionId: msgData.QuestionID,
			}

			_, err := conversationService.AddMessage(ctx, conversation.Id.String(), uint(user.ID), createMsgReq)
			if err != nil {
				return conversations, contextutils.WrapErrorf(err, "failed to add message %d for conversation %d", j, i)
			}
		}

		// Now retrieve all messages for this conversation to get their actual data
		messages, err := conversationService.GetConversationMessages(ctx, conversation.Id.String(), uint(user.ID))
		if err != nil {
			return conversations, contextutils.WrapErrorf(err, "failed to get messages for conversation %d", i)
		}

		// Convert messages to our test data format
		var testMessages []TestMessageData
		for _, msg := range messages {
			testMsg := TestMessageData{
				ID:             msg.Id.String(),
				ConversationID: msg.ConversationId.String(),
				Role:           string(msg.Role),
				Bookmarked:     false, // Default value
				CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
				UpdatedAt:      msg.UpdatedAt.Format(time.RFC3339),
			}

			if msg.QuestionId != nil {
				testMsg.QuestionID = msg.QuestionId
			}

			if msg.Content.Text != nil {
				testMsg.Content = *msg.Content.Text
			}

			testMessages = append(testMessages, testMsg)
		}

		// Update the conversation with the actual messages
		conversations[convKey] = TestConversationData{
			ID:       conversation.Id.String(),
			Username: convData.Username,
			Title:    convData.Title,
			Messages: testMessages,
		}

		logger.Info(ctx, "Created test conversation", map[string]interface{}{
			"username":        convData.Username,
			"title":           convData.Title,
			"conversation_id": conversation.Id,
		})
	}

	return conversations, nil
}

// outputConversationDataForTests outputs the created conversation data to a JSON file for E2E tests to read
func outputConversationDataForTests(conversations map[string]TestConversationData, rootDir string, logger *observability.Logger) error {
	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-conversations.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(conversations, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal conversations data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write conversations data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output conversations data for E2E tests", map[string]interface{}{
		"file_path":           outputPath,
		"conversations_count": len(conversations),
	})

	return nil
}

// loadAndCreateFeedback loads and creates feedback reports from test data
func loadAndCreateFeedback(ctx context.Context, filePath string, users map[string]*models.User, db *sql.DB, logger *observability.Logger) (map[string]TestFeedbackData, error) {
	feedback := make(map[string]TestFeedbackData)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Feedback file is optional, so just return if it doesn't exist
		logger.Info(ctx, "Feedback file not found, skipping", map[string]interface{}{
			"file_path": filePath,
		})
		return feedback, nil
	}

	var testFeedback TestFeedback
	if err := yaml.Unmarshal(data, &testFeedback); err != nil {
		return feedback, contextutils.WrapError(err, "failed to parse feedback data")
	}

	for i, feedbackData := range testFeedback.FeedbackReports {
		user, exists := users[feedbackData.Username]
		if !exists {
			return feedback, contextutils.ErrorWithContextf("user not found for feedback: %s", feedbackData.Username)
		}

		// Default values
		feedbackType := feedbackData.FeedbackType
		if feedbackType == "" {
			feedbackType = "general"
		}
		status := feedbackData.Status
		if status == "" {
			status = "new"
		}

		// Marshal context_data to JSON
		contextJSON, err := json.Marshal(feedbackData.ContextData)
		if err != nil {
			return feedback, contextutils.WrapErrorf(err, "failed to marshal context_data for feedback %d", i)
		}

		// Insert feedback directly into database
		var feedbackID int
		err = db.QueryRow(`
			INSERT INTO feedback_reports (user_id, feedback_text, feedback_type, context_data, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			RETURNING id
		`, user.ID, feedbackData.FeedbackText, feedbackType, contextJSON, status).Scan(&feedbackID)
		if err != nil {
			return feedback, contextutils.WrapErrorf(err, "failed to insert feedback %d", i)
		}

		// Store feedback data for test output
		feedbackKey := fmt.Sprintf("%s_%d", feedbackData.Username, i)
		feedback[feedbackKey] = TestFeedbackData{
			ID:           feedbackID,
			Username:     feedbackData.Username,
			FeedbackText: feedbackData.FeedbackText,
			FeedbackType: feedbackType,
			Status:       status,
			ContextData:  feedbackData.ContextData,
		}

		logger.Info(ctx, "Created test feedback", map[string]interface{}{
			"username":      feedbackData.Username,
			"feedback_id":   feedbackID,
			"status":        status,
			"feedback_type": feedbackType,
		})
	}

	return feedback, nil
}

// outputFeedbackDataForTests outputs the created feedback data to a JSON file for E2E tests to read
func outputFeedbackDataForTests(feedback map[string]TestFeedbackData, rootDir string, logger *observability.Logger) error {
	// Write to JSON file in the frontend/tests directory
	outputPath := filepath.Join(rootDir, "..", "frontend", "tests", "test-feedback.json")

	// Ensure the directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return contextutils.WrapErrorf(err, "failed to create output directory: %s", outputDir)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to marshal feedback data to JSON")
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return contextutils.WrapErrorf(err, "failed to write feedback data to file: %s", outputPath)
	}

	logger.Info(context.Background(), "Output feedback data for E2E tests", map[string]interface{}{
		"file_path":      outputPath,
		"feedback_count": len(feedback),
	})

	return nil
}
