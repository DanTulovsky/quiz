package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"quizapp/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockAdminChecker is a minimal mock for authzAdminChecker
type mockAdminChecker struct {
	isAdmin bool
	err     error
}

func (m *mockAdminChecker) IsAdmin(_ context.Context, _ int) (bool, error) {
	return m.isAdmin, m.err
}

func setupGinWithSessions() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))
	return r
}

func TestGetCurrentUserID_Context(t *testing.T) {
	r := setupGinWithSessions()
	r.GET("/test", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, 42)
		id, err := GetCurrentUserID(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "\"id\":42")
}

func TestGetCurrentUserID_SessionFallback(t *testing.T) {
	r := setupGinWithSessions()
	r.GET("/test", func(c *gin.Context) {
		// No context value; set session value and then read via helper
		sess := sessions.Default(c)
		sess.Set(middleware.UserIDKey, 99)
		_ = sess.Save()

		id, err := GetCurrentUserID(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "\"id\":99")
}

func TestGetCurrentUserID_Unauthenticated(t *testing.T) {
	r := setupGinWithSessions()
	r.GET("/test", func(c *gin.Context) {
		_, err := GetCurrentUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), ErrUnauthenticated.Error())
}

func TestGetCurrentUserID_InvalidType(t *testing.T) {
	r := setupGinWithSessions()
	r.GET("/test", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, "not-an-int")
		_, err := GetCurrentUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), ErrInvalidUserID.Error())
}

func TestRequireSelfOrAdmin(t *testing.T) {
	t.Run("self allowed", func(t *testing.T) {
		svc := &mockAdminChecker{isAdmin: false}
		err := RequireSelfOrAdmin(context.Background(), svc, 7, 7)
		assert.NoError(t, err)
	})

	t.Run("admin allowed", func(t *testing.T) {
		svc := &mockAdminChecker{isAdmin: true}
		err := RequireSelfOrAdmin(context.Background(), svc, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("forbidden when not admin and not self", func(t *testing.T) {
		svc := &mockAdminChecker{isAdmin: false}
		err := RequireSelfOrAdmin(context.Background(), svc, 1, 2)
		assert.Error(t, err)
		assert.Equal(t, ErrForbidden, err)
	})
}
