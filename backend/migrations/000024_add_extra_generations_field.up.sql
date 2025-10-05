-- Add extra_generations_today field to stories table to allow one extra generation per day
ALTER TABLE stories ADD COLUMN extra_generations_today INTEGER NOT NULL DEFAULT 0;
