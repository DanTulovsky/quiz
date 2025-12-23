-- Add iOS notification preferences to user_learning_preferences table
-- Add ios_device_tokens table for storing device tokens

DO $$
BEGIN
    -- Add word_of_day_ios_notify_enabled column
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'word_of_day_ios_notify_enabled'
    ) THEN
        ALTER TABLE user_learning_preferences ADD COLUMN word_of_day_ios_notify_enabled BOOLEAN DEFAULT FALSE;
        COMMENT ON COLUMN user_learning_preferences.word_of_day_ios_notify_enabled IS 'Whether to send Word of the Day iOS push notifications to this user';
    END IF;

    -- Add daily_reminder_ios_notify_enabled column
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences' AND column_name = 'daily_reminder_ios_notify_enabled'
    ) THEN
        ALTER TABLE user_learning_preferences ADD COLUMN daily_reminder_ios_notify_enabled BOOLEAN DEFAULT FALSE;
        COMMENT ON COLUMN user_learning_preferences.daily_reminder_ios_notify_enabled IS 'Whether to send daily reminder iOS push notifications to this user';
    END IF;
END $$;

-- Create ios_device_tokens table
CREATE TABLE IF NOT EXISTS ios_device_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_token TEXT NOT NULL,
    device_type VARCHAR(20) DEFAULT 'ios',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, device_token)
);

-- Add index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_ios_device_tokens_user_id ON ios_device_tokens(user_id);

-- Add comment to document the table purpose
COMMENT ON TABLE ios_device_tokens IS 'Stores iOS device tokens for push notifications';

