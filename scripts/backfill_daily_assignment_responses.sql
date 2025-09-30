-- Optional backfill script to populate daily_assignment_responses for historical data
-- This script is safe to run multiple times; it only inserts mappings for assignments
-- that do not already have an entry in daily_assignment_responses.

-- WARNING: This script uses created_at::date to match responses to assignment_date.
-- If you need user-local-day correctness, run a timezone-aware backfill instead
-- (see notes below).

-- Example: run with psql connecting to your DB
-- psql "postgres://user:pass@host:5432/dbname" -f scripts/backfill_daily_assignment_responses.sql

-- Simple single-statement backfill (may be heavy on large DBs)
INSERT INTO daily_assignment_responses (assignment_id, user_response_id, created_at)
SELECT dqa.id, ur.id, ur.created_at
FROM daily_question_assignments dqa
JOIN LATERAL (
  SELECT ur.id, ur.created_at
  FROM user_responses ur
  WHERE ur.user_id = dqa.user_id
    AND ur.question_id = dqa.question_id
    AND ur.created_at::date = dqa.assignment_date
  ORDER BY ur.created_at DESC
  LIMIT 1
) ur ON true
WHERE NOT EXISTS (
  SELECT 1 FROM daily_assignment_responses dar WHERE dar.assignment_id = dqa.id
);

-- Notes on timezone correctness:
-- - The queries above use `created_at::date = assignment_date`, which interprets
--   the response timestamp in the database session/timezone. If your DB is UTC
--   but users are in other timezones, this may mis-assign responses around midnight.
-- - For robust results, consider a script that:
--     1) Loads assignments and their user.timezone from users table.
--     2) For each assignment, computes the user's local date range (assignment_date 00:00 -> +24h in user's timezone)
--     3) Finds the latest user_responses.created_at within that local range and inserts the mapping.
--   This can be implemented as a one-off Go/Python script using the application's DB and timezone data.


