-- Add last_daily_reminder_sent column to user_learning_preferences table
-- This column tracks when the last daily reminder email was sent to each user

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'last_daily_reminder_sent'
    ) THEN
        -- Add the new column with NULL default (no reminder sent yet)
        ALTER TABLE user_learning_preferences ADD COLUMN last_daily_reminder_sent TIMESTAMP;

        -- Add a comment to document the column purpose
        COMMENT ON COLUMN user_learning_preferences.last_daily_reminder_sent IS 'Timestamp of when the last daily reminder email was sent to this user';
    END IF;
END $$;
