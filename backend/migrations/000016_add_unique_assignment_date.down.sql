-- Down: drop unique constraint if it exists

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'daily_question_assignments'::regclass
      AND contype = 'u'
      AND pg_get_constraintdef(oid) LIKE '%(user_id, question_id, assignment_date)%'
  ) THEN
    ALTER TABLE daily_question_assignments DROP CONSTRAINT IF EXISTS daily_question_assignments_user_question_assignment_date_unique;
  END IF;
END$$;
