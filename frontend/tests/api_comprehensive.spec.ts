import {test, expect} from '@playwright/test';
import {assertStatus} from './http-assert';

import * as yaml from 'js-yaml';
import * as fs from 'fs';
import * as path from 'path';
import {fileURLToPath} from 'url';

// Session name constant - should match backend config.SessionName
const SESSION_NAME = 'quiz-session';

// Test users
const REGULAR_USER = {
  username: 'apitestuser',
  password: 'password'
};

const STORY_TEST_USER = {
  username: 'apitestuserstory1',
  password: 'password'
};

const ADMIN_USER = {
  username: 'apitestadmin',
  password: 'password'
};

interface SwaggerPath {
  [method: string]: {
    tags?: string[];
    summary?: string;
    description?: string;
    security?: any[];
    parameters?: any[];
    requestBody?: any;
    responses: {
      [statusCode: string]: {
        description: string;
        content?: any;
      };
    };
  };
}

interface SwaggerSpec {
  paths: {
    [path: string]: SwaggerPath;
  };
  components?: {
    schemas?: {
      [schemaName: string]: any;
    };
  };
}

interface TestCase {
  path: string;
  method: string;
  description: string;
  requiresAuth: boolean;
  requiresAdmin: boolean;
  expectedStatusCodes: string[];
  requestBody?: any;
  pathParams?: Record<string, any>;
  queryParams?: Record<string, any>;
}

// Test data interfaces
interface TestUser {
  username: string;
  email: string;
  password: string;
  preferred_language: string;
  current_level: string;
  ai_provider: string;
  ai_api_key: string;
  ai_model?: string;
}

interface TestLearningPreferences {
  username: string;
  focus_on_weak_areas: boolean;
  fresh_question_ratio: number;
  weak_area_boost: number;
  known_question_penalty: number;
  review_interval_days: number;
  daily_reminder_enabled: boolean;
}

interface TestData {
  users: TestUser[];
  learning_preferences: TestLearningPreferences[];
}

interface TestRoleData {
  id: number;
  name: string;
  description: string;
}

interface TestStorySectionData {
  id: number;
  story_id: number;
  section_number: number;
  content: string;
  language_level: string;
  word_count: number;
  generated_by: string;
}

interface TestStoryData {
  id: number;
  username: string;
  title: string;
  status: string;
  sections: TestStorySectionData[];
}

interface TestRolesData {
  [roleName: string]: TestRoleData;
}

interface TestConversationData {
  id: string;
  username: string;
  title: string;
}

interface TestConversationsData {
  [conversationKey: string]: TestConversationData;
}


/**
 * Comprehensive API E2E tests that dynamically build test cases from swagger.yaml
 * Tests all endpoints for both regular and admin users
 */

