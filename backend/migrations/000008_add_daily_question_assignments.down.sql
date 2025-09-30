-- Remove daily question assignments table and related indexes
-- This migration rolls back the addition of daily question assignments

-- Drop indexes first
DROP INDEX IF EXISTS idx_daily_assignments_date;
DROP INDEX IF EXISTS idx_daily_assignments_completed;
DROP INDEX IF EXISTS idx_daily_assignments_user_question;
DROP INDEX IF EXISTS idx_daily_assignments_user_date;

-- Drop the table
DROP TABLE IF EXISTS daily_question_assignments;
