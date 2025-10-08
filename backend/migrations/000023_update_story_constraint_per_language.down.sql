-- Revert story constraint back to per user only
-- First drop the new constraint
DROP INDEX IF EXISTS unique_active_story_per_user_language;

-- Create old constraint for per user only (only if is_current column exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'stories' AND column_name = 'is_current') THEN
        CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user
        ON stories (user_id)
        WHERE is_current = true;
    END IF;
END $$;
