-- Down: revert daily_question_assignments.assignment_date (date) -> timestamptz (UTC midnight)

DO $$
BEGIN
  -- Add a temporary timestamptz column
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='daily_question_assignments' AND column_name='assignment_date_tz'
  ) THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ADD COLUMN assignment_date_tz timestamptz';
  END IF;

  -- Populate assignment_date_tz by casting date at midnight UTC
  EXECUTE $$
    UPDATE daily_question_assignments
    SET assignment_date_tz = (assignment_date::timestamp AT TIME ZONE 'UTC')
    WHERE assignment_date_tz IS NULL;
  $$;

  -- Drop old date column and rename tz column into place
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments DROP COLUMN assignment_date';
  END IF;
  EXECUTE 'ALTER TABLE daily_question_assignments RENAME COLUMN assignment_date_tz TO assignment_date';

  -- Recreate original indexes for timestamptz queries
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE tablename = 'daily_question_assignments' AND indexname = 'idx_daily_question_assignments_date'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_daily_question_assignments_date ON daily_question_assignments(assignment_date)';
  END IF;

END$$;


