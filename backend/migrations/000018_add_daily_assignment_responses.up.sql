-- Add table to link daily_question_assignments to canonical user_responses
CREATE TABLE IF NOT EXISTS daily_assignment_responses (
    id SERIAL PRIMARY KEY,
    assignment_id INTEGER NOT NULL REFERENCES daily_question_assignments(id) ON DELETE CASCADE,
    user_response_id INTEGER NOT NULL REFERENCES user_responses(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(assignment_id),
    UNIQUE(user_response_id)
);

CREATE INDEX IF NOT EXISTS idx_daily_assignment_responses_assignment_id ON daily_assignment_responses(assignment_id);


