-- Revert user_answer_index column back to user_answer TEXT
-- First, add the old column back
ALTER TABLE user_responses ADD COLUMN user_answer TEXT;

-- Update existing data to set a default text value for all existing records
-- This is a safe default since we can't determine the original text from index
UPDATE user_responses SET user_answer = 'default_answer' WHERE user_answer IS NULL;

-- Make the old column NOT NULL
ALTER TABLE user_responses ALTER COLUMN user_answer SET NOT NULL;

-- Drop the new column
ALTER TABLE user_responses DROP COLUMN user_answer_index;
