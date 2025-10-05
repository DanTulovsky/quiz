-- Revert the extra_generations_today field addition
ALTER TABLE stories DROP COLUMN IF EXISTS extra_generations_today;
