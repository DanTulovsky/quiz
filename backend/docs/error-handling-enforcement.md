# Error Handling Enforcement

This document describes the enforcement mechanisms for maintaining consistent error handling across the backend codebase.

## Overview

We have standardized error handling to use the `contextutils` package instead of `fmt.Errorf`. This ensures:
- Consistent error types and messages
- Better error context and traceability
- Easier error handling and debugging
- Centralized error management

## Enforcement Mechanisms

### 1. Automated Script Check

The `scripts/check-fmt-errorf.sh` script automatically checks for `fmt.Errorf` usage in production code:

```bash
# Run the check
./scripts/check-fmt-errorf.sh

# Expected output if compliant:
# ✅ No fmt.Errorf usage found in production code
# ✅ All error handling follows the standardized pattern
```

### 2. CI/CD Integration

The error handling check is integrated into the CI/CD pipeline to prevent merging code with inconsistent error handling.

### 3. Pre-commit Hooks

Developers can set up pre-commit hooks to automatically check for `fmt.Errorf` usage before committing code.

## Standardized Error Handling

### Available Functions

1. **WrapError(err, context)** - Wrap an error with simple context
   ```go
   return contextutils.WrapError(err, "failed to connect to database")
   ```

2. **WrapErrorf(err, format, args...)** - Wrap an error with formatted context
   ```go
   return contextutils.WrapErrorf(err, "failed to create user %s", username)
   ```

3. **ErrorWithContextf(format, args...)** - Create a new error with formatted message
   ```go
   return contextutils.ErrorWithContextf("user not found: %s", username)
   ```

### Predefined Error Types

Use predefined error types for common scenarios:

```go
// Database errors
return contextutils.ErrDatabaseConnection
return contextutils.ErrRecordNotFound

// Validation errors
return contextutils.ErrInvalidInput
return contextutils.ErrMissingRequired

// Authentication errors
return contextutils.ErrUnauthorized
return contextutils.ErrInvalidCredentials

// Service errors
return contextutils.ErrServiceUnavailable
return contextutils.ErrTimeout
```

## Migration Guide

### Before (Inconsistent)
```go
if err != nil {
    return fmt.Errorf("failed to save user: %w", err)
}

if user == nil {
    return fmt.Errorf("user not found: %s", username)
}
```

### After (Standardized)
```go
if err != nil {
    return contextutils.WrapError(err, "failed to save user")
}

if user == nil {
    return contextutils.ErrorWithContextf("user not found: %s", username)
}
```

## Exceptions

The following files are allowed to use `fmt.Errorf`:
- `internal/utils/errors.go` - Implementation of error utilities
- `*_test.go` files - Test files for testing error scenarios

## Best Practices

1. **Always wrap errors with context** - Don't return raw errors
2. **Use descriptive error messages** - Make errors actionable
3. **Use predefined error types** - For common error scenarios
4. **Include relevant data** - Username, ID, etc. in error messages
5. **Log errors appropriately** - Use `LogAndWrapError` for logging

## Troubleshooting

### Common Issues

1. **Import missing** - Add `contextutils "quizapp/internal/utils"` to imports
2. **Wrong function** - Use `WrapError` for wrapping, `ErrorWithContextf` for new errors
3. **Format string issues** - Ensure proper format string usage

### Debugging

To debug error handling issues:

```bash
# Check for fmt.Errorf usage
./scripts/check-fmt-errorf.sh

# Run tests to ensure error handling works
go test ./...

# Check for linting issues
golangci-lint run
```

## Maintenance

- Run the check script regularly during development
- Update error types in `internal/utils/errors.go` as needed
- Review error messages for clarity and consistency
- Ensure all new code follows the standardized pattern

## Benefits

1. **Consistency** - All errors follow the same pattern
2. **Traceability** - Better error context for debugging
3. **Maintainability** - Centralized error management
4. **Type Safety** - Predefined error types prevent typos
5. **Documentation** - Self-documenting error handling
