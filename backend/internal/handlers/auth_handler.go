package handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/otel/attribute"
)

// AuthHandler handles authentication related HTTP requests
type AuthHandler struct {
	userService  services.UserServiceInterface
	oauthService *services.OAuthService
	config       *config.Config
	logger       *observability.Logger
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(userService services.UserServiceInterface, oauthService *services.OAuthService, cfg *config.Config, logger *observability.Logger) *AuthHandler {
	return &AuthHandler{
		userService:  userService,
		oauthService: oauthService,
		config:       cfg,
		logger:       logger,
	}
}

// Login handles user login requests
func (h *AuthHandler) Login(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "login")
	defer observability.FinishSpan(span, nil)

	var req api.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	// Set span attributes for observability
	span.SetAttributes(
		attribute.String("auth.username", req.Username),
		attribute.Bool("auth.password_provided", req.Password != ""),
	)

	// Authenticate user against database
	user, err := h.userService.AuthenticateUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Authentication failed for user", err, map[string]interface{}{"username": req.Username})
		HandleAppError(c, contextutils.ErrInvalidCredentials)
		return
	}

	if user == nil {
		HandleAppError(c, contextutils.ErrInvalidCredentials)
		return
	}

	// Update span attributes with user info
	span.SetAttributes(
		attribute.Int("user.id", user.ID),
		attribute.String("user.username", user.Username),
		attribute.Bool("user.email_provided", user.Email.Valid),
		attribute.String("user.language", user.PreferredLanguage.String),
		attribute.String("user.level", user.CurrentLevel.String),
	)

	// Update last active
	if err := h.userService.UpdateLastActive(c.Request.Context(), user.ID); err != nil {
		// Log error but don't fail login
		// In production, you'd want proper logging here
		h.logger.Warn(c.Request.Context(), "Failed to update last active for user", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
	}

	// Create session
	session := sessions.Default(c)
	session.Set(middleware.UserIDKey, user.ID)
	session.Set(middleware.UsernameKey, user.Username)

	if err := session.Save(); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to save session", err, map[string]interface{}{"error": err.Error()})
		HandleAppError(c, contextutils.WrapError(err, "failed to create session"))
		return
	}

	// Convert models.User to api.User with proper API key checking
	apiUser := convertUserToAPIWithService(c.Request.Context(), user, h.userService)

	// Return user info (without API key)
	c.JSON(http.StatusOK, api.LoginResponse{
		Success: boolPtr(true),
		Message: stringPtr("Login successful"),
		User:    &apiUser,
	})
}

// Logout handles user logout requests
func (h *AuthHandler) Logout(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "logout")
	defer observability.FinishSpan(span, nil)

	// Get user info before clearing session for tracing
	session := sessions.Default(c)
	userID := session.Get(middleware.UserIDKey)
	username := session.Get(middleware.UsernameKey)

	// Set span attributes
	if userID != nil {
		span.SetAttributes(attribute.Int("user.id", userID.(int)))
	}
	if username != nil {
		span.SetAttributes(attribute.String("user.username", username.(string)))
	}

	session.Clear()

	if err := session.Save(); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to clear session"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{
		Success: true,
		Message: stringPtr("Logout successful"),
	})
}

// Status returns the current authentication status
func (h *AuthHandler) Status(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "status")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID := session.Get(middleware.UserIDKey)

	if userID == nil {
		span.SetAttributes(attribute.Bool("auth.authenticated", false))
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	span.SetAttributes(
		attribute.Bool("auth.authenticated", true),
		attribute.Int("user.id", userID.(int)),
	)

	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error getting user by ID", err, map[string]interface{}{"user_id": userID.(int)})
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	if user == nil {
		// User not found, clear session
		session.Clear()
		if err := session.Save(); err != nil {
			h.logger.Error(c.Request.Context(), "Error saving session", err, map[string]interface{}{"error": err.Error()})
		}
		span.SetAttributes(attribute.Bool("auth.user_found", false))
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	// Update span attributes with user info
	span.SetAttributes(
		attribute.Bool("auth.user_found", true),
		attribute.String("user.username", user.Username),
		attribute.Bool("user.email_provided", user.Email.Valid),
		attribute.String("user.language", user.PreferredLanguage.String),
		attribute.String("user.level", user.CurrentLevel.String),
		attribute.Bool("user.ai_enabled", user.AIEnabled.Bool),
		attribute.String("user.ai_provider", user.AIProvider.String),
		attribute.String("user.ai_model", user.AIModel.String),
	)

	// Update last active timestamp
	if err := h.userService.UpdateLastActive(c.Request.Context(), user.ID); err != nil {
		h.logger.Error(c.Request.Context(), "Error updating last active", err, map[string]interface{}{"user_id": user.ID})
		// Don't fail the request for this error
	}

	// Convert models.User to api.User with proper API key checking
	apiUser := convertUserToAPIWithService(c.Request.Context(), user, h.userService)

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user":          &apiUser,
	})
}

