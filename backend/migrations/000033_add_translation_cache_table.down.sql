-- Drop translation_cache table and indexes
DROP INDEX IF EXISTS idx_translation_cache_expires_at;
DROP INDEX IF EXISTS idx_translation_cache_lookup;
DROP TABLE IF EXISTS translation_cache;

