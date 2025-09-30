-- Down: revert assignment_date timestamptz -> date (best-effort)

DO $$
BEGIN
  -- Add backup column
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date_date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments ADD COLUMN assignment_date_date date';
  END IF;

  -- Populate from timestamptz by converting to UTC date
  EXECUTE $$
    UPDATE daily_question_assignments
    SET assignment_date_date = (assignment_date AT TIME ZONE 'UTC')::date
    WHERE assignment_date_date IS NULL;
  $$;

  -- Drop current assignment_date and rename backup into place
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='daily_question_assignments' AND column_name='assignment_date') THEN
    EXECUTE 'ALTER TABLE daily_question_assignments DROP COLUMN assignment_date';
  END IF;
  EXECUTE 'ALTER TABLE daily_question_assignments RENAME COLUMN assignment_date_date TO assignment_date';

END$$;


