// Package middleware provides authentication and authorization middleware for the Gin web framework.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"quizapp/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Session keys for storing user information
const (
	// UserIDKey is the key used to store user ID in session
	UserIDKey = "user_id"
	// UsernameKey is the key used to store username in session
	UsernameKey = "username"
	// AuthMethodKey is the key used to store authentication method
	AuthMethodKey = "auth_method"
	// APIKeyIDKey is the key used to store API key ID (for API key auth)
	APIKeyIDKey = "api_key_id"
)

// AuthMethod constants
const (
	AuthMethodSession = "session"
	AuthMethodAPIKey  = "api_key"
)

// AuthAPIKeyValidator is an interface for validating API keys
type AuthAPIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (*models.AuthAPIKey, error)
	UpdateLastUsed(ctx context.Context, keyID int) error
}

// AuthUserServiceGetter is an interface for getting user info
type AuthUserServiceGetter interface {
	GetUserByID(ctx context.Context, userID int) (*models.User, error)
}

// RequireAuth returns a middleware that requires authentication
// This version only supports session-based auth for backward compatibility
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fall back to session authentication
		session := sessions.Default(c)
		userID := session.Get(UserIDKey)

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Validate user_id is an integer
		userIDInt, ok := userID.(int)
		if !ok {
			// Try to convert from float64 (JSON numbers are often stored as float64)
			if userIDFloat, ok := userID.(float64); ok {
				userIDInt = int(userIDFloat)
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
					"code":  "UNAUTHORIZED",
				})
				c.Abort()
				return
			}
		}

		// Validate username is a string and not empty
		username := session.Get(UsernameKey)
		if username == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		usernameStr, ok := username.(string)
		if !ok || usernameStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Store user info in context for handlers to use
		c.Set(UserIDKey, userIDInt)
		c.Set(UsernameKey, usernameStr)
		c.Set(AuthMethodKey, AuthMethodSession)

		c.Next()
	}
}

// RequireAuthWithAPIKey returns a middleware that requires authentication via API key or session
// It checks for API key authentication first, then falls back to session authentication
func RequireAuthWithAPIKey(apiKeyService AuthAPIKeyValidator, userService AuthUserServiceGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key authentication first
		var rawKey string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			rawKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Check for API key in query parameter
			rawKey = c.Query("api_key")
		}

		if rawKey != "" {
			// Validate API key
			apiKey, err := apiKeyService.ValidateAPIKey(c.Request.Context(), rawKey)
			if err == nil && apiKey != nil {
				// Check permission level against request method
				if !apiKey.CanPerformMethod(c.Request.Method) {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "This API key does not have permission for this operation",
						"code":  "FORBIDDEN",
					})
					c.Abort()
					return
				}

				// Get user info to set username in context
				user, err := userService.GetUserByID(c.Request.Context(), apiKey.UserID)
				if err != nil || user == nil {
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": "Invalid API key - user not found",
						"code":  "UNAUTHORIZED",
					})
					c.Abort()
					return
				}

				// Set user context
				c.Set(UserIDKey, apiKey.UserID)
				c.Set(UsernameKey, user.Username)
				c.Set(AuthMethodKey, AuthMethodAPIKey)
				c.Set(APIKeyIDKey, apiKey.ID)

				// Update last used timestamp asynchronously
				go func() {
					_ = apiKeyService.UpdateLastUsed(context.Background(), apiKey.ID)
				}()

				c.Next()
				return
			}
			// If we got here with a key (from header or query), it's invalid
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Fall back to session authentication
		session := sessions.Default(c)
		userID := session.Get(UserIDKey)

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Validate user_id is an integer
		userIDInt, ok := userID.(int)
		if !ok {
			// Try to convert from float64 (JSON numbers are often stored as float64)
			if userIDFloat, ok := userID.(float64); ok {
				userIDInt = int(userIDFloat)
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
					"code":  "UNAUTHORIZED",
				})
				c.Abort()
				return
			}
		}

		// Validate username is a string and not empty
		username := session.Get(UsernameKey)
		if username == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		usernameStr, ok := username.(string)
		if !ok || usernameStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Store user info in context for handlers to use
		c.Set(UserIDKey, userIDInt)
		c.Set(UsernameKey, usernameStr)
		c.Set(AuthMethodKey, AuthMethodSession)

		c.Next()
	}
}

// RequireAdmin returns a middleware that requires authentication and admin role
func RequireAdmin(userService interface{}) gin.HandlerFunc {
	// Type assertion to get the user service
	us, ok := userService.(interface {
		IsAdmin(ctx context.Context, userID int) (bool, error)
	})
	if !ok {
		panic("userService must implement IsAdmin method")
	}

	return func(c *gin.Context) {
		// First check authentication
		session := sessions.Default(c)
		userID := session.Get(UserIDKey)

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Validate user_id is an integer
		userIDInt, ok := userID.(int)
		if !ok {
			// Try to convert from float64 (JSON numbers are often stored as float64)
			if userIDFloat, ok := userID.(float64); ok {
				userIDInt = int(userIDFloat)
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
					"code":  "UNAUTHORIZED",
				})
				c.Abort()
				return
			}
		}

		// Validate username is a string and not empty
		username := session.Get(UsernameKey)
		if username == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		usernameStr, ok := username.(string)
		if !ok || usernameStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Check if user has admin role
		isAdmin, err := us.IsAdmin(c.Request.Context(), userIDInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check admin status",
				"code":  "INTERNAL_ERROR",
			})
			c.Abort()
			return
		}

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
				"code":  "FORBIDDEN",
			})
			c.Abort()
			return
		}

		// Store user info in context for handlers to use
		c.Set(UserIDKey, userIDInt)
		c.Set(UsernameKey, usernameStr)

		c.Next()
	}
}
