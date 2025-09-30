package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewRouteListingHandler(t *testing.T) {
	handler := NewRouteListingHandler("Test Service")
	assert.NotNil(t, handler)
	assert.Equal(t, "Test Service", handler.serviceName)
	assert.NotNil(t, handler.routes)
}

func TestRouteListingHandler_CollectRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add some test routes
	router.GET("/", func(_ *gin.Context) {})
	router.POST("/test", func(_ *gin.Context) {})
	router.PUT("/test/:id", func(_ *gin.Context) {})
	router.DELETE("/test/:id", func(_ *gin.Context) {})

	// Add grouped routes
	v1 := router.Group("/v1")
	{
		v1.GET("/users", func(_ *gin.Context) {})
		v1.POST("/users", func(_ *gin.Context) {})
		v1.PUT("/users/:id", func(_ *gin.Context) {})
		v1.DELETE("/users/:id", func(_ *gin.Context) {})
	}

	handler := NewRouteListingHandler("Test Service")
	handler.CollectRoutes(router)

	// Check that routes were collected
	assert.Len(t, handler.routes, 8)

	// Check for specific routes by method and path combination
	foundRoutes := make(map[string]bool)
	for _, route := range handler.routes {
		key := route.Method + " " + route.Path
		foundRoutes[key] = true
	}

	assert.True(t, foundRoutes["GET /"])
	assert.True(t, foundRoutes["POST /test"])
	assert.True(t, foundRoutes["PUT /test/:id"])
	assert.True(t, foundRoutes["DELETE /test/:id"])
	assert.True(t, foundRoutes["GET /v1/users"])
	assert.True(t, foundRoutes["POST /v1/users"])
	assert.True(t, foundRoutes["PUT /v1/users/:id"])
	assert.True(t, foundRoutes["DELETE /v1/users/:id"])
}

func TestRouteListingHandler_GetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add test routes
	router.GET("/", func(_ *gin.Context) {})
	router.POST("/test", func(_ *gin.Context) {})

	handler := NewRouteListingHandler("Test Service")
	handler.CollectRoutes(router)

	// Check that routes were collected
	assert.Len(t, handler.routes, 2)

	// Check route structure
	for _, route := range handler.routes {
		assert.NotEmpty(t, route.Method)
		assert.NotEmpty(t, route.Path)
	}
}

func TestRouteListingHandler_GetRouteListingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add test routes
	router.GET("/", func(_ *gin.Context) {})
	router.POST("/test", func(_ *gin.Context) {})
	router.PUT("/test/:id", func(_ *gin.Context) {})

	handler := NewRouteListingHandler("Test Service")
	handler.CollectRoutes(router)

	// Test the handler endpoint
	req, _ := http.NewRequest("GET", "/routes", nil)
	w := httptest.NewRecorder()

	router.GET("/routes", handler.GetRouteListingJSON)
	router.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Check content type
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Parse response
	var routes []RouteInfo
	err := json.Unmarshal(w.Body.Bytes(), &routes)
	assert.NoError(t, err)

	// Check response structure
	assert.Len(t, routes, 3)

	// Check route structure
	for _, route := range routes {
		assert.NotEmpty(t, route.Method)
		assert.NotEmpty(t, route.Path)
	}
}

func TestRouteListingHandler_EmptyRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewRouteListingHandler("Empty Service")
	handler.CollectRoutes(router)

	assert.Len(t, handler.routes, 0)
}

func TestRouteListingHandler_ComplexRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add complex nested routes
	api := router.Group("/api")
	v1 := api.Group("/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("", func(_ *gin.Context) {})
			users.POST("", func(_ *gin.Context) {})
			users.GET("/:id", func(_ *gin.Context) {})
			users.PUT("/:id", func(_ *gin.Context) {})
			users.DELETE("/:id", func(_ *gin.Context) {})
		}

		posts := v1.Group("/posts")
		{
			posts.GET("", func(_ *gin.Context) {})
			posts.POST("", func(_ *gin.Context) {})
			posts.GET("/:id", func(_ *gin.Context) {})
			posts.PUT("/:id", func(_ *gin.Context) {})
			posts.DELETE("/:id", func(_ *gin.Context) {})
		}
	}

	handler := NewRouteListingHandler("Complex Service")
	handler.CollectRoutes(router)

	assert.Len(t, handler.routes, 10)

	// Check for specific nested routes by method and path combination
	foundRoutes := make(map[string]bool)
	for _, route := range handler.routes {
		key := route.Method + " " + route.Path
		foundRoutes[key] = true
	}

	assert.True(t, foundRoutes["GET /api/v1/users"])
	assert.True(t, foundRoutes["POST /api/v1/users"])
	assert.True(t, foundRoutes["GET /api/v1/users/:id"])
	assert.True(t, foundRoutes["PUT /api/v1/users/:id"])
	assert.True(t, foundRoutes["DELETE /api/v1/users/:id"])
	assert.True(t, foundRoutes["GET /api/v1/posts"])
	assert.True(t, foundRoutes["POST /api/v1/posts"])
	assert.True(t, foundRoutes["GET /api/v1/posts/:id"])
	assert.True(t, foundRoutes["PUT /api/v1/posts/:id"])
	assert.True(t, foundRoutes["DELETE /api/v1/posts/:id"])
}

func TestRouteListingHandler_DifferentHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add routes with different HTTP methods
	router.GET("/test", func(_ *gin.Context) {})
	router.POST("/test", func(_ *gin.Context) {})
	router.PUT("/test", func(_ *gin.Context) {})
	router.PATCH("/test", func(_ *gin.Context) {})
	router.DELETE("/test", func(_ *gin.Context) {})
	router.HEAD("/test", func(_ *gin.Context) {})
	router.OPTIONS("/test", func(_ *gin.Context) {})

	handler := NewRouteListingHandler("HTTP Methods Service")
	handler.CollectRoutes(router)

	assert.Len(t, handler.routes, 7)

	// Check for all HTTP methods
	methods := make(map[string]bool)
	for _, route := range handler.routes {
		methods[route.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["HEAD"])
	assert.True(t, methods["OPTIONS"])
}

func TestRouteListingHandler_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add a simple route
	router.GET("/test", func(_ *gin.Context) {})

	handler := NewRouteListingHandler("Format Test Service")
	handler.CollectRoutes(router)

	// Test the handler endpoint
	req, _ := http.NewRequest("GET", "/routes", nil)
	w := httptest.NewRecorder()

	router.GET("/routes", handler.GetRouteListingJSON)
	router.ServeHTTP(w, req)

	var routes []RouteInfo
	err := json.Unmarshal(w.Body.Bytes(), &routes)
	assert.NoError(t, err)

	// Check response format
	assert.Len(t, routes, 1)

	// Check first route structure
	route := routes[0]
	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/test", route.Path)
}
