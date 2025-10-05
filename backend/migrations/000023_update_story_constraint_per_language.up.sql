-- Update story constraint to be per user per language instead of just per user
-- First drop the old constraint
DROP INDEX IF EXISTS unique_current_story_per_user;

-- Create new constraint for per user per language
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user_per_language
ON stories (user_id, language)
WHERE is_current = true;
