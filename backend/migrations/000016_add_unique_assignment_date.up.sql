-- Up: add unique constraint on (user_id, question_id, assignment_date) if no duplicates

DO $$
DECLARE
  dup_count int;
  has_constraint int;
BEGIN
  -- Check if the unique constraint already exists
  SELECT COUNT(*) INTO has_constraint
  FROM pg_constraint
  WHERE conrelid = 'daily_question_assignments'::regclass
    AND contype = 'u'
    AND pg_get_constraintdef(oid) LIKE '%(user_id, question_id, assignment_date)%';

  IF has_constraint = 0 THEN
    -- Count duplicate groups
    SELECT COUNT(*) INTO dup_count FROM (
      SELECT 1 FROM daily_question_assignments
      GROUP BY user_id, question_id, assignment_date
      HAVING COUNT(*) > 1
    ) t;

    IF dup_count = 0 THEN
      ALTER TABLE daily_question_assignments
        ADD CONSTRAINT daily_question_assignments_user_question_assignment_date_unique
        UNIQUE (user_id, question_id, assignment_date);
    ELSE
      RAISE NOTICE 'Found % duplicate (user_id,question_id,assignment_date) groups; skipping unique constraint creation', dup_count;
    END IF;
  ELSE
    RAISE NOTICE 'Unique constraint already exists; nothing to do';
  END IF;
END$$;


