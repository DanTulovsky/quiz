-- Remove daily_goal column from user_learning_preferences
DO $$ BEGIN IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'user_learning_preferences'
        AND column_name = 'daily_goal'
) THEN
ALTER TABLE user_learning_preferences DROP COLUMN daily_goal;
END IF;
END $$;