test.describe('Comprehensive API Tests', () => {
  const baseURL = process.env.TEST_BASE_URL || 'http://localhost:3001';
  let swaggerSpec: SwaggerSpec;
  let testCases: TestCase[] = [];
  let happyPathCases: TestCase[] = [];
  let errorCases: TestCase[] = [];
  let testData: TestData;
  let testRolesData: TestRolesData;
  let testConversationsData: TestConversationsData;
  let testStoriesData: TestStoriesData;

  test.beforeAll(async () => {
    // Load swagger.yaml
    const swaggerPath = process.env.SWAGGER_FILE_PATH;
    if (!swaggerPath) {
      throw new Error('SWAGGER_FILE_PATH environment variable is required');
    }
    const swaggerContent = fs.readFileSync(swaggerPath, 'utf8');
    swaggerSpec = yaml.load(swaggerContent) as SwaggerSpec;

    // Load test data files
    const dataDir = path.join(process.cwd(), '..', 'backend', 'data');

    // Verify required test data files exist
    const requiredFiles = [
      path.join(dataDir, 'test_users.yaml'),
      path.join(dataDir, 'test_analytics.yaml'),
      path.join(process.cwd(), 'tests', 'test-users.json')
    ];

    for (const file of requiredFiles) {
      if (!fs.existsSync(file)) {
        throw new Error(`Required test data file not found: ${file}`);
      }
    }

    testData = await loadTestData(dataDir);

    // Load roles data
    const rolesPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'test-roles.json');
    if (fs.existsSync(rolesPath)) {
      const rolesContent = fs.readFileSync(rolesPath, 'utf8');
      testRolesData = JSON.parse(rolesContent);
    } else {
      throw new Error('test-roles.json not found');
    }

    // Load conversations data
    const conversationsPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'test-conversations.json');

    if (fs.existsSync(conversationsPath)) {
      try {
        const conversationsContent = fs.readFileSync(conversationsPath, 'utf8');
        testConversationsData = JSON.parse(conversationsContent);
        // console.log(`Loaded ${Object.keys(testConversationsData).length} conversations from test data`);
        // console.log(`Sample conversation: ${JSON.stringify(Object.values(testConversationsData)[0], null, 2)}`);
      } catch (error) {
        throw new Error('Error loading conversations data:', error);
      }
    } else {
      throw new Error('test-conversations.json not found');
    }

    // Load stories data
    const storiesPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'test-stories.json');

    if (fs.existsSync(storiesPath)) {
      try {
        const storiesContent = fs.readFileSync(storiesPath, 'utf8');
        testStoriesData = JSON.parse(storiesContent);
        console.log(`Loaded ${Object.keys(testStoriesData).length} stories from test data`);
      } catch (error) {
        throw new Error('Error loading stories data:', error);
      }
    } else {
      throw new Error('test-stories.json not found');
    }

    // Initialize available user IDs
    initializeAvailableUserIds();

    // Initialize available story IDs
    initializeAvailableStoryIds();
    initializeAvailableQuestionIds();
    initializeAvailableSectionIds();
    initializeAvailableSnippetIds();

    // Generate test cases from swagger spec
    testCases = generateTestCases(swaggerSpec);
    happyPathCases = generateHappyPathTestCases(swaggerSpec);
    errorCases = generateErrorTestCases(swaggerSpec);

    console.log(`\n📊 Generated ${testCases.length} test cases:`);
    console.log(`- Public endpoints: ${testCases.filter(tc => !tc.requiresAuth).length}`);
    console.log(`- Protected endpoints: ${testCases.filter(tc => tc.requiresAuth && !tc.requiresAdmin).length}`);
    console.log(`- Admin endpoints: ${testCases.filter(tc => tc.requiresAdmin).length}`);
    console.log(`- Happy path cases: ${happyPathCases.length}`);
    console.log(`- Error test cases: ${errorCases.length}\n`);
  });

  async function loadTestData(dataDir: string): Promise<TestData> {
    const testData: TestData = {
      users: [],
      learning_preferences: []
    };

    try {
      // Load users
      const usersPath = path.join(dataDir, 'test_users.yaml');
      if (fs.existsSync(usersPath)) {
        const usersContent = fs.readFileSync(usersPath, 'utf8');
        const usersData = yaml.load(usersContent) as any;
        testData.users = usersData.users || [];
      }

      // Load learning preferences
      const analyticsPath = path.join(dataDir, 'test_analytics.yaml');
      if (fs.existsSync(analyticsPath)) {
        const analyticsContent = fs.readFileSync(analyticsPath, 'utf8');
        const analyticsData = yaml.load(analyticsContent) as any;
        testData.learning_preferences = analyticsData.learning_preferences || [];
      }
    } catch (error) {
      console.warn('Failed to load test data files:', error);
    }

    return testData;
  }

  function shouldSkipEndpoint(path: string, isAdmin: boolean = false): boolean {
    const shouldSkip = path.includes('/google/') ||
      path.includes('/stream') ||
      path.includes('/health') ||
      path.includes('/clear-database') ||
      path.includes('/clear-user-data') ||
      path.includes('/test-email');

    if (shouldSkip) return true;

    // Only skip story endpoints for admin users
    if (isAdmin && path.includes('/story')) {
      return true;
    }

    return false;
  }

  function generateTestCases(spec: SwaggerSpec): TestCase[] {
    const cases: TestCase[] = [];

    for (const [path, pathObj] of Object.entries(spec.paths)) {
      for (const [method, methodObj] of Object.entries(pathObj)) {
        const requiresAuth = !!(methodObj.security && methodObj.security.length > 0);
        const requiresAdmin = !!(path.includes('/admin/') || path.includes('/v1/admin/'));
        // For main test cases, we'll use the appropriate success status code based on HTTP method
        const allStatusCodes = Object.keys(methodObj.responses);
        let successStatusCode = '200';

        if (method === 'post') {
          // POST endpoints that create resources should expect 201
          successStatusCode = allStatusCodes.find(code => code === '201') ||
            allStatusCodes.find(code => code.startsWith('2')) || '200';
        } else if (method === 'delete') {
          // DELETE endpoints should expect 204
          successStatusCode = allStatusCodes.find(code => code === '204') ||
            allStatusCodes.find(code => code.startsWith('2')) || '200';
        } else {
          // GET, PUT, PATCH should expect 200
          successStatusCode = allStatusCodes.find(code => code.startsWith('2')) || '200';
        }

        const expectedStatusCodes = [successStatusCode];

        // Skip some endpoints that require special handling
        if (shouldSkipEndpoint(path, requiresAdmin)) {
          continue;
        }

        // Generate test data based on path and method
        const testCase: TestCase = {
          path,
          method: method.toUpperCase(),
          description: methodObj.summary || `${method.toUpperCase()} ${path}`,
          requiresAuth,
          requiresAdmin,
          expectedStatusCodes,
          requestBody: generateRequestBody(methodObj.requestBody, path, method),
          pathParams: generatePathParams(methodObj.parameters || [], path),
          queryParams: generateQueryParams(methodObj.parameters || [])
        };

        cases.push(testCase);
      }
    }

    return cases;
  }

  function generateHappyPathTestCases(spec: SwaggerSpec): TestCase[] {
    const cases: TestCase[] = [];

    for (const [path, pathObj] of Object.entries(spec.paths)) {
      for (const [method, methodObj] of Object.entries(pathObj)) {
        const requiresAuth = !!(methodObj.security && methodObj.security.length > 0);
        const requiresAdmin = !!(path.includes('/admin/') || path.includes('/v1/admin/'));

        // Skip some endpoints that require special handling
        if (shouldSkipEndpoint(path, requiresAdmin)) {
          continue;
        }

        // Include endpoints that have any 2xx response for valid requests
        const successResponses = Object.keys(methodObj.responses).filter(code => code.startsWith('2'));
        if (successResponses.length > 0) {
          const testCase: TestCase = {
            path,
            method: method.toUpperCase(),
            description: methodObj.summary || `${method.toUpperCase()} ${path}`,
            requiresAuth,
            requiresAdmin,
            expectedStatusCodes: successResponses, // Expect any 2xx status code for happy path
            requestBody: generateRequestBody(methodObj.requestBody, path, method),
            pathParams: generatePathParams(methodObj.parameters || [], path),
            queryParams: generateQueryParams(methodObj.parameters || [])
          };

          cases.push(testCase);
        }
      }
    }

    return cases;
  }

  function generateErrorTestCases(spec: SwaggerSpec): TestCase[] {
    const cases: TestCase[] = [];

    for (const [path, pathObj] of Object.entries(spec.paths)) {
      for (const [method, methodObj] of Object.entries(pathObj)) {
        const requiresAuth = !!(methodObj.security && methodObj.security.length > 0);
        const requiresAdmin = !!(path.includes('/admin/') || path.includes('/v1/admin/'));

        // Skip some endpoints that require special handling
        if (shouldSkipEndpoint(path, requiresAdmin)) {
          continue;
        }

        // Generate error test cases based on the endpoint's error responses
        const errorResponses = Object.keys(methodObj.responses).filter(code => code !== '200');

        for (const errorCode of errorResponses) {
          // Skip 400 error-case generation for TTS endpoint which is proxied to an external service
          // We don't control its validation behavior and invalid payloads can cause 500s instead of 400s
          if (path === '/v1/audio/speech' && errorCode === '400') {
            continue;
          }
          let errorPath = path;
          let errorRequestBody = undefined;
          let errorDescription = '';

          // Generate appropriate error test data based on the error code and endpoint
          switch (errorCode) {
            case '400':
              errorDescription = 'Invalid request';
              if (path.includes('/{id}')) {
                errorPath = path.replace('{id}', 'invalid');
              }
              if (path.includes('/{questionId}')) {
                errorPath = path.replace('{questionId}', 'invalid');
              }
              if (path.includes('/{service}')) {
                errorPath = path.replace('{service}', 'invalid-service');
              }
              if (path.includes('/{date}')) {
                errorPath = path.replace('{date}', 'invalid-date');
              }
              // For 400 errors, generate invalid request body
              if (methodObj.requestBody) {
                errorRequestBody = generateInvalidRequestBody(methodObj.requestBody, path, method);
              }
              break;
            case '401':
              errorDescription = 'Unauthorized';
              // This will be tested without authentication
              break;
            case '404':
              errorDescription = 'Resource not found';
              if (path.includes('/{id}')) {
                // For conversation endpoints, use a fake UUID that doesn't exist
                if (path.includes('/conversations/')) {
                  errorPath = path.replace('{id}', '00000000-0000-0000-0000-000000000001');
                } else {
                  errorPath = path.replace('{id}', '999999999'); // Use a much larger number
                }
              }
              // For 404 errors, still need valid request body
              if (methodObj.requestBody) {
                errorRequestBody = generateRequestBody(methodObj.requestBody, path, method);
              }
              break;
            case '403':
              errorDescription = 'Forbidden';
              // This will be tested with regular user accessing admin endpoints
              break;
            case '503':
              errorDescription = 'Service unavailable';
              // This is for endpoints that might be disabled (like email service)
              // For 503 errors, generate valid request body since the error should be due to service unavailability
              if (methodObj.requestBody) {
                errorRequestBody = generateRequestBody(methodObj.requestBody, path, method);
              }
              break;
          }

          // Only create error test cases for meaningful scenarios
          if (errorCode === '400') {
            // Only create 400 test cases - skip 503 since we can't easily trigger service unavailability in tests
            const testCase: TestCase = {
              path: errorPath,
              method: method.toUpperCase(),
              description: `${method.toUpperCase()} ${path} - ${errorDescription}`,
              requiresAuth,
              requiresAdmin,
              expectedStatusCodes: [errorCode], // Expect exactly one error code
              requestBody: errorRequestBody,
              // For error cases where we've already modified the path, don't generate path params
              pathParams: errorPath === path ? generatePathParams(methodObj.parameters || [], errorPath) : {},
              queryParams: generateQueryParams(methodObj.parameters || [])
            };

            cases.push(testCase);
          } else if (errorCode === '404' && path.includes('/{id}')) {
            // Only create 404 test cases for endpoints with path parameters
            // Skip problematic endpoints that don't properly validate IDs
            if (!path.includes('/report') && !path.includes('/mark-known')) {
              const testCase: TestCase = {
                path: errorPath,
                method: method.toUpperCase(),
                description: `${method.toUpperCase()} ${path} - ${errorDescription}`,
                requiresAuth,
                requiresAdmin,
                expectedStatusCodes: [errorCode], // Expect exactly one error code
                requestBody: errorRequestBody,
                pathParams: generatePathParams(methodObj.parameters || [], errorPath),
                queryParams: generateQueryParams(methodObj.parameters || [])
              };

              cases.push(testCase);
            }
          }
        }
      }
    }

    return cases;
  }

  function generateRequestBody(requestBody: any, path: string, method: string, userContext?: {username: string; password: string}): any {
    if (!requestBody) return undefined;

    // Get the schema from the request body
    const schema = requestBody.content?.['application/json']?.schema;
    if (!schema) return {};

    // Handle $ref schemas
    if (schema.$ref) {
      const schemaName = schema.$ref.split('/').pop();
      // Look up the actual schema from the swagger spec
      return generateFromSchemaRef(schemaName, path, userContext);
    }

    // Handle inline schemas
    return generateFromSchema(schema, path, userContext);
  }

  function generateInvalidRequestBody(requestBody: any, path: string, method: string): any {
    if (!requestBody) return {};

    // Get the schema from the request body
    const schema = requestBody.content?.['application/json']?.schema;
    if (!schema) return {};

    // Handle $ref schemas
    if (schema.$ref) {
      const schemaName = schema.$ref.split('/').pop();
      // Look up the actual schema from the swagger spec
      return generateInvalidFromSchemaRef(schemaName, path);
    }

    // Handle inline schemas
    return generateInvalidFromSchema(schema, path);
  }

  function generateFromSchemaRef(schemaName: string, path: string, userContext?: {username: string; password: string}): any {
    // Look up the actual schema from the swagger spec
    const schema = swaggerSpec.components?.schemas?.[schemaName];
    if (!schema) {
      console.warn(`Schema ${schemaName} not found in swagger spec`);
      return {};
    }

    // Generate from the actual schema for all cases
    return generateFromSchema(schema, path, userContext);
  }

  function generateInvalidFromSchemaRef(schemaName: string, path: string): any {
    // Look up the actual schema from the swagger spec
    const schema = swaggerSpec.components?.schemas?.[schemaName];
    if (!schema) {
      console.warn(`Schema ${schemaName} not found in swagger spec`);
      return {};
    }

    // Generate invalid data for specific schemas
    switch (schemaName) {
      case 'AnswerRequest':
        return {
          question_id: 'invalid', // Should be number
          user_answer_index: 'invalid', // Should be number
          response_time_ms: 'invalid' // Should be number
        };

      default:
        // For unknown schemas, generate invalid data
        return generateInvalidFromSchema(schema, path);
    }
  }

  function generateFromSchema(schema: any, path: string, userContext?: {username: string; password: string}): any {
    const result: any = {};

    if (schema.properties) {
      for (const [key, prop] of Object.entries(schema.properties as any)) {
        const required = schema.required?.includes(key) || false;

        if (required || shouldIncludeOptionalProperty(key, path)) {
          result[key] = generatePropertyValue(prop, key, path, userContext);
        }
      }
    }

    return result;
  }

  function generateInvalidFromSchema(schema: any, path: string): any {
    const result: any = {};

    if (schema.properties) {
      for (const [key, prop] of Object.entries(schema.properties as any)) {
        const required = schema.required?.includes(key) || false;

        if (required || shouldIncludeOptionalProperty(key, path)) {
          result[key] = generateInvalidPropertyValue(prop, key);
        }
      }
    }

    return result;
  }

  function generatePropertyValue(prop: any, key: string, path: string = '', userContext?: {username: string; password: string}): any {
    if (prop.type === 'string') {
      if (prop.format === 'email') {
        return `test-${Math.random().toString(36).substring(2, 8)}@example.com`;
      }
      if (prop.format === 'uuid') {
        // Generate a valid UUID for UUID format strings
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
          const r = Math.random() * 16 | 0;
          const v = c === 'x' ? r : (r & 0x3 | 0x8);
          return v.toString(16);
        });
      }
      if (prop.enum && prop.enum.length > 0) {
        return prop.enum[0];
      }
      if (key.includes('password')) {
        // For login endpoints, use the user context if available
        if (path.includes('/auth/login') && userContext) {
          return userContext.password;
        }
        // For login endpoints without user context, use the correct password from test data
        if (path.includes('/auth/login')) {
          const adminUser = testData.users.find(u => u.username === 'apitestadmin');
          const regularUser = testData.users.find(u => u.username === 'apitestuser');
          return adminUser?.password || regularUser?.password || 'password';
        }
        return 'password123';
      }
      if (key.includes('username')) {
        // For login endpoints, use the user context if available
        if (path.includes('/auth/login') && userContext) {
          return userContext.username;
        }
        // For login endpoints without user context, use an existing user from test data
        if (path.includes('/auth/login')) {
          // Find a user that matches the test context
          const adminUser = testData.users.find(u => u.username === 'apitestadmin');
          const regularUser = testData.users.find(u => u.username === 'apitestuser');
          return adminUser?.username || regularUser?.username || 'apitestuser';
        }
        // For profile update endpoints, don't change the username
        if (path.includes('/userz/profile')) {
          return userContext?.username || 'apitestuser';
        }
        // For force-send notification endpoints, use a user with daily reminders enabled
        if (path.includes('/force-send')) {
          return 'reminderuser';
        }
        // For signup endpoints, use a unique username with valid characters only
        return `testuser_${Math.random().toString(36).substring(2, 8)}`;
      }
      if (key.includes('reason')) {
        return 'Test reason';
      }
      if (key.includes('language') || key.includes('_language')) {
        // For language fields, use a valid language code
        // Common language codes that are likely to be supported
        const commonLanguages = ['en', 'es', 'fr', 'de', 'it', 'pt', 'ru', 'ja', 'ko', 'zh'];
        return commonLanguages[Math.floor(Math.random() * commonLanguages.length)];
      }
      return 'test_value';
    }

    if (prop.type === 'number' || prop.type === 'integer') {
      // Special handling for question_id - use a valid question ID
      if (key === 'question_id') {
        // For question_id, we need to get a valid question ID
        // Since this is a synchronous function, we'll use a fallback approach
        // The actual question ID will be resolved in the test execution
        return 1; // This will be overridden in the test execution
      }

      // Handle numeric types with min/max constraints
      if (prop.minimum !== undefined && prop.maximum !== undefined) {
        // Return a value within the range
        const value = Math.max(prop.minimum, Math.min(prop.maximum, (prop.minimum + prop.maximum) / 2));
        // For integer types, ensure we return an integer
        return prop.type === 'integer' ? Math.floor(value) : value;
      }
      if (prop.minimum !== undefined) {
        return prop.type === 'integer' ? Math.floor(prop.minimum) : prop.minimum;
      }
      if (prop.maximum !== undefined) {
        return prop.type === 'integer' ? Math.floor(prop.maximum) : prop.maximum;
      }
      return prop.type === 'integer' ? 1 : 1.0; // Default numeric value
    }

    if (prop.type === 'boolean') {
      return true; // Default to true for boolean values
    }

    if (prop.type === 'array') {
      const minItems = typeof (prop as any).minItems === 'number' ? (prop as any).minItems : 0;
      // If array is allowed to be empty, return empty to avoid unintended side-effects in tests
      if (minItems <= 0) {
        return [];
      }
      const length = Math.max(minItems, 1);
      const itemsSchema = (prop as any).items || {};

      // Special handling for user_ids arrays used by admin assign/unassign endpoints
      if (key === 'user_ids') {
        const arr: number[] = [];
        for (let i = 0; i < length; i++) {
          // Prefer distinct available user IDs when possible
          const id = availableUserIds[i] ?? getAvailableUserId();
          arr.push(id);
        }
        return arr;
      }

      // Generic array value generation based on items schema
      const values: any[] = [];
      for (let i = 0; i < length; i++) {
        values.push(generateArrayItemValue(itemsSchema));
      }
      return values;
    }

    if (prop.type === 'object') {
      return {}; // Default empty object
    }

    // Handle $ref properties
    if (prop.$ref) {
      const refSchemaName = prop.$ref.split('/').pop();
      const refSchema = swaggerSpec.components?.schemas?.[refSchemaName];
      if (refSchema) {
        // If the referenced schema is an enum, return the first enum value
        if (refSchema.enum && refSchema.enum.length > 0) {
          return refSchema.enum[0];
        }
        // Special-case primitive refs without properties
        if (refSchema.type === 'string') {
          if (refSchemaName === 'Language' || key === 'language') {
            return testData.users[0]?.preferred_language || 'italian';
          }
          if (refSchemaName === 'Level' || key === 'level') {
            return 'B1';
          }
          return 'test_value';
        }
        if (refSchema.type === 'integer') {
          return 1;
        }
        if (refSchema.type === 'number') {
          return 1.0;
        }
        if (refSchema.type === 'boolean') {
          return true;
        }
        // Otherwise, generate from the referenced schema (objects)
        return generateFromSchema(refSchema, key);
      }
    }

    return null; // Fallback
  }

  function generateArrayItemValue(itemSchema: any): any {
    if (!itemSchema) return null;
    if (itemSchema.type === 'integer') {
      const min = typeof itemSchema.minimum === 'number' ? itemSchema.minimum : 1;
      return Math.max(1, Math.floor(min));
    }
    if (itemSchema.type === 'number') {
      const min = typeof itemSchema.minimum === 'number' ? itemSchema.minimum : 1.0;
      return min;
    }
    if (itemSchema.type === 'string') {
      if (itemSchema.enum && itemSchema.enum.length > 0) return itemSchema.enum[0];
      return 'item';
    }
    if (itemSchema.type === 'boolean') {
      return true;
    }
    // Fallback for complex types
    return {};
  }

  function generateInvalidPropertyValue(prop: any, key: string): any {
    if (prop.type === 'string') {
      if (prop.format === 'email') {
        return 'invalid-email'; // Invalid email format
      }
      if (prop.enum && prop.enum.length > 0) {
        return 'invalid-enum-value'; // Invalid enum value
      }
      if (key.includes('password')) {
        return 123; // Wrong type
      }
      if (key.includes('username')) {
        return null; // Null value
      }
      if (key.includes('reason')) {
        return 456; // Wrong type
      }
      return 789; // Wrong type for string
    }

    if (prop.type === 'number' || prop.type === 'integer') {
      return 'invalid-number'; // String instead of number
    }

    if (prop.type === 'boolean') {
      return 'invalid-boolean'; // String instead of boolean
    }

    if (prop.type === 'array') {
      return 'invalid-array'; // String instead of array
    }

    if (prop.type === 'object') {
      return 'invalid-object'; // String instead of object
    }

    // Handle $ref properties
    if (prop.$ref) {
      const refSchemaName = prop.$ref.split('/').pop();
      const refSchema = swaggerSpec.components?.schemas?.[refSchemaName];
      if (refSchema) {
        return generateInvalidFromSchema(refSchema, key);
      }
    }

    return 'invalid-value'; // Fallback
  }

  function shouldIncludeOptionalProperty(key: string, path: string): boolean {
    // Include certain optional properties that are commonly needed
    const importantProps = [
      'email', 'username', 'password', 'question_id', 'user_answer_index',
      // UserSettings properties
      'language', 'level', 'ai_provider', 'ai_model', 'ai_enabled', 'api_key',
      // Snippet update properties
      'original_text', 'translated_text', 'source_language', 'target_language', 'context'
    ];
    return importantProps.includes(key);
  }

  function generatePathParams(parameters: any[], path: string): Record<string, any> {
    if (!parameters) return {};

    const pathParams: Record<string, any> = {};

    for (const param of parameters) {
      if (param.in === 'path') {
        // Generate appropriate test values based on parameter name
        if (param.name === 'id') {
          // For question-related endpoints, use a placeholder that will be replaced during test execution
          if (path.includes('/questions/') && path.includes('/ai-fix')) {
            pathParams[param.name] = 'QUESTION_ID_PLACEHOLDER';
          } else if (path.includes('/story/')) {
            // For story endpoints, use a valid story ID based on the operation
            // We need to determine if this is a DELETE operation to use archived stories
            // For now, default to active story (will be overridden in custom replacement logic)
            pathParams[param.name] = 'STORY_ID_PLACEHOLDER';
          } else {
            pathParams[param.name] = 1; // Default ID, will be overridden if needed
          }
        } else if (param.name === 'questionId') {
          pathParams[param.name] = 'QUESTION_ID_PLACEHOLDER'; // Use placeholder for question ID
        } else if (param.name === 'conversationId') {
          pathParams[param.name] = 'CONVERSATION_ID_PLACEHOLDER'; // Use placeholder for conversation ID
        } else if (param.name === 'userId') {
          pathParams[param.name] = 1; // Default user ID for testing
        } else if (param.name === 'provider') {
          pathParams[param.name] = 'ollama';
        } else if (param.name === 'service') {
          pathParams[param.name] = 'google';
        } else if (param.name === 'roleId') {
          pathParams[param.name] = 1;
        } else if (param.name === 'date') {
          // Use the seeded test assignment date to match created daily assignments
          pathParams[param.name] = '2025-08-04';
        }
      }
    }

    return pathParams;
  }

  function generateQueryParams(parameters: any[]): Record<string, any> {
    if (!parameters) return {};

    const queryParams: Record<string, any> = {};

    for (const param of parameters) {
      if (param.in === 'query') {
        // Generate appropriate test values based on parameter name
        if (param.name === 'page') {
          queryParams[param.name] = 1;
        } else if (param.name === 'page_size') {
          queryParams[param.name] = 10;
        } else if (param.name === 'language') {
          // Use the user's preferred language from test data instead of hardcoding
          queryParams[param.name] = testData.users[0]?.preferred_language || 'italian';
        } else if (param.name === 'level') {
          queryParams[param.name] = 'B1';
        } else if (param.name === 'type') {
          queryParams[param.name] = 'vocabulary';
        } else if (param.name === 'search') {
          queryParams[param.name] = 'test';
        } else if (param.name === 'q') {
          queryParams[param.name] = 'test query';
        } else if (param.name === 'user_id') {
          queryParams[param.name] = 1;
        }
      }
    }

    return queryParams;
  }

