-- Revert the unique constraint fix

-- Drop the new constraint
DROP INDEX IF EXISTS unique_snippet_per_user_text_language_context;

-- Recreate the old constraint (without context fields)
CREATE UNIQUE INDEX IF NOT EXISTS snippets_user_id_original_text_source_language_target_language_key
ON snippets (user_id, original_text, source_language, target_language)
WHERE original_text IS NOT NULL;







