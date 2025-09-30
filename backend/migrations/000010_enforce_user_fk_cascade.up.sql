-- Ensure all user-related foreign keys have ON DELETE CASCADE for legacy databases
-- This migration is safe to run even if schema.sql already created them with CASCADE
-- because it drops and re-creates constraints idempotently.
BEGIN;
-- user_api_keys.user_id -> users.id
ALTER TABLE IF EXISTS user_api_keys DROP CONSTRAINT IF EXISTS user_api_keys_user_id_fkey;
ALTER TABLE user_api_keys
ADD CONSTRAINT user_api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- user_questions.user_id -> users.id
ALTER TABLE IF EXISTS user_questions DROP CONSTRAINT IF EXISTS user_questions_user_id_fkey;
ALTER TABLE user_questions
ADD CONSTRAINT user_questions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- user_question_metadata.user_id -> users.id
ALTER TABLE IF EXISTS user_question_metadata DROP CONSTRAINT IF EXISTS user_question_metadata_user_id_fkey;
ALTER TABLE user_question_metadata
ADD CONSTRAINT user_question_metadata_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- question_priority_scores.user_id -> users.id
ALTER TABLE IF EXISTS question_priority_scores DROP CONSTRAINT IF EXISTS question_priority_scores_user_id_fkey;
ALTER TABLE question_priority_scores
ADD CONSTRAINT question_priority_scores_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- user_learning_preferences.user_id -> users.id
ALTER TABLE IF EXISTS user_learning_preferences DROP CONSTRAINT IF EXISTS user_learning_preferences_user_id_fkey;
ALTER TABLE user_learning_preferences
ADD CONSTRAINT user_learning_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- performance_metrics.user_id -> users.id
ALTER TABLE IF EXISTS performance_metrics DROP CONSTRAINT IF EXISTS performance_metrics_user_id_fkey;
ALTER TABLE performance_metrics
ADD CONSTRAINT performance_metrics_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- user_roles.user_id -> users.id
ALTER TABLE IF EXISTS user_roles DROP CONSTRAINT IF EXISTS user_roles_user_id_fkey;
ALTER TABLE user_roles
ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- question_reports.reported_by_user_id -> users.id
ALTER TABLE IF EXISTS question_reports DROP CONSTRAINT IF EXISTS question_reports_reported_by_user_id_fkey;
ALTER TABLE question_reports
ADD CONSTRAINT question_reports_reported_by_user_id_fkey FOREIGN KEY (reported_by_user_id) REFERENCES users(id) ON DELETE CASCADE;
-- upcoming_notifications.user_id -> users.id
ALTER TABLE IF EXISTS upcoming_notifications DROP CONSTRAINT IF EXISTS upcoming_notifications_user_id_fkey;
ALTER TABLE upcoming_notifications
ADD CONSTRAINT upcoming_notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- sent_notifications.user_id -> users.id
ALTER TABLE IF EXISTS sent_notifications DROP CONSTRAINT IF EXISTS sent_notifications_user_id_fkey;
ALTER TABLE sent_notifications
ADD CONSTRAINT sent_notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- daily_question_assignments.user_id -> users.id (if table exists)
ALTER TABLE IF EXISTS daily_question_assignments DROP CONSTRAINT IF EXISTS daily_question_assignments_user_id_fkey;
ALTER TABLE IF EXISTS daily_question_assignments
ADD CONSTRAINT daily_question_assignments_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
COMMIT;
