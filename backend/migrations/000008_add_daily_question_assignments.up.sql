-- Add daily question assignments table
-- This table tracks which questions are assigned to which users for which dates

CREATE TABLE IF NOT EXISTS daily_question_assignments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    assignment_date DATE NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    UNIQUE(user_id, question_id, assignment_date)
);

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_daily_assignments_user_date ON daily_question_assignments(user_id, assignment_date);
CREATE INDEX IF NOT EXISTS idx_daily_assignments_user_question ON daily_question_assignments(user_id, question_id);
CREATE INDEX IF NOT EXISTS idx_daily_assignments_completed ON daily_question_assignments(user_id, assignment_date, is_completed);
CREATE INDEX IF NOT EXISTS idx_daily_assignments_date ON daily_question_assignments(assignment_date);

-- Add comments to document the table purpose
COMMENT ON TABLE daily_question_assignments IS 'Tracks daily question assignments for users';
COMMENT ON COLUMN daily_question_assignments.user_id IS 'The user who was assigned the question';
COMMENT ON COLUMN daily_question_assignments.question_id IS 'The question that was assigned';
COMMENT ON COLUMN daily_question_assignments.assignment_date IS 'The date for which the question was assigned';
COMMENT ON COLUMN daily_question_assignments.is_completed IS 'Whether the user has completed this question for this date';
COMMENT ON COLUMN daily_question_assignments.completed_at IS 'When the user completed this question (if completed)';
COMMENT ON COLUMN daily_question_assignments.created_at IS 'When this assignment was created';
