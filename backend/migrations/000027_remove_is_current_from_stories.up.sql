-- Remove is_current column and simplify logic to use status='active' instead
-- Since an active story is by definition the current story for that user

-- Drop the old constraint that was based on is_current
DROP INDEX IF EXISTS unique_current_story_per_user;

-- Drop the is_current column
ALTER TABLE stories DROP COLUMN IF EXISTS is_current;

-- The unique_active_story_per_user_language index should already exist from migration 000023
-- But we'll ensure it exists in case migration 000023 didn't run
CREATE UNIQUE INDEX IF NOT EXISTS unique_active_story_per_user_language
ON stories (user_id, language)
WHERE status = 'active';

-- Update the index that was previously based on is_current
DROP INDEX IF EXISTS idx_stories_user_current;
CREATE INDEX IF NOT EXISTS idx_stories_user_status ON stories(user_id, status);
