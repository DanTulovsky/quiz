-- Add stories table
CREATE TABLE IF NOT EXISTS stories (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    language VARCHAR(10) NOT NULL,
    subject TEXT,
    author_style TEXT,
    time_period TEXT,
    genre TEXT,
    tone TEXT,
    character_names TEXT,
    custom_instructions TEXT,
    section_length_override VARCHAR(10) CHECK (section_length_override IN ('short', 'medium', 'long')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'archived', 'completed')) DEFAULT 'active',
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    last_section_generated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for stories table
CREATE INDEX IF NOT EXISTS idx_stories_user_id ON stories(user_id);
CREATE INDEX IF NOT EXISTS idx_stories_status ON stories(status);
CREATE INDEX IF NOT EXISTS idx_stories_user_current ON stories(user_id, is_current);
CREATE INDEX IF NOT EXISTS idx_stories_user_status ON stories(user_id, status);

-- Create partial unique index to ensure only one current story per user
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user
ON stories (user_id)
WHERE is_current = true;

-- Add story_sections table
CREATE TABLE IF NOT EXISTS story_sections (
    id SERIAL PRIMARY KEY,
    story_id INTEGER NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    section_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    language_level VARCHAR(5) NOT NULL,
    word_count INTEGER NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    generation_date DATE NOT NULL DEFAULT CURRENT_DATE,

    CONSTRAINT unique_story_section_number UNIQUE (story_id, section_number) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT unique_story_generation_date UNIQUE (story_id, generation_date) DEFERRABLE INITIALLY DEFERRED
);

-- Create indexes for story_sections table
CREATE INDEX IF NOT EXISTS idx_story_sections_story_id ON story_sections(story_id);
CREATE INDEX IF NOT EXISTS idx_story_sections_number ON story_sections(story_id, section_number);
CREATE INDEX IF NOT EXISTS idx_story_sections_date ON story_sections(generation_date);

-- Add story_section_questions table
CREATE TABLE IF NOT EXISTS story_section_questions (
    id SERIAL PRIMARY KEY,
    section_id INTEGER NOT NULL REFERENCES story_sections(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_answer_index INTEGER NOT NULL CHECK (correct_answer_index >= 0 AND correct_answer_index <= 3),
    explanation TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for story_section_questions table
CREATE INDEX IF NOT EXISTS idx_story_section_questions_section_id ON story_section_questions(section_id);

-- Add trigger to update updated_at timestamp on stories table
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_stories_updated_at BEFORE UPDATE ON stories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
