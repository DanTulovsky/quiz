-- Revert the story constraint fix
-- Drop the correct partial unique index and recreate the incorrect constraint

-- Drop the correct partial unique index
DROP INDEX IF EXISTS unique_current_story_per_user;

-- Recreate the incorrect constraint (for rollback purposes)
ALTER TABLE stories ADD CONSTRAINT unique_current_story_per_user 
UNIQUE (user_id, is_current) DEFERRABLE INITIALLY DEFERRED;
