-- Add feedback_reports table for generic user feedback
CREATE TABLE IF NOT EXISTS feedback_reports (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    feedback_text TEXT NOT NULL,
    feedback_type VARCHAR(50) DEFAULT 'general',
    -- bug, feature_request, general, improvement
    context_data JSONB NOT NULL DEFAULT '{}',
    screenshot_data TEXT,
    screenshot_url TEXT,
    status VARCHAR(50) DEFAULT 'new',
    -- new, in_progress, resolved, dismissed
    admin_notes TEXT,
    assigned_to_user_id INTEGER,
    resolved_at TIMESTAMPTZ,
    resolved_by_user_id INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to_user_id) REFERENCES users(id) ON DELETE
    SET NULL,
        FOREIGN KEY (resolved_by_user_id) REFERENCES users(id) ON DELETE
    SET NULL
);
-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_feedback_reports_user_id ON feedback_reports(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_status ON feedback_reports(status);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_created_at ON feedback_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_type ON feedback_reports(feedback_type);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_context_data ON feedback_reports USING GIN(context_data);
