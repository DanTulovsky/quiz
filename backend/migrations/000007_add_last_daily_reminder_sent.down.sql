-- Remove last_daily_reminder_sent column from user_learning_preferences table

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'last_daily_reminder_sent'
    ) THEN
        ALTER TABLE user_learning_preferences DROP COLUMN last_daily_reminder_sent;
    END IF;
END $$;
