package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	contextutils "quizapp/internal/utils"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

// SchemaLoader loads JSON schemas from the Swagger specification
type SchemaLoader struct {
	schemas               map[string]*gojsonschema.Schema
	jsonCompatibleSchemas map[string]interface{}
	swaggerData           map[string]interface{}
}

// NewSchemaLoader creates a new schema loader
func NewSchemaLoader() *SchemaLoader {
	return &SchemaLoader{
		schemas:               make(map[string]*gojsonschema.Schema),
		jsonCompatibleSchemas: make(map[string]interface{}),
	}
}

// LoadSchemasFromSwagger loads all schemas from the Swagger specification
func (sl *SchemaLoader) LoadSchemasFromSwagger(swaggerPath string) error {
	// Read the Swagger file
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		return contextutils.WrapError(err, "failed to read swagger file")
	}

	return sl.LoadSchemasFromSwaggerFromData(data)
}

// LoadSchemasFromSwaggerFromData loads all schemas from swagger data bytes
func (sl *SchemaLoader) LoadSchemasFromSwaggerFromData(data []byte) error {
	// Parse the Swagger spec (YAML only)
	var swagger map[string]interface{}

	if err := yaml.Unmarshal(data, &swagger); err != nil {
		return contextutils.WrapError(err, "failed to parse swagger file as YAML")
	}

	fmt.Printf("✅ Successfully parsed swagger file as YAML\n")

	// Store the parsed swagger data for later use
	sl.swaggerData = swagger

	// Extract components/schemas
	components, ok := swagger["components"].(map[interface{}]interface{})
	if !ok {
		fmt.Printf("❌ No components section found. Available keys: %v\n", getKeys(swagger))
		fmt.Printf("❌ Components type: %T, value: %v\n", swagger["components"], swagger["components"])
		return contextutils.ErrorWithContextf("no components section found in swagger")
	}

	schemas, ok := components["schemas"].(map[interface{}]interface{})
	if !ok {
		fmt.Printf("❌ No schemas section found in components. Available keys: %v\n", getKeysInterface(components))
		fmt.Printf("❌ Schemas type: %T, value: %v\n", components["schemas"], components["schemas"])
		return contextutils.ErrorWithContextf("no schemas section found in swagger")
	}

	// Convert schemas to JSON-compatible format
	jsonCompatibleSchemas := make(map[string]interface{})
	for schemaName, schemaData := range schemas {
		schemaNameStr, ok := schemaName.(string)
		if !ok {
			fmt.Printf("Warning: schema name is not a string: %v\n", schemaName)
			continue
		}

		convertedSchema := convertToJSONCompatible(schemaData)

		jsonCompatibleSchemas[schemaNameStr] = convertedSchema
	}

	// Store jsonCompatibleSchemas for creating array schemas later
	sl.jsonCompatibleSchemas = jsonCompatibleSchemas

	// Load each schema
	for schemaNameStr := range jsonCompatibleSchemas {
		// Create a schema document with the full swagger context for $ref resolution
		completeSchemaDoc := map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"components": map[string]interface{}{
				"schemas": jsonCompatibleSchemas,
			},
			"$ref": "#/components/schemas/" + schemaNameStr,
		}

		schemaBytes, err := json.Marshal(completeSchemaDoc)
		if err != nil {
			fmt.Printf("Warning: failed to marshal schema %s: %v\n", schemaNameStr, err)
			continue
		}

		// Load the schema
		schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
		schema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			fmt.Printf("Warning: failed to load schema %s: %v\n", schemaNameStr, err)
			continue
		}

		sl.schemas[schemaNameStr] = schema
		fmt.Printf("✅ Loaded schema: %s\n", schemaNameStr)
	}

	return nil
}

// getKeys returns the keys of a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// getKeysInterface returns the keys of a map with interface{} keys
func getKeysInterface(m map[interface{}]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if keyStr, ok := k.(string); ok {
			keys = append(keys, keyStr)
		}
	}
	return keys
}

// convertInterfaceMapToStringMap converts a map[interface{}]interface{} to map[string]interface{}
func convertInterfaceMapToStringMap(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		keyStr := fmt.Sprintf("%v", k) // Convert any key type to string
		result[keyStr] = convertToJSONCompatible(v)
	}
	return result
}

