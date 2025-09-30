-- Change user_answer column from TEXT to INTEGER and rename to user_answer_index
-- First, check if the new column already exists (for test databases that were created with new schema)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_responses' AND column_name = 'user_answer_index'
    ) THEN
        -- Add the new column only if it doesn't exist
        ALTER TABLE user_responses ADD COLUMN user_answer_index INTEGER;

        -- Update existing data to set a default index (0) for all existing records
        -- This is a safe default since we can't determine the original index from text
        UPDATE user_responses SET user_answer_index = 0 WHERE user_answer_index IS NULL;

        -- Make the new column NOT NULL
        ALTER TABLE user_responses ALTER COLUMN user_answer_index SET NOT NULL;
    END IF;
END $$;

-- Drop the old column if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_responses' AND column_name = 'user_answer'
    ) THEN
        ALTER TABLE user_responses DROP COLUMN user_answer;
    END IF;
END $$;
