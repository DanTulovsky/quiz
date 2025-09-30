-- Remove tts_voice column from user_learning_preferences table

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'user_learning_preferences'
          AND column_name = 'tts_voice'
    ) THEN
        ALTER TABLE user_learning_preferences
        DROP COLUMN tts_voice;
    END IF;
END $$;