// convertToJSONCompatible converts a map[interface{}]interface{} to map[string]interface{}
func convertToJSONCompatible(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		hasNullable := false

		for k, val := range v {
			keyStr := fmt.Sprintf("%v", k) // Convert any key type to string

			// Check for nullable field
			if keyStr == "nullable" {
				nullable, ok := val.(bool)
				if ok && nullable {
					hasNullable = true
					continue // Skip the nullable field as we'll handle it in the type conversion
				}
			}

			convertedVal := convertToJSONCompatible(val)
			result[keyStr] = convertedVal
		}

		// Handle nullable fields by converting to union type
		if hasNullable {
			// If there's a $ref field, create a union type with null
			if ref, hasRef := result["$ref"].(string); hasRef {
				// Create a union type that allows both the referenced type and null
				result["oneOf"] = []interface{}{
					map[string]interface{}{"$ref": ref},
					map[string]interface{}{"enum": []interface{}{nil}},
				}
				// Remove the original $ref field
				delete(result, "$ref")
			} else if typeVal, hasType := result["type"].(string); hasType {
				// If there's a type field, convert to array of types including null
				result["type"] = []interface{}{typeVal, "null"}
			}
		}

		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			convertedVal := convertToJSONCompatible(val)
			result[i] = convertedVal
		}
		return result
	default:
		return data
	}
}

// ValidateData validates data against a schema
func (sl *SchemaLoader) ValidateData(data interface{}, schemaName string) error {
	schema, exists := sl.schemas[schemaName]
	if !exists {
		return contextutils.ErrorWithContextf("schema %s not found", schemaName)
	}

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return contextutils.WrapError(err, "failed to marshal data")
	}

	// Create document loader
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return contextutils.WrapError(err, "validation error")
	}

	if !result.Valid() {
		var validationErrors []string
		for _, validationErr := range result.Errors() {
			errorMsg := fmt.Sprintf("%s: %s", validationErr.Field(), validationErr.Description())
			// Include the actual value that failed validation if available
			if validationErr.Value() != nil {
				errorMsg += fmt.Sprintf(" (received: %v)", validationErr.Value())
			}
			validationErrors = append(validationErrors, errorMsg)
		}
		return contextutils.ErrorWithContextf("schema validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// AutoLoadSchemas automatically loads schemas from the swagger file path
func AutoLoadSchemas() *SchemaLoader {
	loader := NewSchemaLoader()

	// Get swagger file path from environment variable
	swaggerPath := os.Getenv("SWAGGER_FILE_PATH")
	if swaggerPath == "" {
		fmt.Printf("❌ SWAGGER_FILE_PATH environment variable not set\n")
		return loader
	}

	if _, err := os.Stat(swaggerPath); err == nil {
		if err := loader.LoadSchemasFromSwagger(swaggerPath); err != nil {
			fmt.Printf("Warning: failed to load schemas from %s: %v\n", swaggerPath, err)
		} else {
			fmt.Printf("✅ Successfully loaded schemas from %s\n", swaggerPath)
			return loader
		}
	} else {
		fmt.Printf("⚠️  Swagger file not found at %s: %v\n", swaggerPath, err)
	}

	return loader
}

// IsEndpointDocumented checks if an endpoint is documented in the swagger spec
func (sl *SchemaLoader) IsEndpointDocumented(path, method string) bool {
	// Use cached swagger data if available
	if sl.swaggerData == nil {
		return false
	}
	swagger := sl.swaggerData

	// Extract paths
	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathsInterface, ok := swagger["paths"].(map[interface{}]interface{})
		if !ok {
			return false
		}
		// Convert to string keys
		paths = convertInterfaceMapToStringMap(pathsInterface)
	}

	// First, try exact match
	pathInfo, exists := paths[path]
	if exists {
		pathMap, ok := pathInfo.(map[string]interface{})
		if !ok {
			// Try with interface{} keys
			pathMapInterface, ok := pathInfo.(map[interface{}]interface{})
			if !ok {
				return false
			}
			// Convert to string keys
			pathMap = convertInterfaceMapToStringMap(pathMapInterface)
		}

		// Look for the specific HTTP method
		_, exists = pathMap[strings.ToLower(method)]
		if exists {
			return true
		}
	}

	// If exact match fails, try pattern matching for path parameters
	for swaggerPath := range paths {
		if sl.pathMatchesPattern(path, swaggerPath) {
			pathInfo := paths[swaggerPath]
			pathMap, ok := pathInfo.(map[string]interface{})
			if !ok {
				// Try with interface{} keys
				pathMapInterface, ok := pathInfo.(map[interface{}]interface{})
				if !ok {
					continue
				}
				// Convert to string keys
				pathMap = convertInterfaceMapToStringMap(pathMapInterface)
			}

			// Look for the specific HTTP method
			_, exists = pathMap[strings.ToLower(method)]
			if exists {
				return true
			}
		}
	}

	return false
}

// pathMatchesPattern checks if a request path matches a swagger path pattern
func (sl *SchemaLoader) pathMatchesPattern(requestPath, swaggerPath string) bool {
	// Split paths into segments
	requestSegments := strings.Split(requestPath, "/")
	swaggerSegments := strings.Split(swaggerPath, "/")

	// Paths must have the same number of segments
	if len(requestSegments) != len(swaggerSegments) {
		return false
	}

	// Compare each segment
	for i, swaggerSegment := range swaggerSegments {
		requestSegment := requestSegments[i]

		// If swagger segment is a parameter (starts with { and ends with })
		if strings.HasPrefix(swaggerSegment, "{") && strings.HasSuffix(swaggerSegment, "}") {
			// Any value is acceptable for parameters
			continue
		}

		// Otherwise, segments must match exactly
		if swaggerSegment != requestSegment {
			return false
		}
	}

	return true
}

// DetermineRequestSchemaFromPath automatically determines the schema name from the API path and method
func (sl *SchemaLoader) DetermineRequestSchemaFromPath(path, method string) string {
	// Use cached swagger data if available
	if sl.swaggerData == nil {
		return ""
	}
	swagger := sl.swaggerData

	// Extract paths
	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathsInterface, ok := swagger["paths"].(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		paths = convertInterfaceMapToStringMap(pathsInterface)
	}

	// First, try exact match
	pathInfo, exists := paths[path]
	if !exists {
		// If exact match fails, try pattern matching for path parameters
		for swaggerPath := range paths {
			if sl.pathMatchesPattern(path, swaggerPath) {
				pathInfo = paths[swaggerPath]
				break
			}
		}
		if pathInfo == nil {
			return ""
		}
	}

	pathMap, ok := pathInfo.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathMapInterface, ok := pathInfo.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		pathMap = convertInterfaceMapToStringMap(pathMapInterface)
	}

	// Look for the specific HTTP method
	methodInfo, exists := pathMap[strings.ToLower(method)]
	if !exists {
		return ""
	}

	methodMap, ok := methodInfo.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		methodMapInterface, ok := methodInfo.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		methodMap = convertInterfaceMapToStringMap(methodMapInterface)
	}

	// Extract the request body schema
	requestBody, exists := methodMap["requestBody"]
	if !exists {
		return ""
	}

	requestBodyMap, ok := requestBody.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		requestBodyMapInterface, ok := requestBody.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		requestBodyMap = convertInterfaceMapToStringMap(requestBodyMapInterface)
	}

	// Extract content
	content, ok := requestBodyMap["content"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		contentInterface, ok := requestBodyMap["content"].(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		content = convertInterfaceMapToStringMap(contentInterface)
	}

	// Look for application/json content
	jsonContent, exists := content["application/json"]
	if !exists {
		return ""
	}

	jsonContentMap, ok := jsonContent.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		jsonContentMapInterface, ok := jsonContent.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		jsonContentMap = convertInterfaceMapToStringMap(jsonContentMapInterface)
	}

	// Extract schema
	schema, exists := jsonContentMap["schema"]
	if !exists {
		return ""
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		schemaMapInterface, ok := schema.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		schemaMap = convertInterfaceMapToStringMap(schemaMapInterface)
	}

	// Extract $ref
	ref, exists := schemaMap["$ref"]
	if !exists {
		return ""
	}

	refStr, ok := ref.(string)
	if !ok {
		return ""
	}

	// Extract schema name from $ref
	// $ref format: "#/components/schemas/SchemaName"
	parts := strings.Split(refStr, "/")
	if len(parts) < 4 {
		return ""
	}

	return parts[len(parts)-1]
}

