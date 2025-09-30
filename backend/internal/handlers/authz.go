package handlers

import (
	"context"
	"errors"

	"quizapp/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var (
	// ErrUnauthenticated indicates no current user could be determined
	ErrUnauthenticated = errors.New("user not authenticated")
	// ErrInvalidUserID indicates the stored user identifier is malformed
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrForbidden indicates the user lacks permissions for the operation
	ErrForbidden = errors.New("forbidden")
)

// GetCurrentUserID returns the current authenticated user's ID.
// It first checks the Gin context (set by RequireAuth/RequireAdmin),
// then falls back to the session store. Returns an error if unauthenticated
// or if the stored value is invalid.
func GetCurrentUserID(c *gin.Context) (int, error) {
	if rawID, exists := c.Get(middleware.UserIDKey); exists {
		if id, ok := rawID.(int); ok {
			return id, nil
		}
		return 0, ErrInvalidUserID
	}

	// Fallback to session lookup if context not populated
	session := sessions.Default(c)
	userID := session.Get(middleware.UserIDKey)
	if userID == nil {
		return 0, ErrUnauthenticated
	}
	id, ok := userID.(int)
	if !ok {
		return 0, ErrInvalidUserID
	}
	return id, nil
}

// authzAdminChecker is the minimal capability needed from user service for admin checks.
// Any concrete user service that implements IsAdmin satisfies this interface.
type authzAdminChecker interface {
	IsAdmin(ctx context.Context, userID int) (bool, error)
}

// RequireSelfOrAdmin permits the action if the current user is the target user
// or has admin privileges. Returns ErrForbidden when neither condition is met.
func RequireSelfOrAdmin(ctx context.Context, svc authzAdminChecker, currentID, targetID int) error {
	if currentID == 0 {
		return ErrUnauthenticated
	}
	if currentID == targetID {
		return nil
	}

	isAdmin, err := svc.IsAdmin(ctx, currentID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrForbidden
	}
	return nil
}
