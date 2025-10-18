-- Add snippets table for storing user-saved words and phrases
-- This table allows users to save vocabulary items from translation lookups

CREATE TABLE IF NOT EXISTS snippets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_text TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    source_language VARCHAR(10) NOT NULL,
    target_language VARCHAR(10) NOT NULL,
    question_id INTEGER REFERENCES questions(id) ON DELETE SET NULL,
    context TEXT,
    difficulty_level VARCHAR(20), -- CEFR level (A1, A2, B1, B2, C1, C2) from question or default
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    -- Ensure one snippet per user per original text (case-insensitive)
    UNIQUE(user_id, original_text, source_language, target_language)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);
CREATE INDEX IF NOT EXISTS idx_snippets_user_created ON snippets(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snippets_source_language ON snippets(source_language);
CREATE INDEX IF NOT EXISTS idx_snippets_target_language ON snippets(target_language);
CREATE INDEX IF NOT EXISTS idx_snippets_question_id ON snippets(question_id);
CREATE INDEX IF NOT EXISTS idx_snippets_search_text ON snippets USING gin(to_tsvector('english', original_text || ' ' || translated_text));