// Check is a lightweight auth-check endpoint intended for reverse proxy auth_request.
// It requires authentication via middleware and returns 204 when authenticated.
// Unauthenticated requests are rejected by the RequireAuth middleware with 401.
func (h *AuthHandler) Check(c *gin.Context) {
	// If we reached here, authentication succeeded in middleware
	c.Status(http.StatusNoContent)
}

// Signup handles user registration requests
func (h *AuthHandler) Signup(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "signup")
	defer observability.FinishSpan(span, nil)

	// Check if signups are disabled
	if h.config != nil && h.config.IsSignupDisabled() {
		span.SetAttributes(attribute.Bool("auth.signups_disabled", true))
		HandleAppError(c, contextutils.ErrForbidden)
		return
	}

	span.SetAttributes(attribute.Bool("auth.signups_disabled", false))

	var req api.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, openapi_types.ErrValidationEmail) {
			HandleAppError(c, contextutils.ErrInvalidInput)
			return
		}
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	// Set span attributes for request data
	span.SetAttributes(
		attribute.String("signup.username", req.Username),
		attribute.Bool("signup.password_provided", req.Password != ""),
		attribute.Bool("signup.email_provided", req.Email != nil && *req.Email != ""),
		attribute.Bool("signup.language_provided", req.PreferredLanguage != nil && *req.PreferredLanguage != ""),
		attribute.Bool("signup.level_provided", req.CurrentLevel != nil && *req.CurrentLevel != ""),
		attribute.Bool("signup.timezone_provided", req.Timezone != nil && *req.Timezone != ""),
	)

	// Validate required fields
	if req.Username == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	if req.Password == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	if req.Email == nil || *req.Email == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Validate username format (3-50 characters, alphanumeric + underscore)
	if len(req.Username) < 3 || len(req.Username) > 50 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(req.Username) {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Validate password (minimum 8 characters)
	if len(req.Password) < 8 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Validate email format (convert to string)
	if !contextutils.IsValidEmail(string(*req.Email)) {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Normalize email to lowercase
	email := strings.ToLower(string(*req.Email))

	h.logger.Info(c.Request.Context(), "Attempting signup for user", map[string]interface{}{"username": req.Username, "email": email})

	// Check if username already exists
	existingUser, err := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error checking username uniqueness", err, map[string]interface{}{"username": req.Username})
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	if existingUser != nil {
		span.SetAttributes(attribute.Bool("signup.username_exists", true))
		HandleAppError(c, contextutils.ErrRecordExists)
		return
	}

	// Check if email already exists
	existingUserByEmail, err := h.userService.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error checking email uniqueness", err, map[string]interface{}{"email": email})
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	if existingUserByEmail != nil {
		span.SetAttributes(attribute.Bool("signup.email_exists", true))
		HandleAppError(c, contextutils.ErrRecordExists)
		return
	}

	// Set default values for optional fields
	language := "italian" // Default to first language in the list
	if h.config != nil {
		// Get available languages from config
		languages := h.config.GetLanguages()
		if len(languages) > 0 {
			language = languages[0]
		}
	}
	if req.PreferredLanguage != nil && *req.PreferredLanguage != "" {
		language = *req.PreferredLanguage
	}

	// Choose canonical default level for the selected language (first level in config)
	level := ""
	levels := []string{}
	if h.config != nil {
		levels = h.config.GetLevelsForLanguage(language)
		if len(levels) > 0 {
			level = levels[0]
		}
	}

	// If client provided a level, require it to be a canonical code for the language.
	if req.CurrentLevel != nil && *req.CurrentLevel != "" {
		provided := *req.CurrentLevel
		matched := false
		for _, l := range levels {
			if strings.EqualFold(l, provided) {
				level = l
				matched = true
				break
			}
		}
		if !matched {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
	}

	timezone := "UTC" // Default timezone
	if req.Timezone != nil && *req.Timezone != "" {
		timezone = *req.Timezone
	}

	// Update span attributes with final values
	span.SetAttributes(
		attribute.String("signup.language", language),
		attribute.String("signup.level", level),
		attribute.String("signup.timezone", timezone),
	)

	// Create user with email and timezone (no AI settings)
	user, err := h.userService.CreateUserWithEmailAndTimezone(c.Request.Context(), req.Username, email, timezone, language, level)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Error creating user", err, map[string]interface{}{"username": req.Username, "email": email})
		HandleAppError(c, contextutils.WrapError(err, "failed to create user account"))
		return
	}

	// Now set the password hash
	if err := h.userService.UpdateUserPassword(c.Request.Context(), user.ID, req.Password); err != nil {
		h.logger.Error(c.Request.Context(), "Error setting user password", err, map[string]interface{}{"user_id": user.ID})
		// Try to clean up the user we just created
		if deleteErr := h.userService.DeleteUser(c.Request.Context(), user.ID); deleteErr != nil {
			h.logger.Error(c.Request.Context(), "Error cleaning up user after password set failure", err, map[string]interface{}{"user_id": user.ID, "error": deleteErr.Error()})
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to create user account"))
		return
	}

	// Update span attributes with created user info
	span.SetAttributes(
		attribute.Int("user.id", user.ID),
		attribute.String("user.username", user.Username),
		attribute.String("user.email", email),
	)

	h.logger.Info(c.Request.Context(), "Successfully created user", map[string]interface{}{"username": req.Username, "user_id": user.ID})

	// Return success response (no session created, no auto-login)
	c.JSON(http.StatusCreated, api.SuccessResponse{
		Success: true,
		Message: stringPtr("Account created successfully. Please log in."),
	})
}

// GoogleLogin initiates Google OAuth flow
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "google_login")
	defer observability.FinishSpan(span, nil)

	// Generate a state parameter for security
	state := generateRandomState()

	// Get the redirect URI from query parameters
	redirectURI := c.Query("redirect_uri")

	// Set span attributes
	span.SetAttributes(
		attribute.String("oauth.provider", "google"),
		attribute.String("oauth.state", state),
		attribute.String("oauth.redirect_uri", redirectURI),
	)

	// Store state and redirect URI in session for verification
	session := sessions.Default(c)
	session.Set("oauth_state", state)
	if redirectURI != "" {
		session.Set("oauth_redirect_uri", redirectURI)
	}
	if err := session.Save(); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to save session"))
		return
	}

	// Check if request is from iOS (only via platform query param to avoid false positives from web browsers on iOS devices)
	isIOS := c.Query("platform") == "ios"

	// Log iOS detection and client ID availability
	if isIOS {
		iosClientID := h.oauthService.GetConfig().GoogleOAuthIOSClientID
		if iosClientID == "" {
			h.logger.Warn(c.Request.Context(), "iOS OAuth request detected but GOOGLE_OAUTH_IOS_CLIENT_ID is not set - will use web client ID", nil)
		} else {
			h.logger.Info(c.Request.Context(), "iOS OAuth request detected, using iOS client ID", map[string]interface{}{
				"ios_client_id": iosClientID,
			})
		}
	}

	// Generate Google OAuth URL (with iOS client ID if available and request is from iOS)
	authURL := h.oauthService.GetGoogleAuthURL(c.Request.Context(), state, isIOS)

	// Store the redirect URI that will be used (for iOS, this is the custom URL scheme)
	if isIOS && h.oauthService.GetConfig().GoogleOAuthIOSClientID != "" {
		// For iOS, the redirect URI is the custom URL scheme with path component
		// Format: com.googleusercontent.apps.{CLIENT_ID}:/oauth2redirect
		// Strip .apps.googleusercontent.com suffix if present
		// Keep the hyphen - it's part of the iOS URL scheme as shown in Google Console
		iosClientID := h.oauthService.GetConfig().GoogleOAuthIOSClientID
		iosClientIDForRedirect := strings.TrimSuffix(iosClientID, ".apps.googleusercontent.com")
		iosRedirectURI := fmt.Sprintf("com.googleusercontent.apps.%s:/oauth2redirect", iosClientIDForRedirect)
		session.Set("oauth_redirect_uri", iosRedirectURI)
		if err := session.Save(); err != nil {
			HandleAppError(c, contextutils.WrapError(err, "failed to save session"))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// GoogleCallback handles the OAuth callback from Google
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "google_callback")
	defer observability.FinishSpan(span, nil)

	// Get the authorization code and state from query parameters
	code := c.Query("code")
	state := c.Query("state")

	// Set span attributes
	span.SetAttributes(
		attribute.String("oauth.provider", "google"),
		attribute.Bool("oauth.code_provided", code != ""),
		attribute.String("oauth.state", state),
	)

	h.logger.Info(c.Request.Context(), "Google OAuth callback received", map[string]interface{}{"code": code, "state": state})

	if code == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Verify state parameter for OAuth security (CSRF protection)
	session := sessions.Default(c)
	storedState := session.Get("oauth_state")

	h.logger.Info(c.Request.Context(), "OAuth state verification", map[string]interface{}{"stored_state": storedState, "received_state": state})

	// Enforce strict state verification for security
	if storedState == nil {
		h.logger.Error(c.Request.Context(), "No OAuth state found in session - possible CSRF attack or session issue", nil, map[string]interface{}{"state": state})
		span.SetAttributes(attribute.Bool("oauth.state_valid", false))
		HandleAppError(c, contextutils.ErrOAuthStateMismatch)
		return
	}

	if storedState.(string) != state {
		h.logger.Error(c.Request.Context(), "OAuth state mismatch - possible CSRF attack", nil, map[string]interface{}{"stored_state": storedState.(string), "received_state": state})
		span.SetAttributes(attribute.Bool("oauth.state_valid", false))
		HandleAppError(c, contextutils.ErrOAuthStateMismatch)
		return
	}

	span.SetAttributes(attribute.Bool("oauth.state_valid", true))
	h.logger.Info(c.Request.Context(), "OAuth state verification successful")

	// Check if user is already authenticated (prevent duplicate callbacks)
	existingUserID := session.Get(middleware.UserIDKey)
	if existingUserID != nil {
		h.logger.Info(c.Request.Context(), "User already authenticated during OAuth callback", map[string]interface{}{
			"user_id": existingUserID.(int),
		})
		span.SetAttributes(attribute.Bool("oauth.duplicate_callback", true))

		// Get user information for the response
		user, err := h.userService.GetUserByID(c.Request.Context(), existingUserID.(int))
		if err != nil {
			h.logger.Error(c.Request.Context(), "Error getting user by ID", err, map[string]interface{}{"user_id": existingUserID.(int)})
			HandleAppError(c, contextutils.ErrInternalError)
			return
		}

		if user == nil {
			h.logger.Error(c.Request.Context(), "User not found", nil, map[string]interface{}{"user_id": existingUserID.(int)})
			HandleAppError(c, contextutils.ErrInternalError)
			return
		}

		// Convert models.User to api.User with proper API key checking
		apiUser := convertUserToAPIWithService(c.Request.Context(), user, h.userService)

		// Return success response for already authenticated user
		response := api.LoginResponse{
			Success: boolPtr(true),
			Message: stringPtr("Already authenticated"),
			User:    &apiUser,
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Get the stored redirect URI from session
	// This is used for post-authentication redirect (where to send user after login)
	// For token exchange, we only use it if it's an iOS redirect URI
	storedRedirectURI := session.Get("oauth_redirect_uri")
	var redirectURI string
	if storedRedirectURI != nil {
		redirectURI = storedRedirectURI.(string)
		h.logger.Info(c.Request.Context(), "Retrieved stored redirect URI from session", map[string]interface{}{
			"redirect_uri": redirectURI,
		})
	} else {
		h.logger.Warn(c.Request.Context(), "No redirect URI stored in session, will use default for token exchange", nil)
	}

	// Clear the state and redirect URI from session
	session.Delete("oauth_state")
	session.Delete("oauth_redirect_uri")
	if err := session.Save(); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to save session", err, map[string]interface{}{"error": err.Error()})
		HandleAppError(c, contextutils.WrapError(err, "failed to save session"))
		return
	}

	// Authenticate user with Google OAuth
	// Only use stored redirect URI for token exchange if it's iOS (starts with com.googleusercontent.apps.)
	// For web OAuth, don't pass it so it uses the configured GoogleOAuthRedirectURL
	var redirectURIArg string
	if redirectURI != "" && strings.HasPrefix(redirectURI, "com.googleusercontent.apps.") {
		redirectURIArg = redirectURI
		h.logger.Info(c.Request.Context(), "Using iOS redirect URI for token exchange", map[string]interface{}{
			"redirect_uri": redirectURI,
		})
	} else if redirectURI != "" {
		h.logger.Info(c.Request.Context(), "Ignoring stored redirect URI for token exchange (web OAuth, will use config default)", map[string]interface{}{
			"stored_redirect_uri": redirectURI,
		})
	}
	user, err := h.oauthService.AuthenticateGoogleUser(c.Request.Context(), code, h.userService, redirectURIArg)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Google OAuth authentication failed", err, map[string]interface{}{"error": err.Error()})

		// Check if this is a signup disabled error (structured)
		if errors.Is(err, services.ErrSignupsDisabled) {
			span.SetAttributes(attribute.Bool("oauth.signups_disabled", true))
			HandleAppError(c, contextutils.ErrForbidden)
			return
		}

		// Provide better error messages to the frontend using structured error checking
		errorMessage := "Authentication failed"
		if errors.Is(err, services.ErrOAuthCodeAlreadyUsed) {
			errorMessage = "This authentication link has already been used. Please try signing in again."
		} else if errors.Is(err, services.ErrOAuthClientConfig) {
			errorMessage = "OAuth configuration error. Please contact support."
		} else if errors.Is(err, services.ErrOAuthInvalidRequest) {
			errorMessage = "Invalid authentication request. Please try again."
		} else if errors.Is(err, services.ErrOAuthUnauthorized) {
			errorMessage = "OAuth client is not authorized. Please contact support."
		} else if errors.Is(err, services.ErrOAuthUnsupportedGrant) {
			errorMessage = "Unsupported OAuth grant type. Please contact support."
		}

		HandleAppError(c, contextutils.WrapError(err, errorMessage))
		return
	}

	// Update span attributes with user info
	span.SetAttributes(
		attribute.Int("user.id", user.ID),
		attribute.String("user.username", user.Username),
		attribute.Bool("user.email_provided", user.Email.Valid),
		attribute.String("user.language", user.PreferredLanguage.String),
		attribute.String("user.level", user.CurrentLevel.String),
		attribute.Bool("user.is_new", user.CreatedAt.After(time.Now().Add(-5*time.Minute))), // Rough check if user was just created
	)

	// Update last active
	if err := h.userService.UpdateLastActive(c.Request.Context(), user.ID); err != nil {
		h.logger.Warn(c.Request.Context(), "Failed to update last active for user", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
	}

	// Create session
	session.Set(middleware.UserIDKey, user.ID)
	session.Set(middleware.UsernameKey, user.Username)

	h.logger.Info(c.Request.Context(), "Setting session for user", map[string]interface{}{"user_id": user.ID, "username": user.Username})

	if err := session.Save(); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to save session", err, map[string]interface{}{"error": err.Error()})
		HandleAppError(c, contextutils.WrapError(err, "failed to create session"))
		return
	}

	// Convert models.User to api.User with proper API key checking
	apiUser := convertUserToAPIWithService(c.Request.Context(), user, h.userService)

	h.logger.Info(c.Request.Context(), "Google OAuth successful for user", map[string]interface{}{"username": user.Username, "user_id": user.ID})

	// Return user info with redirect URI if available
	response := api.LoginResponse{
		Success: boolPtr(true),
		Message: stringPtr("Google authentication successful"),
		User:    &apiUser,
	}

	// Add redirect URI to response if it was stored
	if redirectURI != "" {
		response.RedirectUri = &redirectURI
	}

	c.JSON(http.StatusOK, response)
}

