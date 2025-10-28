-- Remove word_of_day_email_enabled field from users table
ALTER TABLE users DROP COLUMN IF EXISTS word_of_day_email_enabled;

-- Drop indexes
DROP INDEX IF EXISTS idx_word_of_the_day_user_date;
DROP INDEX IF EXISTS idx_word_of_the_day_assignment_date;
DROP INDEX IF EXISTS idx_word_of_the_day_user_id;

-- Drop word_of_the_day table
DROP TABLE IF EXISTS word_of_the_day;

