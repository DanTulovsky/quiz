// Package main provides the entry point for the Quiz Application worker service.
package main

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/version"
	"quizapp/internal/worker"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// fatalIfErr logs the error with context and panics with a consistent message
func fatalIfErr(ctx context.Context, logger *observability.Logger, msg string, err error, fields map[string]interface{}) {
	logger.Error(ctx, msg, err, fields)
	panic(msg + ": " + err.Error())
}

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Setup observability (tracing/metrics/logging)
	tp, mp, logger, err := observability.SetupObservability(&cfg.OpenTelemetry, "quiz-worker")
	if err != nil {
		panic("Failed to initialize observability: " + err.Error())
	}
	defer func() {
		if tp != nil {
			if err := tp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down tracer provider", map[string]interface{}{"error": err.Error(), "provider": "tracer"})
			}
		}
		if mp != nil {
			if err := mp.Shutdown(context.TODO()); err != nil {
				logger.Warn(ctx, "Error shutting down meter provider", map[string]interface{}{"error": err.Error(), "provider": "meter"})
			}
		}
	}()

	logger.Info(ctx, "Starting quiz worker service", map[string]interface{}{
		"port":     cfg.Server.WorkerPort,
		"logLevel": cfg.Server.LogLevel,
		"debug":    cfg.Server.Debug,
	})

	// Initialize database manager with logger
	dbManager := database.NewManager(logger)

	// Initialize database connection without running migrations (migrations are managed elsewhere)
	db, err := dbManager.InitDBWithoutMigrations(cfg.Database)
	if err != nil {
		fatalIfErr(ctx, logger, "Failed to initialize database", err, map[string]interface{}{"db_url": cfg.Database.URL})
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn(ctx, "Warning: failed to close database", map[string]interface{}{"error": err.Error(), "db_url": cfg.Database.URL})
		}
	}()

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	// Create question service
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	// Create usage stats service
	usageStatsService := services.NewUsageStatsService(cfg, db, logger)
	aiService := services.NewAIService(cfg, logger, usageStatsService)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	generationHintService := services.NewGenerationHintService(db, logger)
	emailService := services.CreateEmailServiceWithDB(cfg, logger, db)
	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Initialize worker with the observability logger
	workerInstance := worker.NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, storyService, emailService, generationHintService, "default", cfg, logger)
	go workerInstance.Start(ctx)

	// Initialize admin handler for worker UI
	adminHandler := handlers.NewWorkerAdminHandlerWithLogger(userService, questionService, aiService, cfg, workerInstance, workerService, learningService, dailyQuestionService, logger)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())

	// Add HTTP request logging middleware using our observability logger
	router.Use(func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log request details using our observability logger
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Create structured log entry
		fields := map[string]interface{}{
			"http.method":      method,
			"http.path":        path,
			"http.status_code": statusCode,
			"http.latency_ms":  latency.Milliseconds(),
			"http.client_ip":   clientIP,
			"http.user_agent":  c.Request.UserAgent(),
		}

		// Add error message if present
		if len(c.Errors) > 0 {
			fields["http.error"] = c.Errors.String()
		}

		// Log using our observability logger (goes to both stdout and OTLP)
		// Use appropriate log level based on status code
		if statusCode >= 500 {
			logger.Error(c.Request.Context(), "HTTP request failed", nil, fields)
		} else if statusCode >= 400 {
			logger.Warn(c.Request.Context(), "HTTP request warning", fields)
		} else {
			logger.Info(c.Request.Context(), "HTTP request", fields)
		}
	})

	// Add OpenTelemetry middleware for HTTP tracing with automatic error attributes
	router.Use(observability.GinMiddlewareWithErrorHandling("quiz-worker"))

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup session middleware
	store := cookie.NewStore([]byte(cfg.Server.SessionSecret))
	router.Use(sessions.Sessions(config.SessionName, store))

	// Setup routes
	v1 := router.Group("/v1")
	{
		// Health check route
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Version route
		v1.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":   "worker",
				"version":   version.Version,
				"commit":    version.Commit,
				"buildTime": version.BuildTime,
			})
		})
	}

	// Serve static assets (CSS/JS) for worker admin dashboard
	staticFS, _ := fs.Sub(handlers.AssetsFS, "templates/assets")
	router.StaticFS("/worker", http.FS(staticFS))

	// Config dump endpoint
	router.GET("/configz", adminHandler.GetConfigz)

	// API routes for worker management
	api := router.Group("/v1")
	{
		// Admin worker endpoints (for frontend)
		adminWorker := api.Group("/admin/worker")
		adminWorker.Use(middleware.RequireAuth())
		{
			adminWorker.GET("/details", adminHandler.GetWorkerDetails)
			adminWorker.GET("/status", adminHandler.GetWorkerStatus)
			adminWorker.GET("/logs", adminHandler.GetActivityLogs)
			adminWorker.POST("/pause", adminHandler.PauseWorker)
			adminWorker.POST("/resume", adminHandler.ResumeWorker)
			adminWorker.POST("/trigger", adminHandler.TriggerWorkerRun)
			adminWorker.GET("/ai-concurrency", adminHandler.GetAIConcurrencyStats)
		}

		// Worker user control endpoints (for pausing/resuming user question generation)
		workerUsers := api.Group("/admin/worker/users")
		workerUsers.Use(middleware.RequireAuth())
		{
			workerUsers.GET("/", adminHandler.GetWorkerUsers)
			workerUsers.POST("/pause", adminHandler.PauseWorkerUser)
			workerUsers.POST("/resume", adminHandler.ResumeWorkerUser)
		}

		// System health for worker
		system := api.Group("/system")
		{
			system.GET("/health", adminHandler.GetSystemHealth)
		}

		// Admin analytics endpoints (for frontend)
		adminAnalytics := api.Group("/admin/worker/analytics")
		adminAnalytics.Use(middleware.RequireAuth())
		{
			adminAnalytics.GET("/priority-scores", adminHandler.GetPriorityAnalytics)
			adminAnalytics.GET("/user-performance", adminHandler.GetUserPerformanceAnalytics)
			adminAnalytics.GET("/generation-intelligence", adminHandler.GetGenerationIntelligence)
			adminAnalytics.GET("/system-health", adminHandler.GetSystemHealthAnalytics)
			adminAnalytics.GET("/comparison", adminHandler.GetUserComparisonAnalytics)
			adminAnalytics.GET("/user/:userID", adminHandler.GetUserPriorityAnalytics)
		}

		// Admin daily questions endpoints (for frontend)
		adminDaily := api.Group("/admin/worker/daily")
		adminDaily.Use(middleware.RequireAuth())
		{
			adminDaily.GET("/users/:userId/questions/:date", adminHandler.GetUserDailyQuestions)
			adminDaily.POST("/users/:userId/questions/:date/regenerate", adminHandler.RegenerateUserDailyQuestions)
		}

		// Admin notification endpoints (for frontend)
		adminNotifications := api.Group("/admin/worker/notifications")
		adminNotifications.Use(middleware.RequireAuth())
		{
			adminNotifications.GET("/stats", adminHandler.GetNotificationStats)
			adminNotifications.GET("/errors", adminHandler.GetNotificationErrors)
			adminNotifications.GET("/sent", adminHandler.GetSentNotifications)
			adminNotifications.POST("/test/create-sent", adminHandler.CreateTestSentNotification)
			adminNotifications.POST("/force-send", adminHandler.ForceSendNotification)
		}
	}

	// Automatic route listing at root path
	routeListing := handlers.NewRouteListingHandler("Worker")
	routeListing.CollectRoutes(router)

	// Root path shows all available routes
	router.GET("/", func(c *gin.Context) {
		// Support JSON output via query parameter
		if c.Query("json") == "true" {
			routeListing.GetRouteListingJSON(c)
		} else {
			routeListing.GetRouteListingPage(c)
		}
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.WorkerPort,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info(ctx, "Worker server starting", map[string]interface{}{"port": cfg.Server.WorkerPort})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatalIfErr(ctx, logger, "Failed to start worker server", err, map[string]interface{}{"port": cfg.Server.WorkerPort})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "Worker server shutting down", map[string]interface{}{"service": "worker"})

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, config.WorkerShutdownTimeout)
	defer shutdownCancel()

	// Shutdown the worker first
	if err := workerInstance.Shutdown(shutdownCtx); err != nil {
		logger.Warn(ctx, "Warning: failed to shutdown worker", map[string]interface{}{"error": err.Error(), "service": "worker"})
	}

	// Then shutdown the server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fatalIfErr(ctx, logger, "Worker server forced to shutdown", err, map[string]interface{}{"service": "worker"})
	}

	logger.Info(ctx, "Worker server exited", map[string]interface{}{"service": "worker"})
}
