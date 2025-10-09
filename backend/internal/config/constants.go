package config

import "time"

// Timeout constants
const (
	// HTTP timeouts
	DefaultHTTPTimeout    = 60 * time.Second
	OAuthHTTPTimeout      = 10 * time.Second
	AIRequestTimeout      = 3 * time.Minute
	AIShutdownTimeout     = 30 * time.Second
	WorkerShutdownTimeout = 30 * time.Second
	CLIWorkerTimeout      = 10 * time.Minute
	TestTimeout           = 100 * time.Millisecond
	AITestTimeout         = 1 * time.Second

	// Database timeouts
	DatabaseConnMaxLifetime = 5 * time.Minute

	// Session timeouts
	SessionMaxAge = 7 * 24 * time.Hour // 7 days

	// Worker timeouts
	WorkerCheckInterval     = 15 * time.Second
	WorkerHeartbeatInterval = 30 * time.Second
	WorkerTriggerThrottle   = 30 * time.Second
	WorkerSleepDuration     = 100 * time.Millisecond

	// Quiz timeouts
	QuizStreamTimeout = 60 * time.Second

	// Retry timeouts
	UserRetryBackoffBase = 1 * time.Hour
)

// Batch and size constants
const (
	// AI batch sizes (these should be configurable per provider)
	DefaultAIBatchSize = 3
)

// Session configuration constants
const (
	// Session settings
	SessionPath     = "/"
	SessionHTTPOnly = true
	SessionSecure   = false // Set to true in production with HTTPS

	// Session name
	SessionName = "quiz-session"
)

// Security configuration constants
const (
	// Content Security Policy
	DefaultCSP = "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; media-src 'self' blob: data:;"
)

// AI service constants
const (
	// Polling intervals
	AIShutdownPollInterval = 100 * time.Millisecond
)

// Logging constants
const (

	// Log prefixes
	NoActionPrefix = "NOACTION:"
)
