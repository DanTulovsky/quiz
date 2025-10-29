package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"quizapp/internal/observability"

	"github.com/gin-gonic/gin"
)

// Global schema loader instance
var globalSchemaLoader *SchemaLoader

// initSchemaLoader initializes the global schema loader once
func initSchemaLoader() *SchemaLoader {
	if globalSchemaLoader == nil {
		globalSchemaLoader = AutoLoadSchemas()
	}
	return globalSchemaLoader
}

// ResponseValidationMiddleware creates middleware that automatically validates responses
func ResponseValidationMiddleware(logger *observability.Logger) gin.HandlerFunc {
	// Initialize schema loader once
	schemaLoader := initSchemaLoader()

	return func(c *gin.Context) {
		// Start tracing span for validation
		ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "response_validation")
		defer span.End()

		// Store the original response writer
		originalWriter := c.Writer

		// Create a custom response writer that captures the response
		responseWriter := &responseCaptureWriter{
			ResponseWriter: originalWriter,
			body:           &bytes.Buffer{},
			status:         0,
		}

		// Replace the response writer
		c.Writer = responseWriter

		// Continue to the next handler
		c.Next()

		// After the response is written, validate it
		statusCode := responseWriter.status
		if statusCode == 0 {
			statusCode = c.Writer.Status()
		}

		// Only validate 2xx responses
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			// Skip validation for streaming responses
			contentType := c.Writer.Header().Get("Content-Type")
			if contentType == "text/event-stream" {
				span.SetAttributes(
					observability.AttributeTypeFilter("streaming_response"),
				)
				logger.Debug(ctx, "Skipping validation for streaming response", map[string]interface{}{
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
				})
				// Write the buffered response to the real writer
				c.Writer = originalWriter
				c.Writer.WriteHeader(statusCode)
				_, _ = c.Writer.Write(responseWriter.body.Bytes())
				return
			}

			// Try to parse the response as JSON
			var responseData interface{}
			err := json.Unmarshal(responseWriter.body.Bytes(), &responseData)
			if err == nil {
				// Automatically determine schema name from the endpoint
				schemaName := schemaLoader.DetermineSchemaFromPath(c.Request.URL.Path, c.Request.Method)

				// Add tracing attributes
				span.SetAttributes(
					observability.AttributeSearch(c.Request.URL.Path),
					observability.AttributeTypeFilter(c.Request.Method),
				)

				if schemaName != "" {
					span.SetAttributes(observability.AttributeSearch(schemaName))

					if err := schemaLoader.ValidateData(responseData, schemaName); err != nil {
						// Log the validation error and add tracing attributes
						span.SetAttributes(
							observability.AttributeTypeFilter("validation_failed"),
						)

						// Log the validation error and fail the request
						logger.Error(ctx, "Response validation failed", err, map[string]interface{}{
							"method":        c.Request.Method,
							"path":          c.Request.URL.Path,
							"schema_name":   schemaName,
							"error":         err.Error(),
							"response_data": responseWriter.body.String()[:int(math.Min(200, float64(responseWriter.body.Len())))],
						})

						// Write a 400 error response instead of the original response
						c.Writer = originalWriter
						c.Writer.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(c.Writer).Encode(gin.H{
							"error":   "Response validation failed",
							"message": "API response does not match the specification",
							"method":  c.Request.Method,
							"path":    c.Request.URL.Path,
							"schema":  schemaName,
							"details": err.Error(),
						})
						return
					}
					// Add success tracing attributes
					span.SetAttributes(
						observability.AttributeTypeFilter("validation_passed"),
					)

					// Write the buffered response to the real writer
					c.Writer = originalWriter
					c.Writer.WriteHeader(statusCode)
					_, _ = c.Writer.Write(responseWriter.body.Bytes())
					return
				}
				// No schema found for this endpoint
				span.SetAttributes(
					observability.AttributeTypeFilter("no_schema_found"),
				)

				logger.Warn(ctx, "No schema found for endpoint", map[string]interface{}{
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
				})
				// Write the buffered response to the real writer
				c.Writer = originalWriter
				c.Writer.WriteHeader(statusCode)
				_, _ = c.Writer.Write(responseWriter.body.Bytes())
				return
			}
			// Failed to parse JSON response
			span.SetAttributes(
				observability.AttributeTypeFilter("json_parse_failed"),
			)

			logger.Error(ctx, "Failed to parse JSON response", err, map[string]interface{}{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			})
			// Write the buffered response to the real writer
			c.Writer = originalWriter
			c.Writer.WriteHeader(statusCode)
			_, _ = c.Writer.Write(responseWriter.body.Bytes())
			return
		}
		// Non-200 status code, skip validation
		span.SetAttributes(
			observability.AttributeTypeFilter("non_200_status"),
		)
		// Write the buffered response to the real writer
		c.Writer = originalWriter
		c.Writer.WriteHeader(statusCode)
		_, _ = c.Writer.Write(responseWriter.body.Bytes())
	}
}

