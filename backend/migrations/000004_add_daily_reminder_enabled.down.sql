-- Remove daily_reminder_enabled column from user_learning_preferences table

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'daily_reminder_enabled'
    ) THEN
        ALTER TABLE user_learning_preferences DROP COLUMN daily_reminder_enabled;
    END IF;
END $$;
