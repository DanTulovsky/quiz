package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"quizapp/internal/observability"

	"github.com/gin-gonic/gin"
)

// RouteInfo represents information about a single route
type RouteInfo struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	HandlerName string `json:"handler_name"`
}

// RouteListingHandler generates automatic route listings
type RouteListingHandler struct {
	serviceName string
	routes      []RouteInfo
}

// NewRouteListingHandler creates a new route listing handler
func NewRouteListingHandler(serviceName string) *RouteListingHandler {
	return &RouteListingHandler{
		serviceName: serviceName,
		routes:      []RouteInfo{},
	}
}

// CollectRoutes extracts all routes from a Gin engine
func (h *RouteListingHandler) CollectRoutes(engine *gin.Engine) {
	h.routes = []RouteInfo{}

	// Get all routes from the Gin engine
	routes := engine.Routes()

	for _, route := range routes {
		// Skip internal Gin routes
		if strings.HasPrefix(route.Path, "/debug/") {
			continue
		}

		h.routes = append(h.routes, RouteInfo{
			Method:      route.Method,
			Path:        route.Path,
			HandlerName: route.Handler,
		})
	}

	// Sort routes by path for better organization
	sort.Slice(h.routes, func(i, j int) bool {
		return h.routes[i].Path < h.routes[j].Path
	})
}

// GetRouteListingPage shows all available routes as HTML
func (h *RouteListingHandler) GetRouteListingPage(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_route_listing_page")
	defer observability.FinishSpan(span, nil)
	html := h.generateHTML()
	// Add no-cache headers
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.String(http.StatusOK, html)
}

// GetRouteListingJSON returns the route listing as JSON
func (h *RouteListingHandler) GetRouteListingJSON(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_route_listing_json")
	defer observability.FinishSpan(span, nil)
	c.JSON(http.StatusOK, h.routes)
}

// generateHTML creates an HTML page listing all routes
func (h *RouteListingHandler) generateHTML() string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + h.serviceName + ` - Available Routes</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; padding: 20px; background-color: #f8f9fa; color: #212529; }
        .container { max-width: 1200px; margin: auto; background: #fff; padding: 30px; border-radius: 8px; box-shadow: 0 4px 8px rgba(0,0,0,0.05); }
        h1 { color: #0056b3; border-bottom: 2px solid #dee2e6; padding-bottom: 10px; margin-bottom: 30px; }
        .service-info { background: #e7f3ff; padding: 15px; border-radius: 5px; margin-bottom: 30px; }
        .route-table { width: 100%; border-collapse: collapse; margin-bottom: 30px; }
        .route-table th, .route-table td { padding: 12px; text-align: left; border-bottom: 1px solid #dee2e6; }
        .route-table th { background-color: #f8f9fa; font-weight: 600; color: #495057; }
        .route-table tr:nth-child(even) { background-color: #f8f9fa; }
        .route-table tr:hover { background-color: #e9ecef; }
        .method { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; min-width: 60px; text-align: center; }
        .method-get { background-color: #d4edda; color: #155724; }
        .method-post { background-color: #cce5ff; color: #004085; }
        .method-put { background-color: #fff3cd; color: #856404; }
        .method-delete { background-color: #f8d7da; color: #721c24; }
        .method-patch { background-color: #e2e3e5; color: #383d41; }
        .path { font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace; font-size: 14px; color: #6f42c1; }
        .clickable-path { cursor: pointer; text-decoration: underline; }
        .clickable-path:hover { background-color: #f8f9fa; }
        .footer { margin-top: 30px; text-align: center; color: #6c757d; font-size: 14px; }
        .stats { display: flex; gap: 20px; margin-bottom: 20px; }
        .stat-box { background: #ffffff; border: 1px solid #dee2e6; padding: 15px; border-radius: 5px; text-align: center; flex: 1; }
        .stat-number { font-size: 24px; font-weight: bold; color: #0056b3; }
        .stat-label { color: #6c757d; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>` + h.serviceName + ` Service - Available Routes</h1>

        <div class="service-info">
            <strong>Service:</strong> ` + h.serviceName + `<br>
            <strong>Generated:</strong> ` + time.Now().Format("2006-01-02 15:04:05") + `<br>
            <strong>Total Routes:</strong> ` + fmt.Sprintf("%d", len(h.routes)) + `
        </div>

        <div class="stats">
            <div class="stat-box">
                <div class="stat-number">` + fmt.Sprintf("%d", len(h.routes)) + `</div>
                <div class="stat-label">Total Routes</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">` + fmt.Sprintf("%d", h.countMethods("GET")) + `</div>
                <div class="stat-label">GET Routes</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">` + fmt.Sprintf("%d", h.countMethods("POST")) + `</div>
                <div class="stat-label">POST Routes</div>
            </div>
        </div>

        <table class="route-table">
            <thead>
                <tr>
                    <th>Method</th>
                    <th>Path</th>
                    <th>Handler</th>
                </tr>
            </thead>
            <tbody>`)

	for _, route := range h.routes {
		methodClass := "method-" + strings.ToLower(route.Method)
		pathClass := "path"

		// Make paths clickable for GET routes
		if route.Method == "GET" {
			pathClass += " clickable-path"
		}

		html.WriteString(fmt.Sprintf(`
                <tr>
                    <td><span class="method %s">%s</span></td>
                    <td><span class="%s" onclick="navigateToRoute('%s', '%s')">%s</span></td>
                    <td>%s</td>
                </tr>`,
			methodClass, route.Method,
			pathClass, route.Method, route.Path, route.Path,
			route.HandlerName,
		))
	}

	html.WriteString(`
            </tbody>
        </table>

        <div class="footer">
            <p>Click on any GET route path to navigate to it | <a href="/?json=true">View as JSON</a></p>
        </div>
    </div>

    <script>
        function navigateToRoute(method, path) {
            if (method === 'GET') {
                window.location.href = path;
            } else {
                alert('Only GET routes can be navigated to directly. Use API client for ' + method + ' requests.');
            }
        }
    </script>
</body>
</html>`)

	return html.String()
}

// countMethods counts routes by HTTP method
func (h *RouteListingHandler) countMethods(method string) int {
	count := 0
	for _, route := range h.routes {
		if route.Method == method {
			count++
		}
	}
	return count
}
