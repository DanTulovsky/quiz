# Error Handling Standards

This document outlines the standardized error handling approach for the quiz application backend.

## Principles

1. **Always use error wrapping with `%w`**: This preserves the error chain for debugging
2. **Use consistent error types**: Define error variables for common error scenarios
3. **Log errors appropriately**: Use structured logging with context
4. **Handle errors at the right level**: Don't ignore errors, handle them appropriately

## Error Types

### Database Errors
- `ErrDatabaseConnection`: Database connection failures
- `ErrDatabaseQuery`: Query execution failures
- `ErrDatabaseTransaction`: Transaction failures
- `ErrRecordNotFound`: Record not found in database
- `ErrRecordExists`: Record already exists
- `ErrForeignKeyViolation`: Foreign key constraint violations

### Validation Errors
- `ErrInvalidInput`: Invalid input data
- `ErrMissingRequired`: Missing required fields
- `ErrInvalidFormat`: Invalid data format
- `ErrValidationFailed`: General validation failures

### Authentication Errors
- `ErrUnauthorized`: User not authenticated
- `ErrForbidden`: User not authorized for action
- `ErrInvalidCredentials`: Invalid login credentials
- `ErrSessionExpired`: User session has expired

### Service Errors
- `ErrServiceUnavailable`: Service temporarily unavailable
- `ErrTimeout`: Request timeout
- `ErrRateLimit`: Rate limit exceeded
- `ErrInternalError`: Internal server error

### AI Service Errors
- `ErrAIProviderUnavailable`: AI provider is down
- `ErrAIRequestFailed`: AI request failed
- `ErrAIResponseInvalid`: Invalid response from AI
- `ErrAIConfigInvalid`: Invalid AI configuration

### OAuth Errors
- `ErrOAuthCodeExpired`: OAuth code has expired
- `ErrOAuthStateMismatch`: OAuth state mismatch
- `ErrOAuthProviderError`: OAuth provider error

## Error Handling Functions

### WrapError
Wraps an error with additional context using `%w` verb:
```go
err := WrapError(dbErr, "failed to create user")
```

### WrapErrorf
Wraps an error with formatted context:
```go
err := WrapErrorf(dbErr, "failed to create user %s", username)
```

### LogAndWrapError
Logs an error and wraps it with context:
```go
err := LogAndWrapError(dbErr, "database operation failed")
```

### LogAndWrapErrorf
Logs an error and wraps it with formatted context:
```go
err := LogAndWrapErrorf(dbErr, "failed to create user %s", username)
```

### IsError
Checks if an error is of a specific type:
```go
if IsError(err, utils.ErrRecordNotFound) {
    // Handle not found case
}
```

### AsError
Checks if an error can be converted to a specific type:
```go
var pqErr *pq.Error
if AsError(err, &pqErr) {
    // Handle PostgreSQL specific error
}
```

## Best Practices

### 1. Always Wrap Errors
```go
// ❌ Bad - loses error context
return err

// ✅ Good - preserves error chain
return WrapError(err, "failed to create user")
```

### 2. Use Consistent Error Types
```go
// ❌ Bad - inline error creation
return fmt.Errorf("user not found")

// ✅ Good - use predefined error types
return utils.ErrRecordNotFound
```

### 3. Log Errors Appropriately
```go
// ❌ Bad - no logging
if err != nil {
    return err
}

// ✅ Good - structured logging
if err != nil {
    return LogAndWrapError(err, "database operation failed")
}
```

### 4. Handle Errors at the Right Level
```go
// In service layer
func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
    if err := s.db.Create(user); err != nil {
        return WrapError(err, "failed to create user in database")
    }
    return nil
}

// In handler layer
func (h *UserHandler) CreateUser(c *gin.Context) {
    user := &models.User{...}
    if err := h.userService.CreateUser(c.Request.Context(), user); err != nil {
        if IsError(err, utils.ErrRecordExists) {
            c.JSON(http.StatusConflict, api.ErrorResponse{
                Error: stringPtr("User already exists"),
            })
            return
        }
        c.JSON(http.StatusInternalServerError, api.ErrorResponse{
            Error: stringPtr("Failed to create user"),
        })
        return
    }
    c.JSON(http.StatusCreated, user)
}
```

### 5. Use Error Checking in Handlers
```go
// Check for specific error types
if IsError(err, utils.ErrUnauthorized) {
    c.JSON(http.StatusUnauthorized, api.ErrorResponse{
        Error: stringPtr("Authentication required"),
    })
    return
}

if IsError(err, utils.ErrForbidden) {
    c.JSON(http.StatusForbidden, api.ErrorResponse{
        Error: stringPtr("Access denied"),
    })
    return
}
```

## Migration Guide

When updating existing code:

1. Replace `fmt.Errorf("%v", err)` with `WrapError(err, "context")`
2. Replace `fmt.Errorf("message: %v", err)` with `WrapErrorf(err, "message")`
3. Replace inline error creation with predefined error types
4. Add appropriate logging using `LogAndWrapError` or `LogAndWrapErrorf`
5. Use `IsError` and `AsError` for error type checking

## Examples

### Database Operations
```go
func (s *UserService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
    var user models.User
    err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, utils.ErrRecordNotFound
        }
        return nil, WrapError(err, "failed to get user by ID")
    }
    return &user, nil
}
```

### Validation
```go
func validateUserInput(user *models.User) error {
    if user.Username == "" {
        return utils.ErrMissingRequired
    }
    if len(user.Username) < 3 {
        return utils.ErrInvalidInput
    }
    return nil
}
```

### Service Layer
```go
func (s *AIService) GenerateQuestion(ctx context.Context, req *models.AIQuestionGenRequest) (*models.Question, error) {
    if err := validateRequest(req); err != nil {
        return nil, WrapError(err, "invalid AI request")
    }

    response, err := s.callAIProvider(ctx, req)
    if err != nil {
        return nil, LogAndWrapError(err, "AI provider request failed")
    }

    question, err := s.parseResponse(response)
    if err != nil {
        return nil, WrapError(err, "failed to parse AI response")
    }

    return question, nil
}
```
