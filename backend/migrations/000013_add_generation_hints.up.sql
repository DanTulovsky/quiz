-- Create generation_hints table to record short-lived per-user generation priorities
CREATE TABLE IF NOT EXISTS generation_hints (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    level TEXT NOT NULL,
    question_type TEXT NOT NULL,
    priority_weight INTEGER NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT generation_hints_user_lang_level_type_unique UNIQUE (user_id, language, level, question_type)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_generation_hints_user_expires ON generation_hints (user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_generation_hints_expires ON generation_hints (expires_at);

