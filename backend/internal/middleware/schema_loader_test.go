package middleware

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// extractAllPathsFromSwagger reads the swagger file and extracts all paths and their methods
func extractAllPathsFromSwagger(t *testing.T) (map[string][]string, error) {
	// Try multiple possible paths for the swagger file
	possiblePaths := []string{
		"../../swagger.yaml",
		"../../../swagger.yaml",
		"/workspaces/quiz/swagger.yaml",
	}

	var data []byte
	var err error

	for _, path := range possiblePaths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read swagger file: %v", err)
	}

	var swagger map[string]interface{}
	err = yaml.Unmarshal(data, &swagger)
	require.NoError(t, err)

	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathsInterface, ok := swagger["paths"].(map[interface{}]interface{})
		if ok {
			paths = make(map[string]interface{})
			for k, v := range pathsInterface {
				if keyStr, ok := k.(string); ok {
					paths[keyStr] = v
				}
			}
		} else {
			return nil, fmt.Errorf("no paths section found in swagger")
		}
	}

	result := make(map[string][]string)
	for path, pathInfo := range paths {
		pathMap, ok := pathInfo.(map[string]interface{})
		if !ok {
			// Try with interface{} keys
			pathMapInterface, ok := pathInfo.(map[interface{}]interface{})
			if ok {
				pathMap = make(map[string]interface{})
				for k, v := range pathMapInterface {
					if keyStr, ok := k.(string); ok {
						pathMap[keyStr] = v
					}
				}
			} else {
				continue
			}
		}

		var methods []string
		for method := range pathMap {
			if method != "parameters" && method != "summary" && method != "description" && method != "tags" {
				methods = append(methods, strings.ToUpper(method))
			}
		}

		if len(methods) > 0 {
			result[path] = methods
		}
	}

	return result, nil
}

