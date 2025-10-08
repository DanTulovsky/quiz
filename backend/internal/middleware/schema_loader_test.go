package middleware

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaLoader_Integration(t *testing.T) {
	// Set the swagger file path for testing
	originalSwaggerPath := os.Getenv("SWAGGER_FILE_PATH")
	defer os.Setenv("SWAGGER_FILE_PATH", originalSwaggerPath)

	// Set to the actual swagger file in the project root
	os.Setenv("SWAGGER_FILE_PATH", "../../swagger.yaml")

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

	// Test specific key endpoints manually
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
		{"/v1/quiz/question/123", "GET", false, false, "", ""}, // Parameterized paths may not find response schema
	}

	t.Logf("Testing %d specific endpoints", len(testEndpoints))

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
}