// Track available user IDs from test data
let availableUserIds: number[] = [];
let availableQuestionIds: number[] = [];
let availableStoryIds: number[] = [];
let availableSectionIds: number[] = [];
let availableSnippetIds: number[] = [];
let deletedConversationIds: Set<string> = new Set();
let deletedStoryIds: Set<number> = new Set();
let deletedSnippetIds: Set<number> = new Set();

// Helper function to get stories from database for a specific user
async function getStoriesFromDatabase(username?: string, request?: any): Promise<Array<{id: number, status: string}>> {
  if (!request) {
    console.log(`❌ No request object provided for database query`);
    return [];
  }

  try {
    const baseURL = process.env.TEST_BASE_URL || 'http://localhost:3001';

    // Get user ID for the username
    let userId = null;
    if (username) {
      const userDataPath = path.join(process.cwd(), 'tests', 'test-users.json');
      if (fs.existsSync(userDataPath)) {
        const userDataContent = fs.readFileSync(userDataPath, 'utf8');
        const userData = JSON.parse(userDataContent);
        const user = Object.values(userData).find((u: any) => u.username === username) as any;
        userId = user?.id;
      }
    }

    if (!userId) {
      console.log(`❌ Could not find user ID for username: ${username}`);
      return [];
    }

    // Query the database directly using the backend API (requires admin auth)
    const adminSession = await loginUser(request, ADMIN_USER);
    const response = await request.get(`${baseURL}/v1/admin/backend/stories?user_id=${userId}`, {
      headers: {
        'Cookie': adminSession
      }
    });

    if (response.ok) {
      const data = await response.json();
      return data.stories || [];
    } else {
      console.log(`❌ Failed to fetch stories from database: ${response.status()}`);
      return [];
    }
  } catch (error) {
    console.log(`❌ Error fetching stories from database: ${error}`);
    return [];
  }
}

