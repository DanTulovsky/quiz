-- Remove iOS notification preferences and ios_device_tokens table

DO $$
BEGIN
    -- Remove word_of_day_ios_notify_enabled column
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'word_of_day_ios_notify_enabled'
    ) THEN
        ALTER TABLE user_learning_preferences DROP COLUMN word_of_day_ios_notify_enabled;
    END IF;

    -- Remove daily_reminder_ios_notify_enabled column
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'daily_reminder_ios_notify_enabled'
    ) THEN
        ALTER TABLE user_learning_preferences DROP COLUMN daily_reminder_ios_notify_enabled;
    END IF;
END $$;

-- Drop ios_device_tokens table
DROP TABLE IF EXISTS ios_device_tokens;