// generateRandomState generates a cryptographically secure random state parameter for OAuth security
func generateRandomState() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)

	// Use crypto/rand for cryptographically secure random generation
	for i := range b {
		// Generate a random byte and map it to charset
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			// If crypto/rand fails, we have a serious system issue - don't fallback to weaker randomness
			panic("Cryptographic random number generation failed: " + err.Error())
		}
		b[i] = charset[randomByte[0]%byte(len(charset))]
	}
	return string(b)
}

// SignupStatus returns whether signups are enabled or disabled
func (h *AuthHandler) SignupStatus(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "signup_status")
	defer observability.FinishSpan(span, nil)

	signupsDisabled := false
	oauthWhitelistEnabled := false
	var allowedDomains []string
	var allowedEmails []string

	if h.config != nil {
		signupsDisabled = h.config.IsSignupDisabled()
		if h.config.System != nil {
			oauthWhitelistEnabled = len(h.config.System.Auth.AllowedDomains) > 0 || len(h.config.System.Auth.AllowedEmails) > 0
			allowedDomains = h.config.System.Auth.AllowedDomains
			allowedEmails = h.config.System.Auth.AllowedEmails
		}
	}

	span.SetAttributes(
		attribute.Bool("auth.signups_disabled", signupsDisabled),
		attribute.Bool("auth.config_available", h.config != nil),
		attribute.Bool("auth.oauth_whitelist_enabled", oauthWhitelistEnabled),
	)

	c.JSON(http.StatusOK, gin.H{
		"signups_disabled":        signupsDisabled,
		"oauth_whitelist_enabled": oauthWhitelistEnabled,
		"allowed_domains":         allowedDomains,
		"allowed_emails":          allowedEmails,
	})
}
