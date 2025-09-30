-- Add tts_voice column to user_learning_preferences table
-- Stores the preferred TTS voice code (e.g., it-IT-IsabellaNeural)

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences'
          AND column_name = 'tts_voice'
    ) THEN
        ALTER TABLE user_learning_preferences
        ADD COLUMN tts_voice TEXT DEFAULT '';

        COMMENT ON COLUMN user_learning_preferences.tts_voice IS 'Preferred TTS voice (e.g., it-IT-IsabellaNeural)';
    END IF;
END $$;


