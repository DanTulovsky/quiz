package handlers

import (
	"quizapp/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// GetUserIDFromSession retrieves the current user ID from the session.
// Returns (0, false) if not authenticated or if the stored value is invalid.
func GetUserIDFromSession(c *gin.Context) (int, bool) {
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
