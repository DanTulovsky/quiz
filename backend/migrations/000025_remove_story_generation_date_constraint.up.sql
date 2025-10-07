-- Remove the unique constraint on story_id and generation_date to allow multiple sections per day
ALTER TABLE story_sections DROP CONSTRAINT IF EXISTS unique_story_generation_date;
