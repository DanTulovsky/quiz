package middleware

import (
	"fmt"
	"os"
	"sort"
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
		{"/v1/story", "GET", false, true, "", "StoryArray"}, // Test GET /v1/story (returns array of Story)
		{"/v1/quiz/question", "GET", false, true, "", "Question"},
		{"/v1/settings", "PUT", true, true, "UserSettings", "SuccessResponse"},
		{"/v1/settings/languages", "GET", false, true, "", "LanguagesResponse"},
		{"/v1/quiz/question/123", "GET", false, true, "", "Question"},  // Test parameterized path
		{"/v1/story/123", "GET", false, true, "", "StoryWithSections"}, // Test parameterized path
		{"/v1/snippets/by-question/123", "GET", false, true, "", "SnippetsResponse"},
		{"/v1/snippets/by-section/123", "GET", false, true, "", "SnippetsResponse"},
		{"/v1/snippets/by-story/123", "GET", false, true, "", "SnippetsResponse"},
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
					} else {
						// Check if this endpoint should have a response schema by examining the swagger definition
						// Some endpoints legitimately don't have response bodies (204 No Content, etc.)
						t.Logf("No response schema found for %s %s - checking if this is expected...", method, path)

						// Check if the endpoint has any response definitions in swagger
						if paths, ok := loader.swaggerData["paths"].(map[string]interface{}); ok {
							if pathInfo, exists := paths[path]; exists {
								if pathMap, ok := pathInfo.(map[string]interface{}); ok {
									if methodInfo, exists := pathMap[strings.ToLower(method)]; exists {
										if methodMap, ok := methodInfo.(map[string]interface{}); ok {
											if _, exists := methodMap["responses"]; exists {
												t.Errorf("❌ Endpoint %s %s has responses defined in swagger but no schema detected - this indicates a bug in schema detection logic", method, path)
											} else {
												t.Logf("  Endpoint %s %s has no responses section in swagger - this is expected", method, path)
											}
										}
									}
								}
							}
						}
					}
				})
			}
		}

		t.Logf("Automatically tested %d endpoint-method combinations", testedCount)
	})
}

