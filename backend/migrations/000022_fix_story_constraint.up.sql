-- Fix the incorrect unique constraint on stories table
-- The original constraint was UNIQUE (user_id, is_current) which prevented multiple stories with is_current = false
-- We need to replace it with a partial unique index that only applies when is_current = true

-- Drop the incorrect constraint if it exists
ALTER TABLE stories DROP CONSTRAINT IF EXISTS unique_current_story_per_user;

-- Create the correct partial unique index (only if is_current column exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'stories' AND column_name = 'is_current') THEN
        CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user
        ON stories (user_id)
        WHERE is_current = true;
    END IF;
END $$;
