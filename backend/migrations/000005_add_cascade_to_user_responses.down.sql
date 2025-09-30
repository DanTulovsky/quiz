-- Revert ON DELETE CASCADE from user_responses.user_id foreign key
-- This restores the original behavior where user responses are not automatically deleted

-- First, drop the existing foreign key constraint
ALTER TABLE user_responses DROP CONSTRAINT IF EXISTS user_responses_user_id_fkey;

-- Add the foreign key constraint without CASCADE (original behavior)
ALTER TABLE user_responses ADD CONSTRAINT user_responses_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users (id);
