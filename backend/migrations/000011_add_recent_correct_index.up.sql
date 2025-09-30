-- Add partial index to optimize recent-correct exclusion for daily eligibility
CREATE INDEX IF NOT EXISTS idx_user_responses_recent_correct
ON user_responses(user_id, question_id, created_at)
WHERE is_correct = TRUE;


