-- Fix snippets unique constraint to include question_id, section_id, and story_id
-- This allows the same snippet text to exist for different contexts

-- Drop the old unique constraint if it exists (the one without context fields)
ALTER TABLE snippets DROP CONSTRAINT IF EXISTS snippets_user_id_original_text_source_language_target_language_key;

-- Drop the new constraint if it exists (in case we're re-running this migration)
DROP INDEX IF EXISTS unique_snippet_per_user_text_language_context;

-- Create the new unique constraint that includes question_id, section_id, and story_id
-- This allows the same snippet to exist for different contexts (question, section, story)
CREATE UNIQUE INDEX IF NOT EXISTS unique_snippet_per_user_text_language_context
ON snippets (user_id, original_text, source_language, target_language, question_id, section_id, story_id)
WHERE original_text IS NOT NULL;


