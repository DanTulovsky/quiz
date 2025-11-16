-- Add translation practice tables

-- Translation practice sentences table - stores sentences available for translation practice
CREATE TABLE IF NOT EXISTS translation_practice_sentences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sentence_text TEXT NOT NULL,
    source_language VARCHAR(10) NOT NULL,
    target_language VARCHAR(10) NOT NULL,
    language_level VARCHAR(5) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('ai_generated', 'story_section', 'vocabulary_question', 'reading_comprehension', 'snippet', 'phrasebook')),
    source_id INTEGER, -- ID of the source (story_id, question_id, snippet_id, etc.)
    topic TEXT, -- Optional topic or keywords for AI-generated sentences
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for translation_practice_sentences
CREATE INDEX IF NOT EXISTS idx_translation_practice_sentences_user_id ON translation_practice_sentences(user_id);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sentences_languages ON translation_practice_sentences(source_language, target_language);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sentences_level ON translation_practice_sentences(language_level);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sentences_source ON translation_practice_sentences(source_type, source_id);

-- Translation practice sessions table - tracks individual practice attempts
CREATE TABLE IF NOT EXISTS translation_practice_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sentence_id INTEGER NOT NULL REFERENCES translation_practice_sentences(id) ON DELETE CASCADE,
    original_sentence TEXT NOT NULL,
    user_translation TEXT NOT NULL,
    translation_direction VARCHAR(20) NOT NULL CHECK (translation_direction IN ('en_to_learning', 'learning_to_en')),
    ai_feedback TEXT NOT NULL,
    ai_score DECIMAL(3, 2) CHECK (ai_score >= 0 AND ai_score <= 5), -- Score from 0 to 5
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for translation_practice_sessions
CREATE INDEX IF NOT EXISTS idx_translation_practice_sessions_user_id ON translation_practice_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sessions_sentence_id ON translation_practice_sessions(sentence_id);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sessions_created_at ON translation_practice_sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_translation_practice_sessions_user_created ON translation_practice_sessions(user_id, created_at DESC);



