-- Revert story constraint back to per user only
-- First drop the new constraint
DROP INDEX IF EXISTS unique_current_story_per_user_per_language;

-- Create old constraint for per user only
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user
ON stories (user_id)
WHERE is_current = true;
