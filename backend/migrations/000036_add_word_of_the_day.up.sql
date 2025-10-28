-- Create word_of_the_day table
CREATE TABLE IF NOT EXISTS word_of_the_day (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    assignment_date DATE NOT NULL,
    source_type VARCHAR(20) NOT NULL CHECK (
        source_type IN ('vocabulary_question', 'snippet')
    ),
    source_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE(user_id, assignment_date)
);
-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_word_of_the_day_user_id ON word_of_the_day(user_id);
CREATE INDEX IF NOT EXISTS idx_word_of_the_day_assignment_date ON word_of_the_day(assignment_date);
CREATE INDEX IF NOT EXISTS idx_word_of_the_day_user_date ON word_of_the_day(user_id, assignment_date);
-- Add word_of_day_email_enabled field to users table for email preferences
ALTER TABLE users
ADD COLUMN IF NOT EXISTS word_of_day_email_enabled BOOLEAN DEFAULT FALSE;
