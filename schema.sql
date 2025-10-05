-- AI-Powered Quiz Application Database Schema
-- This file contains the complete database schema used by the application
-- It should be the single source of truth for database structure
-- Users table - stores user information and preferences
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    timezone VARCHAR(100),
    password_hash TEXT,
    last_active TIMESTAMPTZ,
    preferred_language VARCHAR(50),
    current_level VARCHAR(10),
    ai_provider VARCHAR(100),
    ai_model VARCHAR(100),
    ai_enabled BOOLEAN DEFAULT FALSE,
    ai_api_key TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- User API Keys table - stores multiple API keys per user per provider
-- This allows users to save different API keys for different AI providers
CREATE TABLE IF NOT EXISTS user_api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    provider VARCHAR(100) NOT NULL,
    api_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE(user_id, provider)
);
-- Questions table - stores generated quiz questions (now global/shared)
CREATE TABLE IF NOT EXISTS questions (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    language VARCHAR(50) NOT NULL,
    level VARCHAR(10) NOT NULL,
    difficulty_score DECIMAL(5, 2),
    content TEXT NOT NULL,
    correct_answer INTEGER NOT NULL,
    explanation TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    -- Variety elements for question generation diversity
    topic_category VARCHAR(100),
    grammar_focus VARCHAR(100),
    vocabulary_domain VARCHAR(100),
    scenario VARCHAR(100),
    style_modifier VARCHAR(100),
    difficulty_modifier VARCHAR(100),
    time_context VARCHAR(100)
);
-- User Questions mapping table - maps questions to users
CREATE TABLE IF NOT EXISTS user_questions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    UNIQUE(user_id, question_id)
);
-- User responses table - tracks user answers to questions
CREATE TABLE IF NOT EXISTS user_responses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    user_answer_index INTEGER NOT NULL,
    is_correct BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    confidence_level INTEGER CHECK (
        confidence_level >= 1
        AND confidence_level <= 5
    ),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id)
);
-- User question metadata table - tracks user-specific question metadata
CREATE TABLE IF NOT EXISTS user_question_metadata (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    question_id INTEGER REFERENCES questions(id) ON DELETE CASCADE,
    marked_as_known BOOLEAN DEFAULT FALSE,
    marked_as_known_at TIMESTAMPTZ,
    confidence_level INTEGER CHECK (
        confidence_level >= 1
        AND confidence_level <= 5
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, question_id)
);
-- Question priority scores table - tracks calculated priority scores for questions
CREATE TABLE IF NOT EXISTS question_priority_scores (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    question_id INTEGER REFERENCES questions(id) ON DELETE CASCADE,
    priority_score DECIMAL(10, 4) NOT NULL DEFAULT 100.0,
    last_calculated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, question_id)
);
-- User learning preferences table - stores user learning preferences
CREATE TABLE IF NOT EXISTS user_learning_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    focus_on_weak_areas BOOLEAN DEFAULT TRUE,
    include_review_questions BOOLEAN DEFAULT TRUE,
    fresh_question_ratio DECIMAL(3, 2) DEFAULT 0.3 CHECK (
        fresh_question_ratio >= 0
        AND fresh_question_ratio <= 1
    ),
    known_question_penalty DECIMAL(3, 2) DEFAULT 0.1 CHECK (
        known_question_penalty >= 0
        AND known_question_penalty <= 1
    ),
    review_interval_days INTEGER DEFAULT 7 CHECK (review_interval_days > 0),
    weak_area_boost DECIMAL(3, 2) DEFAULT 2.0 CHECK (
        weak_area_boost >= 0.1
        AND weak_area_boost <= 10.0
    ),
    -- User-configurable daily question goal
    daily_goal INTEGER DEFAULT 10 CHECK (daily_goal > 0),
    -- Preferred TTS voice (e.g., it-IT-IsabellaNeural)
    tts_voice TEXT DEFAULT '',
    daily_reminder_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);
