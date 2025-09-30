-- Up: convert daily_question_assignments.assignment_date (date) -> timestamptz
-- Populate using the user's timezone so the stored instant represents local-midnight

DO $$
DECLARE
  cs text[];
  c text;
BEGIN
  -- Add a temporary column to hold timestamptz values
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date_tz') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ADD COLUMN assignment_date_tz timestamptz';
  END IF;

  -- Populate assignment_date_tz using the user's timezone (fall back to UTC)
  EXECUTE $upd$
    UPDATE daily_question_assignments d
    SET assignment_date_tz = (d.assignment_date::timestamp AT TIME ZONE COALESCE(u.timezone,'UTC'))
    FROM users u
    WHERE u.id = d.user_id AND d.assignment_date_tz IS NULL;
  $upd$;

  -- Drop any existing constraints or indexes that reference the old date column
  SELECT array_agg(conname) INTO cs FROM pg_constraint WHERE conrelid = 'daily_question_assignments'::regclass AND pg_get_constraintdef(oid) LIKE '%assignment_date%';
  IF cs IS NOT NULL THEN
    FOREACH c IN ARRAY cs LOOP
      EXECUTE format('ALTER TABLE daily_question_assignments DROP CONSTRAINT IF EXISTS %I', c);
    END LOOP;
  END IF;

  -- Drop the old assignment_date column and replace with assignment_date_tz
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments DROP COLUMN assignment_date';
  END IF;
  EXECUTE 'ALTER TABLE daily_question_assignments RENAME COLUMN assignment_date_tz TO assignment_date';

  -- Ensure a unique constraint on (user_id, question_id, assignment_date)
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conrelid = 'daily_question_assignments'::regclass AND contype = 'u' AND pg_get_constraintdef(oid) LIKE '%(user_id, question_id, assignment_date)%'
  ) THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ADD CONSTRAINT daily_question_assignments_user_question_assignment_date_unique UNIQUE (user_id, question_id, assignment_date)';
  END IF;

  -- Convert other timestamp columns on this table if they are still without TZ
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='completed_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN completed_at TYPE timestamptz USING completed_at AT TIME ZONE ''UTC''';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='created_at' AND data_type='timestamp without time zone') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE ''UTC''';
  END IF;

END$$;

-- Note: this migration assumes the user's timezone field contains a valid IANA timezone name.
-- Rows with NULL/invalid timezone will be treated as UTC.


