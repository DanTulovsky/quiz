-- Add generated_by column to story_sections table to track who generated each section
-- Use DO block to safely add column only if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'story_sections' AND column_name = 'generated_by') THEN
        ALTER TABLE story_sections ADD COLUMN generated_by VARCHAR(10) NOT NULL DEFAULT 'user' CHECK (generated_by IN ('worker', 'user'));
    END IF;
END $$;

-- Add index for efficient querying by story, generator type, and date
-- Use CREATE INDEX IF NOT EXISTS to safely create index only if it doesn't exist
CREATE INDEX IF NOT EXISTS idx_story_sections_generated_by ON story_sections(story_id, generated_by, generation_date);
