ALTER TABLE user_question_metadata
    ADD COLUMN IF NOT EXISTS confidence_level INTEGER CHECK (confidence_level >= 1 AND confidence_level <= 5);
