package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_LoadsFromYAML(t *testing.T) {
	// Create a temporary config file
	tempFile := createTempConfigFile(t, `
server:
  port: "9090"
  worker_port: "9091"
  admin_username: "testadmin"
  admin_password: "testpass"
  session_secret: "test-secret"
  debug: true
  log_level: "debug"
  worker_base_url: "http://test:9091"
  backend_base_url: "http://test:9090"
  app_base_url: "http://test:3000"
  max_ai_concurrent: 20
  max_ai_per_user: 5
  cors_origins:
    - "http://test:3000"
    - "http://test:3001"

database:
  url: "postgres://test:test@localhost:5432/testdb"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: "10m"

open_telemetry:
  endpoint: "test:4317"
  protocol: "http"
  insecure: false
  service_name: "test-service"
  service_version: "test-version"
  enable_tracing: false
  enable_metrics: false
  enable_logging: false
  sampling_rate: 0.5

email:
  enabled: true
  daily_reminder:
    enabled: true
    hour: 10
  smtp:
    host: "smtp.test.com"
    port: 465
    username: "test@test.com"
    password: "testpass"
    from_address: "test@test.com"
    from_name: "Test App"

providers:
  - name: Test Provider
    code: test
    url: "http://test:11434/v1"
    supports_grammar: true
    question_batch_size: 3
    models:
      - name: "Test Model"
        code: "test-model"
        max_tokens: 4096

language_levels:
  testlang:
    levels:
      - "A1"
      - "A2"
    descriptions:
      A1: "Test Beginner"
      A2: "Test Elementary"

system:
  auth:
    signups_disabled: true
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	// Clear any environment variables that might interfere
	envVars := []string{
		"OTEL_ENDPOINT", "OTEL_PROTOCOL", "OTEL_INSECURE", "OTEL_SERVICE_NAME",
		"OTEL_SERVICE_VERSION", "OTEL_ENABLE_TRACING", "OTEL_ENABLE_METRICS",
		"OTEL_ENABLE_LOGGING", "OTEL_SAMPLING_RATE", "OTEL_HEADERS",
		"OPEN_TELEMETRY_ENDPOINT", "OPEN_TELEMETRY_PROTOCOL", "OPEN_TELEMETRY_INSECURE", "OPEN_TELEMETRY_SERVICE_NAME",
		"OPEN_TELEMETRY_SERVICE_VERSION", "OPEN_TELEMETRY_ENABLE_TRACING", "OPEN_TELEMETRY_ENABLE_METRICS",
		"OPEN_TELEMETRY_ENABLE_LOGGING", "OPEN_TELEMETRY_SAMPLING_RATE", "OPEN_TELEMETRY_HEADERS",
		"SERVER_PORT", "SERVER_DEBUG", "DATABASE_URL", "EMAIL_ENABLED", "EMAIL_SMTP_PASSWORD",
	}

	// Store original values and clear them
	originalVars := make(map[string]string)
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			originalVars[envVar] = val
			if err := os.Unsetenv(envVar); err != nil {
				t.Logf("Failed to unset env var %s: %v", envVar, err)
			}
		}
	}

	// Restore original values after test
	defer func() {
		for envVar, val := range originalVars {
			if err := os.Setenv(envVar, val); err != nil {
				t.Logf("Failed to set env var %s: %v", envVar, err)
			}
		}
	}()

	// Set environment variable to use our temp file
	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
	}()

	// Set EMAIL_SMTP_PASSWORD to the expected test value to override any .env file value
	if err := os.Setenv("EMAIL_SMTP_PASSWORD", "testpass"); err != nil {
		t.Fatalf("Failed to set EMAIL_SMTP_PASSWORD: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("EMAIL_SMTP_PASSWORD"); err != nil {
			t.Logf("Failed to unset EMAIL_SMTP_PASSWORD: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test server config
	assert.Equal(t, "9090", config.Server.Port)
	assert.Equal(t, "9091", config.Server.WorkerPort)
	assert.Equal(t, "testadmin", config.Server.AdminUsername)
	assert.Equal(t, "testpass", config.Server.AdminPassword)
	assert.Equal(t, "test-secret", config.Server.SessionSecret)
	assert.True(t, config.Server.Debug)
	assert.Equal(t, "debug", config.Server.LogLevel)
	assert.Equal(t, "http://test:9091", config.Server.WorkerBaseURL)
	assert.Equal(t, "http://test:9090", config.Server.BackendBaseURL)
	assert.Equal(t, "http://test:3000", config.Server.AppBaseURL)
	assert.Equal(t, 20, config.Server.MaxAIConcurrent)
	assert.Equal(t, 5, config.Server.MaxAIPerUser)
	assert.Equal(t, []string{"http://test:3000", "http://test:3001"}, config.Server.CORSOrigins)

	// Test database config
	assert.Equal(t, "postgres://test:test@localhost:5432/testdb", config.Database.URL)
	assert.Equal(t, 50, config.Database.MaxOpenConns)
	assert.Equal(t, 10, config.Database.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, config.Database.ConnMaxLifetime)

	// Test OpenTelemetry config
	assert.Equal(t, "test:4317", config.OpenTelemetry.Endpoint)
	assert.Equal(t, "http", config.OpenTelemetry.Protocol)
	assert.False(t, config.OpenTelemetry.Insecure)
	assert.Equal(t, "test-service", config.OpenTelemetry.ServiceName)
	assert.Equal(t, "test-version", config.OpenTelemetry.ServiceVersion)
	assert.False(t, config.OpenTelemetry.EnableTracing)
	assert.False(t, config.OpenTelemetry.EnableMetrics)
	assert.False(t, config.OpenTelemetry.EnableLogging)
	assert.Equal(t, 0.5, config.OpenTelemetry.SamplingRate)

	// Test email config
	assert.True(t, config.Email.Enabled)
	assert.True(t, config.Email.DailyReminder.Enabled)
	assert.Equal(t, 10, config.Email.DailyReminder.Hour)
	assert.Equal(t, "smtp.test.com", config.Email.SMTP.Host)
	assert.Equal(t, 465, config.Email.SMTP.Port)
	assert.Equal(t, "test@test.com", config.Email.SMTP.Username)
	assert.Equal(t, "testpass", config.Email.SMTP.Password)
	assert.Equal(t, "test@test.com", config.Email.SMTP.FromAddress)
	assert.Equal(t, "Test App", config.Email.SMTP.FromName)

	// Test providers
	require.Len(t, config.Providers, 1)
	assert.Equal(t, "Test Provider", config.Providers[0].Name)
	assert.Equal(t, "test", config.Providers[0].Code)
	assert.Equal(t, "http://test:11434/v1", config.Providers[0].URL)
	assert.True(t, config.Providers[0].SupportsGrammar)
	assert.Equal(t, 3, config.Providers[0].QuestionBatchSize)
	require.Len(t, config.Providers[0].Models, 1)
	assert.Equal(t, "Test Model", config.Providers[0].Models[0].Name)
	assert.Equal(t, "test-model", config.Providers[0].Models[0].Code)
	assert.Equal(t, 4096, config.Providers[0].Models[0].MaxTokens)

	// Test language levels
	require.Contains(t, config.LanguageLevels, "testlang")
	assert.Equal(t, []string{"A1", "A2"}, config.LanguageLevels["testlang"].Levels)
	assert.Equal(t, "Test Beginner", config.LanguageLevels["testlang"].Descriptions["A1"])
	assert.Equal(t, "Test Elementary", config.LanguageLevels["testlang"].Descriptions["A2"])

	// Test system config
	require.NotNil(t, config.System)
	assert.True(t, config.System.Auth.SignupsDisabled)
}

func TestNewConfig_EnvironmentVariableOverrides(t *testing.T) {
	// Create a minimal config file
	tempFile := createTempConfigFile(t, `
server:
  port: "8080"
  debug: false
database:
  url: "postgres://default:default@localhost:5432/defaultdb"
email:
  enabled: false
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	// Set environment variables to override YAML values
	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("SERVER_PORT", "9090"); err != nil {
		t.Fatalf("Failed to set SERVER_PORT: %v", err)
	}
	if err := os.Setenv("SERVER_DEBUG", "true"); err != nil {
		t.Fatalf("Failed to set SERVER_DEBUG: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", "postgres://env:env@localhost:5432/envdb"); err != nil {
		t.Fatalf("Failed to set DATABASE_URL: %v", err)
	}
	if err := os.Setenv("EMAIL_ENABLED", "true"); err != nil {
		t.Fatalf("Failed to set EMAIL_ENABLED: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("SERVER_PORT"); err != nil {
			t.Logf("Failed to unset SERVER_PORT: %v", err)
		}
		if err := os.Unsetenv("SERVER_DEBUG"); err != nil {
			t.Logf("Failed to unset SERVER_DEBUG: %v", err)
		}
		if err := os.Unsetenv("DATABASE_URL"); err != nil {
			t.Logf("Failed to unset DATABASE_URL: %v", err)
		}
		if err := os.Unsetenv("EMAIL_ENABLED"); err != nil {
			t.Logf("Failed to unset EMAIL_ENABLED: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	// Environment variables should override YAML values
	assert.Equal(t, "9090", config.Server.Port)
	assert.True(t, config.Server.Debug)
	assert.Equal(t, "postgres://env:env@localhost:5432/envdb", config.Database.URL)
	assert.True(t, config.Email.Enabled)
}

func TestNewConfig_EnvironmentVariableTypes(t *testing.T) {
	tempFile := createTempConfigFile(t, `
server:
  max_ai_concurrent: 10
  max_ai_per_user: 3
open_telemetry:
  sampling_rate: 1.0
  enable_tracing: true
email:
  daily_reminder:
    hour: 9
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("SERVER_MAX_AI_CONCURRENT", "20"); err != nil {
		t.Fatalf("Failed to set SERVER_MAX_AI_CONCURRENT: %v", err)
	}
	if err := os.Setenv("SERVER_MAX_AI_PER_USER", "5"); err != nil {
		t.Fatalf("Failed to set SERVER_MAX_AI_PER_USER: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_SAMPLING_RATE", "0.5"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_ENABLE_TRACING", "false"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
	}
	if err := os.Setenv("EMAIL_DAILY_REMINDER_HOUR", "12"); err != nil {
		t.Fatalf("Failed to set EMAIL_DAILY_REMINDER_HOUR: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("SERVER_MAX_AI_CONCURRENT"); err != nil {
			t.Logf("Failed to unset SERVER_MAX_AI_CONCURRENT: %v", err)
		}
		if err := os.Unsetenv("SERVER_MAX_AI_PER_USER"); err != nil {
			t.Logf("Failed to unset SERVER_MAX_AI_PER_USER: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_SAMPLING_RATE"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_ENABLE_TRACING"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
		}
		if err := os.Unsetenv("EMAIL_DAILY_REMINDER_HOUR"); err != nil {
			t.Logf("Failed to unset EMAIL_DAILY_REMINDER_HOUR: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	// Test integer overrides
	assert.Equal(t, 20, config.Server.MaxAIConcurrent)
	assert.Equal(t, 5, config.Server.MaxAIPerUser)

	// Test float overrides
	assert.Equal(t, 0.5, config.OpenTelemetry.SamplingRate)

	// Test boolean overrides
	assert.False(t, config.OpenTelemetry.EnableTracing)

	// Test nested struct overrides
	assert.Equal(t, 12, config.Email.DailyReminder.Hour)
}

func TestNewConfig_StringSliceOverride(t *testing.T) {
	tempFile := createTempConfigFile(t, `
server:
  cors_origins:
    - "http://default:3000"
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("SERVER_CORS_ORIGINS", "http://env:3000,http://env:3001,http://env:3002"); err != nil {
		t.Fatalf("Failed to set SERVER_CORS_ORIGINS: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("SERVER_CORS_ORIGINS"); err != nil {
			t.Logf("Failed to unset SERVER_CORS_ORIGINS: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	expected := []string{"http://env:3000", "http://env:3001", "http://env:3002"}
	assert.Equal(t, expected, config.Server.CORSOrigins)
}

func TestNewConfig_InvalidEnvironmentVariable(t *testing.T) {
	tempFile := createTempConfigFile(t, `
server:
  max_ai_concurrent: 10
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("SERVER_MAX_AI_CONCURRENT", "invalid"); err != nil {
		t.Fatalf("Failed to set SERVER_MAX_AI_CONCURRENT: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("SERVER_MAX_AI_CONCURRENT"); err != nil {
			t.Logf("Failed to unset SERVER_MAX_AI_CONCURRENT: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	// Should keep the original YAML value when environment variable is invalid
	assert.Equal(t, 10, config.Server.MaxAIConcurrent)
}

func TestNewConfig_ConfigFileNotFound(t *testing.T) {
	if err := os.Setenv("QUIZ_CONFIG_FILE", "/nonexistent/file.yaml"); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
	}()

	_, err := NewConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config from /nonexistent/file.yaml")
}

func TestNewConfig_LoadsFromEnvironmentVariable(t *testing.T) {
	// The test should use the QUIZ_CONFIG_FILE environment variable set by the task
	// which points to the merged config file
	configFile := os.Getenv("QUIZ_CONFIG_FILE")
	t.Logf("QUIZ_CONFIG_FILE environment variable: %s", configFile)

	// If the environment variable is not set, skip this test
	if configFile == "" {
		t.Skip("QUIZ_CONFIG_FILE environment variable not set, skipping test")
	}

	config, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have default values
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "8081", config.Server.WorkerPort)
	assert.Equal(t, "admin", config.Server.AdminUsername)
	assert.Equal(t, "password", config.Server.AdminPassword)
	assert.Equal(t, "your-secret-key", config.Server.SessionSecret)
	assert.False(t, config.Server.Debug)
	assert.Equal(t, "info", config.Server.LogLevel)
}

func TestConfig_GetLanguages(t *testing.T) {
	config := &Config{
		LanguageLevels: map[string]LanguageLevelConfig{
			"english": {Levels: []string{"A1", "A2"}},
			"spanish": {Levels: []string{"B1", "B2"}},
			"french":  {Levels: []string{"C1", "C2"}},
		},
	}

	languages := config.GetLanguages()
	expected := []string{"english", "french", "spanish"} // Should be sorted
	assert.Equal(t, expected, languages)
}

func TestConfig_GetLevelsForLanguage(t *testing.T) {
	config := &Config{
		LanguageLevels: map[string]LanguageLevelConfig{
			"english": {Levels: []string{"A1", "A2", "B1"}},
		},
	}

	levels := config.GetLevelsForLanguage("english")
	assert.Equal(t, []string{"A1", "A2", "B1"}, levels)

	// Test non-existent language
	levels = config.GetLevelsForLanguage("nonexistent")
	assert.Empty(t, levels)
}

func TestConfig_GetLevelDescriptionsForLanguage(t *testing.T) {
	config := &Config{
		LanguageLevels: map[string]LanguageLevelConfig{
			"english": {
				Levels: []string{"A1", "A2"},
				Descriptions: map[string]string{
					"A1": "Beginner",
					"A2": "Elementary",
				},
			},
		},
	}

	descriptions := config.GetLevelDescriptionsForLanguage("english")
	expected := map[string]string{
		"A1": "Beginner",
		"A2": "Elementary",
	}
	assert.Equal(t, expected, descriptions)

	// Test non-existent language
	descriptions = config.GetLevelDescriptionsForLanguage("nonexistent")
	assert.Empty(t, descriptions)
}

func TestConfig_GetAllLevels(t *testing.T) {
	config := &Config{
		LanguageLevels: map[string]LanguageLevelConfig{
			"english": {Levels: []string{"A1", "A2", "B1"}},
			"spanish": {Levels: []string{"A1", "B1", "C1"}},
		},
	}

	levels := config.GetAllLevels()
	expected := []string{"A1", "A2", "B1", "C1"} // Should be unique and sorted
	assert.Equal(t, expected, levels)
}

func TestConfig_GetAllLevelDescriptions(t *testing.T) {
	config := &Config{
		LanguageLevels: map[string]LanguageLevelConfig{
			"english": {
				Descriptions: map[string]string{
					"A1": "English Beginner",
					"A2": "English Elementary",
				},
			},
			"spanish": {
				Descriptions: map[string]string{
					"A1": "Spanish Beginner",
					"B1": "Spanish Intermediate",
				},
			},
		},
	}

	descriptions := config.GetAllLevelDescriptions()
	// Should merge all descriptions, with the last one encountered winning
	// Since map iteration order is not guaranteed, we need to check the actual result
	assert.Equal(t, "English Elementary", descriptions["A2"])
	assert.Equal(t, "Spanish Intermediate", descriptions["B1"])
	// A1 could be either "English Beginner" or "Spanish Beginner" depending on iteration order
	assert.Contains(t, []string{"English Beginner", "Spanish Beginner"}, descriptions["A1"])
	assert.Len(t, descriptions, 3)
}

func TestConfig_IsSignupDisabled(t *testing.T) {
	// Test with signups disabled
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				SignupsDisabled: true,
			},
		},
	}
	assert.True(t, config.IsSignupDisabled())

	// Test with signups enabled
	config.System.Auth.SignupsDisabled = false
	assert.False(t, config.IsSignupDisabled())

	// Test with no system config
	config.System = nil
	assert.False(t, config.IsSignupDisabled())
}

func TestConfig_IsEmailAllowed(t *testing.T) {
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				AllowedEmails: []string{"admin@example.com", "support@quizapp.com"},
			},
		},
	}

	// Test allowed emails
	assert.True(t, config.IsEmailAllowed("admin@example.com"))
	assert.True(t, config.IsEmailAllowed("ADMIN@EXAMPLE.COM"))
	assert.True(t, config.IsEmailAllowed("  admin@example.com  "))
	assert.True(t, config.IsEmailAllowed("support@quizapp.com"))

	// Test non-allowed emails
	assert.False(t, config.IsEmailAllowed("user@example.com"))
	assert.False(t, config.IsEmailAllowed("admin@other.com"))

	// Test with no allowed emails
	config.System.Auth.AllowedEmails = nil
	assert.False(t, config.IsEmailAllowed("admin@example.com"))

	// Test with no system config
	config.System = nil
	assert.False(t, config.IsEmailAllowed("admin@example.com"))
}

func TestConfig_IsDomainAllowed(t *testing.T) {
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				AllowedDomains: []string{"company.com", "trusted-partner.org"},
			},
		},
	}

	// Test allowed domains
	assert.True(t, config.IsDomainAllowed("company.com"))
	assert.True(t, config.IsDomainAllowed("COMPANY.COM"))
	assert.True(t, config.IsDomainAllowed("  company.com  "))
	assert.True(t, config.IsDomainAllowed("trusted-partner.org"))

	// Test non-allowed domains
	assert.False(t, config.IsDomainAllowed("other.com"))
	assert.False(t, config.IsDomainAllowed("company.org"))

	// Test with no allowed domains
	config.System.Auth.AllowedDomains = nil
	assert.False(t, config.IsDomainAllowed("company.com"))

	// Test with no system config
	config.System = nil
	assert.False(t, config.IsDomainAllowed("company.com"))
}

func TestConfig_IsOAuthSignupAllowed(t *testing.T) {
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com"},
				AllowedEmails:   []string{"admin@example.com"},
			},
		},
	}

	// Test when signups are disabled but email is whitelisted
	assert.True(t, config.IsOAuthSignupAllowed("admin@example.com"))
	assert.True(t, config.IsOAuthSignupAllowed("ADMIN@EXAMPLE.COM"))

	// Test when signups are disabled but domain is whitelisted
	assert.True(t, config.IsOAuthSignupAllowed("user@company.com"))
	assert.True(t, config.IsOAuthSignupAllowed("test@COMPANY.COM"))

	// Test when signups are disabled and email/domain not whitelisted
	assert.False(t, config.IsOAuthSignupAllowed("user@other.com"))
	assert.False(t, config.IsOAuthSignupAllowed("other@example.com"))

	// Test when signups are enabled (should always allow)
	config.System.Auth.SignupsDisabled = false
	assert.True(t, config.IsOAuthSignupAllowed("any@email.com"))
	assert.True(t, config.IsOAuthSignupAllowed("user@other.com"))

	// Test with no system config
	config.System = nil
	assert.False(t, config.IsOAuthSignupAllowed("admin@example.com"))
}

func TestConfig_IsOAuthSignupAllowed_EdgeCases(t *testing.T) {
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com"},
				AllowedEmails:   []string{"admin@example.com"},
			},
		},
	}

	// Test invalid email formats
	assert.False(t, config.IsOAuthSignupAllowed("invalid-email"))
	assert.False(t, config.IsOAuthSignupAllowed("@company.com"))
	assert.False(t, config.IsOAuthSignupAllowed("user@"))

	// Test with empty whitelists (empty slices, not nil)
	config.System.Auth.AllowedDomains = []string{}
	config.System.Auth.AllowedEmails = []string{}
	// Empty slices should still allow the check to proceed, but no matches will be found
	assert.False(t, config.IsOAuthSignupAllowed("user@company.com"))
	assert.False(t, config.IsOAuthSignupAllowed("admin@example.com"))

	// Test with nil whitelists
	config.System.Auth.AllowedDomains = nil
	config.System.Auth.AllowedEmails = nil
	assert.False(t, config.IsOAuthSignupAllowed("user@company.com"))
	assert.False(t, config.IsOAuthSignupAllowed("admin@example.com"))
}

func TestOverrideStructFromEnv_ComplexNestedStruct(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:  "8080",
			Debug: false,
		},
		Database: DatabaseConfig{
			URL:          "postgres://default:default@localhost:5432/defaultdb",
			MaxOpenConns: 25,
		},
		Email: EmailConfig{
			Enabled: false,
			SMTP: SMTPConfig{
				Host: "default.com",
				Port: 587,
			},
			DailyReminder: DailyReminderConfig{
				Enabled: false,
				Hour:    9,
			},
		},
	}

	// Set environment variables
	if err := os.Setenv("SERVER_PORT", "9090"); err != nil {
		t.Fatalf("Failed to set SERVER_PORT: %v", err)
	}
	if err := os.Setenv("SERVER_DEBUG", "true"); err != nil {
		t.Fatalf("Failed to set SERVER_DEBUG: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", "postgres://env:env@localhost:5432/envdb"); err != nil {
		t.Fatalf("Failed to set DATABASE_URL: %v", err)
	}
	if err := os.Setenv("DATABASE_MAX_OPEN_CONNS", "50"); err != nil {
		t.Fatalf("Failed to set DATABASE_MAX_OPEN_CONNS: %v", err)
	}
	if err := os.Setenv("EMAIL_ENABLED", "true"); err != nil {
		t.Fatalf("Failed to set EMAIL_ENABLED: %v", err)
	}
	if err := os.Setenv("EMAIL_SMTP_HOST", "smtp.env.com"); err != nil {
		t.Fatalf("Failed to set EMAIL_SMTP_HOST: %v", err)
	}
	if err := os.Setenv("EMAIL_SMTP_PORT", "465"); err != nil {
		t.Fatalf("Failed to set EMAIL_SMTP_PORT: %v", err)
	}
	if err := os.Setenv("EMAIL_DAILY_REMINDER_ENABLED", "true"); err != nil {
		t.Fatalf("Failed to set EMAIL_DAILY_REMINDER_ENABLED: %v", err)
	}
	if err := os.Setenv("EMAIL_DAILY_REMINDER_HOUR", "12"); err != nil {
		t.Fatalf("Failed to set EMAIL_DAILY_REMINDER_HOUR: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("SERVER_PORT"); err != nil {
			t.Logf("Failed to unset SERVER_PORT: %v", err)
		}
		if err := os.Unsetenv("SERVER_DEBUG"); err != nil {
			t.Logf("Failed to unset SERVER_DEBUG: %v", err)
		}
		if err := os.Unsetenv("DATABASE_URL"); err != nil {
			t.Logf("Failed to unset DATABASE_URL: %v", err)
		}
		if err := os.Unsetenv("DATABASE_MAX_OPEN_CONNS"); err != nil {
			t.Logf("Failed to unset DATABASE_MAX_OPEN_CONNS: %v", err)
		}
		if err := os.Unsetenv("EMAIL_ENABLED"); err != nil {
			t.Logf("Failed to unset EMAIL_ENABLED: %v", err)
		}
		if err := os.Unsetenv("EMAIL_SMTP_HOST"); err != nil {
			t.Logf("Failed to unset EMAIL_SMTP_HOST: %v", err)
		}
		if err := os.Unsetenv("EMAIL_SMTP_PORT"); err != nil {
			t.Logf("Failed to unset EMAIL_SMTP_PORT: %v", err)
		}
		if err := os.Unsetenv("EMAIL_DAILY_REMINDER_ENABLED"); err != nil {
			t.Logf("Failed to unset EMAIL_DAILY_REMINDER_ENABLED: %v", err)
		}
		if err := os.Unsetenv("EMAIL_DAILY_REMINDER_HOUR"); err != nil {
			t.Logf("Failed to unset EMAIL_DAILY_REMINDER_HOUR: %v", err)
		}
	}()

	overrideStructFromEnv(config)

	// Verify all overrides worked
	assert.Equal(t, "9090", config.Server.Port)
	assert.True(t, config.Server.Debug)
	assert.Equal(t, "postgres://env:env@localhost:5432/envdb", config.Database.URL)
	assert.Equal(t, 50, config.Database.MaxOpenConns)
	assert.True(t, config.Email.Enabled)
	assert.Equal(t, "smtp.env.com", config.Email.SMTP.Host)
	assert.Equal(t, 465, config.Email.SMTP.Port)
	assert.True(t, config.Email.DailyReminder.Enabled)
	assert.Equal(t, 12, config.Email.DailyReminder.Hour)
}

func TestOverrideStructFromEnv_InvalidValues(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			MaxAIConcurrent: 10,
			MaxAIPerUser:    3,
		},
		OpenTelemetry: OpenTelemetryConfig{
			SamplingRate:  1.0,
			EnableTracing: true,
		},
	}

	// Set invalid environment variables
	if err := os.Setenv("SERVER_MAX_AI_CONCURRENT", "not-a-number"); err != nil {
		t.Fatalf("Failed to set SERVER_MAX_AI_CONCURRENT: %v", err)
	}
	if err := os.Setenv("SERVER_MAX_AI_PER_USER", "also-not-a-number"); err != nil {
		t.Fatalf("Failed to set SERVER_MAX_AI_PER_USER: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_SAMPLING_RATE", "not-a-float"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_ENABLE_TRACING", "not-a-bool"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("SERVER_MAX_AI_CONCURRENT"); err != nil {
			t.Logf("Failed to unset SERVER_MAX_AI_CONCURRENT: %v", err)
		}
		if err := os.Unsetenv("SERVER_MAX_AI_PER_USER"); err != nil {
			t.Logf("Failed to unset SERVER_MAX_AI_PER_USER: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_SAMPLING_RATE"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_ENABLE_TRACING"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
		}
	}()

	overrideStructFromEnv(config)

	// Should keep original values when environment variables are invalid
	assert.Equal(t, 10, config.Server.MaxAIConcurrent)
	assert.Equal(t, 3, config.Server.MaxAIPerUser)
	assert.Equal(t, 1.0, config.OpenTelemetry.SamplingRate)
	assert.True(t, config.OpenTelemetry.EnableTracing)
}

func TestOverrideStructFromEnv_EmptyValues(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:  "8080",
			Debug: false,
		},
	}

	// Set empty environment variables
	if err := os.Setenv("SERVER_PORT", ""); err != nil {
		t.Fatalf("Failed to set SERVER_PORT: %v", err)
	}
	if err := os.Setenv("SERVER_DEBUG", ""); err != nil {
		t.Fatalf("Failed to set SERVER_DEBUG: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("SERVER_PORT"); err != nil {
			t.Logf("Failed to unset SERVER_PORT: %v", err)
		}
		if err := os.Unsetenv("SERVER_DEBUG"); err != nil {
			t.Logf("Failed to unset SERVER_DEBUG: %v", err)
		}
	}()

	overrideStructFromEnv(config)

	// Should keep original values when environment variables are empty
	assert.Equal(t, "8080", config.Server.Port)
	assert.False(t, config.Server.Debug)
}

func TestOverrideStructFromEnv_NonExistentEnvironmentVariables(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:  "8080",
			Debug: false,
		},
	}

	overrideStructFromEnv(config)

	// Should keep original values when environment variables don't exist
	assert.Equal(t, "8080", config.Server.Port)
	assert.False(t, config.Server.Debug)
}

func TestConfig_OpenTelemetryEnvironmentOverrides(t *testing.T) {
	// Create a minimal config file
	tempFile := createTempConfigFile(t, `
open_telemetry:
  endpoint: "localhost:4317"
  protocol: "grpc"
  insecure: true
  service_name: "test-service"
  enable_tracing: true
  enable_metrics: true
  enable_logging: true
  sampling_rate: 0.5
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	// Set environment variables to override YAML values
	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_ENDPOINT", "otel-collector:4317"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_ENDPOINT: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_PROTOCOL", "http"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_PROTOCOL: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_INSECURE", "false"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_INSECURE: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_SERVICE_NAME", "env-service"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_SERVICE_NAME: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_ENABLE_TRACING", "false"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
	}
	if err := os.Setenv("OPEN_TELEMETRY_SAMPLING_RATE", "0.8"); err != nil {
		t.Fatalf("Failed to set OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_ENDPOINT"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_ENDPOINT: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_PROTOCOL"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_PROTOCOL: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_INSECURE"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_INSECURE: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_SERVICE_NAME"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_SERVICE_NAME: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_ENABLE_TRACING"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_ENABLE_TRACING: %v", err)
		}
		if err := os.Unsetenv("OPEN_TELEMETRY_SAMPLING_RATE"); err != nil {
			t.Logf("Failed to unset OPEN_TELEMETRY_SAMPLING_RATE: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	// Environment variables should override YAML values
	assert.Equal(t, "otel-collector:4317", config.OpenTelemetry.Endpoint)
	assert.Equal(t, "http", config.OpenTelemetry.Protocol)
	assert.False(t, config.OpenTelemetry.Insecure)
	assert.Equal(t, "env-service", config.OpenTelemetry.ServiceName)
	assert.False(t, config.OpenTelemetry.EnableTracing)
	assert.Equal(t, 0.8, config.OpenTelemetry.SamplingRate)

	// Values not overridden by environment should keep YAML values
	assert.True(t, config.OpenTelemetry.EnableMetrics)
	assert.True(t, config.OpenTelemetry.EnableLogging)
}

func TestConfig_OpenTelemetryEnvironmentOverrides_OTEL_Prefix_ShouldNotWork(t *testing.T) {
	// Create a minimal config file
	tempFile := createTempConfigFile(t, `
open_telemetry:
  endpoint: "localhost:4317"
  protocol: "grpc"
  service_name: "test-service"
`)

	defer func() {
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	// Set environment variables with OTEL_ prefix (which should NOT work)
	if err := os.Setenv("QUIZ_CONFIG_FILE", tempFile); err != nil {
		t.Fatalf("Failed to set QUIZ_CONFIG_FILE: %v", err)
	}
	if err := os.Setenv("OTEL_ENDPOINT", "otel-collector:4317"); err != nil {
		t.Fatalf("Failed to set OTEL_ENDPOINT: %v", err)
	}
	if err := os.Setenv("OTEL_PROTOCOL", "http"); err != nil {
		t.Fatalf("Failed to set OTEL_PROTOCOL: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("QUIZ_CONFIG_FILE"); err != nil {
			t.Logf("Failed to unset QUIZ_CONFIG_FILE: %v", err)
		}
		if err := os.Unsetenv("OTEL_ENDPOINT"); err != nil {
			t.Logf("Failed to unset OTEL_ENDPOINT: %v", err)
		}
		if err := os.Unsetenv("OTEL_PROTOCOL"); err != nil {
			t.Logf("Failed to unset OTEL_PROTOCOL: %v", err)
		}
	}()

	config, err := NewConfig()
	require.NoError(t, err)

	// OTEL_ prefixed environment variables should NOT override YAML values
	assert.Equal(t, "localhost:4317", config.OpenTelemetry.Endpoint, "OTEL_ENDPOINT should not override the endpoint")
	assert.Equal(t, "grpc", config.OpenTelemetry.Protocol, "OTEL_PROTOCOL should not override the protocol")
	assert.Equal(t, "test-service", config.OpenTelemetry.ServiceName)
}

func TestConfig_IsOAuthSignupAllowed_InvalidEmail(t *testing.T) {
	config := &Config{
		System: &SystemConfig{
			Auth: AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com"},
				AllowedEmails:   []string{"admin@example.com"},
			},
		},
	}

	// Test that invalid email is blocked
	result := config.IsOAuthSignupAllowed("invalid-email")
	t.Logf("IsOAuthSignupAllowed('invalid-email') = %v", result)
	t.Logf("IsEmailAllowed('invalid-email') = %v", config.IsEmailAllowed("invalid-email"))
	t.Logf("SignupsDisabled = %v", config.IsSignupDisabled())

	assert.False(t, result, "Invalid email should be blocked")
}

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() {
		if err := tempFile.Close(); err != nil {
			t.Logf("Failed to close temp file: %v", err)
		}
	}()

	_, err = tempFile.WriteString(content)
	require.NoError(t, err)

	return tempFile.Name()
}
