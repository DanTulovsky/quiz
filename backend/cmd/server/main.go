// Package main provides the main entry point for the quiz application backend server.
// It sets up the HTTP server, database connections, middleware, and API routes.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/di"
	"quizapp/internal/handlers"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
)

// Application encapsulates the main application logic and can be tested
type Application struct {
	container di.ServiceContainerInterface
	router    *gin.Engine
}

// NewApplication creates a new application instance
func NewApplication(container di.ServiceContainerInterface) (*Application, error) {
	// Get services from container
	userService, err := container.GetUserService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user service")
	}

	questionService, err := container.GetQuestionService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get question service")
	}

	learningService, err := container.GetLearningService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get learning service")
	}

	aiService, err := container.GetAIService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get AI service")
	}

	workerService, err := container.GetWorkerService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get worker service")
	}

	dailyQuestionService, err := container.GetDailyQuestionService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get daily question service")
	}

	storyService, err := container.GetStoryService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get story service")
	}

	oauthService, err := container.GetOAuthService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get OAuth service")
	}

	generationHintService, err := container.GetGenerationHintService()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get generation hint service")
	}

	// Use the router factory
	router := handlers.NewRouter(
		container.GetConfig(),
		userService,
		questionService,
		learningService,
		aiService,
		workerService,
		dailyQuestionService,
		storyService,
		oauthService,
		generationHintService,
		container.GetLogger(),
	)

	return &Application{
		container: container,
		router:    router,
	}, nil
}

// Run starts the application and returns an error if it fails to start
func (a *Application) Run(ctx context.Context, port string) error {
	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := a.router.Run(":" + port); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		return nil // Context cancelled, graceful shutdown
	case err := <-serverErr:
		return contextutils.WrapError(err, "server failed")
	}
}

// Shutdown gracefully shuts down the application
func (a *Application) Shutdown(ctx context.Context) error {
	return a.container.Shutdown(ctx)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup observability (tracing/metrics/logging)
	tp, mp, logger, err := observability.SetupObservability(&cfg.OpenTelemetry, "quiz-backend")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if tp != nil {
			if err := tp.Shutdown(shutdownCtx); err != nil {
				logger.Warn(ctx, "Error shutting down tracer provider", map[string]interface{}{"error": err.Error(), "provider": "tracer"})
			}
		}
		if mp != nil {
			if err := mp.Shutdown(shutdownCtx); err != nil {
				logger.Warn(ctx, "Error shutting down meter provider", map[string]interface{}{"error": err.Error(), "provider": "meter"})
			}
		}
	}()

	logger.Info(ctx, "Starting quiz backend service", map[string]interface{}{
		"port":     cfg.Server.Port,
		"logLevel": cfg.Server.LogLevel,
	})

	// Initialize dependency injection container
	container := di.NewServiceContainer(cfg, logger)

	// Initialize all services
	if err := container.Initialize(ctx); err != nil {
		logger.Error(ctx, "Failed to initialize services", err, nil)
		os.Exit(1)
	}

	// Ensure admin user exists
	if err := container.EnsureAdminUser(ctx); err != nil {
		logger.Error(ctx, "Failed to ensure admin user exists", err, map[string]interface{}{"admin_username": cfg.Server.AdminUsername})
		os.Exit(1)
	}

	// Create application instance
	app, err := NewApplication(container)
	if err != nil {
		logger.Error(ctx, "Failed to create application", err, nil)
		os.Exit(1)
	}

	// Start application in a goroutine
	appErr := make(chan error, 1)
	go func() {
		if err := app.Run(ctx, cfg.Server.Port); err != nil {
			appErr <- err
		}
	}()

	// Wait for shutdown signal or application error
	select {
	case <-shutdownCh:
		logger.Info(ctx, "Received shutdown signal, shutting down gracefully", nil)
	case err := <-appErr:
		logger.Error(ctx, "Application failed", err, nil)
		os.Exit(1)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown application
	if err := app.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Error during application shutdown", err, nil)
		os.Exit(1)
	}

	logger.Info(ctx, "Shutdown completed successfully", nil)
}
