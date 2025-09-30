package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultErrorRecoveryConfig(t *testing.T) {
	config := DefaultErrorRecoveryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.RetryDelay)
	assert.Equal(t, 5*time.Second, config.MaxRetryDelay)
	assert.False(t, config.EnableCircuitBreaker)
	assert.Equal(t, 5, config.CircuitBreakerThreshold)
	assert.Equal(t, 30*time.Second, config.CircuitBreakerTimeout)
}

func TestErrorRecoveryMiddleware_PanicRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a router with panic recovery middleware
	router := gin.New()
	router.Use(ErrorRecoveryMiddleware(nil, nil))

	router.GET("/panic", func(_ *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 500 with error message
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestErrorRecoveryMiddleware_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorRecoveryMiddleware(nil, nil))

	router.GET("/normal", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/normal", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCircuitBreaker_CanExecute(t *testing.T) {
	config := &ErrorRecoveryConfig{
		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 2,
		CircuitBreakerTimeout:   100 * time.Millisecond,
	}

	cb := newCircuitBreaker(config)

	// Initially closed, should allow execution
	assert.True(t, cb.canExecute())
	assert.Equal(t, circuitClosed, cb.state)

	// Record failures
	cb.recordFailure()
	cb.recordFailure()

	// Should be open now
	assert.False(t, cb.canExecute())
	assert.Equal(t, circuitOpen, cb.state)

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should be half-open now
	assert.True(t, cb.canExecute())
	assert.Equal(t, circuitHalfOpen, cb.state)

	// Record success
	cb.recordSuccess()

	// Should be closed again
	assert.True(t, cb.canExecute())
	assert.Equal(t, circuitClosed, cb.state)
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errors     []*gin.Error
		expected   bool
	}{
		{
			name:       "5xx error",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "timeout error",
			statusCode: http.StatusRequestTimeout,
			expected:   true,
		},
		{
			name:       "rate limit error",
			statusCode: http.StatusTooManyRequests,
			expected:   true,
		},
		{
			name:       "4xx error",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "2xx success",
			statusCode: http.StatusOK,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.statusCode, tt.errors)
			assert.Equal(t, tt.expected, result)
		})
	}
}
