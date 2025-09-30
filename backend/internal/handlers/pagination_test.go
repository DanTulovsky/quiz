package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParsePagination_DefaultsAndBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var gotPage, gotSize int

	r.GET("/test", func(c *gin.Context) {
		gotPage, gotSize = ParsePagination(c, 1, 20, 100)
		c.Status(http.StatusOK)
	})

	// No params -> defaults
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 1, gotPage)
	assert.Equal(t, 20, gotSize)

	// Invalid -> defaults
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?page=abc&page_size=-5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 1, gotPage)
	assert.Equal(t, 20, gotSize)

	// Valid within bounds
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?page=3&page_size=50", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 3, gotPage)
	assert.Equal(t, 50, gotSize)

	// Over max -> clamped
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?page=2&page_size=5000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 2, gotPage)
	assert.Equal(t, 100, gotSize)
}

func TestParseFilters_OnlyNonEmptyTrimmed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var filters map[string]string

	r.GET("/filters", func(c *gin.Context) {
		filters = ParseFilters(c, "search", "type", "status", "language", "level", "extra")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	// Use percent-encoding for whitespace-only extra and spaces around language to avoid invalid URL
	req, _ := http.NewRequest("GET", "/filters?search=test&type=vocabulary&status=&language=%20it%20&level=B2&extra=%09%0A", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, map[string]string{
		"search":   "test",
		"type":     "vocabulary",
		"language": "it",
		"level":    "B2",
	}, filters)
}

func TestWritePaginated_BuildsStandardResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/paginated", func(c *gin.Context) {
		items := []int{1, 2, 3}
		pagination := map[string]int{"page": 1, "page_size": 20, "total": 3, "total_pages": 1}
		WritePaginated(c, "items", items, pagination, gin.H{"stats": gin.H{"foo": "bar"}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/paginated", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Shallow checks on payload shape
	body := w.Body.String()
	assert.Contains(t, body, "\"items\":[1,2,3]")
	assert.Contains(t, body, "\"pagination\":")
	assert.Contains(t, body, "\"stats\":")
}
