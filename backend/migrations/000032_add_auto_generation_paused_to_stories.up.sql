-- Add auto_generation_paused column to stories table
ALTER TABLE stories ADD COLUMN IF NOT EXISTS auto_generation_paused BOOLEAN NOT NULL DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN stories.auto_generation_paused IS 'When true, the worker will skip automatic section generation for this story. Manual generation still works.';