// DetermineSchemaFromPath determines the schema name for a given path and HTTP method
// by parsing the swagger file and looking up the response schema for the 200 status code.
func (sl *SchemaLoader) DetermineSchemaFromPath(path, method string) string {
	// Use cached swagger data if available
	if sl.swaggerData == nil {
		return ""
	}
	swagger := sl.swaggerData

	// Extract paths
	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathsInterface, ok := swagger["paths"].(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		paths = convertInterfaceMapToStringMap(pathsInterface)
	}

	// First, try exact match
	pathInfo, exists := paths[path]
	if !exists {
		// If exact match fails, try pattern matching for path parameters
		for swaggerPath := range paths {
			if sl.pathMatchesPattern(path, swaggerPath) {
				pathInfo = paths[swaggerPath]
				break
			}
		}
		if pathInfo == nil {
			return ""
		}
	}

	pathMap, ok := pathInfo.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		pathMapInterface, ok := pathInfo.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		pathMap = convertInterfaceMapToStringMap(pathMapInterface)
	}

	// Look for the specific HTTP method
	methodInfo, exists := pathMap[strings.ToLower(method)]
	if !exists {
		return ""
	}

	methodMap, ok := methodInfo.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		methodMapInterface, ok := methodInfo.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		methodMap = convertInterfaceMapToStringMap(methodMapInterface)
	}

	// Extract the response schema
	responses, ok := methodMap["responses"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		responsesInterface, ok := methodMap["responses"].(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		responses = convertInterfaceMapToStringMap(responsesInterface)
	}

	// Look for success response (try 200, 201, etc.)
	var successResponse interface{}

	// Try common success status codes in order of preference
	successCodes := []string{"200", "201", "202"}
	for _, code := range successCodes {
		if resp, exists := responses[code]; exists {
			successResponse = resp
			break
		}
	}

	if successResponse == nil {
		return ""
	}

	responseMap, ok := successResponse.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		responseMapInterface, ok := successResponse.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		responseMap = convertInterfaceMapToStringMap(responseMapInterface)
	}

	// Extract content
	content, ok := responseMap["content"].(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		contentInterface, ok := responseMap["content"].(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		content = convertInterfaceMapToStringMap(contentInterface)
	}

	// Look for application/json
	jsonContent, exists := content["application/json"]
	if !exists {
		return ""
	}

	jsonMap, ok := jsonContent.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		jsonMapInterface, ok := jsonContent.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		jsonMap = convertInterfaceMapToStringMap(jsonMapInterface)
	}

	// Extract schema reference
	schema, exists := jsonMap["schema"]
	if !exists {
		return ""
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		// Try with interface{} keys
		schemaMapInterface, ok := schema.(map[interface{}]interface{})
		if !ok {
			return ""
		}
		// Convert to string keys
		schemaMap = convertInterfaceMapToStringMap(schemaMapInterface)
	}

	// Extract $ref directly
	if ref, exists := schemaMap["$ref"]; exists {
		if refStr, ok := ref.(string); ok {
			// Extract schema name from $ref (e.g., "#/components/schemas/DashboardResponse")
			if strings.HasPrefix(refStr, "#/components/schemas/") {
				schemaName := strings.TrimPrefix(refStr, "#/components/schemas/")
				return schemaName
			}
		}
	}

	// Handle array schemas - check if it's an array with items that have a $ref
	if schemaType, exists := schemaMap["type"]; exists {
		if typeStr, ok := schemaType.(string); ok && typeStr == "array" {
			// Check for items.$ref
			if items, exists := schemaMap["items"]; exists {
				itemsMap, ok := items.(map[string]interface{})
				if !ok {
					// Try with interface{} keys
					itemsMapInterface, ok := items.(map[interface{}]interface{})
					if !ok {
						return ""
					}
					itemsMap = convertInterfaceMapToStringMap(itemsMapInterface)
				}

				if ref, exists := itemsMap["$ref"]; exists {
					if refStr, ok := ref.(string); ok {
						// Extract schema name from $ref (e.g., "#/components/schemas/Story")
						if strings.HasPrefix(refStr, "#/components/schemas/") {
							itemSchemaName := strings.TrimPrefix(refStr, "#/components/schemas/")

							// For array responses, we need to create a synthetic schema that validates arrays
							arraySchemaName := fmt.Sprintf("%sArray", itemSchemaName)

							// Check if we've already created this array schema
							if _, exists := sl.schemas[arraySchemaName]; !exists {
								// Create array schema with full context for $ref resolution
								arraySchema := map[string]interface{}{
									"$schema": "http://json-schema.org/draft-07/schema#",
									"components": map[string]interface{}{
										"schemas": sl.jsonCompatibleSchemas,
									},
									"type": "array",
									"items": map[string]interface{}{
										"$ref": fmt.Sprintf("#/components/schemas/%s", itemSchemaName),
									},
								}

								// Load the array schema
								schemaBytes, err := json.Marshal(arraySchema)
								if err != nil {
									fmt.Printf("Warning: failed to marshal array schema %s: %v\n", arraySchemaName, err)
									return itemSchemaName // Fallback to item schema
								}

								schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
								schema, err := gojsonschema.NewSchema(schemaLoader)
								if err != nil {
									fmt.Printf("Warning: failed to load array schema %s: %v\n", arraySchemaName, err)
									return itemSchemaName // Fallback to item schema
								}

								sl.schemas[arraySchemaName] = schema
								fmt.Printf("✅ Created array schema: %s\n", arraySchemaName)
							}

							return arraySchemaName
						}
					}
				}
			}
		}
	}

	return ""
}
