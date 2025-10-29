-- Drop auth_api_keys table
DROP INDEX IF EXISTS idx_auth_api_keys_key_prefix;
DROP INDEX IF EXISTS idx_auth_api_keys_key_hash;
DROP INDEX IF EXISTS idx_auth_api_keys_user_id;
DROP TABLE IF EXISTS auth_api_keys;
