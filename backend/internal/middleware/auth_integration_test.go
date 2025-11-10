//go:build integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setupGinWithAuth() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Setup sessions
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("test-session", store))

	return router
}

func TestRequireAuth_AuthenticatedUser_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		require.True(t, exists)
		username, exists := c.Get("username")
		require.True(t, exists)

		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"message":  "access granted",
		})
	})

	// First request to set up session
	req1, _ := http.NewRequest("GET", "/setup-session", nil)
	w1 := httptest.NewRecorder()

	router.GET("/setup-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", 1)
		session.Set("username", "testuser")
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "session set"})
	})

	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Second request with session cookie
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify response contains user data
	assert.Contains(t, w2.Body.String(), "testuser")
	assert.Contains(t, w2.Body.String(), "access granted")
}

func TestRequireAuth_UnauthenticatedUser_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Request without authentication
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestRequireAuth_InvalidSession_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Create a request with an invalid session cookie
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "test-session",
		Value: "invalid-session-data",
		Path:  "/",
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestRequireAuth_PartialSession_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Set up session with missing fields
	router.GET("/setup-partial-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", 1)
		// Missing username - should cause auth failure
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "partial session set"})
	})

	// First request to set up partial session
	req1, _ := http.NewRequest("GET", "/setup-partial-session", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Second request with partial session
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
	assert.Contains(t, w2.Body.String(), "Authentication required")
}

func TestAuth_ContextValues_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Route that checks context values
	router.GET("/check-context", RequireAuth(), func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, 42, userID)

		username, exists := c.Get("username")
		assert.True(t, exists)
		assert.Equal(t, "contextuser", username)

		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	// Set up session with specific values
	router.GET("/setup-context-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", 42)
		session.Set("username", "contextuser")
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "context session set"})
	})

	// First request to set up session
	req1, _ := http.NewRequest("GET", "/setup-context-session", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Second request to check context
	req2, _ := http.NewRequest("GET", "/check-context", nil)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "contextuser")
	assert.Contains(t, w2.Body.String(), "42")
}

func TestAuth_SessionCorruption_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	// Set up session with wrong data types
	router.GET("/setup-corrupted-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", "not-an-integer") // Wrong type
		session.Set("username", 12345)           // Wrong type
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "corrupted session set"})
	})

	// First request to set up corrupted session
	req1, _ := http.NewRequest("GET", "/setup-corrupted-session", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Second request with corrupted session
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestAuth_MultipleRequests_SameSession_Integration(t *testing.T) {
	router := setupGinWithAuth()

	// Protected route that increments a counter
	counter := 0
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		counter++
		c.JSON(http.StatusOK, gin.H{
			"counter": counter,
			"user":    c.GetString("username"),
		})
	})

	// Set up session
	router.GET("/setup-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", 1)
		session.Set("username", "testuser")
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "session set"})
	})

	// First request to set up session
	req1, _ := http.NewRequest("GET", "/setup-session", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Multiple requests with same session
	for i := 1; i <= 3; i++ {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "testuser")
		assert.Contains(t, w.Body.String(), `"counter":`+string(rune('0'+i)))
	}
}
