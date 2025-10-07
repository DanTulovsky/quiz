-- Add back the unique constraint on story_id and generation_date
ALTER TABLE story_sections ADD CONSTRAINT unique_story_generation_date UNIQUE (story_id, generation_date) DEFERRABLE INITIALLY DEFERRED;
