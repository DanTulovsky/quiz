-- Drop generation_hints table and indexes
DROP INDEX IF EXISTS idx_generation_hints_user_expires;
DROP INDEX IF EXISTS idx_generation_hints_expires;
DROP TABLE IF EXISTS generation_hints;

