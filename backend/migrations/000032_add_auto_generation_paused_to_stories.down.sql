-- Remove auto_generation_paused column from stories table
ALTER TABLE stories DROP COLUMN IF EXISTS auto_generation_paused;

