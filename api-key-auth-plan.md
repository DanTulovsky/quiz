# API Key Authentication Implementation Plan

## Overview
Implement API key authentication to allow programmatic access to the Quiz App API. Users will be able to generate, manage, and delete multiple API keys with different permission levels (readonly vs full access).

## Architecture

### Database Schema
Create a new table `auth_api_keys` (separate from existing `user_api_keys` which stores AI provider keys):

```sql
CREATE TABLE auth_api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(255) NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,  -- First 8 chars for identification
    permission_level VARCHAR(20) NOT NULL CHECK (permission_level IN ('readonly', 'full')),
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(key_hash)
);

CREATE INDEX idx_auth_api_keys_user_id ON auth_api_keys(user_id);
CREATE INDEX idx_auth_api_keys_key_hash ON auth_api_keys(key_hash);
CREATE INDEX idx_auth_api_keys_key_prefix ON auth_api_keys(key_prefix);
```

**Key Format:** `qapp_` + 32 random characters (e.g., `qapp_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p`)
- Keys are hashed before storage (bcrypt)
- Store key_prefix for user identification in UI
- Never return full key after creation

### Authentication Flow

1. **Request Authentication Order:**
   - First check for `Authorization: Bearer <api_key>` header
   - If no API key, fall back to session-based auth
   - This allows both methods to coexist

2. **API Key Validation:**
   - Extract key from Authorization header
   - Hash and lookup in database
   - Check permission level against request method
   - Set user context (user_id, username) in Gin context
   - Update last_used_at timestamp (async)

3. **Permission Levels:**
   - **readonly**: Only GET requests allowed
   - **full**: All HTTP methods allowed (GET, POST, PUT, DELETE)

### Backend Implementation

#### 1. Database Migration (`000037_add_auth_api_keys.up.sql`)
- Create `auth_api_keys` table
- Add indexes

#### 2. Models (`internal/models/auth_api_key.go`)
```go
type AuthAPIKey struct {
    ID              int
    UserID          int
    KeyName         string
    KeyHash         string
    KeyPrefix       string
    PermissionLevel string  // "readonly" or "full"
    LastUsedAt      sql.NullTime
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

#### 3. Service Layer (`internal/services/auth_api_key_service.go`)
```go
type AuthAPIKeyServiceInterface interface {
    CreateAPIKey(ctx context.Context, userID int, keyName string, permissionLevel string) (*AuthAPIKey, string, error)
    ListAPIKeys(ctx context.Context, userID int) ([]AuthAPIKey, error)
    DeleteAPIKey(ctx context.Context, userID int, keyID int) error
    ValidateAPIKey(ctx context.Context, rawKey string) (*AuthAPIKey, error)
    UpdateLastUsed(ctx context.Context, keyID int) error
}
```

**Key Methods:**
- `CreateAPIKey`: Generates random key, hashes it, stores in DB, returns key once
- `ValidateAPIKey`: Hashes provided key, looks up in DB, returns key info
- `UpdateLastUsed`: Updates last_used_at (call asynchronously to avoid blocking requests)

#### 4. Middleware (`internal/middleware/auth.go`)
Update `RequireAuth` middleware to:
1. Check for `Authorization: Bearer <token>` header first
2. If found, validate API key and set user context
3. If not found or invalid, fall back to session auth
4. For readonly keys, reject non-GET requests

Add new middleware:
```go
func RequireAuthWithAPIKey() gin.HandlerFunc
func checkAPIKeyPermission(permissionLevel string, method string) bool
```

#### 5. Handlers (`internal/handlers/auth_api_key_handler.go`)
```go
type AuthAPIKeyHandler struct {
    apiKeyService services.AuthAPIKeyServiceInterface
    logger        *observability.Logger
}

// POST /v1/api-keys
func (h *AuthAPIKeyHandler) CreateAPIKey(c *gin.Context)

// GET /v1/api-keys
func (h *AuthAPIKeyHandler) ListAPIKeys(c *gin.Context)

