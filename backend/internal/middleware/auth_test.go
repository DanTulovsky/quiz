package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAPIKeyService struct {
	result          *models.AuthAPIKey
	err             error
	lastValidated   string
	updateCallsChan chan int
}

func newMockAPIKeyService(result *models.AuthAPIKey, err error) *mockAPIKeyService {
	return &mockAPIKeyService{
		result:          result,
		err:             err,
		updateCallsChan: make(chan int, 1),
	}
}

func (m *mockAPIKeyService) ValidateAPIKey(_ context.Context, rawKey string) (*models.AuthAPIKey, error) {
	m.lastValidated = rawKey
	return m.result, m.err
}

func (m *mockAPIKeyService) UpdateLastUsed(_ context.Context, keyID int) error {
	if m.updateCallsChan != nil {
		m.updateCallsChan <- keyID
	}
	return nil
}

type mockUserService struct {
	user       *models.User
	err        error
	callCount  int
	lastUserID int
}

func (m *mockUserService) GetUserByID(_ context.Context, userID int) (*models.User, error) {
	m.callCount++
	m.lastUserID = userID
	return m.user, m.err
}

type mockAdminService struct {
	isAdmin    bool
	err        error
	lastUserID int
}

func (m *mockAdminService) IsAdmin(_ context.Context, userID int) (bool, error) {
	m.lastUserID = userID
	return m.isAdmin, m.err
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))
	return router
}

func setSessionCookie(t *testing.T, router *gin.Engine, values map[string]interface{}) *http.Cookie {
	setupPath := "/setup-session-" + t.Name()
	router.GET(setupPath, func(c *gin.Context) {
		session := sessions.Default(c)
		for k, v := range values {
			session.Set(k, v)
		}
		require.NoError(t, session.Save())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", setupPath, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	cookies := w.Result().Cookies()
	require.NotEmpty(t, cookies)
	return cookies[0]
}

func TestRequireAuthWithAPIKey_BearerTokenSuccess(t *testing.T) {
	router := newTestRouter()

	apiKey := &models.AuthAPIKey{
		ID:              10,
		UserID:          42,
		PermissionLevel: models.PermissionLevelReadonly,
	}
	mockAPI := newMockAPIKeyService(apiKey, nil)
	mockUser := &mockUserService{
		user: &models.User{
			ID:       42,
			Username: "apiuser",
		},
	}

	router.GET("/resource", RequireAuthWithAPIKey(mockAPI, mockUser), func(c *gin.Context) {
		assert.Equal(t, 42, c.GetInt(UserIDKey))
		assert.Equal(t, "apiuser", c.GetString(UsernameKey))
		assert.Equal(t, AuthMethodAPIKey, c.GetString(AuthMethodKey))
		assert.Equal(t, apiKey.ID, c.GetInt(APIKeyIDKey))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/resource", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-key", mockAPI.lastValidated)
	assert.Equal(t, 1, mockUser.callCount)
	select {
	case keyID := <-mockAPI.updateCallsChan:
		assert.Equal(t, apiKey.ID, keyID)
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected UpdateLastUsed to be called")
	}
}

func TestRequireAuthWithAPIKey_ForbiddenForReadonlyOnWrite(t *testing.T) {
	router := newTestRouter()

	apiKey := &models.AuthAPIKey{
		ID:              5,
		UserID:          7,
		PermissionLevel: models.PermissionLevelReadonly,
	}
	mockAPI := newMockAPIKeyService(apiKey, nil)
	mockUser := &mockUserService{
		user: &models.User{ID: 7, Username: "readonly"},
	}

	router.POST("/resource", RequireAuthWithAPIKey(mockAPI, mockUser), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should": "not happen"})
	})

	req := httptest.NewRequest("POST", "/resource", nil)
	req.Header.Set("Authorization", "Bearer write-attempt")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "does not have permission")
	assert.Equal(t, "write-attempt", mockAPI.lastValidated)
	assert.Equal(t, 0, mockUser.callCount)
}

func TestRequireAuthWithAPIKey_InvalidKey(t *testing.T) {
	router := newTestRouter()

	mockAPI := newMockAPIKeyService(nil, errors.New("invalid key"))
	mockUser := &mockUserService{}

	router.GET("/resource", RequireAuthWithAPIKey(mockAPI, mockUser), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should": "not happen"})
	})

	req := httptest.NewRequest("GET", "/resource", nil)
	req.Header.Set("Authorization", "Bearer bad-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key")
	assert.Equal(t, "bad-key", mockAPI.lastValidated)
	assert.Equal(t, 0, mockUser.callCount)
}

func TestRequireAuthWithAPIKey_FallsBackToSession(t *testing.T) {
	router := newTestRouter()

	mockAPI := newMockAPIKeyService(nil, nil)
	mockUser := &mockUserService{}

	router.GET("/protected", RequireAuthWithAPIKey(mockAPI, mockUser), func(c *gin.Context) {
		assert.Equal(t, 99, c.GetInt(UserIDKey))
		assert.Equal(t, "session-user", c.GetString(UsernameKey))
		assert.Equal(t, AuthMethodSession, c.GetString(AuthMethodKey))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	sessionCookie := setSessionCookie(t, router, map[string]interface{}{
		UserIDKey:   99,
		UsernameKey: "session-user",
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, mockAPI.lastValidated)
	assert.Equal(t, 0, len(mockAPI.updateCallsChan))
}

func TestRequireAdmin_AllowsAdminUsers(t *testing.T) {
	router := newTestRouter()

	mockAdmin := &mockAdminService{isAdmin: true}

	router.GET("/admin", RequireAdmin(mockAdmin), func(c *gin.Context) {
		assert.Equal(t, 123, c.GetInt(UserIDKey))
		assert.Equal(t, "admin-user", c.GetString(UsernameKey))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	sessionCookie := setSessionCookie(t, router, map[string]interface{}{
		UserIDKey:   123,
		UsernameKey: "admin-user",
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 123, mockAdmin.lastUserID)
}

func TestRequireAdmin_ForbiddenForNonAdmin(t *testing.T) {
	router := newTestRouter()

	mockAdmin := &mockAdminService{isAdmin: false}

	router.GET("/admin", RequireAdmin(mockAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should": "not happen"})
	})

	sessionCookie := setSessionCookie(t, router, map[string]interface{}{
		UserIDKey:   9,
		UsernameKey: "user",
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Admin access required")
	assert.Equal(t, 9, mockAdmin.lastUserID)
}

func TestRequireAdmin_InternalError(t *testing.T) {
	router := newTestRouter()

	mockAdmin := &mockAdminService{
		err: errors.New("lookup failure"),
	}

	router.GET("/admin", RequireAdmin(mockAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should": "not happen"})
	})

	sessionCookie := setSessionCookie(t, router, map[string]interface{}{
		UserIDKey:   55,
		UsernameKey: "user55",
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to check admin status")
	assert.Equal(t, 55, mockAdmin.lastUserID)
}

func TestRequireAdmin_Unauthenticated(t *testing.T) {
	router := newTestRouter()

	mockAdmin := &mockAdminService{isAdmin: true}

	router.GET("/admin", RequireAdmin(mockAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should": "not happen"})
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, mockAdmin.lastUserID)
}

