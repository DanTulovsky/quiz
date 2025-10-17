-- Add story section views tracking table
-- This table tracks when users view/read story sections to enable engagement-based generation

CREATE TABLE IF NOT EXISTS story_section_views (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    section_id INTEGER NOT NULL REFERENCES story_sections(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Ensure one view per user per section (most recent view wins)
    UNIQUE(user_id, section_id)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_story_section_views_user_id ON story_section_views(user_id);
CREATE INDEX IF NOT EXISTS idx_story_section_views_section_id ON story_section_views(section_id);
CREATE INDEX IF NOT EXISTS idx_story_section_views_user_viewed ON story_section_views(user_id, viewed_at);
CREATE INDEX IF NOT EXISTS idx_story_section_views_section_viewed ON story_section_views(section_id, viewed_at);