func TestSwaggerResponsesUseComponentRefs(t *testing.T) {
	// Load swagger file
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
	require.NoError(t, err, "failed to read swagger.yaml for inline schema check")

	var swagger map[string]interface{}
	err = yaml.Unmarshal(data, &swagger)
	require.NoError(t, err, "failed to parse swagger.yaml for inline schema check")

	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		pathsInterface, ok := swagger["paths"].(map[interface{}]interface{})
		require.True(t, ok, "swagger paths section missing or malformed")
		paths = convertInterfaceMapToStringMap(pathsInterface)
	}

	violations := make(map[string]string)

	for pathKey, pathInfo := range paths {
		pathMap, ok := pathInfo.(map[string]interface{})
		if !ok {
			pathInterface, ok := pathInfo.(map[interface{}]interface{})
			if !ok {
				continue
			}
			pathMap = convertInterfaceMapToStringMap(pathInterface)
		}

		for method, methodInfo := range pathMap {
			methodUpper := strings.ToUpper(method)

			// Skip non-method metadata
			switch method {
			case "parameters", "summary", "description", "tags":
				continue
			}

			methodMap, ok := methodInfo.(map[string]interface{})
			if !ok {
				methodInterface, ok := methodInfo.(map[interface{}]interface{})
				if !ok {
					continue
				}
				methodMap = convertInterfaceMapToStringMap(methodInterface)
			}

			responsesVal, ok := methodMap["responses"]
			if !ok {
				continue
			}

			responses, ok := responsesVal.(map[string]interface{})
			if !ok {
				responsesInterface, ok := responsesVal.(map[interface{}]interface{})
				if !ok {
					continue
				}
				responses = convertInterfaceMapToStringMap(responsesInterface)
			}

			for statusCode, responseInfo := range responses {
				responseMap, ok := responseInfo.(map[string]interface{})
				if !ok {
					responseInterface, ok := responseInfo.(map[interface{}]interface{})
					if !ok {
						continue
					}
					responseMap = convertInterfaceMapToStringMap(responseInterface)
				}

				contentVal, ok := responseMap["content"]
				if !ok {
					continue
				}

				contentMap, ok := contentVal.(map[string]interface{})
				if !ok {
					contentInterface, ok := contentVal.(map[interface{}]interface{})
					if !ok {
						continue
					}
					contentMap = convertInterfaceMapToStringMap(contentInterface)
				}

				jsonContent, exists := contentMap["application/json"]
				if !exists {
					continue
				}

				jsonMap, ok := jsonContent.(map[string]interface{})
				if !ok {
					jsonInterface, ok := jsonContent.(map[interface{}]interface{})
					if !ok {
						continue
					}
					jsonMap = convertInterfaceMapToStringMap(jsonInterface)
				}

				schemaVal, exists := jsonMap["schema"]
				if !exists {
					continue
				}

				schemaMap, ok := schemaVal.(map[string]interface{})
				if !ok {
					schemaInterface, ok := schemaVal.(map[interface{}]interface{})
					if !ok {
						continue
					}
					schemaMap = convertInterfaceMapToStringMap(schemaInterface)
				}

				if _, hasRef := schemaMap["$ref"]; hasRef {
					continue
				}

				key := fmt.Sprintf("%s %s %s", methodUpper, pathKey, statusCode)

				if schemaType, ok := schemaMap["type"].(string); ok {
					switch schemaType {
					case "object":
						violations[key] = "inline object schema"
					case "array":
						itemsVal, ok := schemaMap["items"]
						if !ok {
							violations[key] = "array schema missing items $ref"
							continue
						}

						itemsMap, ok := itemsVal.(map[string]interface{})
						if !ok {
							itemsInterface, ok := itemsVal.(map[interface{}]interface{})
							if !ok {
								violations[key] = "array schema items malformed"
								continue
							}
							itemsMap = convertInterfaceMapToStringMap(itemsInterface)
						}

						if _, hasItemRef := itemsMap["$ref"]; !hasItemRef {
							violations[key] = "array schema uses inline items"
						}
					default:
						// Allow primitive schemas without refs (string, integer, etc.)
					}
					continue
				}

				// Schemas without type or $ref should be flagged
				violations[key] = "schema missing $ref and type"
			}
		}
	}

	expectedViolations := map[string]string{
		"POST /v1/admin/worker/users/resume 200":                                     "inline object schema",
		"GET /v1/admin/backend/userz/paginated 200":                                  "inline object schema",
		"GET /v1/admin/worker/logs 200":                                              "inline object schema",
		"POST /v1/admin/backend/userz/{id}/reset-password 200":                       "inline object schema",
		"GET /v1/ai/search 200":                                                      "inline object schema",
		"POST /v1/audio/speech/init 200":                                             "inline object schema",
		"GET /v1/word-of-day/history 200":                                            "inline object schema",
		"GET /v1/daily/dates 200":                                                    "inline object schema",
		"POST /v1/daily/questions/{date}/answer/{questionId} 200":                    "schema missing $ref and type",
		"POST /v1/admin/backend/questions/{id}/assign-users 200":                     "inline object schema",
		"POST /v1/admin/backend/userz/{id}/clear 200":                                "inline object schema",
		"GET /v1/ai/bookmarks 200":                                                   "inline object schema",
		"POST /v1/admin/worker/daily/users/{userId}/questions/{date}/regenerate 200": "inline object schema",
		"POST /v1/admin/worker/pause 200":                                            "inline object schema",
		"GET /v1/admin/worker/notifications/sent 200":                                "inline object schema",
		"GET /v1/admin/worker/analytics/priority-scores 200":                         "inline object schema",
		"POST /v1/admin/backend/questions/{id}/fix 200":                              "inline object schema",
		"GET /v1/snippets/search 200":                                                "inline object schema",
		"GET /v1/admin/backend/questions 200":                                        "inline object schema",
		"GET /v1/daily/history/{questionId} 200":                                     "inline object schema",
		"POST /v1/admin/backend/questions/{id}/unassign-users 200":                   "inline object schema",
		"POST /v1/admin/backend/questions/{id}/ai-fix 200":                           "inline object schema",
		"DELETE /v1/admin/backend/feedback 200":                                      "inline object schema",
		"POST /v1/admin/backend/clear-user-data 200":                                 "inline object schema",
		"GET /v1/daily/questions/{date} 200":                                         "inline object schema",
		"GET /v1/admin/backend/userz 200":                                            "inline object schema",
		"POST /v1/admin/backend/userz 201":                                           "inline object schema",
		"GET /v1/admin/backend/questions/{id}/users 200":                             "inline object schema",
		"POST /v1/admin/worker/trigger 200":                                          "inline object schema",
		"GET /v1/admin/backend/questions/paginated 200":                              "inline object schema",
		"POST /v1/admin/worker/users/pause 200":                                      "inline object schema",
		"GET /health 200":                                                            "inline object schema",
		"GET /health 503":                                                            "inline object schema",
		"PUT /v1/ai/conversations/bookmark 200":                                      "inline object schema",
		"GET /v1/admin/worker/users 200":                                             "inline object schema",
		"PUT /v1/userz/profile 200":                                                  "inline object schema",
		"GET /v1/admin/worker/notifications/errors 200":                              "inline object schema",
		"PUT /v1/admin/backend/questions/{id} 200":                                   "inline object schema",
		"DELETE /v1/admin/backend/questions/{id} 200":                                "inline object schema",
		"GET /v1/admin/backend/userz/{id}/roles 200":                                 "inline object schema",
		"POST /v1/admin/backend/userz/{id}/roles 200":                                "inline object schema",
		"DELETE /v1/admin/backend/userz/{id}/roles/{roleId} 200":                     "inline object schema",
		"POST /v1/admin/worker/resume 200":                                           "inline object schema",
		"POST /v1/admin/backend/clear-database 200":                                  "inline object schema",
		"GET /v1/ai/conversations 200":                                               "inline object schema",
		"GET /v1/admin/worker/ai-concurrency 200":                                    "inline object schema",
		"GET /v1/admin/worker/daily/users/{userId}/questions/{date} 200":             "inline object schema",
		"GET /v1/admin/backend/roles 200":                                            "inline object schema",
		"PUT /v1/admin/backend/userz/{id} 200":                                       "inline object schema",
		"DELETE /v1/admin/backend/userz/{id} 200":                                    "inline object schema",
		"GET /v1/admin/worker/details 200":                                           "inline object schema",
		"GET /v1/admin/backend/reported-questions 200":                               "inline object schema",
		"GET /v1/admin/backend/stories 200":                                          "inline object schema",
	}

	if len(violations) == 0 && len(expectedViolations) == 0 {
		return
	}

	var unexpected []string
	for key, reason := range violations {
		expectedReason, ok := expectedViolations[key]
		if !ok {
			unexpected = append(unexpected, fmt.Sprintf("%s (%s)", key, reason))
			continue
		}
		if expectedReason != reason {
			unexpected = append(unexpected, fmt.Sprintf("%s (expected %s, got %s)", key, expectedReason, reason))
		}
	}

	var missing []string
	for key, reason := range expectedViolations {
		if _, ok := violations[key]; !ok {
			missing = append(missing, fmt.Sprintf("%s (%s)", key, reason))
		}
	}

	if len(unexpected) > 0 || len(missing) > 0 {
		sort.Strings(unexpected)
		sort.Strings(missing)
		message := "Swagger inline response schema check failed."
		if len(unexpected) > 0 {
			message += "\nUnexpected inline schemas:\n" + strings.Join(unexpected, "\n")
		}
		if len(missing) > 0 {
			message += "\nExpected inline schemas no longer present:\n" + strings.Join(missing, "\n")
		}
		t.Fatalf("%s", message)
	}
}
