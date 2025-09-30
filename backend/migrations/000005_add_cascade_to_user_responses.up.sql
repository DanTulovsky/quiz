-- Add ON DELETE CASCADE to user_responses.user_id foreign key
-- This allows automatic deletion of user responses when a user is deleted

-- First, drop the existing foreign key constraint
ALTER TABLE user_responses DROP CONSTRAINT IF EXISTS user_responses_user_id_fkey;

-- Add the foreign key constraint with CASCADE
ALTER TABLE user_responses ADD CONSTRAINT user_responses_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
