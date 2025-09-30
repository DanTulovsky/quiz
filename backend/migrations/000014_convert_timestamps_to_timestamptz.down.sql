-- Down: best-effort revert of timestamptz -> timestamp without time zone
-- WARNING: This will DROP timezone information and should only be used if you know all data is UTC.

DO $$
BEGIN
  -- users
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='last_active' AND data_type='timestamp with time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN last_active TYPE timestamp without time zone USING last_active AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='created_at' AND data_type='timestamp with time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN created_at TYPE timestamp without time zone USING created_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='updated_at' AND data_type='timestamp with time zone') THEN
    EXECUTE 'ALTER TABLE users ALTER COLUMN updated_at TYPE timestamp without time zone USING updated_at AT TIME ZONE ''UTC''';
  END IF;

  -- daily_question_assignments
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date' AND data_type='timestamp with time zone') THEN
    -- Convert timestamptz back to date (drop time) as best-effort
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN assignment_date TYPE date USING (assignment_date AT TIME ZONE ''UTC'')::date';
  END IF;

  -- Note: Other reversions (timestamps -> timestamp without time zone) can be added similarly if needed.
END$$;