// responseCaptureWriter captures the response body for validation
// Add a status field to track the status code
type responseCaptureWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseCaptureWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseCaptureWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *responseCaptureWriter) Status() int {
	if w.status != 0 {
		return w.status
	}
	return w.ResponseWriter.Status()
}

// isStaticFile checks if a path is a static file that should be allowed to pass through
func isStaticFile(path string) bool {
	staticPaths := []string{
		"/swagger.yaml",
		"/swaggerz",
		"/configz",
		"/",
	}

	for _, staticPath := range staticPaths {
		if path == staticPath {
			return true
		}
	}

	// Also allow paths that start with /backend/ (static assets)
	if strings.HasPrefix(path, "/backend/") {
		return true
	}

	return false
}

// RequestValidationMiddleware creates middleware that prevents undocumented API calls
func RequestValidationMiddleware(logger *observability.Logger) gin.HandlerFunc {
	// Initialize schema loader once
	schemaLoader := initSchemaLoader()

	return func(c *gin.Context) {
		// Start tracing span for request validation
		ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "request_validation")
		defer span.End()

		// Check if the endpoint exists in the swagger spec
		path := c.Request.URL.Path
		method := c.Request.Method

		// Log all requests for debugging
		logger.Info(ctx, "Request validation middleware called", map[string]interface{}{
			"method": method,
			"path":   path,
		})

		// Add tracing attributes
		span.SetAttributes(
			observability.AttributeSearch(path),
			observability.AttributeTypeFilter(method),
		)

		// Allow static files to pass through
		if isStaticFile(path) {
			// Continue to the next handler
			c.Next()
			return
		}

		// Check if this endpoint is documented in swagger
		if !schemaLoader.IsEndpointDocumented(path, method) {
			// Log the undocumented API call
			logger.Warn(ctx, "Undocumented API call attempted", map[string]interface{}{
				"method":     method,
				"path":       path,
				"ip":         c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})

			// Return 404 for undocumented endpoints
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Endpoint not found",
				"message": "The requested endpoint is not documented in the API specification",
			})
			c.Abort()
			return
		}

		// Endpoint is documented, continue
		span.SetAttributes(
			observability.AttributeTypeFilter("endpoint_documented"),
		)

		// Validate request body against schema for POST/PUT/PATCH requests
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// Determine the request body schema name for this endpoint
			schemaName := schemaLoader.DetermineRequestSchemaFromPath(path, method)

			// Log the schema determination for debugging
			logger.Info(ctx, "Request validation schema determined", map[string]interface{}{
				"method":      method,
				"path":        path,
				"schema_name": schemaName,
			})

			// Log when no schema is found
			if schemaName == "" {
				logger.Warn(ctx, "No schema found for endpoint", map[string]interface{}{
					"method": method,
					"path":   path,
				})
			}

			// Restore the request body so handlers can read it
			body, err := c.GetRawData()
			if err == nil && len(body) > 0 {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}

			if schemaName != "" {
				// Read the request body without consuming it
				body, err := c.GetRawData()
				if err == nil && len(body) > 0 {
					// Restore the request body so handlers can read it
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

					// Log the raw request body for debugging
					logger.Info(ctx, "Request body received", map[string]interface{}{
						"method":      method,
						"path":        path,
						"schema_name": schemaName,
						"body":        string(body),
					})

					// Parse the JSON
					var requestData interface{}
					if err := json.Unmarshal(body, &requestData); err == nil {
						// Validate the request data against the schema
						if err := schemaLoader.ValidateData(requestData, schemaName); err != nil {
							// Log the validation error and the request data
							logger.Error(ctx, "Request validation failed", err, map[string]interface{}{
								"method":       method,
								"path":         path,
								"schema_name":  schemaName,
								"error":        err.Error(),
								"request_data": requestData,
								"raw_body":     string(body),
							})
							// Add validation error details to tracing span
							span.SetAttributes(
								observability.AttributeTypeFilter("validation_failed"),
								observability.AttributeSearch(path),
								observability.AttributeTypeFilter(method),
								observability.AttributeTypeFilter(schemaName),
								observability.AttributeTypeFilter("validation_error:"+err.Error()),
								observability.AttributeTypeFilter("request_data:"+fmt.Sprintf("%v", requestData)),
								observability.AttributeTypeFilter("raw_body:"+string(body)),
							)
							// Print a concise summary to stdout for test debug
							fmt.Printf("\n[VALIDATION ERROR] %v\n[REQUEST DATA] %v\n[RAW BODY] %s\n\n", err, requestData, string(body))
							// Return 400 for invalid request data
							c.JSON(http.StatusBadRequest, gin.H{
								"error":   "Invalid request data",
								"message": "Request data does not match the API specification",
								"method":  method,
								"path":    path,
								"schema":  schemaName,
								"details": err.Error(),
							})
							c.Abort()
							return
						}
					}

					// Restore the request body so handlers can read it
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				}
			}
		}

		// Continue to the next handler
		c.Next()
	}
}
