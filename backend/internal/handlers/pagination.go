package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParsePagination parses standard pagination query params from the request.
// It enforces bounds and applies defaults when values are missing or invalid.
func ParsePagination(c *gin.Context, defaultPage, defaultSize, maxSize int) (int, int) {
	pageStr := c.DefaultQuery("page", strconv.Itoa(defaultPage))
	sizeStr := c.DefaultQuery("page_size", strconv.Itoa(defaultSize))

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = defaultPage
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}

	return page, size
}

// ParseFilters returns a map of non-empty trimmed query params for the given keys.
func ParseFilters(c *gin.Context, keys ...string) map[string]string {
	filters := make(map[string]string, len(keys))
	for _, key := range keys {
		if val := strings.TrimSpace(c.Query(key)); val != "" {
			filters[key] = val
		}
	}
	return filters
}

// WritePaginated standardizes paginated responses with a flexible items key, pagination block, and optional extras.
// It preserves existing API response shapes by allowing the caller to specify the items key.
func WritePaginated(c *gin.Context, itemsKey string, items, pagination any, extra gin.H) {
	response := gin.H{
		itemsKey:     items,
		"pagination": pagination,
	}
	for k, v := range extra {
		response[k] = v
	}
	c.JSON(http.StatusOK, response)
}
