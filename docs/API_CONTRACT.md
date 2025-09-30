# API Contract Documentation

## Overview

This document describes the OpenAPI specification (`swagger.yaml`) that serves as the single source of truth for the API contract between the backend and frontend of the Quiz Application.

## Purpose

The OpenAPI specification ensures:

1. **Type Safety**: Both backend and frontend agree on data structures
2. **API Documentation**: Self-documenting API with examples
3. **Prevention of Mismatches**: Catches issues like the "Unsupported question type" error
4. **Code Generation**: Can generate TypeScript types and client SDKs
5. **Testing**: Can validate API responses against the schema

## Key Features

### Question Types

The specification defines the exact question types that both backend and frontend must support:

- `vocabulary`
- `fill_blank`
- `qa`
- `reading_comprehension`

### Languages

Supported languages:

- `italian`
- `spanish`
- `french`
- `german`
- `english`

### Levels

CEFR levels:

- `A1`, `A2`, `B1`, `B1+`, `B1++`, `B2`, `C1`, `C2`

## Implementation Guidelines

### Backend (Go)

The backend should implement handlers that strictly adhere to the OpenAPI spec:

1. **Response Format**: All responses must match the defined schemas
2. **Question Types**: Must use the exact enum values defined in `QuestionType`
3. **Error Handling**: Should return `ErrorResponse` schema for errors

### Frontend (TypeScript)

The frontend should use interfaces that match the OpenAPI schemas:

1. **Type Definitions**: Use the exact types from the spec
2. **API Calls**: Structure requests according to the defined schemas
3. **Response Handling**: Handle all response types documented in the spec

## Usage

### Viewing the API Documentation

You can view the API documentation by:

1. **Swagger UI**: Load `swagger.yaml` in [Swagger Editor](https://editor.swagger.io/)
2. **Local Tools**: Use tools like `swagger-ui-serve`

### Generating TypeScript Types

You can generate TypeScript interfaces from the spec:

```bash
# Install openapi-typescript
npm install -g openapi-typescript

# Generate types
openapi-typescript swagger.yaml --output frontend/src/types/api.ts
```

### Validating API Responses

Use tools like `swagger-jsdoc` and `swagger-ui-express` to validate responses.

## Benefits Demonstrated

### Before (Type Mismatch Issue)

- Backend sent: `"fill_blank"`, `"qa"`
- Frontend expected: `"fill_in_blank"`, `"q_and_a"`
- Result: "Unsupported question type" error

### After (OpenAPI Contract)

- Single source of truth in `swagger.yaml`
- Both backend and frontend reference the same enum values
- Automatic validation and type checking prevents mismatches

## Maintenance

1. **Updates**: All API changes must be reflected in `swagger.yaml` first
2. **Versioning**: Use semantic versioning for the API
3. **Validation**: Validate both backend responses and frontend requests against the schema
4. **Documentation**: Keep the spec up-to-date with implementation changes

## Next Steps

1. Set up automated type generation from the OpenAPI spec
2. Add API response validation in tests
3. Integrate Swagger UI for interactive documentation
4. Add OpenAPI validation middleware to the backend
