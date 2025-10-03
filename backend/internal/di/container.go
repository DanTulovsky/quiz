// Package di provides dependency injection container for managing service lifecycle and dependencies.
package di

import (
	"context"
	"database/sql"
	"sync"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"
)

// ServiceContainerInterface defines the interface for service containers
type ServiceContainerInterface interface {
	GetService(name string) (interface{}, error)
	GetUserService() (services.UserServiceInterface, error)
	GetQuestionService() (services.QuestionServiceInterface, error)
	GetLearningService() (services.LearningServiceInterface, error)
	GetAIService() (services.AIServiceInterface, error)
	GetWorkerService() (services.WorkerServiceInterface, error)
	GetDailyQuestionService() (services.DailyQuestionServiceInterface, error)
	GetStoryService() (services.StoryServiceInterface, error)
	GetOAuthService() (*services.OAuthService, error)
	GetGenerationHintService() (services.GenerationHintServiceInterface, error)
	GetEmailService() (services.EmailServiceInterface, error)
	GetDatabase() *sql.DB
	GetConfig() *config.Config
	GetLogger() *observability.Logger
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	EnsureAdminUser(ctx context.Context) error
}

// ServiceContainer manages all service dependencies and lifecycle
type ServiceContainer struct {
	cfg           *config.Config
	logger        *observability.Logger
	dbManager     *database.Manager
	db            *sql.DB
	services      map[string]interface{}
	mu            sync.RWMutex
	shutdownFuncs []func(context.Context) error
}

// NewServiceContainer creates a new dependency injection container
func NewServiceContainer(cfg *config.Config, logger *observability.Logger) *ServiceContainer {
	return &ServiceContainer{
		cfg:      cfg,
		logger:   logger,
		services: make(map[string]interface{}),
	}
}

// Initialize sets up all services and their dependencies
func (sc *ServiceContainer) Initialize(ctx context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Initialize database
	sc.dbManager = database.NewManager(sc.logger)
	db, err := sc.dbManager.InitDBWithConfig(sc.cfg.Database)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to initialize database")
	}
	sc.db = db
	sc.shutdownFuncs = append(sc.shutdownFuncs, func(_ context.Context) error {
		return db.Close()
	})

	// Initialize core services
	sc.initializeServices(ctx)

	// Startup lifecycle services
	if err := sc.startupServices(ctx); err != nil {
		// Cleanup on failure
		_ = sc.cleanup(ctx)
		return contextutils.WrapErrorf(err, "failed to startup services")
	}

	return nil
}

// GetService retrieves a service by name with type assertion
func (sc *ServiceContainer) GetService(name string) (interface{}, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	service, exists := sc.services[name]
	if !exists {
		return nil, contextutils.ErrorWithContextf("service %s not found", name)
	}
	return service, nil
}

// GetServiceAs performs type-safe service retrieval
func GetServiceAs[T any](sc *ServiceContainer, name string) (T, error) {
	var zero T
	service, err := sc.GetService(name)
	if err != nil {
		return zero, err
	}

	typed, ok := service.(T)
	if !ok {
		return zero, contextutils.ErrorWithContextf("service %s is not of expected type %T", name, zero)
	}
	return typed, nil
}

// GetUserService returns the user service
func (sc *ServiceContainer) GetUserService() (services.UserServiceInterface, error) {
	return GetServiceAs[services.UserServiceInterface](sc, "user")
}

// GetQuestionService returns the question service
func (sc *ServiceContainer) GetQuestionService() (services.QuestionServiceInterface, error) {
	return GetServiceAs[services.QuestionServiceInterface](sc, "question")
}

// GetLearningService returns the learning service
func (sc *ServiceContainer) GetLearningService() (services.LearningServiceInterface, error) {
	return GetServiceAs[services.LearningServiceInterface](sc, "learning")
}

// GetAIService returns the AI service
func (sc *ServiceContainer) GetAIService() (services.AIServiceInterface, error) {
	return GetServiceAs[services.AIServiceInterface](sc, "ai")
}

// GetWorkerService returns the worker service
func (sc *ServiceContainer) GetWorkerService() (services.WorkerServiceInterface, error) {
	return GetServiceAs[services.WorkerServiceInterface](sc, "worker")
}

// GetDailyQuestionService returns the daily question service
func (sc *ServiceContainer) GetDailyQuestionService() (services.DailyQuestionServiceInterface, error) {
	return GetServiceAs[services.DailyQuestionServiceInterface](sc, "daily_question")
}

// GetStoryService returns the story service
func (sc *ServiceContainer) GetStoryService() (services.StoryServiceInterface, error) {
	return GetServiceAs[services.StoryServiceInterface](sc, "story")
}

// GetOAuthService returns the OAuth service
func (sc *ServiceContainer) GetOAuthService() (*services.OAuthService, error) {
	service, err := sc.GetService("oauth")
	if err != nil {
		return nil, err
	}
	oauthService, ok := service.(*services.OAuthService)
	if !ok {
		return nil, contextutils.ErrorWithContextf("oauth service has incorrect type")
	}
	return oauthService, nil
}