// DELETE /v1/api-keys/:id
func (h *AuthAPIKeyHandler) DeleteAPIKey(c *gin.Context)
```

#### 6. API Specification (`swagger.yaml`)
Add endpoints:
- `POST /v1/api-keys`: Create new API key
- `GET /v1/api-keys`: List user's API keys
- `DELETE /v1/api-keys/{id}`: Delete API key

Add security scheme:
```yaml
securitySchemes:
  ApiKeyAuth:
    type: apiKey
    in: header
    name: Authorization
```

### Frontend Implementation

#### 1. API Client Updates
- Generate TypeScript types from updated swagger.yaml
- Create API hooks for CRUD operations

#### 2. Settings Page UI (`frontend/src/pages/SettingsPage.tsx`)
Add new section "API Keys" in the Account section:

**Components:**
- List of existing API keys with:
  - Key name
  - Permission level badge
  - Key prefix (e.g., `qapp_1a2b3c4d...`)
  - Last used date
  - Delete button
- "Create New API Key" button
- Modal for creating new key:
  - Name input
  - Permission level selector (readonly/full)
  - Display full key ONCE with copy button
  - Warning: "Save this key now, you won't see it again"

#### 3. API Key Display Modal
- Show full key in monospace font
- Copy to clipboard button
- Security warning
- Confirmation checkbox before closing

### Security Considerations

1. **Key Generation:**
   - Use `crypto/rand` for cryptographically secure random keys
   - 32+ character length
   - Prefix for easy identification

2. **Key Storage:**
   - Hash keys using bcrypt before storage
   - Never log full keys
   - Only show full key once during creation

3. **Key Usage:**
   - Update last_used_at for monitoring
   - Rate limiting applies to API keys same as session auth
   - Include API key ID in traces for debugging

4. **Permissions:**
   - Readonly keys strictly limited to GET requests
   - Full keys have same permissions as session auth
   - Both inherit user's role-based permissions

### Testing Strategy

#### Unit Tests
- Key generation and hashing
- Permission level validation
- API key service methods

#### Integration Tests
- Full authentication flow with API keys
- Permission enforcement (readonly vs full)
- Fallback to session auth
- CRUD operations for API keys
- Multiple keys per user
- Key deletion

#### Manual Testing
- Create API key in UI
- Use API key with curl/Postman
- Test readonly restrictions
- Test full access
- Delete key and verify revocation

### Migration Path

1. Deploy database migration
2. Deploy backend with new endpoints and middleware
3. Deploy frontend with UI changes
4. Document API key usage in README/docs

### API Usage Examples

```bash
# Using API key for readonly access
curl -H "Authorization: Bearer qapp_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p" \
  https://api.example.com/v1/quiz/question

# Using API key for full access
curl -X POST \
  -H "Authorization: Bearer qapp_abcdef123456..." \
  -H "Content-Type: application/json" \
  -d '{"language":"italian","level":"A1"}' \
  https://api.example.com/v1/quiz/generate
```

### Rollout Plan

1. ✅ Create implementation plan
2. ⏳ Create database migration
3. ⏳ Implement backend services and models
4. ⏳ Update authentication middleware
5. ⏳ Implement API handlers
6. ⏳ Update OpenAPI specification
7. ⏳ Implement frontend UI
8. ⏳ Write comprehensive tests
9. ⏳ Run full test suite
10. ⏳ Manual testing and validation
11. ⏳ Documentation update
12. ⏳ Deploy to production

### Future Enhancements (Out of Scope)

- API key expiration dates
- API key usage analytics dashboard
- Rate limiting per API key
- Scope-based permissions (e.g., only quiz access)
- Webhook support with API keys
- API key rotation functionality

## Questions/Decisions

1. ✅ Key prefix: `qapp_` chosen for "Quiz App"
2. ✅ Permission levels: readonly and full (simple model)
3. ✅ Separate table from AI API keys to avoid confusion
4. ✅ Hash keys with bcrypt (same as passwords)
5. ✅ No expiration dates in v1 (can add later)

## Estimated Effort

- Backend: 4-6 hours
- Frontend: 2-3 hours
- Testing: 2-3 hours
- Total: 8-12 hours

---

**Status:** Ready for implementation
**Last Updated:** 2025-10-29
