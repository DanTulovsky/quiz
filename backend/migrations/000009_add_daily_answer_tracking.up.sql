-- Add user answer tracking to daily question assignments
-- This allows us to track what answer the user selected and when they submitted it

ALTER TABLE daily_question_assignments 
ADD COLUMN user_answer_index INTEGER,
ADD COLUMN submitted_at TIMESTAMP;

-- Add comments to document the new columns
COMMENT ON COLUMN daily_question_assignments.user_answer_index IS 'The index of the answer option the user selected (0-based)';
COMMENT ON COLUMN daily_question_assignments.submitted_at IS 'When the user submitted their answer';

-- Add index for querying by submission status
CREATE INDEX IF NOT EXISTS idx_daily_assignments_submitted ON daily_question_assignments(user_id, assignment_date, submitted_at); 