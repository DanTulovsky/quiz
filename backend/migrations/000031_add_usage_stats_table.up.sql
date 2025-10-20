-- Add usage_stats table for tracking translation service usage and quotas
CREATE TABLE IF NOT EXISTS usage_stats (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL, -- e.g., 'google', 'azure', 'aws'
    usage_type VARCHAR(50) NOT NULL,    -- e.g., 'translation', 'tts', 'ai_generation'
    usage_month DATE NOT NULL,          -- First day of the month (YYYY-MM-01)
    characters_used INTEGER NOT NULL DEFAULT 0,
    requests_made INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    -- Ensure one record per service per usage type per month
    UNIQUE(service_name, usage_type, usage_month)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_usage_stats_service_month ON usage_stats(service_name, usage_month);
CREATE INDEX IF NOT EXISTS idx_usage_stats_usage_type ON usage_stats(usage_type);
CREATE INDEX IF NOT EXISTS idx_usage_stats_month ON usage_stats(usage_month);
CREATE INDEX IF NOT EXISTS idx_usage_stats_created_at ON usage_stats(created_at);

-- Add comments for documentation
COMMENT ON TABLE usage_stats IS 'Tracks usage statistics for various services (translation, TTS, AI) by month';
COMMENT ON COLUMN usage_stats.service_name IS 'Name of the service provider (google, azure, etc.)';
COMMENT ON COLUMN usage_stats.usage_type IS 'Type of usage being tracked (translation, tts, ai_generation)';
COMMENT ON COLUMN usage_stats.usage_month IS 'First day of the month for which usage is being tracked';
COMMENT ON COLUMN usage_stats.characters_used IS 'Total characters processed by the service in this month';
COMMENT ON COLUMN usage_stats.requests_made IS 'Total number of requests made to the service in this month';
