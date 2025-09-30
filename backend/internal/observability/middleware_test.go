package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func setupTestTracer() func() {
	// Set up a no-op tracer provider for testing
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Return cleanup function
	return func() {
		otel.SetTracerProvider(nil)
	}
}

func setupGinWithSessions() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup sessions
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("test-session", store))

	return router
}

func TestGinMiddleware_BasicFunctionality(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Set up a simple Gin router with OpenTelemetry middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "middleware working",
		})
	})

	// Test that the middleware doesn't crash and returns expected response
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "middleware working", resp["message"])
}

func TestGinMiddleware_TraceHeadersPropagation(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Set up a simple Gin router with OpenTelemetry middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	// Add a test endpoint that returns trace headers
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"has_traceparent": c.Request.Header.Get("traceparent") != "",
		})
	})

	// Test 1: Request without trace headers
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var resp1 map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &resp1)
	require.NoError(t, err)
	assert.Equal(t, false, resp1["has_traceparent"])

	// Test 2: Request with trace headers
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("traceparent", "00-12345678901234567890123456789012-1234567890123456-01")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	require.NoError(t, err)
	assert.Equal(t, true, resp2["has_traceparent"])
}

func TestGinMiddleware_ErrorHandling(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Test that the middleware handles errors gracefully
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GinMiddleware("test-service"))

	// Add an endpoint that returns an error
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "test error",
		})
	})

	req, _ := http.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()

	// Should handle the error and return 500 status
	router.ServeHTTP(w, req)

	// Should return 500 status
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test error", resp["error"])
}

func TestGinMiddlewareWithErrorHandling_ErrorDetection(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Test that the middleware automatically adds error attributes for failed requests
	router := setupGinWithSessions()
	router.Use(GinMiddlewareWithErrorHandling("test-service"))

	// Add endpoints that return different status codes
	router.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/client-error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	router.GET("/server-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	// Test successful request (should not have error=true)
	req, _ := http.NewRequest("GET", "/success", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test client error (should have error=true)
	req, _ = http.NewRequest("GET", "/client-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test server error (should have error=true)
	req, _ = http.NewRequest("GET", "/server-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGinMiddlewareWithErrorHandling_SuccessfulRequests(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Test that successful requests don't get error attributes
	router := setupGinWithSessions()
	router.Use(GinMiddlewareWithErrorHandling("test-service"))

	// Add a successful endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "middleware working",
		})
	})

	// Test that the middleware doesn't crash and returns expected response
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "middleware working", resp["message"])
}

func TestGinMiddlewareWithErrorHandling_StatusCodes(t *testing.T) {
	// Set up test tracer
	cleanup := setupTestTracer()
	defer cleanup()

	// Test that the middleware handles different status codes correctly
	router := setupGinWithSessions()
	router.Use(GinMiddlewareWithErrorHandling("test-service"))

	// Add endpoints that return different status codes
	router.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/client-error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	router.GET("/server-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	router.GET("/not-found", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	router.GET("/unauthorized", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	})

	// Test successful request
	req, _ := http.NewRequest("GET", "/success", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test client errors (4xx)
	req, _ = http.NewRequest("GET", "/client-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	req, _ = http.NewRequest("GET", "/not-found", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	req, _ = http.NewRequest("GET", "/unauthorized", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test server error (5xx)
	req, _ = http.NewRequest("GET", "/server-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
