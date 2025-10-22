-- Remove section_id and story_id fields from snippets table

-- Drop indexes
DROP INDEX IF EXISTS idx_snippets_section_id;
DROP INDEX IF EXISTS idx_snippets_story_id;
DROP INDEX IF EXISTS unique_snippet_per_user_text_language_context;

-- Drop columns
ALTER TABLE snippets DROP COLUMN IF EXISTS section_id;
ALTER TABLE snippets DROP COLUMN IF EXISTS story_id;

-- Restore original unique constraint
CREATE UNIQUE INDEX IF NOT EXISTS snippets_user_id_original_text_source_language_target_language_key
ON snippets (user_id, original_text, source_language, target_language);
