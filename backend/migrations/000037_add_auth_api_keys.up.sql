-- Add auth_api_keys table for API key authentication
-- This is separate from user_api_keys which stores AI provider API keys
CREATE TABLE IF NOT EXISTS auth_api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(255) NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    permission_level VARCHAR(20) NOT NULL CHECK (permission_level IN ('readonly', 'full')),
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(key_hash)
);

-- Create indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_user_id ON auth_api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_hash ON auth_api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_prefix ON auth_api_keys(key_prefix);

-- Add comment for documentation
COMMENT ON TABLE auth_api_keys IS 'Stores API keys for programmatic authentication (separate from AI provider keys)';
COMMENT ON COLUMN auth_api_keys.key_hash IS 'Bcrypt hash of the API key';
COMMENT ON COLUMN auth_api_keys.key_prefix IS 'First 8 characters of the key for UI display';
COMMENT ON COLUMN auth_api_keys.permission_level IS 'readonly: GET only, full: all methods';
