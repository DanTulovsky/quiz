-- Add daily_reminder_enabled column to user_learning_preferences table
-- This column controls whether users receive daily reminder emails

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'daily_reminder_enabled'
    ) THEN
        -- Add the new column with default value FALSE
        ALTER TABLE user_learning_preferences ADD COLUMN daily_reminder_enabled BOOLEAN DEFAULT FALSE;

        -- Add a comment to document the column purpose
        COMMENT ON COLUMN user_learning_preferences.daily_reminder_enabled IS 'Whether to send daily reminder emails to this user';
    END IF;
END $$;
