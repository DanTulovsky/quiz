-- Up: convert daily_question_assignments.assignment_date (timestamptz) -> date
-- Populate using UTC date so the stored value represents the calendar day in UTC.

DO $$
BEGIN
  -- Add a temporary column to hold date values
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='daily_question_assignments' AND column_name='assignment_date_date'
  ) THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ADD COLUMN assignment_date_date date';
  END IF;

  -- Populate assignment_date_date from existing timestamptz by converting to UTC date
  UPDATE daily_question_assignments
  SET assignment_date_date = (assignment_date AT TIME ZONE 'UTC')::date
  WHERE assignment_date_date IS NULL;

  -- Drop constraints/indexes that reference the old column if necessary is left to migration framework

  -- Replace old column with the date column
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments DROP COLUMN assignment_date';
  END IF;
  EXECUTE 'ALTER TABLE daily_question_assignments RENAME COLUMN assignment_date_date TO assignment_date';

  -- Recreate indexes used by queries (if they do not exist)
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE tablename = 'daily_question_assignments' AND indexname = 'idx_daily_assignments_user_date'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_daily_assignments_user_date ON daily_question_assignments(user_id, assignment_date)';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE tablename = 'daily_question_assignments' AND indexname = 'idx_daily_assignments_date'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_daily_assignments_date ON daily_question_assignments(assignment_date)';
  END IF;

END$$;