// GetGenerationHintService returns the generation hint service
func (sc *ServiceContainer) GetGenerationHintService() (services.GenerationHintServiceInterface, error) {
	return GetServiceAs[services.GenerationHintServiceInterface](sc, "generation_hint")
}

// GetEmailService returns the email service
func (sc *ServiceContainer) GetEmailService() (services.EmailServiceInterface, error) {
	return GetServiceAs[services.EmailServiceInterface](sc, "email")
}

// GetDatabase returns the database instance
func (sc *ServiceContainer) GetDatabase() *sql.DB {
	return sc.db
}

// GetConfig returns the configuration
func (sc *ServiceContainer) GetConfig() *config.Config {
	return sc.cfg
}

// GetLogger returns the logger
func (sc *ServiceContainer) GetLogger() *observability.Logger {
	return sc.logger
}

// Shutdown gracefully shuts down all services
func (sc *ServiceContainer) Shutdown(ctx context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	return sc.cleanup(ctx)
}

// startupServices starts all services that implement the Lifecycle interface
func (sc *ServiceContainer) startupServices(ctx context.Context) error {
	// Check each service to see if it implements Lifecycle interface
	for name, service := range sc.services {
		if lifecycleService, ok := service.(interface{ Startup(context.Context) error }); ok {
			sc.logger.Info(ctx, "Starting service", map[string]interface{}{"service": name})
			if err := lifecycleService.Startup(ctx); err != nil {
				return contextutils.WrapErrorf(err, "failed to startup service %s", name)
			}
			sc.logger.Info(ctx, "Service started successfully", map[string]interface{}{"service": name})
		}
	}
	return nil
}

// cleanup handles shutdown of all services
func (sc *ServiceContainer) cleanup(ctx context.Context) error {
	var errors []error

	// Shutdown lifecycle services first (in reverse order)
	for name := range sc.services {
		if lifecycleService, ok := sc.services[name].(interface{ Shutdown(context.Context) error }); ok {
			sc.logger.Info(ctx, "Shutting down service", map[string]interface{}{"service": name})
			if err := lifecycleService.Shutdown(ctx); err != nil {
				sc.logger.Error(ctx, "Failed to shutdown service", err, map[string]interface{}{"service": name})
				errors = append(errors, contextutils.WrapErrorf(err, "service %s shutdown failed", name))
			} else {
				sc.logger.Info(ctx, "Service shutdown successfully", map[string]interface{}{"service": name})
			}
		}
	}

	// Shutdown services in reverse order of initialization
	for i := len(sc.shutdownFuncs) - 1; i >= 0; i-- {
		if err := sc.shutdownFuncs[i](ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return contextutils.ErrorWithContextf("shutdown errors: %v", errors)
	}
	return nil
}

// initializeServices sets up all service dependencies
func (sc *ServiceContainer) initializeServices(_ context.Context) {
	// Core services that don't depend on other services
	userService := services.NewUserServiceWithLogger(sc.db, sc.cfg, sc.logger)
	sc.services["user"] = userService

	// Learning service depends on user service
	learningService := services.NewLearningServiceWithLogger(sc.db, sc.cfg, sc.logger)
	sc.services["learning"] = learningService

	// Question service depends on learning service
	questionService := services.NewQuestionServiceWithLogger(sc.db, learningService, sc.cfg, sc.logger)
	sc.services["question"] = questionService

	// Daily question service depends on question and learning services
	dailyQuestionService := services.NewDailyQuestionService(sc.db, sc.logger, questionService, learningService)
	sc.services["daily_question"] = dailyQuestionService

	// Story service
	storyService := services.NewStoryService(sc.db, sc.cfg, sc.logger)
	sc.services["story"] = storyService

	// AI service
	aiService := services.NewAIService(sc.cfg, sc.logger)
	sc.services["ai"] = aiService

	// Worker service
	workerService := services.NewWorkerServiceWithLogger(sc.db, sc.logger)
	sc.services["worker"] = workerService

	// Generation hint service
	generationHintService := services.NewGenerationHintService(sc.db, sc.logger)
	sc.services["generation_hint"] = generationHintService

	// OAuth service
	oauthService := services.NewOAuthServiceWithLogger(sc.cfg, sc.logger)
	sc.services["oauth"] = oauthService

	// Email service
	emailService := services.CreateEmailService(sc.cfg, sc.logger)
	sc.services["email"] = emailService

	// Register shutdown functions
	sc.shutdownFuncs = append(sc.shutdownFuncs,
		func(_ context.Context) error { return nil }, // placeholder for future service shutdowns
	)
}

// EnsureAdminUser creates the admin user if it doesn't exist
func (sc *ServiceContainer) EnsureAdminUser(ctx context.Context) error {
	userService, err := sc.GetUserService()
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to get user service")
	}

	return userService.EnsureAdminUserExists(ctx, sc.cfg.Server.AdminUsername, sc.cfg.Server.AdminPassword)
}
