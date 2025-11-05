package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/version"
)

// IMPORTANT: When adding new API endpoints, make sure to:
// 1. Add them to swagger.yaml with proper documentation
// 2. Run `task generate-api-types` to regenerate types
// 3. Update any relevant tests
// 4. Consider if the endpoint should be public or admin-only

// NewRouter creates a new router factory with all the necessary middleware and routes
func NewRouter(
	cfg *config.Config,
	userService services.UserServiceInterface,
	questionService services.QuestionServiceInterface,
	learningService services.LearningServiceInterface,
	aiService services.AIServiceInterface,
	workerService services.WorkerServiceInterface,
	dailyQuestionService services.DailyQuestionServiceInterface,
	storyService services.StoryServiceInterface,
	conversationService services.ConversationServiceInterface,
	oauthService *services.OAuthService,
	generationHintService services.GenerationHintServiceInterface,
	translationService services.TranslationServiceInterface,
	snippetsService services.SnippetsServiceInterface,
	usageStatsService services.UsageStatsServiceInterface,
	wordOfTheDayService services.WordOfTheDayServiceInterface,
	authAPIKeyService services.AuthAPIKeyServiceInterface,
	logger *observability.Logger,
) *gin.Engine {
	// Setup Gin router
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

		// For failed requests (4xx and 5xx), capture response body for debugging
		if statusCode >= 400 {
			// Get response body for error requests
			if c.Writer.Size() > 0 {
				// Try to capture response body for debugging
				// Note: This is a best effort since the response may have already been written
				fields["http.response_size"] = c.Writer.Size()
			}

			// Add more context for 5xx errors
			if statusCode >= 500 {
				fields["http.error_type"] = "server_error"
				// Log additional context that might help debugging
				if c.Request.Body != nil {
					fields["http.request_has_body"] = true
				}
			} else {
				fields["http.error_type"] = "client_error"
			}
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

	// Health check endpoint (defined before any middleware)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "backend"})
	})

	// Add OpenTelemetry middleware for HTTP tracing and context propagation with automatic error attributes
	router.Use(observability.GinMiddlewareWithErrorHandling("quiz-backend"))

	// Add response validation middleware for API endpoints
	router.Use(middleware.ResponseValidationMiddleware(logger))

	// Swagger documentation (defined before middleware)
	router.StaticFile("/swagger.yaml", "./swagger.yaml")
	router.StaticFile("/swaggerz", "./swaggerz.html")

	// Disable automatic redirection for trailing slashes, which is better for APIs
	router.RedirectTrailingSlash = false

	// Setup CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.Server.CORSOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(corsConfig))

	// Setup session middleware
	store := cookie.NewStore([]byte(cfg.Server.SessionSecret))
	// Configure session options for security
	sessionOpts := sessions.Options{
		Path:     config.SessionPath,
		MaxAge:   int(config.SessionMaxAge.Seconds()),
		HttpOnly: config.SessionHTTPOnly,
		Secure:   config.SessionSecure, // Set to true in production with HTTPS
	}
	if cfg.Server.Debug {
		sessionOpts.SameSite = http.SameSiteDefaultMode
	} else {
		sessionOpts.SameSite = http.SameSiteLaxMode
		sessionOpts.Secure = true
	}
	store.Options(sessionOpts)
	router.Use(sessions.Sessions(config.SessionName, store))

	// Setup Gin mode
	gin.SetMode(gin.ReleaseMode)
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	}

	// Security middleware
	secureConfig := secure.DefaultConfig()
	secureConfig.SSLRedirect = false
	secureConfig.ContentSecurityPolicy = config.DefaultCSP
	router.Use(secure.New(secureConfig))

	// Serve all static assets (JS, fonts, CSS, etc.) from /backend/*filepath
	// Note: Static assets are now served from the frontend build

	// Initialize handlers
	authHandler := NewAuthHandler(userService, oauthService, cfg, logger)
	authAPIKeyHandler := NewAuthAPIKeyHandler(authAPIKeyService, logger)
	emailService := services.CreateEmailService(cfg, logger)
	settingsHandler := NewSettingsHandler(userService, storyService, conversationService, aiService, learningService, emailService, usageStatsService, cfg, logger)
	quizHandler := NewQuizHandler(userService, questionService, aiService, learningService, workerService, generationHintService, usageStatsService, cfg, logger)
	dailyQuestionHandler := NewDailyQuestionHandler(userService, dailyQuestionService, cfg, logger)
	storyHandler := NewStoryHandler(storyService, userService, aiService, cfg, logger)
	aiConversationHandler := NewAIConversationHandler(conversationService, cfg, logger)
	translationHandler := NewTranslationHandler(translationService, cfg, logger)
	snippetsHandler := NewSnippetsHandler(snippetsService, cfg, logger)
	wordOfTheDayHandler := NewWordOfTheDayHandler(userService, wordOfTheDayService, cfg, logger)
	adminHandler := NewAdminHandlerWithLogger(userService, questionService, aiService, cfg, learningService, workerService, logger, usageStatsService)
	// Inject story service into admin handler via exported field
	adminHandler.storyService = storyService
	userAdminHandler := NewUserAdminHandler(userService, cfg, logger)
	verbConjugationHandler := NewVerbConjugationHandler(logger)
	feedbackService := services.NewFeedbackService(userService.GetDB(), logger)

	// Initialize Linear service if enabled
	var linearService *services.LinearService
	if cfg.Linear.Enabled {
		linearService = services.NewLinearService(cfg, logger)
	}

	feedbackHandler := NewFeedbackHandler(feedbackService, linearService, userService, cfg, logger)

	// V1 routes (matching swagger spec)
	v1 := router.Group("/v1")
	{
		// Version aggregation endpoint (no auth)
		v1.GET("/version", func(c *gin.Context) {
			backendVersion := gin.H{
				"service":   "backend",
				"version":   version.Version,
				"commit":    version.Commit,
				"buildTime": version.BuildTime,
			}
			workerInternalURL := os.Getenv("WORKER_INTERNAL_URL")
			if workerInternalURL == "" {
				workerInternalURL = cfg.Server.WorkerInternalURL // fallback
			}
			// Use instrumented HTTP client for tracing
			client := &http.Client{
				Transport: otelhttp.NewTransport(http.DefaultTransport),
			}
			req, err := http.NewRequest("GET", workerInternalURL+"/v1/version", nil)
			var workerResp *http.Response
			if err == nil {
				req = req.WithContext(c.Request.Context())
				workerResp, err = client.Do(req)
			}
			var workerVersion interface{}
			if err == nil && workerResp.StatusCode == http.StatusOK {
				defer func() { _ = workerResp.Body.Close() }()
				if err := json.NewDecoder(workerResp.Body).Decode(&workerVersion); err != nil {
					workerVersion = gin.H{"error": "Failed to decode worker version"}
				}
			} else {
				workerVersion = gin.H{"error": "Worker unavailable"}
			}
			c.JSON(http.StatusOK, gin.H{
				"backend": backendVersion,
				"worker":  workerVersion,
			})
		})
		auth := v1.Group("/auth")
		{
			auth.POST("/login", middleware.RequestValidationMiddleware(logger), authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/status", authHandler.Status)
			auth.GET("/check", middleware.RequireAuth(), authHandler.Check)
			auth.POST("/signup", middleware.RequestValidationMiddleware(logger), authHandler.Signup)
			auth.GET("/signup/status", authHandler.SignupStatus)
			auth.GET("/google/login", authHandler.GoogleLogin)
			auth.GET("/google/callback", authHandler.GoogleCallback)
		}

		// API Keys routes (for programmatic API access)
		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(middleware.RequireAuth()) // Keep session-only auth for managing API keys
		{
			apiKeys.POST("", middleware.RequestValidationMiddleware(logger), authAPIKeyHandler.CreateAPIKey)
			apiKeys.GET("", authAPIKeyHandler.ListAPIKeys)
			apiKeys.DELETE("/:id", authAPIKeyHandler.DeleteAPIKey)
		}

		// API Key test endpoints using API key auth (no cookies)
		apiKeysTest := v1.Group("/api-keys")
		apiKeysTest.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		{
			apiKeysTest.GET("/test-read", authAPIKeyHandler.TestRead)
			apiKeysTest.POST("/test-write", authAPIKeyHandler.TestWrite)
		}

		// Translation routes
		v1.POST("/translate", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), translationHandler.TranslateText)

		// Feedback routes
		v1.POST("/feedback", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), feedbackHandler.SubmitFeedback)

		// Snippets routes
		v1.POST("/snippets", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), snippetsHandler.CreateSnippet)
		v1.GET("/snippets", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.GetSnippets)
		v1.DELETE("/snippets", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.DeleteAllSnippets)
		v1.GET("/snippets/search", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.SearchSnippets)
		v1.GET("/snippets/by-question/:question_id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.GetSnippetsByQuestion)
		v1.GET("/snippets/by-section/:section_id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.GetSnippetsBySection)
		v1.GET("/snippets/by-story/:story_id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.GetSnippetsByStory)
		v1.GET("/snippets/:id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.GetSnippet)
		v1.PUT("/snippets/:id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), snippetsHandler.UpdateSnippet)
		v1.DELETE("/snippets/:id", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), snippetsHandler.DeleteSnippet)

		quiz := v1.Group("/quiz")
		quiz.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		quiz.Use(middleware.RequestValidationMiddleware(logger))
		{
			quiz.GET("/question", quizHandler.GetQuestion)
			quiz.GET("/question/:id", quizHandler.GetQuestion)
			quiz.POST("/question/:id/report", quizHandler.ReportQuestion)
			quiz.POST("/question/:id/mark-known", quizHandler.MarkQuestionAsKnown)
			quiz.POST("/answer", quizHandler.SubmitAnswer)
			quiz.GET("/progress", quizHandler.GetProgress)
			quiz.GET("/ai-token-usage", quizHandler.GetAITokenUsage)
			quiz.GET("/ai-token-usage/daily", quizHandler.GetAITokenUsageDaily)
			quiz.GET("/ai-token-usage/hourly", quizHandler.GetAITokenUsageHourly)
			quiz.GET("/worker-status", quizHandler.GetWorkerStatus)
			quiz.POST("/chat/stream", quizHandler.ChatStream)
		}
		daily := v1.Group("/daily")
		daily.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		daily.Use(middleware.RequestValidationMiddleware(logger))
		{
			daily.GET("/questions/:date", dailyQuestionHandler.GetDailyQuestions)
			daily.POST("/questions/:date/complete/:questionId", dailyQuestionHandler.MarkQuestionCompleted)
			daily.DELETE("/questions/:date/complete/:questionId", dailyQuestionHandler.ResetQuestionCompleted)
			daily.POST("/questions/:date/answer/:questionId", dailyQuestionHandler.SubmitDailyQuestionAnswer)
			daily.GET("/history/:questionId", dailyQuestionHandler.GetQuestionHistory)
			daily.GET("/dates", dailyQuestionHandler.GetAvailableDates)
			daily.GET("/progress/:date", dailyQuestionHandler.GetDailyProgress)
			// Note: Assignment is handled automatically by the worker
		}

		wordOfDay := v1.Group("/word-of-day")
		{
			// Protected endpoints requiring authentication (API key or session)
			wordOfDay.GET("", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), wordOfTheDayHandler.GetWordOfTheDayToday)
			wordOfDay.GET("/history", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), wordOfTheDayHandler.GetWordOfTheDayHistory)
			// Embed endpoint (public - no authentication required) - must come before /:date to avoid route matching
			wordOfDay.GET("/:date/embed", wordOfTheDayHandler.GetWordOfTheDayEmbed)
			wordOfDay.GET("/:date", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), wordOfTheDayHandler.GetWordOfTheDay)
		}

		story := v1.Group("/story")
		story.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		story.Use(middleware.RequestValidationMiddleware(logger))
		{
			story.POST("", storyHandler.CreateStory)
			story.GET("", storyHandler.GetUserStories)
			story.GET("/current", storyHandler.GetCurrentStory)
			story.GET("/:id", storyHandler.GetStory)
			story.GET("/section/:id", storyHandler.GetSection)
			story.POST("/:id/generate", storyHandler.GenerateNextSection)
			story.POST("/:id/archive", storyHandler.ArchiveStory)
			story.POST("/:id/complete", storyHandler.CompleteStory)
			story.POST("/:id/set-current", storyHandler.SetCurrentStory)
			story.POST("/:id/toggle-auto-generation", storyHandler.ToggleAutoGeneration)
			story.DELETE("/:id", storyHandler.DeleteStory)
			story.GET("/:id/export", storyHandler.ExportStory)
		}
		settings := v1.Group("/settings")
		{
			settings.GET("/ai-providers", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), settingsHandler.GetProviders)
			settings.GET("/levels", settingsHandler.GetLevels)
			settings.GET("/languages", settingsHandler.GetLanguages)
			settings.POST("/test-ai", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.TestAIConnection)
			settings.POST("/test-email", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.SendTestEmail)
			settings.PUT("", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.UpdateUserSettings)
			settings.PUT("/word-of-day-email", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.UpdateWordOfDayEmailPreference)
			// User data management endpoints
			settings.POST("/clear-stories", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.ClearAllStories)
			settings.POST("/clear-ai-chats", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.ClearAllAIChats)
			settings.POST("/reset-account", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), settingsHandler.ResetAccount)
			settings.GET("/api-key/:provider", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), settingsHandler.CheckAPIKeyAvailability)
		}

		// Verb conjugation endpoints
		verbConjugations := v1.Group("/verb-conjugations")
		verbConjugations.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		{
			verbConjugations.GET("/info", verbConjugationHandler.GetVerbConjugationInfo)
			verbConjugations.GET("/languages", verbConjugationHandler.GetAvailableLanguages)
			verbConjugations.GET("/:language", verbConjugationHandler.GetVerbConjugations)
			verbConjugations.GET("/:language/:verb", verbConjugationHandler.GetVerbConjugation)
		}

		// AI conversation endpoints
		ai := v1.Group("/ai")
		ai.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		ai.Use(middleware.RequestValidationMiddleware(logger))
		{
			ai.GET("/conversations", aiConversationHandler.GetConversations)
			ai.POST("/conversations", aiConversationHandler.CreateConversation)
			ai.GET("/conversations/:id", aiConversationHandler.GetConversation)
			ai.PUT("/conversations/:id", aiConversationHandler.UpdateConversation)
			ai.DELETE("/conversations/:id", aiConversationHandler.DeleteConversation)
			ai.POST("/conversations/:conversationId/messages", aiConversationHandler.AddMessage)
			ai.PUT("/conversations/bookmark", aiConversationHandler.ToggleMessageBookmark)
			ai.GET("/search", aiConversationHandler.SearchConversations)
			ai.GET("/bookmarks", aiConversationHandler.GetBookmarkedMessages)
		}
		preferences := v1.Group("/preferences")
		preferences.Use(middleware.RequireAuthWithAPIKey(authAPIKeyService, userService))
		preferences.Use(middleware.RequestValidationMiddleware(logger))
		{
			preferences.GET("/learning", settingsHandler.GetLearningPreferences)
			preferences.PUT("/learning", settingsHandler.UpdateLearningPreferences)
		}

		// User management endpoints (non-admin)
		userz := v1.Group("/userz")
		{
			userz.PUT("/profile", middleware.RequireAuthWithAPIKey(authAPIKeyService, userService), middleware.RequestValidationMiddleware(logger), userAdminHandler.UpdateCurrentUserProfile)
		}

		// Admin endpoints
		admin := v1.Group("/admin")
		admin.Use(middleware.RequireAdmin(userService))
		admin.Use(middleware.RequestValidationMiddleware(logger))
		{
			// Backend admin endpoints
			backend := admin.Group("/backend")
			{
				// Backend admin page
				backend.GET("", adminHandler.GetBackendAdminPage)
				// Feedback management (admin only)
				backend.GET("/feedback", feedbackHandler.ListFeedback)
				backend.GET("/feedback/:id", feedbackHandler.GetFeedback)
				backend.PATCH("/feedback/:id", feedbackHandler.UpdateFeedback)
				backend.DELETE("/feedback/:id", feedbackHandler.DeleteFeedback)
				backend.DELETE("/feedback", func(c *gin.Context) {
					// Check if it's a delete all request
					if c.Query("all") == "true" {
						feedbackHandler.DeleteAllFeedback(c)
					} else {
						feedbackHandler.DeleteFeedbackByStatus(c)
					}
				})
				backend.POST("/feedback/:id/linear-issue", feedbackHandler.CreateLinearIssue)
				// User management (admin only)
				backend.GET("/userz", userAdminHandler.GetAllUsers)
				backend.GET("/userz/paginated", userAdminHandler.GetUsersPaginated)
				backend.POST("/userz", userAdminHandler.CreateUser)
				backend.PUT("/userz/:id", userAdminHandler.UpdateUser)
				backend.DELETE("/userz/:id", userAdminHandler.DeleteUser)
				backend.POST("/userz/:id/reset-password", userAdminHandler.ResetUserPassword)

				// Role management endpoints
				backend.GET("/roles", adminHandler.GetRoles)
				backend.GET("/userz/:id/roles", adminHandler.GetUserRoles)
				backend.POST("/userz/:id/roles", adminHandler.AssignRole)
				backend.DELETE("/userz/:id/roles/:roleId", adminHandler.RemoveRole)

				// Admin dashboard data
				backend.GET("/dashboard", adminHandler.GetBackendAdminData)
				backend.GET("/ai-concurrency", adminHandler.GetAIConcurrencyStats)

				// Question management
				backend.GET("/questions/:id", adminHandler.GetQuestion)
				backend.GET("/questions/:id/users", adminHandler.GetUsersForQuestion)
				backend.PUT("/questions/:id", adminHandler.UpdateQuestion)
				backend.DELETE("/questions/:id", adminHandler.DeleteQuestion)
				backend.POST("/questions/:id/assign-users", adminHandler.AssignUsersToQuestion)
				backend.POST("/questions/:id/unassign-users", adminHandler.UnassignUsersFromQuestion)
				backend.GET("/questions/paginated", adminHandler.GetQuestionsPaginated)
				backend.GET("/questions", adminHandler.GetAllQuestions)
				backend.GET("/reported-questions", adminHandler.GetReportedQuestionsPaginated)
				backend.POST("/questions/:id/fix", adminHandler.MarkQuestionAsFixed)
				backend.POST("/questions/:id/ai-fix", adminHandler.FixQuestionWithAI)

				// Data management
				backend.POST("/clear-user-data", adminHandler.ClearUserData)
				backend.POST("/clear-database", adminHandler.ClearDatabase)
				backend.POST("/userz/:id/clear", adminHandler.ClearUserDataForUser)

				// Story explorer (admin)
				backend.GET("/stories", adminHandler.GetStoriesPaginated)
				backend.GET("/stories/:id", adminHandler.GetStoryAdmin)
				backend.DELETE("/stories/:id", adminHandler.DeleteStoryAdmin)
				backend.GET("/story-sections/:id", adminHandler.GetSectionAdmin)

				// Usage stats (admin)
				backend.GET("/usage-stats", adminHandler.GetUsageStats)
				backend.GET("/usage-stats/:service", adminHandler.GetUsageStatsByService)
			}

		}
	}

	// Config dump endpoint
	router.GET("/configz", adminHandler.GetConfigz)

	// Serve frontend static files
	router.Static("/assets", "./frontend/dist/assets")
	router.StaticFile("/favicon.svg", "./frontend/dist/favicon.svg")
	router.StaticFile("/fonts", "./frontend/dist/fonts")

	// Catch-all route for SPA - serve index.html for any route that doesn't match API routes
	router.NoRoute(func(c *gin.Context) {
		// Don't serve index.html for API routes
		if strings.HasPrefix(c.Request.URL.Path, "/v1/") ||
			strings.HasPrefix(c.Request.URL.Path, "/configz") ||
			strings.HasPrefix(c.Request.URL.Path, "/swagger") ||
			strings.HasPrefix(c.Request.URL.Path, "/backend/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// Serve the frontend's index.html for all other routes
		c.File("./frontend/dist/index.html")
	})

	// Automatic route listing at root path
	routeListing := NewRouteListingHandler("Backend")
	routeListing.CollectRoutes(router)

	// Root path shows all available routes
	router.GET("/", func(c *gin.Context) {
		if c.Query("json") == "true" {
			routeListing.GetRouteListingJSON(c)
		} else {
			routeListing.GetRouteListingPage(c)
		}
	})

	return router
}
