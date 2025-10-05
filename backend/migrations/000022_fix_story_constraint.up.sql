-- Fix the incorrect unique constraint on stories table
-- The original constraint was UNIQUE (user_id, is_current) which prevented multiple stories with is_current = false
-- We need to replace it with a partial unique index that only applies when is_current = true

-- Drop the incorrect constraint if it exists
ALTER TABLE stories DROP CONSTRAINT IF EXISTS unique_current_story_per_user;

-- Create the correct partial unique index
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user 
ON stories (user_id) 
WHERE is_current = true;
