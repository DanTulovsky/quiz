package handlers

import (
	"quizapp/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// GetUserIDFromSession retrieves the current user ID from the session or context.
// Returns (0, false) if not authenticated or if the stored value is invalid.
func GetUserIDFromSession(c *gin.Context) (int, bool) {
	// First check if user ID is already in context (set by auth middleware)
	if userIDVal, exists := c.Get(middleware.UserIDKey); exists {
		if id, ok := userIDVal.(int); ok {
			return id, true
		}
		// Try to convert from uint (common in tests)
		if idUint, ok := userIDVal.(uint); ok {
			return int(idUint), true
		}
		// If it's some other type in context, it's invalid
		return 0, false
	}

	// Fall back to session if not in context (maintain original behavior for sessions)
	session := sessions.Default(c)
	userID := session.Get(middleware.UserIDKey)
	if userID == nil {
		return 0, false
	}
	id, ok := userID.(int)
	if !ok {
		return 0, false
	}
	return id, true
}

// GetUsernameFromSession retrieves the current user username from the session or context.
// Returns (0, false) if not authenticated or if the stored value is invalid.
func GetUsernameFromSession(c *gin.Context) (string, bool) {
	// First check if user ID is already in context (set by auth middleware)
	if usernameVal, exists := c.Get(middleware.UsernameKey); exists {
		if username, ok := usernameVal.(string); ok {
			return username, true
		}
		return "", false
	}

	// Fall back to session if not in context (maintain original behavior for sessions)
	session := sessions.Default(c)
	username := session.Get(middleware.UsernameKey)
	if username == nil {
		return "", false
	}
	usernameStr, ok := username.(string)
	if !ok {
		return "", false
	}
	return usernameStr, true
}
