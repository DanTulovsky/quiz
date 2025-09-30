// Package main provides a CLI tool for running the worker to generate questions for a specific user.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/worker"
)

func main() {
	ctx := context.Background()
	// Define command line flags
	var (
		username     = flag.String("username", "", "Username to generate questions for (required)")
		level        = flag.String("level", "", "Override user's current level (optional)")
		language     = flag.String("language", "", "Override user's preferred language (optional)")
		questionType = flag.String("type", "vocabulary", "Question type: vocabulary, fill_blank, qa, reading_comprehension")
		topic        = flag.String("topic", "", "Specific topic for questions (optional)")
		count        = flag.Int("count", 5, "Number of questions to generate")
		aiProvider   = flag.String("ai-provider", "", "Override AI provider (optional)")
		aiModel      = flag.String("ai-model", "", "Override AI model (optional)")
		aiAPIKey     = flag.String("ai-api-key", "", "Override AI API key (optional)")
		help         = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		printUsage(nil)
		return
	}

	if *username == "" {
		fmt.Fprintln(os.Stderr, "Error: --username flag is required")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup observability (tracing/metrics/logging)
	tp, mp, logger, err := observability.SetupObservability(&cfg.OpenTelemetry, "quiz-cli-worker")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
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

	logger.Info(ctx, "Starting quiz CLI worker", map[string]interface{}{
		"username":      *username,
		"question_type": *questionType,
		"count":         *count,
	})

	// Validate question type
	validTypes := map[string]models.QuestionType{
		"vocabulary":            models.Vocabulary,
		"fill_blank":            models.FillInBlank,
		"qa":                    models.QuestionAnswer,
		"reading_comprehension": models.ReadingComprehension,
	}

	qType, valid := validTypes[strings.ToLower(*questionType)]
	if !valid {
		logger.Error(ctx, "Invalid question type", nil, map[string]interface{}{"question_type": *questionType})
		fmt.Fprintf(os.Stderr, "Error: Invalid question type '%s'\n", *questionType)
		os.Exit(1)
	}

	// Validate level if provided
	if *level != "" {
		if !isValidLevel(*level, cfg.GetAllLevels()) {
			logger.Error(ctx, "Invalid level", nil, map[string]interface{}{"level": *level})
			fmt.Fprintf(os.Stderr, "Error: Invalid level '%s'\n", *level)
			os.Exit(1)
		}
	}

	// Validate language if provided (use dynamic list from config)
	validLanguages := cfg.GetLanguages()
	if *language != "" {
		if !isValidLanguage(*language, validLanguages) {
			logger.Error(ctx, "Invalid language", nil, map[string]interface{}{"language": *language})
			fmt.Fprintf(os.Stderr, "Error: Invalid language '%s'\n", *language)
			os.Exit(1)
		}
	}

	// Initialize database manager with logger
	dbManager := database.NewManager(logger)

	// Initialize database connection with configuration
	db, err := dbManager.InitDBWithoutMigrations(cfg.Database)
	if err != nil {
		logger.Error(ctx, "Failed to connect to database", err, map[string]interface{}{"db_url": cfg.Database.URL})
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close database connection", map[string]interface{}{"error": err.Error(), "db_url": cfg.Database.URL})
		}
	}()

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	// Create question service
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Get user by username
	user, err := userService.GetUserByUsername(ctx, *username)
	if err != nil {
		logger.Error(ctx, "Failed to get user", err)
		fmt.Fprintf(os.Stderr, "Failed to get user: %v\n", err)
		os.Exit(1)
	}
	if user == nil {
		logger.Error(ctx, "User not found", nil, map[string]interface{}{"username": *username})
		fmt.Fprintf(os.Stderr, "User not found: %s\n", *username)
		os.Exit(1)
		return
	}
	logger.Info(ctx, "Found user", map[string]interface{}{"username": user.Username, "user_id": user.ID})

	// Apply AI overrides if provided
	if *aiProvider != "" {
		user.AIProvider.String = *aiProvider
		user.AIProvider.Valid = true
		user.AIEnabled.Bool = true
		user.AIEnabled.Valid = true
	}
	if *aiModel != "" {
		user.AIModel.String = *aiModel
		user.AIModel.Valid = true
	}
	if *aiAPIKey != "" {
		// Set AI provider and API key if provided
		if *aiProvider != "" && *aiAPIKey != "" {
			if err := userService.SetUserAPIKey(ctx, user.ID, *aiProvider, *aiAPIKey); err != nil {
				logger.Error(ctx, "Failed to set API key", err)
				fmt.Fprintf(os.Stderr, "Failed to set API key: %v\n", err)
				os.Exit(1)
			}
		} else if *aiAPIKey != "" {
			// If only API key is provided, use the user's current AI provider
			if err := userService.SetUserAPIKey(ctx, user.ID, user.AIProvider.String, *aiAPIKey); err != nil {
				logger.Error(ctx, "Failed to set API key", err)
				fmt.Fprintf(os.Stderr, "Failed to set API key: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Check if user has AI enabled (after potential overrides)
	if !user.AIEnabled.Valid || !user.AIEnabled.Bool {
		logger.Warn(ctx, "User does not have AI enabled", map[string]interface{}{"username": user.Username, "user_id": user.ID})
		logger.Info(ctx, "You may want to enable AI for this user first or use --ai-provider flag")
	}

	// Determine language and level to use
	languageToUse := user.PreferredLanguage.String
	if *language != "" {
		languageToUse = *language
	}

	levelToUse := user.CurrentLevel.String
	if *level != "" {
		levelToUse = *level
	}

	// Validate that we have required settings
	if languageToUse == "" {
		logger.Error(ctx, "No language specified", nil, map[string]interface{}{"username": user.Username, "user_id": user.ID})
		fmt.Fprintln(os.Stderr, "Error: No language specified. User has no preferred language and --language flag not provided")
		os.Exit(1)
	}
	if levelToUse == "" {
		logger.Error(ctx, "No level specified", nil, map[string]interface{}{"username": user.Username, "user_id": user.ID})
		fmt.Fprintln(os.Stderr, "Error: No level specified. User has no current level and --level flag not provided")
		os.Exit(1)
	}

	// Print configuration
	fmt.Printf("=== CLI Worker Configuration ===\n")
	fmt.Printf("User: %s (ID: %d)\n", user.Username, user.ID)
	fmt.Printf("Language: %s\n", languageToUse)
	fmt.Printf("Level: %s\n", levelToUse)
	fmt.Printf("Question Type: %s\n", qType)
	fmt.Printf("Count: %d\n", *count)
	if *topic != "" {
		fmt.Printf("Topic: %s\n", *topic)
	}
	if user.AIProvider.Valid && user.AIProvider.String != "" {
		fmt.Printf("AI Provider: %s\n", user.AIProvider.String)
	}
	if user.AIModel.Valid && user.AIModel.String != "" {
		fmt.Printf("AI Model: %s\n", user.AIModel.String)
	}
	fmt.Printf("===============================\n\n")

	// Create email service
	emailService := services.CreateEmailService(cfg, logger)
	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create a minimal worker instance for question generation
	workerInstance := worker.NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, emailService, nil, "cli", cfg, logger)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, config.CLIWorkerTimeout)
	defer cancel()

	// Log CLI worker start with structured logging
	logger.Info(ctx, "CLI worker starting question generation", map[string]interface{}{
		"user_id":       user.ID,
		"username":      user.Username,
		"question_type": qType,
		"count":         *count,
		"language":      languageToUse,
		"level":         levelToUse,
	})

	// Generate questions
	fmt.Printf("Starting question generation...\n")
	startTime := time.Now()

	result, err := workerInstance.GenerateQuestionsForUser(ctx, user, languageToUse, levelToUse, qType, *count, *topic)

	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("\n❌ Question generation failed after %v\n", duration)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Question generation completed successfully in %v\n", duration)
	fmt.Printf("Result: %s\n", result)
}

func isValidLevel(level string, validLevels []string) bool {
	for _, validLevel := range validLevels {
		if strings.EqualFold(level, validLevel) {
			return true
		}
	}
	return false
}

func isValidLanguage(language string, validLanguages []string) bool {
	for _, validLang := range validLanguages {
		if strings.EqualFold(language, validLang) {
			return true
		}
	}
	return false
}

func printUsage(cfg *config.Config) {
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "Error: Configuration is missing or invalid.\n")
		return
	}
	fmt.Printf("Usage: cli-worker [flags]\n")
	fmt.Printf("Flags:\n")
	fmt.Printf("  -language string\tLanguage to generate questions for\n")
	fmt.Printf("  -level string\tLevel to generate questions for\n")
	fmt.Printf("  -type string\tQuestion type (vocabulary, fill_in_blank, qa, reading_comprehension)\n")
	fmt.Printf("  -count int\tNumber of questions to generate (default 1)\n")
	fmt.Printf("  -topic string\tTopic for question generation\n")
	fmt.Printf("  -provider string\tAI provider to use\n")
	fmt.Printf("  -model string\tAI model to use\n")
	fmt.Printf("  -help\tShow this help message\n\n")

	fmt.Printf("Valid levels: %s\n", strings.Join(cfg.GetAllLevels(), ", "))
	fmt.Printf("Valid languages: %s\n", strings.Join(cfg.GetLanguages(), ", "))
	if cfg.Providers != nil {
		providerNames := make([]string, 0, len(cfg.Providers))
		for _, p := range cfg.Providers {
			providerNames = append(providerNames, p.Code)
		}
		fmt.Printf("Valid providers: %s\n", strings.Join(providerNames, ", "))
	} else {
		fmt.Printf("Valid providers: \n")
	}
}
