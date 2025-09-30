// Package middleware provides authentication and authorization middleware for the Gin web framework.
package middleware

import (
	"context"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Session keys for storing user information
const (
	// UserIDKey is the key used to store user ID in session
	UserIDKey = "user_id"
	// UsernameKey is the key used to store username in session
	UsernameKey = "username"
)

// RequireAuth returns a middleware that requires authentication
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
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
