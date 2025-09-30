package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStandardizeHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid input", "Field 'name' is required")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid input", response["message"])
	assert.Equal(t, "Field 'name' is required", response["details"])
}

func TestStandardizeHTTPError_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		StandardizeHTTPError(c, http.StatusInternalServerError, "Database error", "Connection timeout")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Database error", response["message"])
	assert.Equal(t, "Connection timeout", response["details"])
}

func TestStandardizeHTTPError_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		StandardizeHTTPError(c, http.StatusNotFound, "Resource not found", "User with ID 123 does not exist")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Resource not found", response["message"])
	assert.Equal(t, "User with ID 123 does not exist", response["details"])
}

func TestHandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		HandleValidationError(c, "email", "invalid-email", "must be a valid email address")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid email", response["message"])
	assert.Equal(t, "Value 'invalid-email' is invalid: must be a valid email address", response["details"])
}

func TestHandleValidationError_EmptyValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		HandleValidationError(c, "username", "", "cannot be empty")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid username", response["message"])
	assert.Equal(t, "Value '' is invalid: cannot be empty", response["details"])
}

func TestHandleValidationError_NumericValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		HandleValidationError(c, "age", 150, "must be between 0 and 120")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid age", response["message"])
	assert.Equal(t, "Value '150' is invalid: must be between 0 and 120", response["details"])
}

func TestHandleValidationError_ComplexValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		HandleValidationError(c, "settings", map[string]interface{}{"key": "value"}, "must be a valid JSON object")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid settings", response["message"])
	assert.Contains(t, response["details"], "must be a valid JSON object")
}

func TestErrorUtils_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		StandardizeHTTPError(c, http.StatusBadRequest, "Test error", "Test details")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestErrorUtils_ResponseStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		StandardizeHTTPError(c, http.StatusBadRequest, "Test error", "Test details")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check that fields exist
	assert.Contains(t, response, "code")
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "severity")
	assert.Contains(t, response, "retryable")
}

func TestBadRequestHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/bad", func(c *gin.Context) {
		HandleAppError(c, contextutils.ErrInvalidInput)
	})

	req, _ := http.NewRequest("GET", "/bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_INPUT", response["code"])
	assert.Contains(t, response, "message")
}

func TestUnauthorizedHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/unauth", func(c *gin.Context) {
		HandleAppError(c, contextutils.ErrUnauthorized)
	})

	req, _ := http.NewRequest("GET", "/unauth", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response["code"])
	assert.Contains(t, response, "message")
}
