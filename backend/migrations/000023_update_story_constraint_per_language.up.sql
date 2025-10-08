-- Update story constraint to be per user per language instead of just per user
-- First drop the old constraint
DROP INDEX IF EXISTS unique_current_story_per_user;

-- Create new constraint for per user per language based on active status
-- (This replaces the old is_current logic)
CREATE UNIQUE INDEX IF NOT EXISTS unique_active_story_per_user_language
ON stories (user_id, language)
WHERE status = 'active';
