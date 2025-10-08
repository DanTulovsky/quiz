-- Revert the removal of is_current column
-- Restore the is_current field and related constraints

-- Add back the is_current column
ALTER TABLE stories ADD COLUMN IF NOT EXISTS is_current BOOLEAN NOT NULL DEFAULT FALSE;

-- Drop the new constraint based on active status
DROP INDEX IF EXISTS unique_active_story_per_user_language;

-- Recreate the old constraint based on is_current
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user
ON stories (user_id)
WHERE is_current = true;

-- Recreate the index that was based on is_current
DROP INDEX IF EXISTS idx_stories_user_status;
CREATE INDEX IF NOT EXISTS idx_stories_user_current ON stories(user_id, is_current);
