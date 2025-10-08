-- Revert the story constraint fix
-- Drop the correct partial unique index and recreate the incorrect constraint

-- Drop the correct partial unique index
DROP INDEX IF EXISTS unique_current_story_per_user;

-- Recreate the incorrect constraint (for rollback purposes, only if is_current column exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'stories' AND column_name = 'is_current') THEN
        ALTER TABLE stories ADD CONSTRAINT unique_current_story_per_user
        UNIQUE (user_id, is_current) DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;