func TestSchemaLoader_Integration(t *testing.T) {
	// Set the swagger file path for testing
	originalSwaggerPath := os.Getenv("SWAGGER_FILE_PATH")
	defer func() {
		_ = os.Setenv("SWAGGER_FILE_PATH", originalSwaggerPath)
	}()

	// Set to the actual swagger file in the project root
	_ = os.Setenv("SWAGGER_FILE_PATH", "../../swagger.yaml")

	loader := NewSchemaLoader()

	// Try multiple possible paths for the swagger file
	possiblePaths := []string{
		"../../swagger.yaml",
		"../../../swagger.yaml",
		"/workspaces/quiz/swagger.yaml",
	}

	var swaggerData []byte
	var swaggerErr error

	for _, path := range possiblePaths {
		swaggerData, swaggerErr = os.ReadFile(path)
		if swaggerErr == nil {
			break
		}
	}

	require.NoError(t, swaggerErr)

	testErr := loader.LoadSchemasFromSwaggerFromData(swaggerData)
	require.NoError(t, testErr)
	require.NotNil(t, loader.swaggerData)
	require.NotEmpty(t, loader.schemas)

	// Test a representative sample of endpoints manually
	testEndpoints := []struct {
		path       string
		method     string
		expectReq  bool
		expectResp bool
		reqSchema  string
		respSchema string
	}{
		{"/v1/auth/login", "POST", true, true, "LoginRequest", "LoginResponse"},
		{"/v1/story", "POST", true, true, "CreateStoryRequest", "Story"},
		{"/v1/quiz/question", "GET", false, true, "", "Question"},
		{"/v1/settings", "PUT", true, true, "UserSettings", "SuccessResponse"},
		{"/v1/settings/languages", "GET", false, true, "", "LanguagesResponse"},
		{"/v1/quiz/question/123", "GET", false, true, "", "Question"},  // Test parameterized path
		{"/v1/story/123", "GET", false, true, "", "StoryWithSections"}, // Test parameterized path
	}

	// Also test all endpoints automatically
	allPaths, err := extractAllPathsFromSwagger(t)
	require.NoError(t, err)

	t.Logf("Testing %d manually selected endpoints and %d total endpoints from swagger", len(testEndpoints), len(allPaths))

	t.Logf("Testing %d representative endpoints", len(testEndpoints))

	for _, endpoint := range testEndpoints {
		t.Run(fmt.Sprintf("%s_%s", endpoint.method, endpoint.path), func(t *testing.T) {
			// Test that endpoint is documented
			isDocumented := loader.IsEndpointDocumented(endpoint.path, endpoint.method)
			assert.True(t, isDocumented, "Endpoint %s %s should be documented", endpoint.method, endpoint.path)

			// Test request schema determination
			requestSchema := loader.DetermineRequestSchemaFromPath(endpoint.path, endpoint.method)
			if endpoint.expectReq {
				assert.NotEmpty(t, requestSchema, "Should find request schema for %s %s", endpoint.method, endpoint.path)
				if requestSchema != "" {
					assert.Equal(t, endpoint.reqSchema, requestSchema, "Request schema should match expected")
					assert.Contains(t, loader.schemas, requestSchema, "Request schema %s should be loaded", requestSchema)
				}
			} else {
				assert.Empty(t, requestSchema, "Should not find request schema for %s %s", endpoint.method, endpoint.path)
			}

			// Test response schema determination
			responseSchema := loader.DetermineSchemaFromPath(endpoint.path, endpoint.method)
			if endpoint.expectResp {
				assert.NotEmpty(t, responseSchema, "Should find response schema for %s %s", endpoint.method, endpoint.path)
				if responseSchema != "" {
					assert.Equal(t, endpoint.respSchema, responseSchema, "Response schema should match expected")
					assert.Contains(t, loader.schemas, responseSchema, "Response schema %s should be loaded", responseSchema)
				}
			}
		})
	}

	// Test schema validation for a few key schemas
	t.Run("SchemaValidation", func(t *testing.T) {
		// Test LoginRequest validation
		loginRequest := map[string]interface{}{
			"username": "testuser",
			"password": "testpass123",
		}

		err := loader.ValidateData(loginRequest, "LoginRequest")
		assert.NoError(t, err, "Valid LoginRequest should pass validation")

		// Test invalid LoginRequest
		invalidLoginRequest := map[string]interface{}{
			"username": "", // Should fail validation
		}

		err = loader.ValidateData(invalidLoginRequest, "LoginRequest")
		assert.Error(t, err, "Invalid LoginRequest should fail validation")

		// Test Story schema validation
		story := map[string]interface{}{
			"id":       1,
			"user_id":  1,
			"title":    "Test Story",
			"status":   "active",
			"language": "en",
		}

		err = loader.ValidateData(story, "Story")
		assert.NoError(t, err, "Valid Story should pass validation")
	})

	// Test that undocumented endpoint is correctly rejected
	t.Run("UndocumentedEndpoint", func(t *testing.T) {
		assert.False(t, loader.IsEndpointDocumented("/v1/nonexistent", "GET"))
	})

	// Test all endpoints automatically
	t.Run("AllEndpoints", func(t *testing.T) {
		testedCount := 0
		for path, methods := range allPaths {
			for _, method := range methods {
				t.Run(fmt.Sprintf("%s_%s", method, path), func(t *testing.T) {
					testedCount++

					// Test that endpoint is documented
					isDocumented := loader.IsEndpointDocumented(path, method)
					assert.True(t, isDocumented, "Endpoint %s %s should be documented", method, path)
					if !isDocumented {
						t.Logf("Debug: Endpoint %s %s not found as documented", method, path)

						// Debug: Check if swaggerData is loaded
						if loader.swaggerData == nil {
							t.Logf("  ERROR: swaggerData is nil")
						} else {
							// Debug: Check if the path exists in swagger
							if paths, ok := loader.swaggerData["paths"].(map[string]interface{}); ok {
								if _, exists := paths[path]; exists {
									t.Logf("  Path exists in swagger: %s", path)
								} else {
									t.Logf("  Path does not exist in swagger: %s", path)
									// Try pattern matching manually
									for swaggerPath := range paths {
										if strings.Contains(swaggerPath, "{") && strings.Contains(swaggerPath, "}") {
											if loader.pathMatchesPattern(path, swaggerPath) {
												t.Logf("  Pattern match found: %s matches %s", path, swaggerPath)
												break
											}
										}
									}
								}
							} else {
								t.Logf("  ERROR: paths section not found in swagger data")
							}
						}
					}
					assert.True(t, isDocumented, "Endpoint %s %s should be documented", method, path)

					// Test request schema determination
					requestSchema := loader.DetermineRequestSchemaFromPath(path, method)
					if requestSchema != "" {
						assert.Contains(t, loader.schemas, requestSchema, "Request schema %s should be loaded", requestSchema)
					}

					// Test response schema determination
					responseSchema := loader.DetermineSchemaFromPath(path, method)
					if responseSchema != "" {
						assert.Contains(t, loader.schemas, responseSchema, "Response schema %s should be loaded", responseSchema)
					}
				})
			}
		}

		t.Logf("Automatically tested %d endpoint-method combinations", testedCount)
	})
}
