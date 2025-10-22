-- Add section_id and story_id fields to snippets table
-- This allows snippets to be associated with story sections or entire stories

-- Add section_id column (references story_sections.id)
ALTER TABLE snippets ADD COLUMN section_id INTEGER REFERENCES story_sections(id) ON DELETE SET NULL;

-- Add story_id column (references stories.id) 
ALTER TABLE snippets ADD COLUMN story_id INTEGER REFERENCES stories(id) ON DELETE SET NULL;

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_snippets_section_id ON snippets(section_id);
CREATE INDEX IF NOT EXISTS idx_snippets_story_id ON snippets(story_id);

-- Update the unique constraint to include the new fields
-- Drop the existing unique constraint
ALTER TABLE snippets DROP CONSTRAINT IF EXISTS snippets_user_id_original_text_source_language_target_language_key;

-- Add new unique constraint that includes section_id and story_id
-- This allows the same snippet to exist for different contexts (question, section, story)
CREATE UNIQUE INDEX IF NOT EXISTS unique_snippet_per_user_text_language_context
ON snippets (user_id, original_text, source_language, target_language, question_id, section_id, story_id)
WHERE original_text IS NOT NULL;
