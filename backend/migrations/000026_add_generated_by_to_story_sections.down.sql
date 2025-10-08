-- Remove generated_by column from story_sections table (if it exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'story_sections' AND column_name = 'generated_by') THEN
        ALTER TABLE story_sections DROP COLUMN generated_by;
    END IF;
END $$;

-- Remove the index that was created for the generated_by column (if it exists)
DROP INDEX IF EXISTS idx_story_sections_generated_by;
