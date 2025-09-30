-- Up: convert various TIMESTAMP and DATE columns to TIMESTAMPTZ
-- This migration is idempotent and attempts to preserve instants assuming stored values are UTC.

DO $$
BEGIN
  -- Drop materialized/view dependencies first to allow altering underlying column types
  IF EXISTS (SELECT 1 FROM pg_matviews WHERE matviewname = 'user_performance_analytics') THEN
    EXECUTE 'DROP MATERIALIZED VIEW IF EXISTS user_performance_analytics CASCADE';
  END IF;

  -- users
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='last_active' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN last_active TYPE timestamptz USING last_active AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_api_keys
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_api_keys' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_api_keys ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_api_keys' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_api_keys ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- questions
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='questions' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE questions ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_questions
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_questions' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_questions ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_responses
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_responses' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_responses ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_question_metadata
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_question_metadata' AND column_name='marked_as_known_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_question_metadata ALTER COLUMN marked_as_known_at TYPE timestamptz USING marked_as_known_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_question_metadata' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_question_metadata ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_question_metadata' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_question_metadata ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- question_priority_scores
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='question_priority_scores' AND column_name='last_calculated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE question_priority_scores ALTER COLUMN last_calculated_at TYPE timestamptz USING last_calculated_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='question_priority_scores' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE question_priority_scores ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='question_priority_scores' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE question_priority_scores ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_learning_preferences
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_learning_preferences' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_learning_preferences ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_learning_preferences' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_learning_preferences ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- worker_settings
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_settings' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_settings ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_settings' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_settings ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- worker_status
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_status' AND column_name='last_heartbeat' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_status ALTER COLUMN last_heartbeat TYPE timestamptz USING last_heartbeat AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_status' AND column_name='last_run_start' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_status ALTER COLUMN last_run_start TYPE timestamptz USING last_run_start AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_status' AND column_name='last_run_finish' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_status ALTER COLUMN last_run_finish TYPE timestamptz USING last_run_finish AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_status' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_status ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='worker_status' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE worker_status ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- roles
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='roles' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE roles ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='roles' AND column_name='updated_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE roles ALTER COLUMN updated_at TYPE timestamptz USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- user_roles
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_roles' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_roles ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- question_reports
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='question_reports' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE question_reports ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- notification_errors
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='notification_errors' AND column_name='occurred_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE notification_errors ALTER COLUMN occurred_at TYPE timestamptz USING occurred_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='notification_errors' AND column_name='resolved_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE notification_errors ALTER COLUMN resolved_at TYPE timestamptz USING resolved_at AT TIME ZONE ''UTC''';
  END IF;

  -- upcoming_notifications
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='upcoming_notifications' AND column_name='scheduled_for' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE upcoming_notifications ALTER COLUMN scheduled_for TYPE timestamptz USING scheduled_for AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='upcoming_notifications' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE upcoming_notifications ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- sent_notifications
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sent_notifications' AND column_name='sent_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE sent_notifications ALTER COLUMN sent_at TYPE timestamptz USING sent_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sent_notifications' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE sent_notifications ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- generation_hints
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='generation_hints' AND column_name='expires_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE generation_hints ALTER COLUMN expires_at TYPE timestamptz USING expires_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='generation_hints' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE generation_hints ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- daily_question_assignments: date -> timestamptz (cast date to timestamp at midnight UTC)
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date' AND data_type='date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN assignment_date TYPE timestamptz USING (assignment_date::timestamp AT TIME ZONE ''UTC'')';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='completed_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN completed_at TYPE timestamptz USING completed_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

  -- performance_metrics.last_updated
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='performance_metrics' AND column_name='last_updated' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE performance_metrics ALTER COLUMN last_updated TYPE timestamptz USING last_updated AT TIME ZONE ''UTC''';
  END IF;

  -- user_questions.created_at
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='user_questions' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE user_questions ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

END$$;

-- NOTE: Adjust AT TIME ZONE if data was stored in a non-UTC zone to preserve instants.


-- Recreate materialized view that we dropped earlier
CREATE MATERIALIZED VIEW IF NOT EXISTS user_performance_analytics AS
SELECT u.id as user_id,
    u.username,
    COUNT(ur.id) as total_responses,
    AVG(
        CASE
            WHEN ur.is_correct THEN 1.0
            ELSE 0.0
        END
    ) as accuracy_rate,
    AVG(ur.response_time_ms) as avg_response_time,
    COUNT(uqm.question_id) FILTER (
        WHERE uqm.marked_as_known = TRUE
    ) as marked_questions,
    COUNT(
        CASE
            WHEN ur.created_at > NOW() - INTERVAL '7 days' THEN 1
        END
    ) as recent_responses,
    MAX(ur.created_at) as last_response_at
FROM users u
    LEFT JOIN user_responses ur ON u.id = ur.user_id
    LEFT JOIN user_question_metadata uqm ON u.id = uqm.user_id
    AND ur.question_id = uqm.question_id
GROUP BY u.id,
    u.username;

-- Recreate index on materialized view
CREATE INDEX IF NOT EXISTS idx_user_performance_analytics_user ON user_performance_analytics(user_id);