-- Performance metrics table - tracks user performance across categories
CREATE TABLE IF NOT EXISTS performance_metrics (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    topic VARCHAR(100) NOT NULL,
    language VARCHAR(50) NOT NULL,
    level VARCHAR(10) NOT NULL,
    total_attempts INTEGER NOT NULL,
    correct_attempts INTEGER NOT NULL,
    average_response_time_ms DECIMAL(10, 2) NOT NULL,
    difficulty_adjustment DECIMAL(5, 2) DEFAULT 0.0,
    last_updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, topic, language, level)
);
-- Worker settings table - stores worker control settings
CREATE TABLE IF NOT EXISTS worker_settings (
    id SERIAL PRIMARY KEY,
    setting_key VARCHAR(255) UNIQUE NOT NULL,
    setting_value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- Worker status table - tracks worker health and activity
CREATE TABLE IF NOT EXISTS worker_status (
    id SERIAL PRIMARY KEY,
    worker_instance VARCHAR(255) NOT NULL DEFAULT 'default',
    is_running BOOLEAN NOT NULL DEFAULT false,
    is_paused BOOLEAN NOT NULL DEFAULT false,
    current_activity TEXT,
    last_heartbeat TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_run_start TIMESTAMPTZ,
    last_run_finish TIMESTAMPTZ,
    last_run_error TEXT,
    total_questions_generated INTEGER DEFAULT 0,
    total_runs INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(worker_instance)
);
-- Roles table - stores available roles in the system
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- User roles table - maps users to their roles
CREATE TABLE IF NOT EXISTS user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id)
);
-- Question reports table - tracks which users reported which questions
CREATE TABLE IF NOT EXISTS question_reports (
    id SERIAL PRIMARY KEY,
    question_id INTEGER NOT NULL,
    reported_by_user_id INTEGER NOT NULL,
    report_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    FOREIGN KEY (reported_by_user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE(question_id, reported_by_user_id)
);
-- Notification errors table - tracks notification errors for debugging
CREATE TABLE IF NOT EXISTS notification_errors (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    notification_type VARCHAR(50) NOT NULL,
    error_type VARCHAR(50) NOT NULL,
    -- 'smtp_error', 'template_error', 'user_not_found', etc.
    error_message TEXT NOT NULL,
    email_address VARCHAR(255),
    occurred_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE
    SET NULL
);
-- Upcoming notifications table - tracks notifications scheduled to be sent
CREATE TABLE IF NOT EXISTS upcoming_notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    scheduled_for TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- 'pending', 'sent', 'cancelled'
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS sent_notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    template_name VARCHAR(100) NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'sent',
    -- 'sent', 'failed', 'bounced'
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
-- Daily question assignments - track which questions are assigned to which users on which day
CREATE TABLE IF NOT EXISTS daily_question_assignments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    -- Use DATE to represent a logical calendar day for assignment (YYYY-MM-DD).
    -- Clients should request by local YYYY-MM-DD and server will match by this date.
    assignment_date DATE NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    UNIQUE(user_id, question_id, assignment_date)
);
-- Create indexes for better query performance
-- Note: These are created after all tables to ensure proper table existence
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_ai_enabled ON users(ai_enabled);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_user_provider ON user_api_keys(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_questions_type_lang_level ON questions(type, language, level);
CREATE INDEX IF NOT EXISTS idx_questions_status ON questions(status);
CREATE INDEX IF NOT EXISTS idx_questions_usage_count ON questions(usage_count);
CREATE INDEX IF NOT EXISTS idx_user_questions_user_id ON user_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_questions_question_id ON user_questions(question_id);
CREATE INDEX IF NOT EXISTS idx_user_questions_user_question ON user_questions(user_id, question_id);
-- Variety field indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_questions_topic_category ON questions(topic_category);
CREATE INDEX IF NOT EXISTS idx_questions_grammar_focus ON questions(grammar_focus);
CREATE INDEX IF NOT EXISTS idx_questions_vocabulary_domain ON questions(vocabulary_domain);
CREATE INDEX IF NOT EXISTS idx_questions_scenario ON questions(scenario);
CREATE INDEX IF NOT EXISTS idx_user_responses_user_id ON user_responses(user_id);
CREATE INDEX IF NOT EXISTS idx_user_responses_question_id ON user_responses(question_id);
CREATE INDEX IF NOT EXISTS idx_user_responses_created_at ON user_responses(created_at);
CREATE INDEX IF NOT EXISTS idx_user_responses_confidence ON user_responses(confidence_level);
CREATE INDEX IF NOT EXISTS idx_user_responses_user_created ON user_responses(user_id, created_at);
-- Optimize recent-correct exclusion in daily eligibility
CREATE INDEX IF NOT EXISTS idx_user_responses_recent_correct ON user_responses(user_id, question_id, created_at)
WHERE is_correct = TRUE;
-- Priority system indexes
CREATE INDEX IF NOT EXISTS idx_user_question_metadata_user_id ON user_question_metadata(user_id);
CREATE INDEX IF NOT EXISTS idx_user_question_metadata_question_id ON user_question_metadata(question_id);
CREATE INDEX IF NOT EXISTS idx_user_question_metadata_marked_known ON user_question_metadata(user_id, question_id)
WHERE marked_as_known = true;
CREATE INDEX IF NOT EXISTS idx_question_priority_scores_user_id ON question_priority_scores(user_id);
CREATE INDEX IF NOT EXISTS idx_question_priority_scores_question_id ON question_priority_scores(question_id);
CREATE INDEX IF NOT EXISTS idx_question_priority_scores_user_score ON question_priority_scores(user_id, priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_user_learning_preferences_user ON user_learning_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_user_id ON performance_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_worker_settings_key ON worker_settings(setting_key);
CREATE INDEX IF NOT EXISTS idx_worker_status_instance ON worker_status(worker_instance);
CREATE INDEX IF NOT EXISTS idx_worker_status_heartbeat ON worker_status(last_heartbeat);
-- Role system indexes
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_role ON user_roles(user_id, role_id);
CREATE INDEX IF NOT EXISTS idx_question_reports_question_id ON question_reports(question_id);
CREATE INDEX IF NOT EXISTS idx_question_reports_reported_by ON question_reports(reported_by_user_id);
CREATE INDEX IF NOT EXISTS idx_question_reports_created_at ON question_reports(created_at);
-- Notification tracking indexes
CREATE INDEX IF NOT EXISTS idx_notification_errors_user_id ON notification_errors(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_errors_type ON notification_errors(notification_type);
CREATE INDEX IF NOT EXISTS idx_notification_errors_occurred_at ON notification_errors(occurred_at);
CREATE INDEX IF NOT EXISTS idx_notification_errors_resolved ON notification_errors(resolved_at)
WHERE resolved_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_upcoming_notifications_user_id ON upcoming_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_upcoming_notifications_scheduled_for ON upcoming_notifications(scheduled_for);
CREATE INDEX IF NOT EXISTS idx_upcoming_notifications_status ON upcoming_notifications(status);
CREATE INDEX IF NOT EXISTS idx_sent_notifications_user_id ON sent_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_sent_notifications_type ON sent_notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_sent_notifications_sent_at ON sent_notifications(sent_at);
CREATE INDEX IF NOT EXISTS idx_sent_notifications_status ON sent_notifications(status);
-- Generation hints - prioritize generation per user/type when waiting
CREATE TABLE IF NOT EXISTS generation_hints (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL,
    level VARCHAR(10) NOT NULL,
    question_type VARCHAR(50) NOT NULL,
    priority_weight INTEGER NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, language, level, question_type)
);
CREATE INDEX IF NOT EXISTS idx_generation_hints_user_expires ON generation_hints(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_generation_hints_expires ON generation_hints(expires_at);
-- Daily assignments indexes
CREATE INDEX IF NOT EXISTS idx_daily_question_assignments_user_id ON daily_question_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_question_assignments_question_id ON daily_question_assignments(question_id);
CREATE INDEX IF NOT EXISTS idx_daily_question_assignments_date ON daily_question_assignments(assignment_date);
-- Mapping table between daily assignments and canonical user responses
CREATE TABLE IF NOT EXISTS daily_assignment_responses (
    id SERIAL PRIMARY KEY,
    assignment_id INTEGER NOT NULL REFERENCES daily_question_assignments(id) ON DELETE CASCADE,
    user_response_id INTEGER NOT NULL REFERENCES user_responses(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assignment_id),
    UNIQUE(user_response_id)
);
CREATE INDEX IF NOT EXISTS idx_daily_assignment_responses_assignment_id ON daily_assignment_responses(assignment_id);
-- Stories table indexes
CREATE INDEX IF NOT EXISTS idx_stories_user_id ON stories(user_id);
CREATE INDEX IF NOT EXISTS idx_stories_status ON stories(status);
CREATE INDEX IF NOT EXISTS idx_stories_user_current ON stories(user_id, is_current);
CREATE INDEX IF NOT EXISTS idx_stories_user_status ON stories(user_id, status);
-- Story sections table indexes
CREATE INDEX IF NOT EXISTS idx_story_sections_story_id ON story_sections(story_id);
CREATE INDEX IF NOT EXISTS idx_story_sections_number ON story_sections(story_id, section_number);
CREATE INDEX IF NOT EXISTS idx_story_sections_date ON story_sections(generation_date);
-- Story section questions table indexes
CREATE INDEX IF NOT EXISTS idx_story_section_questions_section_id ON story_section_questions(section_id);
-- Create materialized view for analytics performance
CREATE MATERIALIZED VIEW IF NOT EXISTS user_performance_analytics AS
SELECT u.id as user_id,
    u.username,
    COUNT(ur.id) as total_responses,
    AVG(
        CASE
            WHEN ur.is_correct THEN 1.0
            ELSE 0.0
        END
    ) as accuracy_rate,
    AVG(ur.response_time_ms) as avg_response_time,
    COUNT(uqm.question_id) FILTER (
        WHERE uqm.marked_as_known = TRUE
    ) as marked_questions,
    COUNT(
        CASE
            WHEN ur.created_at > NOW() - INTERVAL '7 days' THEN 1
        END
    ) as recent_responses,
    MAX(ur.created_at) as last_response_at
FROM users u
    LEFT JOIN user_responses ur ON u.id = ur.user_id
    LEFT JOIN user_question_metadata uqm ON u.id = uqm.user_id
    AND ur.question_id = uqm.question_id
GROUP BY u.id,
    u.username;
-- Create index on materialized view
CREATE INDEX IF NOT EXISTS idx_user_performance_analytics_user ON user_performance_analytics(user_id);
-- Add column comments for documentation
COMMENT ON COLUMN questions.topic_category IS 'General topic category for question context (e.g., daily_life, travel, work)';
COMMENT ON COLUMN questions.grammar_focus IS 'Grammar focus area for the question (e.g., present_perfect, conditionals)';
COMMENT ON COLUMN questions.vocabulary_domain IS 'Vocabulary domain for the question (e.g., food_and_dining, transportation)';
COMMENT ON COLUMN questions.scenario IS 'Scenario context for the question (e.g., at_the_airport, in_a_restaurant)';
COMMENT ON COLUMN questions.style_modifier IS 'Style modifier for the question (e.g., conversational, formal)';
COMMENT ON COLUMN questions.difficulty_modifier IS 'Difficulty modifier for the question (e.g., basic, intermediate)';
COMMENT ON COLUMN questions.time_context IS 'Time context for the question (e.g., morning_routine, workday)';
-- Insert default worker settings
INSERT INTO worker_settings (setting_key, setting_value)
VALUES ('global_pause', 'false'),
    ('heartbeat_interval_seconds', '30'),
    ('max_heartbeat_age_minutes', '5') ON CONFLICT (setting_key) DO NOTHING;
-- Insert default worker status
INSERT INTO worker_status (worker_instance, is_running, is_paused)
VALUES ('default', false, false) ON CONFLICT (worker_instance) DO NOTHING;
-- Stories table - stores user-created stories with metadata
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

-- Create partial unique index to ensure only one current story per user per language
CREATE UNIQUE INDEX IF NOT EXISTS unique_current_story_per_user_per_language
ON stories (user_id, language)
WHERE is_current = true;

-- Story sections table - stores individual sections of stories
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

-- Story section questions table - stores comprehension questions for each section
CREATE TABLE IF NOT EXISTS story_section_questions (
    id SERIAL PRIMARY KEY,
    section_id INTEGER NOT NULL REFERENCES story_sections(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_answer_index INTEGER NOT NULL CHECK (correct_answer_index >= 0 AND correct_answer_index <= 3),
    explanation TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert default roles and assign them to existing users via migration files
