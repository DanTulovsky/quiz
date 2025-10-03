-- Drop trigger and function
DROP TRIGGER IF EXISTS update_stories_updated_at ON stories;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (questions, sections, stories)
DROP TABLE IF EXISTS story_section_questions;
DROP TABLE IF EXISTS story_sections;
DROP TABLE IF EXISTS stories;
