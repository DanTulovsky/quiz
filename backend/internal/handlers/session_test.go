package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quizapp/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))
	return r
}

func TestGetUserIDFromSession_NoUser(t *testing.T) {
	router := setupSessionTestRouter()

	router.GET("/check", func(c *gin.Context) {
		id, ok := GetUserIDFromSession(c)
		c.JSON(http.StatusOK, gin.H{"ok": ok, "id": id})
	})

	req, _ := http.NewRequest("GET", "/check", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])
	assert.Equal(t, float64(0), resp["id"]) // json unmarshals numbers as float64
}

func TestGetUserIDFromSession_ValidInt(t *testing.T) {
	router := setupSessionTestRouter()

	router.GET("/set-and-check", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(middleware.UserIDKey, 42)
		_ = session.Save()
		id, ok := GetUserIDFromSession(c)
		c.JSON(http.StatusOK, gin.H{"ok": ok, "id": id})
	})

	req, _ := http.NewRequest("GET", "/set-and-check", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["ok"])
	assert.Equal(t, float64(42), resp["id"]) // json unmarshals numbers as float64
}

func TestGetUserIDFromSession_InvalidTypeFloat(t *testing.T) {
	router := setupSessionTestRouter()

	router.GET("/set-float-and-check", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(middleware.UserIDKey, float64(7))
		_ = session.Save()
		id, ok := GetUserIDFromSession(c)
		c.JSON(http.StatusOK, gin.H{"ok": ok, "id": id})
	})

	req, _ := http.NewRequest("GET", "/set-float-and-check", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])      // helper only accepts int type
	assert.Equal(t, float64(0), resp["id"]) // 0 when invalid
}