// Helper function to find a conversation for a specific user
function findConversationForUser(username: string): any {
  if (!testConversationsData) {
    console.warn(`No conversation data available for user ${username}`);
    return null;
  }

  // Look for conversations belonging to this user
  for (const [key, conversation] of Object.entries(testConversationsData)) {
    if (conversation.username === username && !deletedConversationIds.has(conversation.id)) {
      return conversation;
    }
  }

  // If no conversation found for this specific user, try to find any non-deleted conversation
  for (const [key, conversation] of Object.entries(testConversationsData)) {
    if (!deletedConversationIds.has(conversation.id)) {
      console.warn(`Using conversation ${conversation.id} for user ${username} (not their own conversation)`);
      return conversation;
    }
  }

  console.warn(`No available conversations found for user ${username}`);
  return null;
}

  let userIdToUsername: Record<number, string> = {};
  let usernameToStoryIds: Record<string, number[]> = {};

  function initializeAvailableUserIds() {
    // Load actual user IDs from the JSON file created by the test data setup
    const userDataPath = path.join(process.cwd(), 'tests', 'test-users.json');

    if (!fs.existsSync(userDataPath)) {
      throw new Error(`test-users.json not found at ${userDataPath}. Test data setup may have failed.`);
    }

    const userDataContent = fs.readFileSync(userDataPath, 'utf8');
    const userData = JSON.parse(userDataContent);

    // Extract user IDs and build ID->username mapping from the loaded data
    availableUserIds = Object.values(userData).map((user: any) => user.id);
    userIdToUsername = {};
    Object.values(userData).forEach((user: any) => {
      userIdToUsername[(user as any).id] = (user as any).username;
    });
  }

  function initializeAvailableStoryIds() {
    // Load actual story IDs from the JSON file created by the test data setup
    const storyDataPath = path.join(process.cwd(), 'tests', 'test-stories.json');

    if (!fs.existsSync(storyDataPath)) {
      throw new Error(`test-stories.json not found at ${storyDataPath}. Test data setup may have failed.`);
    }

    const storyDataContent = fs.readFileSync(storyDataPath, 'utf8');
    const storyData = JSON.parse(storyDataContent);

    // Extract story IDs and build username-to-story-IDs mapping
    availableStoryIds = [];
    usernameToStoryIds = {};

    Object.values(storyData).forEach((story: any) => {
      const storyId = story.id;
      const username = story.username;

      // Skip stories that are already marked as deleted
      if (!deletedStoryIds.has(storyId)) {
        availableStoryIds.push(storyId);
      }

      if (!usernameToStoryIds[username]) {
        usernameToStoryIds[username] = [];
      }
      // Only add to username mapping if not deleted
      if (!deletedStoryIds.has(storyId)) {
        usernameToStoryIds[username].push(storyId);
      }
    });

  }

  function initializeAvailableSectionIds() {
    // Load section IDs from the stories data
    availableSectionIds = [];

    if (!testStoriesData) {
      console.warn('No stories data available for section IDs');
      return;
    }

    // Extract all section IDs from all stories
    for (const story of Object.values(testStoriesData)) {
      if (story.sections && story.sections.length > 0) {
        for (const section of story.sections) {
          if (!availableSectionIds.includes(section.id)) {
            availableSectionIds.push(section.id);
          }
        }
      }
    }

  }

  function initializeAvailableSnippetIds() {
    // Load actual snippet IDs from the JSON file created by the test data setup
    const snippetDataPath = path.join(process.cwd(), 'tests', 'test-snippets.json');

    if (!fs.existsSync(snippetDataPath)) {
      throw new Error(`test-snippets.json not found at ${snippetDataPath}. Test data setup may have failed.`);
    }

    const snippetDataContent = fs.readFileSync(snippetDataPath, 'utf8');
    const snippetData = JSON.parse(snippetDataContent);

    // Extract snippet IDs
    availableSnippetIds = [];
    Object.values(snippetData).forEach((snippet: any) => {
      const snippetId = snippet.id;
      // Skip snippets that are already marked as deleted
      if (!deletedSnippetIds.has(snippetId)) {
        availableSnippetIds.push(snippetId);
      }
    });
  }

  // Helper to get a user ID by username from tests/test-users.json
  function getUserIdByUsername(username: string): number | undefined {
    const userDataPath = path.join(process.cwd(), 'tests', 'test-users.json');
    if (!fs.existsSync(userDataPath)) {
      return undefined;
    }
    const userDataContent = fs.readFileSync(userDataPath, 'utf8');
    const userData = JSON.parse(userDataContent);
    const user = Object.values(userData).find((u: any) => u.username === username) as any;
    return user?.id as number | undefined;
  }

  // Ensure a user exists (by logging in which auto-creates if missing) and fetch their actual ID via admin API
  async function ensureUserExistsAndGetId(username: string, request: any): Promise<number | undefined> {
    try {
      // Login as the target user to ensure the account exists (login creates user if missing)
      await loginUser(request, {username, password: 'password'});
    } catch {
      // Ignore login failures here; we'll still try to fetch via admin API
    }

    try {
      // Get admin session to list users and retrieve the actual ID
      const adminSession = await loginUser(request, ADMIN_USER);
      const resp = await request.get(`${baseURL}/v1/admin/backend/userz`, {
        headers: { 'Cookie': adminSession }
      });
      if (resp.status() === 200) {
        const data = await resp.json();
        const users = (data as any).users || [];
        const user = users.find((u: any) => u.username === username);
        return user?.id as number | undefined;
      }
    } catch {
      // Fall through to undefined
    }
    return undefined;
  }

  // Prefer a stable, non-admin user for admin user operations, chosen from current availableUserIds
  function getSafeNonAdminUserId(): number {
    const bannedUsernames = new Set(['admin', 'adminuser', 'apitestadmin', 'apitestuser', 'testuser', 'roletestuser']);
    for (const id of availableUserIds) {
      const name = userIdToUsername[id];
      if (name && !bannedUsernames.has(name)) {
        return id;
      }
    }
    return getAvailableUserId();
  }

  // Choose a load-test user (loaduserXXX) from currently available IDs
  function getLoadTestUserId(): number | undefined {
    for (const id of availableUserIds) {
      const name = userIdToUsername[id];
      if (typeof name === 'string' && name.startsWith('loaduser')) {
        return id;
      }
    }
    // Fallback to a known one if mapping is missing
    const fallback = getUserIdByUsername('loaduser100');
    return fallback ?? undefined;
  }

  function initializeAvailableQuestionIds() {
    // Start with question IDs 1-100 (based on the test data structure)
    // The test data creates many questions, so we have a good range to work with
    availableQuestionIds = Array.from({length: 100}, (_, i) => i + 1);
  }

  // Pick a random safe user ID from the available list (excluding admin/test control users)
  function getRandomSafeUserId(): number {
    const bannedUsernames = new Set(['admin', 'adminuser', 'apitestadmin', 'apitestuser', 'testuser', 'roletestuser']);
    const candidates = availableUserIds.filter((id) => {
      const name = userIdToUsername[id];
      return name && !bannedUsernames.has(name);
    });
    if (candidates.length === 0) {
      return getAvailableUserId();
    }
    const idx = Math.floor(Math.random() * candidates.length);
    return candidates[idx];
  }

  function getAvailableUserId(): number {
    if (availableUserIds.length === 0) {
      throw new Error('No available user IDs for testing');
    }
    return availableUserIds[0]; // Use the first available ID
  }

  function getAvailableConversationId(username?: string, forDeletion: boolean = false): string {
    if (!testConversationsData || Object.keys(testConversationsData).length === 0) {
      throw new Error('No conversation data available!');
      throw new Error('No conversation data available. Test conversations may not have been created during setup.');
    }

    // Find conversations for the given username or any conversation if no username specified
    const conversationKeys = Object.keys(testConversationsData);
    const matchingConversations = [];

    for (const key of conversationKeys) {
      const conversation = testConversationsData[key];
      // Skip deleted conversations
      if (deletedConversationIds.has(conversation.id)) {
        continue;
      }
      if (!username || conversation.username === username) {
        matchingConversations.push(conversation);
      }
    }

    if (matchingConversations.length === 0) {
      throw new Error(`No conversations found for user '${username}'. Available conversations: ${conversationKeys.join(', ')}`);
    }

    // If forDeletion is true, prefer conversations with titles indicating they can be deleted
    if (forDeletion) {
      const deletableConversations = matchingConversations.filter(conv =>
        conv.title.includes('To Be Deleted')
      );

      if (deletableConversations.length > 0) {
        const conversation = deletableConversations[0];
        return conversation.id;
      }

      // If no deletable conversations for this user, throw an error
      throw new Error(`No deletable conversations found for user '${username}'. Test setup should ensure each user has at least one conversation marked for deletion.`);
    }

    // For non-deletion scenarios, prefer conversations that are NOT marked for deletion
    const nonDeletableConversations = matchingConversations.filter(conv =>
      !conv.title.includes('To Be Deleted')
    );

    if (nonDeletableConversations.length > 0) {
      const conversation = nonDeletableConversations[0];
      return conversation.id;
    }

    // Fallback to any conversation if no non-deletable ones exist
    const conversation = matchingConversations[0];
    return conversation.id;
  }

  async function getAvailableStoryId(username?: string, includeArchived: boolean = false, storyData?: any, request?: any): Promise<number> {

    if (availableStoryIds.length === 0) {
      throw new Error('No available story IDs. Test data setup may have failed.');
    }

    // Load story data if not provided
    if (!storyData) {
      const storyDataPath = path.join(process.cwd(), 'tests', 'test-stories.json');
      if (fs.existsSync(storyDataPath)) {
        const storyDataContent = fs.readFileSync(storyDataPath, 'utf8');
        storyData = JSON.parse(storyDataContent);
      }
    }

    // Also check the database for stories that actually exist
    const existingStories = await getStoriesFromDatabase(username, request);

    if (username) {
      // First, filter database stories for this user
      const userDbStories = existingStories.filter(story => {
        if (includeArchived) {
          return story.status === 'archived' || story.status === 'active';
        } else {
          return story.status === 'active';
        }
      });

      if (userDbStories.length > 0) {
        // Return the first available story from database
        const selectedStory = userDbStories[0];
        return selectedStory.id;
      }

      // Fallback to test data filtering
      const userStoryIds = usernameToStoryIds[username];

      if (userStoryIds && userStoryIds.length > 0) {
        // Filter for stories based on status and exclude deleted ones
        const filteredUserStoryIds = userStoryIds.filter(storyId => {
          // Skip deleted stories
          if (deletedStoryIds.has(storyId)) {
            return false;
          }

          // Check if story exists in database
          const dbStory = existingStories.find(s => s.id === storyId);
          if (!dbStory) {
            return false;
          }

          if (storyData) {
            const story = Object.values(storyData).find((s: any) => (s as TestStoryData).id === storyId);
            if (includeArchived) {
              return story && (story as TestStoryData).status === 'archived';
            } else {
              return story && (story as TestStoryData).status === 'active';
            }
          }
          return false; // If we can't read the data, assume it's not active
        });

        if (filteredUserStoryIds.length > 0) {
          // Return the first story ID from the test data
          // The stories should exist in the database based on test setup
          return filteredUserStoryIds[0];
        } else {
          console.log(`❌ NO USER STORIES FOUND for ${username}, falling back to global list`);
          const statusType = includeArchived ? 'archived' : 'active';

          // If we're looking for stories for the story test user and none exist,
          // try to use a user that has stories instead
          if (username === 'apitestuserstory1') {
            // Fall back to regular user who should have stories
            const regularUserStoryIds = usernameToStoryIds['apitestuser'];
            if (regularUserStoryIds && regularUserStoryIds.length > 0) {
              const fallbackFilteredIds = regularUserStoryIds.filter(storyId => {
                if (deletedStoryIds.has(storyId)) return false;
                if (storyData) {
                  const story = Object.values(storyData).find((s: any) => (s as TestStoryData).id === storyId);
                  if (includeArchived) {
                    return story && (story as TestStoryData).status === 'archived';
                  } else {
                    return story && (story as TestStoryData).status === 'active';
                  }
                }
                return false;
              });

              if (fallbackFilteredIds.length > 0) {
                return fallbackFilteredIds[0];
              }
            }
          }

          throw new Error(`No ${statusType} stories found for user '${username}'. This suggests that the test data setup did not create stories for this user.`);
        }
      } else {
        throw new Error(`No stories found for user '${username}'. Available users with stories: ${Object.keys(usernameToStoryIds).join(', ')}`);
      }
    }

    // If no username specified, filter database stories globally
    const globalDbStories = existingStories.filter(story => {
      if (includeArchived) {
        return story.status === 'archived' || story.status === 'active';
      } else {
        return story.status === 'active';
      }
    });

    if (globalDbStories.length > 0) {
      // Return the first available story from database
      const selectedStory = globalDbStories[0];
      return selectedStory.id;
    }

    // Fallback to test data filtering
    const filteredStoryIds = availableStoryIds.filter(storyId => {
      // Skip deleted stories
      if (deletedStoryIds.has(storyId)) {
        return false;
      }

      if (storyData) {
        const story = Object.values(storyData).find((s: any) => (s as TestStoryData).id === storyId);
        if (includeArchived) {
          return story && (story as TestStoryData).status === 'archived';
        } else {
          return story && (story as TestStoryData).status === 'active';
        }
      }
      return false; // If we can't read the data, assume it's not active
    });

    if (filteredStoryIds.length === 0) {
      const statusType = includeArchived ? 'archived' : 'active';
      throw new Error(`No ${statusType} story IDs available. Available: ${availableStoryIds.filter(id => !deletedStoryIds.has(id)).join(', ')}`);
    }

    // Return the first story ID matching the criteria
    const selectedStoryId = filteredStoryIds[0];
    return selectedStoryId;
  }

  function getUserIdWithDailyAssignments(date: string = '2025-08-04'): number {
    // Load the daily assignments data to find a user with assignments for the given date
    const dataDir = path.join(process.cwd(), '..', 'backend', 'data');
    const dailyAssignmentsPath = path.join(dataDir, 'test_daily_assignments.yaml');

    if (!fs.existsSync(dailyAssignmentsPath)) {
      return getAvailableUserId();
    }

    try {
      const dailyAssignmentsContent = fs.readFileSync(dailyAssignmentsPath, 'utf8');
      const dailyAssignments = yaml.load(dailyAssignmentsContent) as any;

      // Find a user with assignments for the given date
      const assignment = dailyAssignments.daily_assignments?.find((a: any) => a.date === date);

      if (assignment) {
        // Get the user ID from test-users.json
        const userDataPath = path.join(process.cwd(), 'tests', 'test-users.json');
        if (fs.existsSync(userDataPath)) {
          const userDataContent = fs.readFileSync(userDataPath, 'utf8');
          const userData = JSON.parse(userDataContent);

          // Find the user by username
          const user = Object.values(userData).find((u: any) => u.username === assignment.username) as any;
          if (user && user.id) {
            return user.id;
          }
        }
      }

      return getAvailableUserId();
    } catch (error) {
      console.log('Error loading daily assignments, falling back to available user ID:', error);
      return getAvailableUserId();
    }
  }

  async function getAvailableQuestionId(request: any, userContext?: {username: string}): Promise<number> {
    // Use provided user context or default to regular user
    const targetUser = userContext || REGULAR_USER;

    // For daily question endpoints, get a question ID from the daily questions endpoint
    // This ensures the test uses a question that's actually assigned to the user
    try {
      const sessionCookie = await loginUser(request, targetUser);
      const response = await request.get(`${baseURL}/v1/daily/questions/2025-08-04`, {
        headers: {
          'Cookie': sessionCookie
        }
      });

      if (response.status() === 200) {
        const data = await response.json();
        if (data.questions && data.questions.length > 0) {
          return data.questions[0].question_id;
        }
      }
    } catch (error) {
      console.warn('Failed to get question from daily questions endpoint, trying quiz endpoint');
    }

    // For admin endpoints, try to get a question from the quiz endpoint using the target user
    try {
      const sessionCookie = await loginUser(request, targetUser);
      const response = await request.get(`${baseURL}/v1/quiz/question`, {
        headers: {
          'Cookie': sessionCookie
        }
      });

      if (response.status() === 200) {
        const question = await response.json();
        if (question && question.id) {
          return question.id;
        }
      }
    } catch (error) {
      console.warn('Failed to get question from quiz endpoint, falling back to hardcoded ID');
    }

    // Fallback to hardcoded ID if all else fails
    if (availableQuestionIds.length === 0) {
      throw new Error('No available question IDs for testing');
    }
    return availableQuestionIds[0];
  }

  async function getAvailableSectionId(username?: string, request?: any): Promise<number> {
    if (availableSectionIds.length === 0) {
      throw new Error('No available section IDs for testing');
    }

    if (!username) {
      return availableSectionIds[0]; // Use the first available section ID if no username specified
    }

    // Get sections that belong to this user's stories in the database
    const existingStories = await getStoriesFromDatabase(username, request);
    // console.log(`🔍 DATABASE STORIES for ${username}: ${existingStories.map(s => `${s.id}(${s.status})`).join(', ')}`);

    // Find sections that belong to stories that exist in the database
    const userSections: number[] = [];

    if (!testStoriesData) {
      console.warn('No stories data available for section filtering');
      return availableSectionIds[0];
    }

    for (const story of Object.values(testStoriesData)) {
      if (story.username === username && story.sections && story.sections.length > 0) {
        // Check if this story exists in the database
        const dbStory = existingStories.find(s => s.id === story.id);
        if (dbStory) {
          for (const section of story.sections) {
            if (!userSections.includes(section.id)) {
              userSections.push(section.id);
            }
          }
        }
      }
    }

    if (userSections.length === 0) {
      console.warn(`No sections found for user ${username} in database, using first available section`);
      return availableSectionIds[0];
    }

    return userSections[0]; // Return the first section belonging to this user
  }

  function removeUserId(id: number) {
    availableUserIds = availableUserIds.filter(userId => userId !== id);
  }

  function removeQuestionId(id: number) {
    availableQuestionIds = availableQuestionIds.filter(questionId => questionId !== id);
  }

  function markConversationAsDeleted(conversationId: string) {
    deletedConversationIds.add(conversationId);
  }

  function markStoryAsDeleted(storyId: number) {
    // console.log(`🔴 MARKING STORY AS DELETED: ${storyId}, current deleted stories: ${Array.from(deletedStoryIds).join(', ')}`);
    deletedStoryIds.add(storyId);

    // Also remove from available story IDs and username mappings
    availableStoryIds = availableStoryIds.filter(id => id !== storyId);
    Object.values(usernameToStoryIds).forEach(userStories => {
      const index = userStories.indexOf(storyId);
      if (index > -1) {
        userStories.splice(index, 1);
      }
    });

    // console.log(`🔴 AFTER DELETION - Available story IDs: ${availableStoryIds.join(', ')}`);
  }

  function markSnippetAsDeleted(snippetId: number) {
    // console.log(`🔴 MARKING SNIPPET AS DELETED: ${snippetId}, current deleted snippets: ${Array.from(deletedSnippetIds).join(', ')}`);
    deletedSnippetIds.add(snippetId);

    // Also remove from available snippet IDs
    availableSnippetIds = availableSnippetIds.filter(id => id !== snippetId);

    // console.log(`🔴 AFTER DELETION - Available snippet IDs: ${availableSnippetIds.join(', ')}`);
  }

  // Helper function to determine what type of ID we need and get the appropriate value
  async function getReplacementId(pathString: string, paramKey: string, request: any, userContext?: {username: string}, method?: string, isErrorCase: boolean = false): Promise<{value: number | string; type: 'user' | 'question' | 'daily_user' | 'date' | 'role' | 'provider' | 'story' | 'conversation' | 'snippet'}> {
    if (paramKey === 'userId') {
      return {value: getUserIdWithDailyAssignments(), type: 'daily_user'};
    }

    if (paramKey === 'date') {
      // Use a valid date format for date parameters
      return {value: '2025-08-04', type: 'date'};
    }

    if (paramKey === 'provider') {
      // Use a valid provider code defined in config and tests
      return {value: 'ollama', type: 'provider'};
    }

    if (paramKey === 'service') {
      // For error cases, use an invalid service name to trigger validation errors
      if (isErrorCase) {
        return {value: 'invalid-service', type: 'provider'};
      }
      // Use a valid service code for translation services
      return {value: 'google', type: 'provider'};
    }

    if (paramKey === 'questionId') {
      // For error cases, use 'invalid' to trigger validation errors
      if (isErrorCase) {
        return {value: 'invalid', type: 'question'};
      }
      const questionId = await getAvailableQuestionId(request, userContext);
      return {value: questionId, type: 'question'};
    }

    if (paramKey === 'roleId') {
      // For roleId parameters, we need to get a valid role ID
      // For role removal operations, use the actual 'user' role ID from test roles data to ensure correctness
      if (pathString.includes('/roles/') && pathString.includes('/{roleId}')) {
        const userRoleId = testRolesData?.['user']?.id;
        return {value: userRoleId ?? 1, type: 'role'};
      }

      // Use the first available role from the test data for other role operations
      const roleNames = Object.keys(testRolesData);
      if (roleNames.length > 0) {
        const firstRole = testRolesData[roleNames[0]];
        return {value: firstRole.id, type: 'role'};
      }
      // Fallback to admin role ID if no roles found
      return {value: 1, type: 'role'};
    }

    // Section endpoints are now handled in the main test loop to avoid conflicts
    // with the story endpoint logic

    if (paramKey === 'id' || paramKey === 'conversationId') {
      // console.log(`🔍 GETTING REPLACEMENT ID for param: ${paramKey}, path: ${pathString}, method: ${method}, isErrorCase: ${isErrorCase}`);
      // Check if this is a conversation-related endpoint
      if (pathString.includes('/conversations/') || pathString.includes('/ai/conversations/')) {
        // For error cases (like 404 tests), use a fake UUID that doesn't exist
        if (isErrorCase) {
          const fakeUuid = '00000000-0000-0000-0000-000000000001';
          return {value: fakeUuid, type: 'conversation'};
        }
        // For DELETE operations, use a conversation that can be safely deleted
        const forDeletion = method === 'DELETE';
        let username = userContext?.username;
        // If the current user doesn't have conversations, use the dedicated test user
        if (username && !Object.keys(testConversationsData || {}).some(key => testConversationsData[key].username === username)) {
          username = 'apitestuser';
        }
        const conversationId = getAvailableConversationId(username || 'apitestuser', forDeletion);
        return {value: conversationId, type: 'conversation'};
      }

      // Check if this is a section-related endpoint first
      if (pathString.includes('/story/section/') || pathString.includes('/section/')) {
        // console.log(`🔍 SECTION ENDPOINT DETECTED: ${pathString}, method: ${method}`);
        // For section endpoints, we need to get a section ID that belongs to the user's stories
        const sectionId = await getAvailableSectionId(userContext?.username, request);
        // console.log(`📋 RETURNING SECTION ID: ${sectionId} for user: ${userContext?.username}, path: ${pathString}`);
        return {value: sectionId, type: 'section'};
      }

      // Check if this is a story-related endpoint
      if (pathString.includes('/story/') || pathString.includes('/stories/')) {
        // console.log(`🔍 STORY ENDPOINT DETECTED: ${pathString}, method: ${method}`);
        // For admin users, use their own stories; for regular users, use story test user
        // For DELETE and set-current operations, we need an archived story
        const includeArchived = method === 'DELETE' || pathString.includes('/set-current');
        let username = userContext?.username;

        // Admin users should use their own stories, not story test user's stories
        if (username === 'apitestadmin') {
          // Use admin's stories directly - get available story ID for admin
          const storyId = await getAvailableStoryId(username, includeArchived, undefined, request);
          // console.log(`📋 RETURNING STORY ID: ${storyId} for admin user: ${username}, path: ${pathString}`);
          return {value: storyId, type: 'story'};
        }

        // For story operations, use the dedicated story test user if the current user doesn't have stories
        const userStoryIds = usernameToStoryIds[username || ''];
        if (!userStoryIds || userStoryIds.length === 0) {
          // Try the story test user first, then fall back to regular user if needed
          let storyTestUserStoryIds = usernameToStoryIds['apitestuserstory1'];
          if (storyTestUserStoryIds && storyTestUserStoryIds.length > 0) {
            username = 'apitestuserstory1';
          } else {
            // Fall back to regular user if story test user doesn't have stories
            username = 'apitestuser';
          }
        }
        const storyId = await getAvailableStoryId(username || 'apitestuserstory1', includeArchived, undefined, request);
        // console.log(`📋 RETURNING STORY ID: ${storyId} for user: ${username}, path: ${pathString}`);
        return {value: storyId, type: 'story'};
      }

      // Check if this is a snippet-related endpoint
      if (pathString.includes('/snippets/')) {
        // console.log(`🔍 SNIPPET ENDPOINT DETECTED: ${pathString}, method: ${method}`);
        // For error cases (like 404 tests), use a fake ID that doesn't exist
        if (isErrorCase) {
          return {value: 999999999, type: 'snippet'};
        }

        // Filter available snippet IDs to only include those belonging to the current user
        const currentUsername = userContext?.username || 'apitestuser';
        const userSnippetIds = availableSnippetIds.filter(id => {
          // Find the snippet by ID and check if it belongs to the current user
          const snippetDataPath = path.join(process.cwd(), 'tests', 'test-snippets.json');
          if (fs.existsSync(snippetDataPath)) {
            const snippetDataContent = fs.readFileSync(snippetDataPath, 'utf8');
            const snippetData = JSON.parse(snippetDataContent);
            const snippetKey = Object.keys(snippetData).find(key => snippetData[key].id === id);
            return snippetKey && snippetData[snippetKey].username === currentUsername;
          }
          return false;
        });

        // console.log(`📋 Available snippets for user ${currentUsername}: ${userSnippetIds.join(', ')}`);

        // For DELETE operations, use a snippet that can be safely deleted
        const forDeletion = method === 'DELETE';
        if (forDeletion && userSnippetIds.length > 0) {
          const snippetId = userSnippetIds[0];
          return {value: snippetId, type: 'snippet'};
        }

        // Return the first available snippet ID for this user
        if (userSnippetIds.length === 0) {
          throw new Error(`No available snippet IDs for user ${currentUsername}. Test data setup may have failed or user may not have any snippets.`);
        }
        const snippetId = userSnippetIds[0];
        // console.log(`📋 RETURNING SNIPPET ID: ${snippetId} for user: ${currentUsername}, path: ${pathString}`);
        return {value: snippetId, type: 'snippet'};
      }

      // Check if this is a question-related endpoint
      if (pathString.includes('/questions/') || pathString.includes('/quiz/question/') || pathString.includes('/daily/questions/')) {
        const questionId = await getAvailableQuestionId(request, userContext);
        return {value: questionId, type: 'question'};
      }

      // Check if this is a user-related endpoint (most endpoints with {id} are user-related)
      if (pathString.includes('/userz/') || pathString.includes('/admin/backend/userz/') ||
        pathString.includes('/reset-password') || pathString.includes('/roles') ||
        pathString.includes('/clear') || pathString.includes('/daily/') ||
        pathString.includes('/admin/backend/stories/') || pathString.includes('/admin/backend/questions/')) {

        // For roles listing endpoint, pick an existing user ID from the seeded data
        if (pathString.includes('/admin/backend/userz/') && pathString.endsWith('/roles') && !pathString.includes('/{roleId}')) {
          return {value: getAvailableUserId(), type: 'user'};
        }

        // Special handling for role removal operations to avoid removing admin role from admin user
        if (pathString.includes('/roles/') && pathString.includes('/{roleId}')) {
          // For role removal, use a different user to avoid affecting the admin user
          // Find the role test user ID from test-users.json
          const userDataPath = process.cwd() + '/tests/test-users.json';
          if (fs.existsSync(userDataPath)) {
            const userDataContent = fs.readFileSync(userDataPath, 'utf8');
            const userData = JSON.parse(userDataContent);

            // Find the role test user by username
            const roleTestUser = Object.values(userData).find((u: any) => u.username === 'roletestuser') as any;
            const roleTestUserId = roleTestUser?.id;

            if (roleTestUserId) {
              return {value: roleTestUserId, type: 'user'};
            }
          }
        }

        // Special handling: for password reset, operate on a load-test user to avoid flakiness
        if (pathString.includes('/reset-password')) {
          const loadId = getLoadTestUserId();
          if (loadId) {
            return {value: loadId, type: 'user'};
          }
        }

        // For any admin user operations, avoid admin/test control accounts and pick a random safe user ID
        if (pathString.includes('/admin/backend/userz/')) {
          const randomSafeId = getRandomSafeUserId();
          return {value: randomSafeId, type: 'user'};
        }

        return {value: getAvailableUserId(), type: 'user'};
      }

      // Default to user ID for any other {id} parameter
      return {value: getAvailableUserId(), type: 'user'};
    }

    // If we reach here, we have an unknown parameter type - this should fail the test
    throw new Error(`Unknown parameter type '${paramKey}' for path '${pathString}'. This indicates a missing case in getReplacementId.`);
  }

  // Helper function to replace path parameters with appropriate IDs
  async function replacePathParameters(pathString: string, pathParams: Record<string, any> | undefined, request: any, userContext?: {username: string}, method?: string, isErrorCase: boolean = false): Promise<string> {
    if (!pathParams) return pathString;

    // console.log(`🔄 REPLACING PATH PARAMETERS for path: ${pathString}, method: ${method}`);
    let resultPath = pathString;
    for (const [key, value] of Object.entries(pathParams)) {
      const replacement = await getReplacementId(pathString, key, request, userContext, method, isErrorCase);
      // console.log(`🔄 REPLACED {${key}}=${value} with ${replacement.value} (type: ${replacement.type})`);
      resultPath = resultPath.replace(`{${key}}`, replacement.value.toString());
    }
    // console.log(`🔄 FINAL PATH: ${resultPath}`);
    return resultPath;
  }

  async function loginUser(request: any, user: {username: string; password: string}): Promise<string> {
    // Generate the proper login request body using the user context
    const loginRequestBody = generateRequestBody(
      swaggerSpec.paths['/v1/auth/login']['post'].requestBody,
      '/v1/auth/login',
      'post',
      user
    );

    const loginResponse = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginRequestBody,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    await assertStatus(loginResponse, 200, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginRequestBody
    });

    const cookies = loginResponse.headers()['set-cookie'];
    const sessionCookie = cookies?.split(',').find(cookie =>
      cookie.trim().startsWith(`${SESSION_NAME}=`)
    );

    return sessionCookie || '';
  }

  function generateRequestBodyForTest(testCase: TestCase, userContext: {username: string; password: string}): any {
    if (!testCase.requestBody) return undefined;

    // For login endpoints, generate request body with current user context
    if (testCase.path.includes('/auth/login')) {
      return generateRequestBody(
        swaggerSpec.paths[testCase.path][testCase.method.toLowerCase()].requestBody,
        testCase.path,
        testCase.method,
        userContext
      );
    }

    // For profile update endpoints, generate request body with current user context
    if (testCase.path.includes('/userz/profile')) {
      return generateRequestBody(
        swaggerSpec.paths[testCase.path][testCase.method.toLowerCase()].requestBody,
        testCase.path,
        testCase.method,
        userContext
      );
    }

    // For bookmark endpoints, we need to handle conversation_id and message_id dynamically
    if (testCase.path.includes('/conversations/bookmark')) {
      const requestBody = { ...testCase.requestBody };

      // Replace conversation_id and message_id with actual values from test data
      if (requestBody.conversation_id) {
        const conversation = findConversationForUser(userContext.username);
        if (conversation) {
          requestBody.conversation_id = conversation.id;
        } else {
          throw new Error(`No conversation available for user ${userContext.username} for bookmark test.`);
        }
      }

      if (requestBody.message_id) {
        const conversation = findConversationForUser(userContext.username);
        if (conversation && conversation.messages && conversation.messages.length > 0) {
          requestBody.message_id = conversation.messages[0].id;
        } else {
          throw new Error(`No message available for user ${userContext.username} for bookmark test.`);
        }
      }

      return requestBody;
    }

    // For quiz answer endpoints, we need to handle question_id dynamically
    if (testCase.path.includes('/quiz/answer')) {
      // This will be handled in the test execution where we can get a valid question ID
      return testCase.requestBody;
    }

    return testCase.requestBody;
  }

  async function logResponse(testCase: TestCase, response: any, userType: string, username: string, actualUrl?: string) {
    const statusCode = response.status().toString();
    const responseBody = await response.text();
    const urlToLog = actualUrl || testCase.path;

    // Determine whether this path is excluded (use pathname when possible)
    let pathForCheck = testCase.path;
    try {
      if (urlToLog) {
        pathForCheck = new URL(urlToLog).pathname;
      }
    } catch (e) {
      // ignore and fallback to testCase.path
    }
    const excluded = isExcludedPath(pathForCheck);

    if (!testCase.expectedStatusCodes.includes(statusCode)) {
      console.log(`\n❌ UNEXPECTED STATUS CODE:`);
      console.log(`  Endpoint: ${testCase.method} ${urlToLog}`);
      console.log(`  User: ${userType} (${username})`);
      console.log(`  Payload: ${JSON.stringify(testCase.requestBody, null, 2)}`);
      console.log(`  Expected Status Codes: ${testCase.expectedStatusCodes.join(', ')}`);
      console.log(`  Actual Status Code: ${statusCode}`);
      console.log(`  Response Body: ${responseBody}`);
      console.log(``);

      // Fail the test on unexpected status code
      throw new Error(`Unexpected status code ${statusCode} for ${testCase.method} ${urlToLog}. Expected: ${testCase.expectedStatusCodes.join(', ')}`);
    } else {
      // Columnar success log with URL last; mark excluded tests with a different symbol and tag
      const expectedText = testCase.expectedStatusCodes.join(', ');
      const symbol = excluded ? '⏭️' : '✅';
      const excludedTag = excluded ? ' [EXCLUDED]' : '';
      const columns = [
        symbol,
        testCase.method.padEnd(6),
        statusCode.padEnd(3),
        `[Expected: ${expectedText}]`.padEnd(20),
        `${userType}: ${username}`.padEnd(24),
        urlToLog + excludedTag
      ];
      console.log(columns.join('  '));
    }

    // Removed test throttling: no delay needed now that nginx rate limits are disabled in tests
  }

  function sanitizeHeaders(headers: Record<string, any>): Record<string, any> {
    const clone: Record<string, any> = {...headers};
    if (clone['Cookie']) clone['Cookie'] = '[REDACTED]';
    if (clone['cookie']) clone['cookie'] = '[REDACTED]';
    if (clone['Authorization']) clone['Authorization'] = '[REDACTED]';
    if (clone['authorization']) clone['authorization'] = '[REDACTED]';
    return clone;
  }

  // Removed delay helper: no longer needed without rate limiting in tests

  // Replace path parameters without making any authenticated requests
  function replacePathParametersUnauthenticated(path: string, pathParams?: Record<string, any>): string {
    if (!pathParams) return path;
    let result = path;
    for (const key of Object.keys(pathParams)) {
      let value: string | number = 1;
      if (key === 'date') value = '2025-08-04';
      else if (key === 'provider') value = 'ollama';
      else if (key === 'service') value = 'google';
      else if (key === 'roleId') value = 1;
      else if (key === 'questionId') value = 1;
      else if (key === 'userId') value = 1;
      else if (key === 'id' || key === 'conversationId') {
        // Check if this is a conversation-related endpoint
        if (path.includes('/conversations/')) {
          // For conversation endpoints, use an invalid UUID to test 404 scenarios
          value = '00000000-0000-0000-0000-000000000001';
        } else if (path.includes('/story/')) {
          // For unauthenticated path replacement, we need an active story ID that exists
          // but we want to test 404 scenarios, so we'll use a large number that doesn't exist
          value = 999999999;
        } else {
          value = 1;
        }
      }
      result = result.replace(`{${key}}`, String(value));
    }
    return result;
  }

  // E2E exclusion helpers: allow test runner to skip problematic endpoints via
  // the E2E_EXCLUDE_PATHS environment variable (comma-separated list).
  // Each entry may be a substring match or a regex prefixed with `re:`.
  //
  // README NOTE:
  // To temporarily skip flaky or external-dependent endpoints during E2E runs,
  // set E2E_EXCLUDE_PATHS before running tests. Examples:
  //
  //   # Skip the AI-fix endpoint by substring
  //   export E2E_EXCLUDE_PATHS="/questions/ai-fix"
  //
  //   # Skip multiple endpoints including a regex match
  //   export E2E_EXCLUDE_PATHS="/questions/ai-fix,re:^/v1/settings/test-ai$"
  //
  // This is intended as a temporary measure while we add a mock AI backend
  // or otherwise stabilize those endpoints. Prefer adding mocks or enabling
  // provider stubs for CI instead of long-term exclusions.
  function getExcludedPaths(): string[] {
    const raw = process.env.E2E_EXCLUDE_PATHS || '';
    if (!raw) return [];
    return raw.split(',').map(s => s.trim()).filter(Boolean);
  }

  const EXCLUDED_PATHS = getExcludedPaths();

  // Log the effective exclusion list at test startup so CI logs show what was skipped
  console.log('E2E excluded paths:', EXCLUDED_PATHS.length ? EXCLUDED_PATHS.join(', ') : '(none)');

  function isExcludedPath(path: string): boolean {
    if (!EXCLUDED_PATHS || EXCLUDED_PATHS.length === 0) return false;
    for (const pattern of EXCLUDED_PATHS) {
      if (pattern.startsWith('re:')) {
        try {
          const re = new RegExp(pattern.slice(3));
          if (re.test(path)) return true;
        } catch (e) {
          // Invalid regex - ignore this pattern
          console.warn(`Invalid E2E_EXCLUDE_PATH pattern: ${pattern}`);
          continue;
        }
      } else {
        if (path.includes(pattern)) return true;
      }
    }
    return false;
  }

  test.describe('Regular User Tests', () => {
    test('should test endpoints expecting 200 status for regular user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      const sessionCookie = await loginUser(request, REGULAR_USER);

      const regularUserCases = happyPathCases.filter(testCase =>
        !testCase.requiresAdmin && testCase.requiresAuth
      );

      for (const testCase of regularUserCases) {
        // Determine which user to use for this test case
        const isStoryEndpoint = testCase.path.includes('/story/') || testCase.path.includes('/stories/');
        const currentUser = isStoryEndpoint ? STORY_TEST_USER : REGULAR_USER;

        // Check if this endpoint should be excluded
        const expandedPath = await replacePathParameters(testCase.path, testCase.pathParams, request, currentUser, testCase.method, false);
        if (isExcludedPath(expandedPath)) continue;

        let endpointPath = expandedPath;
        const currentSessionCookie = await loginUser(request, currentUser);

        // Add path parameters
        if (testCase.pathParams) {
          for (const [key, value] of Object.entries(testCase.pathParams)) {
            if (key === 'questionId') {
              const questionId = await getAvailableQuestionId(request, currentUser);
              endpointPath = endpointPath.replace(`{${key}}`, questionId.toString());
            } else if (key === 'id' && value === 'STORY_ID_PLACEHOLDER') {
              // For story endpoints, use appropriate story ID based on operation
              // Update endpointPath to include the resolved story ID for pattern matching
              const tempEndpointPath = endpointPath.replace(`{${key}}`, '123'); // temporary replacement for pattern matching
              // DELETE and set-current operations need archived stories to restore
              const includeArchived = testCase.method === 'DELETE' || tempEndpointPath.includes('/set-current');
              const isStoryEndpoint = tempEndpointPath.includes('/story/') || tempEndpointPath.includes('/stories/');
              // Load story data for synchronous access
              const storyDataPath = path.join(process.cwd(), 'tests', 'test-stories.json');
              let storyData = null;
              if (fs.existsSync(storyDataPath)) {
                const storyDataContent = fs.readFileSync(storyDataPath, 'utf8');
                storyData = JSON.parse(storyDataContent);
              }
              let storyId;
              // For all story operations, use the getAvailableStoryId function which handles deleted story tracking
              let username = currentUser.username;

              // Admin users should use their own stories, not story test user's stories
              if (username === 'apitestadmin') {
                // Use admin's stories directly - get available story ID for admin
                storyId = await getAvailableStoryId(username, includeArchived, storyData, request);
              } else {
                // For story operations, use the dedicated story test user if the current user doesn't have stories
                const userStoryIds = usernameToStoryIds[username || ''];
                if (!userStoryIds || userStoryIds.length === 0) {
                  // Try the story test user first, then fall back to regular user if needed
                  let storyTestUserStoryIds = usernameToStoryIds['apitestuserstory1'];
                  if (storyTestUserStoryIds && storyTestUserStoryIds.length > 0) {
                    username = 'apitestuserstory1';
                  } else {
                    // Fall back to regular user if story test user doesn't have stories
                    username = 'apitestuser';
                  }
                }
                storyId = await getAvailableStoryId(username, includeArchived, storyData, request);
              }
              endpointPath = endpointPath.replace(`{${key}}`, storyId.toString());
            } else if (key === 'id' && endpointPath.includes('/conversations/')) {
              // For conversation endpoints, use appropriate conversation ID based on operation
              // DELETE operations need conversations that can be safely deleted
              const forDeletion = testCase.method === 'DELETE';
              const conversationId = getAvailableConversationId(currentUser.username, forDeletion);
              endpointPath = endpointPath.replace(`{${key}}`, conversationId);
            } else if (key === 'conversationId') {
              // For conversationId parameters, use appropriate conversation ID based on operation
              // DELETE operations need conversations that can be safely deleted
              const forDeletion = testCase.method === 'DELETE';
              const conversationId = getAvailableConversationId(currentUser.username, forDeletion);
              endpointPath = endpointPath.replace(`{${key}}`, conversationId);
            } else {
              endpointPath = endpointPath.replace(`{${key}}`, String(value));
            }
          }
        }

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, value.toString());
          }
        }

        const requestOptions: any = {
          headers: {
            'Cookie': currentSessionCookie
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, currentUser);
          requestOptions.headers['Content-Type'] = 'application/json';
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Log the response details and assert status
        await logResponse(testCase, response, isStoryEndpoint ? 'Story Test User' : 'Regular User', currentUser.username, url.toString());
        await assertStatus(response, testCase.expectedStatusCodes.map(code => parseInt(code, 10)), {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });

        // Mark conversation as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && (endpointPath.includes('/conversations/') || endpointPath.includes('/ai/conversations/'))) {
          // Extract conversation ID from the URL - handle both /conversations/ and /ai/conversations/ patterns
          const conversationMatch = endpointPath.match(/\/conversations\/([^\/]+)/) || endpointPath.match(/\/ai\/conversations\/([^\/]+)/);
          if (conversationMatch) {
            const deletedConversationId = conversationMatch[1];
            markConversationAsDeleted(deletedConversationId);
          }
        }

        // Mark story as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 200 && (endpointPath.includes('/story/') || endpointPath.includes('/stories/'))) {
          // Extract story ID from the URL - handle both /story/ and /stories/ patterns
          const storyMatch = endpointPath.match(/\/(?:story|stories)\/(\d+)/);
          if (storyMatch) {
            const deletedStoryId = parseInt(storyMatch[1]);
            markStoryAsDeleted(deletedStoryId);
          }
        }

        // Mark snippet as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && endpointPath.includes('/snippets/')) {
          // Extract snippet ID from the URL
          const snippetMatch = endpointPath.match(/\/snippets\/(\d+)/);
          if (snippetMatch) {
            const deletedSnippetId = parseInt(snippetMatch[1]);
            markSnippetAsDeleted(deletedSnippetId);
          } else {
            console.log(`❌ Could not extract snippet ID from path: ${endpointPath}`);
          }
        }
      }
    });

    test('should test public endpoints without authentication', async ({request}) => {
      const publicCases = testCases.filter(testCase => !testCase.requiresAuth);

      for (const testCase of publicCases) {
        // Check if this endpoint should be excluded
        const expandedPath = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        if (isExcludedPath(expandedPath)) continue;

        let endpointPath = expandedPath;

        // Add path parameters
        if (testCase.pathParams) {
          for (const [key, value] of Object.entries(testCase.pathParams)) {
            if (key === 'questionId') {
              const questionId = await getAvailableQuestionId(request, ADMIN_USER);
              endpointPath = endpointPath.replace(`{${key}}`, questionId.toString());
            } else {
              endpointPath = endpointPath.replace(`{${key}}`, value.toString());
            }
          }
        }

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, value.toString());
          }
        }

        const requestOptions: any = {};

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, ADMIN_USER);
          requestOptions.headers = {'Content-Type': 'application/json'};
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        await logResponse(testCase, response, 'Admin User', 'apitestadmin', url.toString());
        await assertStatus(response, testCase.expectedStatusCodes.map(code => parseInt(code, 10)), {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });
      }
    });

    test('should test specific error scenarios for regular user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      // Filter out admin endpoints for regular user error tests
      const regularUserErrorCases = errorCases.filter(testCase => !testCase.requiresAdmin);

      for (const testCase of regularUserErrorCases) {
        // Check if this endpoint should be excluded
        const expandedPath = await replacePathParameters(testCase.path, testCase.pathParams, request, REGULAR_USER, testCase.method, true);
        if (isExcludedPath(expandedPath)) continue;

        // Use the expanded path that already has the correct parameters replaced
        let endpointPath = expandedPath;

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, String(value));
          }
        }

        const requestOptions: any = {};

        // For 400 error tests on authenticated endpoints, we still need to authenticate
        // For 400 error tests on unauthenticated endpoints, we don't provide auth
        if (testCase.expectedStatusCodes.includes('400') && testCase.requiresAuth) {
          // For 400 error tests on authenticated endpoints, login first
          const sessionCookie = await loginUser(request, REGULAR_USER);
          requestOptions.headers = {
            'Cookie': sessionCookie
          };
        } else if (testCase.requiresAuth) {
          // For other authenticated endpoints, login first
          const sessionCookie = await loginUser(request, REGULAR_USER);
          requestOptions.headers = {
            'Cookie': sessionCookie
          };
        }

        if (testCase.requestBody) {
          // For error test cases, use the original requestBody (invalid data)
          // For happy path cases, use generateRequestBodyForTest (valid data)
          if (testCase.expectedStatusCodes.includes('400')) {
            requestOptions.data = testCase.requestBody;
          } else {
            requestOptions.data = generateRequestBodyForTest(testCase, REGULAR_USER);
          }
          requestOptions.headers = requestOptions.headers || {};
          requestOptions.headers['Content-Type'] = 'application/json';
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        await logResponse(testCase, response, 'Regular User', 'apitestuser', url.toString());
        await assertStatus(response, testCase.expectedStatusCodes.map(code => parseInt(code, 10)), {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });
      }
    });
  });

  test.describe('Admin User Tests', () => {
    test('should test endpoints expecting 200 status for admin user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      const sessionCookie = await loginUser(request, ADMIN_USER);
      // Exclude AI-fix endpoint because it requires a live AI backend which may not be available in CI
      // Build adminCases by expanding path parameters so regex exclusions like
      // re:/v1/admin/backend/questions/[0-9]+/ai-fix match correctly.
      const adminCases: TestCase[] = [];
      for (const testCase of happyPathCases) {
        if (!testCase.requiresAdmin) continue;
        // Skip story endpoints for admin users
        if (testCase.path.includes('/story')) continue;
        const expandedPath = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        if (isExcludedPath(expandedPath)) continue;
        adminCases.push(testCase);
      }
      console.log(`Processing ${adminCases.length} admin happy path cases`);

      for (const testCase of adminCases) {
        // console.log(`Processing test case: ${testCase.method} ${testCase.path}`);

        // Replace path parameters with appropriate IDs
        const path = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        const endpointPath = path;

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, value.toString());
          }
        }

        const requestOptions: any = {
          headers: {
            'Cookie': sessionCookie
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, ADMIN_USER);
          requestOptions.headers['Content-Type'] = 'application/json';
        }

        // Precondition: for role removal, ensure the user actually has the role first
        if (
          testCase.method === 'DELETE' &&
          (testCase.path.includes('/v1/admin/backend/userz/{id}/roles/{roleId}') ||
           (path.includes('/v1/admin/backend/userz/') && path.includes('/roles/')))
        ) {
          const match = path.match(/\/v1\/admin\/backend\/userz\/(\d+)\/roles\/(\d+)/);
          if (match) {
            const userIdForRole = Number(match[1]);
            const roleIdForRemoval = Number(match[2]);
            // Assign role before attempting removal (idempotent if already assigned)
            await request.post(`${baseURL}/v1/admin/backend/userz/${userIdForRole}/roles`, {
              headers: {
                'Cookie': sessionCookie,
                'Content-Type': 'application/json'
              },
              data: {role_id: roleIdForRemoval}
            });
          }
        }

        // For force-send endpoint, check if reminderuser exists first
        if (path.includes('/force-send')) {
          try {
            // Try to find reminderuser in the database first
            const userCheckResponse = await request.get(`${baseURL}/v1/admin/backend/userz`, {
              headers: {
                'Cookie': sessionCookie
              }
            });

            if (userCheckResponse.status() === 200) {
              const responseData = await userCheckResponse.json();
              const users = responseData.users || [];
              const reminderUser = users.find((u: any) => u.username === 'reminderuser');
              if (!reminderUser) {
                throw new Error(`Test data not properly loaded: reminderuser not found in database users list. Available users: ${users.map((u: any) => u.username).join(', ')}`);
              }
            } else if (userCheckResponse.status() === 404) {
              const responseText = await userCheckResponse.text();
              throw new Error(`Test data not properly loaded: reminderuser not found in database. Response: ${responseText}. This indicates the setup-test-db command may have failed or the database state is inconsistent.`);
            } else {
              throw new Error(`Unexpected status code when checking for reminderuser: ${userCheckResponse.status()}`);
            }
          } catch (error) {
            if (error instanceof Error && error.message.includes('Test data not properly loaded')) {
              throw error; // Re-throw our custom error
            }
            console.log('⚠️  Failed to check reminderuser existence:', error);
            // Continue with the test anyway for other errors
          }
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Remove user ID from available list if this was a DELETE operation on a user endpoint
        if (testCase.method === 'DELETE' && testCase.path.includes('/{id}') &&
          (testCase.path.includes('/userz/') || (testCase.path.includes('/admin/backend/userz/') && !testCase.path.includes('/roles/')))) {
          // Extract the user ID that was used in the path
          const pathMatch = path.match(/\/(\d+)(?:\/|$)/);
          if (pathMatch) {
            const deletedUserId = parseInt(pathMatch[1]);
            removeUserId(deletedUserId);
            // console.log(`Removed user ID ${deletedUserId} from available list after DELETE`);
          }
        }

        // Mark conversation as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && (endpointPath.includes('/conversations/') || endpointPath.includes('/ai/conversations/'))) {
          // Extract conversation ID from the URL - handle both /conversations/ and /ai/conversations/ patterns
          const conversationMatch = endpointPath.match(/\/conversations\/([^\/]+)/) || endpointPath.match(/\/ai\/conversations\/([^\/]+)/);
          if (conversationMatch) {
            const deletedConversationId = conversationMatch[1];
            markConversationAsDeleted(deletedConversationId);
            // console.log(`Marked conversation as deleted: ${deletedConversationId}`);
          }
        }

        await logResponse(testCase, response, 'Admin User', 'apitestadmin', url.toString());
        await assertStatus(response, testCase.expectedStatusCodes.map(code => parseInt(code, 10)), {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });
      }
    });

    test('should test all non-admin authenticated endpoints for admin user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      const sessionCookie = await loginUser(request, ADMIN_USER);

      const regularUserCases = testCases.filter(testCase =>
        !testCase.requiresAdmin && testCase.requiresAuth && !testCase.path.includes('/story') && !testCase.path.includes('/conversations/') && !testCase.path.includes('/userz/profile') && !testCase.path.includes('/quiz/') && !testCase.path.includes('/progress/') && !testCase.path.includes('/settings/')
      );

      for (const testCase of regularUserCases) {
        // console.log(`Processing test case: ${testCase.method} ${testCase.path}`);

        // Replace path parameters with appropriate IDs
        const path = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        const endpointPath = path;

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, value.toString());
          }
        }

        const requestOptions: any = {
          headers: {
            'Cookie': sessionCookie
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, ADMIN_USER);

          // Handle question_id in request body for quiz answer endpoints
          if (testCase.path.includes('/quiz/answer') && requestOptions.data && requestOptions.data.question_id === 1) {
            const questionId = await getAvailableQuestionId(request, ADMIN_USER);
            requestOptions.data.question_id = questionId;
            // console.log(`Replaced question_id in request body with ${questionId} for ${testCase.path}`);
          }

          requestOptions.headers['Content-Type'] = 'application/json';
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Remove user ID from available list if this was a DELETE operation
        if (testCase.method === 'DELETE' && testCase.path.includes('/{id}') && !testCase.path.includes('/roles/')) {
          const deletedUserId = testCase.pathParams?.id || 1; // Default to 1 if not found
          removeUserId(deletedUserId);
          // console.log(`Removed user ID ${deletedUserId} from available list after DELETE`);
        }

        // Mark conversation as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && (endpointPath.includes('/conversations/') || endpointPath.includes('/ai/conversations/'))) {
          // Extract conversation ID from the URL - handle both /conversations/ and /ai/conversations/ patterns
          const conversationMatch = endpointPath.match(/\/conversations\/([^\/]+)/) || endpointPath.match(/\/ai\/conversations\/([^\/]+)/);
          if (conversationMatch) {
            const deletedConversationId = conversationMatch[1];
            markConversationAsDeleted(deletedConversationId);
            // console.log(`Marked conversation as deleted: ${deletedConversationId}`);
          }
        }

        await logResponse(testCase, response, 'Admin User', 'apitestadmin', url.toString());
        await assertStatus(response, testCase.expectedStatusCodes.map(code => parseInt(code, 10)), {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });
      }
    });

    test('should test admin error scenarios for admin user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      const sessionCookie = await loginUser(request, ADMIN_USER);

      const adminErrorCases = errorCases.filter(testCase => testCase.requiresAdmin && !testCase.path.includes('/story'));

      for (const testCase of adminErrorCases) {
        // console.log(`Processing test case: ${testCase.method} ${testCase.path}`);

        // Replace path parameters with appropriate IDs (use error case logic for error tests)
        const path = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, true);
        const endpointPath = path;

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, String(value));
          }
        }

        const requestOptions: any = {
          headers: {
            'Cookie': sessionCookie
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = testCase.requestBody;
          requestOptions.headers['Content-Type'] = 'application/json';
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Log the response details
        await logResponse(testCase, response, 'Admin User', 'apitestadmin', url.toString());
      }
    });

    test('should test all admin endpoints for admin user', async ({request}) => {
      // Clear any existing cookies by making a request without cookies
      await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

      const sessionCookie = await loginUser(request, ADMIN_USER);

      // Build adminCases by expanding path parameters and applying exclusions
      const adminCases: TestCase[] = [];
      for (const testCase of testCases) {
        if (!testCase.requiresAdmin) continue;
        // Skip story endpoints for admin users
        if (testCase.path.includes('/story')) continue;
        const expandedPath = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        const fullExpanded = `${baseURL}${expandedPath}`;
        if (isExcludedPath(expandedPath) || isExcludedPath(fullExpanded)) continue;
        adminCases.push(testCase);
      }

      for (const testCase of adminCases) {
        // console.log(`Processing test case: ${testCase.method} ${testCase.path}`);

        // Replace path parameters with appropriate IDs
        const path = await replacePathParameters(testCase.path, testCase.pathParams, request, ADMIN_USER, testCase.method, false);
        const endpointPath = path;

        const url = new URL(`${baseURL}${endpointPath}`);

        // Add query parameters
        if (testCase.queryParams) {
          for (const [key, value] of Object.entries(testCase.queryParams)) {
            url.searchParams.append(key, value.toString());
          }
        }

        const requestOptions: any = {
          headers: {
            'Cookie': sessionCookie
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, ADMIN_USER);

          // Handle question_id in request body for quiz answer endpoints
          if (testCase.path.includes('/quiz/answer') && requestOptions.data && requestOptions.data.question_id === 1) {
            const questionId = await getAvailableQuestionId(request, ADMIN_USER);
            requestOptions.data.question_id = questionId;
            // console.log(`Replaced question_id in request body with ${questionId} for ${testCase.path}`);
          }

          requestOptions.headers['Content-Type'] = 'application/json';
        }

        // Precondition: for role removal, ensure the user actually has the role first
        if (
          testCase.method === 'DELETE' &&
          (testCase.path.includes('/v1/admin/backend/userz/{id}/roles/{roleId}') ||
           (path.includes('/v1/admin/backend/userz/') && path.includes('/roles/')))
        ) {
          const match = path.match(/\/v1\/admin\/backend\/userz\/(\d+)\/roles\/(\d+)/);
          if (match) {
            const userIdForRole = Number(match[1]);
            const roleIdForRemoval = Number(match[2]);
            // Assign role before attempting removal (idempotent if already assigned)
            await request.post(`${baseURL}/v1/admin/backend/userz/${userIdForRole}/roles`, {
              headers: {
                'Cookie': sessionCookie,
                'Content-Type': 'application/json'
              },
              data: {role_id: roleIdForRemoval}
            });
          }
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Remove user ID from available list if this was a DELETE operation on a user endpoint
        if (testCase.method === 'DELETE' && testCase.path.includes('/{id}') &&
          (testCase.path.includes('/userz/') || testCase.path.includes('/admin/backend/userz/'))) {
          // Extract the user ID that was used in the path
          const pathMatch = path.match(/\/(\d+)(?:\/|$)/);
          if (pathMatch) {
            const deletedUserId = parseInt(pathMatch[1]);
            removeUserId(deletedUserId);
            // console.log(`Removed user ID ${deletedUserId} from available list after DELETE`);
          }
        }

        // Mark conversation as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && (endpointPath.includes('/conversations/') || endpointPath.includes('/ai/conversations/'))) {
          // Extract conversation ID from the URL - handle both /conversations/ and /ai/conversations/ patterns
          const conversationMatch = endpointPath.match(/\/conversations\/([^\/]+)/) || endpointPath.match(/\/ai\/conversations\/([^\/]+)/);
          if (conversationMatch) {
            const deletedConversationId = conversationMatch[1];
            markConversationAsDeleted(deletedConversationId);
            // console.log(`Marked conversation as deleted: ${deletedConversationId}`);
          }
        }

        // Mark story as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 200 && (endpointPath.includes('/story/') || endpointPath.includes('/stories/'))) {
          // Extract story ID from the URL - handle both /story/ and /stories/ patterns
          // console.log(`🔍 CHECKING FOR STORY DELETION: method=${testCase.method}, status=${response.status()}, path=${endpointPath}`);
          const storyMatch = endpointPath.match(/\/(?:story|stories)\/(\d+)/);
          if (storyMatch) {
            const deletedStoryId = parseInt(storyMatch[1]);
            // console.log(`📝 EXTRACTED STORY ID: ${deletedStoryId} from path: ${endpointPath}`);
            markStoryAsDeleted(deletedStoryId);
          } else {
            console.log(`❌ Could not extract story ID from path: ${endpointPath}`);
          }
        }

        // Mark snippet as deleted if this was a successful DELETE operation
        if (testCase.method === 'DELETE' && response.status() === 204 && endpointPath.includes('/snippets/')) {
          // Extract snippet ID from the URL
          // console.log(`🔍 CHECKING FOR SNIPPET DELETION: method=${testCase.method}, status=${response.status()}, path=${endpointPath}`);
          const snippetMatch = endpointPath.match(/\/snippets\/(\d+)/);
          if (snippetMatch) {
            const deletedSnippetId = parseInt(snippetMatch[1]);
            // console.log(`📝 EXTRACTED SNIPPET ID: ${deletedSnippetId} from path: ${endpointPath}`);
            markSnippetAsDeleted(deletedSnippetId);
          } else {
            console.log(`❌ Could not extract snippet ID from path: ${endpointPath}`);
          }
        }

        // Log the response details
        await logResponse(testCase, response, 'Admin User', 'apitestadmin', url.toString());
      }
    });
  });

  test.describe('Error Cases', () => {
    test('should test unauthorized access to admin endpoints', async ({request}) => {
      const regularUserSession = await loginUser(request, REGULAR_USER);
      // Skip worker admin endpoints here because the worker service currently requires only authentication, not admin role
      const adminCases = testCases.filter(testCase => testCase.requiresAdmin && !testCase.path.startsWith('/v1/admin/worker'));

      for (const testCase of adminCases) { // Test all admin endpoints
        let endpointPath = testCase.path;
        let userId: number | undefined;

        // Get an available user ID for operations that need an existing user
        // console.log(`Processing test case: ${testCase.method} ${endpointPath}`);
        // console.log(`Processing path: ${endpointPath}, includes /{id}: ${endpointPath.includes('/{id}')}, includes /reset-password: ${endpointPath.includes('/reset-password')}`);

        if (endpointPath.includes('/{id}') && (endpointPath.includes('/reset-password') || endpointPath.includes('/roles') || endpointPath.includes('/clear'))) {
          userId = getAvailableUserId();
          // console.log(`Using available user ID ${userId} for path: ${endpointPath}`);
        } else {
          // console.log(`Not getting available user ID for path: ${endpointPath}`);
        }

        // Add path parameters
        if (testCase.pathParams) {
          for (const [key, value] of Object.entries(testCase.pathParams)) {
            if (key === 'id' && userId) {
              endpointPath = endpointPath.replace(`{${key}}`, userId.toString());
              // console.log(`Replaced {${key}} with ${userId} in path: ${endpointPath}`);
            } else {
              endpointPath = endpointPath.replace(`{${key}}`, String(value));
            }
          }
        }

        const url = new URL(`${baseURL}${endpointPath}`);

        const requestOptions: any = {
          headers: {
            'Cookie': regularUserSession
          }
        };

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, REGULAR_USER);
          requestOptions.headers['Content-Type'] = 'application/json';
        }

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Expect 403 Forbidden. If not, log the full request/response for debugging before failing.
        const status = response.status();
        if (status !== 403) {
          const respHeaders = response.headers();
          const respBody = await response.text();
          console.error(`\n❌ Unauthorized access NOT blocked`);
          console.error(`  Method: ${testCase.method}`);
          console.error(`  URL: ${url.toString()}`);
          console.error(`  Request headers: ${JSON.stringify(sanitizeHeaders(requestOptions.headers || {}), null, 2)}`);
          console.error(`  Request body: ${requestOptions.data ? JSON.stringify(requestOptions.data, null, 2) : 'none'}`);
          console.error(`  Response status: ${status}`);
          console.error(`  Response headers: ${JSON.stringify(respHeaders, null, 2)}`);
          console.error(`  Response body: ${respBody}`);
        }

        await assertStatus(response, 403, {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });

        console.log(`✅ Unauthorized access blocked: ${testCase.method} ${testCase.path}`);
      }
    });

    test('should test unauthenticated access to protected endpoints', async ({request}) => {
      const protectedCases = testCases.filter(testCase => testCase.requiresAuth);

      for (const testCase of protectedCases) {

        // Replace path parameters without triggering any authenticated calls
        const path = replacePathParametersUnauthenticated(testCase.path, testCase.pathParams);
        const endpointPath = path;

        const url = new URL(`${baseURL}${endpointPath}`);

        const requestOptions: any = {};

        if (testCase.requestBody) {
          requestOptions.data = generateRequestBodyForTest(testCase, REGULAR_USER);
          requestOptions.headers = {'Content-Type': 'application/json'};
        }

        // Ensure no lingering session from earlier helper calls
        await request.get(`${baseURL}/v1/auth/logout`, {failOnStatusCode: false});

        const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Should get 401 Unauthorized; if not, dump diagnostics
        if (response.status() !== 401) {
          const respHeaders = response.headers();
          const respBody = await response.text();
          console.error(`\n❌ Unauthenticated access NOT blocked`);
          console.error(`  Method: ${testCase.method}`);
          console.error(`  URL: ${url.toString()}`);
          console.error(`  Request headers: ${JSON.stringify(sanitizeHeaders(requestOptions.headers || {}), null, 2)}`);
          console.error(`  Request body: ${requestOptions.data ? JSON.stringify(requestOptions.data, null, 2) : 'none'}`);
          console.error(`  Response status: ${response.status()}`);
          console.error(`  Response headers: ${JSON.stringify(respHeaders, null, 2)}`);
          console.error(`  Response body: ${respBody}`);
        }
        await assertStatus(response, 401, {
          method: testCase.method,
          url: url.toString(),
          requestHeaders: requestOptions.headers,
          requestBody: requestOptions.data
        });

        console.log(`✅ Unauthenticated access blocked: ${testCase.method} ${testCase.path}`);
      }
    });
  });

  test.describe('Test Summary', () => {
    test('should provide test coverage summary', async () => {
      const totalEndpoints = testCases.length;
      const publicEndpoints = testCases.filter(tc => !tc.requiresAuth).length;
      const protectedEndpoints = testCases.filter(tc => tc.requiresAuth && !tc.requiresAdmin).length;
      const adminEndpoints = testCases.filter(tc => tc.requiresAdmin).length;

      console.log('\n📊 API Test Coverage Summary:');
      console.log(`Total endpoints tested: ${totalEndpoints}`);
      console.log(`Public endpoints: ${publicEndpoints}`);
      console.log(`Protected endpoints: ${protectedEndpoints}`);
      console.log(`Admin endpoints: ${adminEndpoints}`);

      expect(totalEndpoints).toBeGreaterThan(0);
      expect(publicEndpoints).toBeGreaterThan(0);
      expect(protectedEndpoints).toBeGreaterThan(0);
      expect(adminEndpoints).toBeGreaterThan(0);
    });
  });

  test.describe('Destructive Operations (Run Last)', () => {
    test('should test database clear operations', async ({request}) => {
      const adminSession = await loginUser(request, ADMIN_USER);

      // Test clear user data
      const clearUserDataResponse = await request.post(`${baseURL}/v1/admin/backend/clear-user-data`, {
        headers: {
          'Cookie': adminSession
        }
      });
      expect(clearUserDataResponse.status()).toBe(200);
      console.log('✅ Clear user data endpoint tested');

      // Test clear database
      const clearDatabaseResponse = await request.post(`${baseURL}/v1/admin/backend/clear-database`, {
        headers: {
          'Cookie': adminSession
        }
      });
      expect(clearDatabaseResponse.status()).toBe(200);
      console.log('✅ Clear database endpoint tested');
    });
  });
});
