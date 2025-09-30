-- Remove user answer tracking from daily question assignments

DROP INDEX IF EXISTS idx_daily_assignments_submitted;

ALTER TABLE daily_question_assignments 
DROP COLUMN IF EXISTS user_answer_index,
DROP COLUMN IF EXISTS submitted_at; 