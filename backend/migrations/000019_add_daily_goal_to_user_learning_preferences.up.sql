-- Add daily_goal column to user_learning_preferences
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'user_learning_preferences'
        AND column_name = 'daily_goal'
) THEN
ALTER TABLE user_learning_preferences
ADD COLUMN daily_goal INTEGER DEFAULT 10;
END IF;
END $$;
-- Ensure existing rows have a value
UPDATE user_learning_preferences
SET daily_goal = 10
WHERE daily_goal IS NULL;
