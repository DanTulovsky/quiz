// Package config handles application configuration loading from environment variables.
package config

import (
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	contextutils "quizapp/internal/utils"

	"gopkg.in/yaml.v3"
)

// ProviderConfig defines the structure for a single provider
type ProviderConfig struct {
	Name              string    `json:"name" yaml:"name"`
	Code              string    `json:"code" yaml:"code"`
	URL               string    `json:"url,omitempty" yaml:"url,omitempty"`
	SupportsGrammar   bool      `json:"supports_grammar,omitempty" yaml:"supports_grammar,omitempty"`
	QuestionBatchSize int       `json:"question_batch_size,omitempty" yaml:"question_batch_size,omitempty"`
	Models            []AIModel `json:"models" yaml:"models"`
}

// AIModel represents an AI model configuration
type AIModel struct {
	Name      string `json:"name" yaml:"name"`
	Code      string `json:"code" yaml:"code"`
	MaxTokens int    `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
}

// QuestionVarietyConfig defines the variety configuration for question generation
type QuestionVarietyConfig struct {
	TopicCategories     []string            `json:"topic_categories" yaml:"topic_categories"`
	GrammarFocusByLevel map[string][]string `json:"grammar_focus_by_level" yaml:"grammar_focus_by_level"`
	GrammarFocus        []string            `json:"grammar_focus" yaml:"grammar_focus"`
	VocabularyDomains   []string            `json:"vocabulary_domains" yaml:"vocabulary_domains"`
	Scenarios           []string            `json:"scenarios" yaml:"scenarios"`
	StyleModifiers      []string            `json:"style_modifiers" yaml:"style_modifiers"`
	DifficultyModifiers []string            `json:"difficulty_modifiers" yaml:"difficulty_modifiers"`
	TimeContexts        []string            `json:"time_contexts" yaml:"time_contexts"`
}

// LanguageLevelConfig represents the levels and descriptions for a specific language
type LanguageLevelConfig struct {
	Levels       []string          `json:"levels" yaml:"levels"`
	Descriptions map[string]string `json:"descriptions" yaml:"descriptions"`
}

// AuthConfig represents authentication-related configuration
type AuthConfig struct {
	SignupsDisabled bool     `json:"signups_disabled" yaml:"signups_disabled"`
	AllowedDomains  []string `json:"allowed_domains,omitempty" yaml:"allowed_domains,omitempty"`
	AllowedEmails   []string `json:"allowed_emails,omitempty" yaml:"allowed_emails,omitempty"`
}

// SystemConfig represents system-wide configuration
type SystemConfig struct {
	Auth AuthConfig `json:"auth" yaml:"auth"`
}

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server" yaml:"server"`

	// Database configuration
	Database DatabaseConfig `json:"database" yaml:"database"`

	// AI Providers and Language Levels
	Providers      []ProviderConfig               `json:"providers" yaml:"providers"`
	LanguageLevels map[string]LanguageLevelConfig `json:"language_levels" yaml:"language_levels"`
	Variety        *QuestionVarietyConfig         `json:"variety,omitempty" yaml:"variety,omitempty"`
	System         *SystemConfig                  `json:"system,omitempty" yaml:"system,omitempty"`

	// OAuth Configuration
	GoogleOAuthClientID     string `json:"google_oauth_client_id" yaml:"google_oauth_client_id"`
	GoogleOAuthClientSecret string `json:"google_oauth_client_secret" yaml:"google_oauth_client_secret"`
	GoogleOAuthRedirectURL  string `json:"google_oauth_redirect_url" yaml:"google_oauth_redirect_url"`

	// OpenTelemetry Configuration
	OpenTelemetry OpenTelemetryConfig `json:"open_telemetry" yaml:"open_telemetry"`

	// Email Configuration
	Email EmailConfig `json:"email" yaml:"email"`

	// Story Configuration
	Story StoryConfig `json:"story" yaml:"story"`

	// Internal fields
	IsTest bool `json:"is_test" yaml:"is_test"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port                    string   `json:"port" yaml:"port"`
	WorkerPort              string   `json:"worker_port" yaml:"worker_port"`
	AdminUsername           string   `json:"admin_username" yaml:"admin_username"`
	AdminPassword           string   `json:"admin_password" yaml:"admin_password"`
	SessionSecret           string   `json:"session_secret" yaml:"session_secret"`
	Debug                   bool     `json:"debug" yaml:"debug"`
	LogLevel                string   `json:"log_level" yaml:"log_level"`
	WorkerBaseURL           string   `json:"worker_base_url" yaml:"worker_base_url"`
	WorkerInternalURL       string   `json:"worker_internal_url" yaml:"worker_internal_url"`
	BackendBaseURL          string   `json:"backend_base_url" yaml:"backend_base_url"`
	AppBaseURL              string   `json:"app_base_url" yaml:"app_base_url"`
	MaxAIConcurrent         int      `json:"max_ai_concurrent" yaml:"max_ai_concurrent"`
	MaxAIPerUser            int      `json:"max_ai_per_user" yaml:"max_ai_per_user"`
	CORSOrigins             []string `json:"cors_origins" yaml:"cors_origins"`
	QuestionRefillThreshold int      `json:"question_refill_threshold" yaml:"question_refill_threshold"`
	// DailyFreshQuestionRatio controls the minimum fraction of fresh (never-seen)
	// questions to aim for when refilling question pools (0.0 - 1.0). Example: 0.35
	// means at least 35% fresh questions when refilling.
	DailyFreshQuestionRatio float64 `json:"daily_fresh_question_ratio" yaml:"daily_fresh_question_ratio"`
	MaxHistory              int     `json:"max_history" yaml:"max_history"`
	MaxActivityLogs         int     `json:"max_activity_logs" yaml:"max_activity_logs"`
	DailyRepeatAvoidDays    int     `json:"daily_repeat_avoid_days" yaml:"daily_repeat_avoid_days"`
	// DailyHorizonDays controls how many days ahead the worker will assign
	// daily questions (e.g. 0 = today only, 1 = today+1, ...). If unset or
	// <= 0 the worker will fall back to the DAILY_HORIZON_DAYS environment
	// variable (default 1).
	DailyHorizonDays int `json:"daily_horizon_days" yaml:"daily_horizon_days"`
}

// GetLanguages returns a slice of all supported languages (derived from language_levels keys)
func (c *Config) GetLanguages() []string {
	if c.LanguageLevels == nil {
		return []string{}
	}

	languages := make([]string, 0, len(c.LanguageLevels))
	for lang := range c.LanguageLevels {
		languages = append(languages, lang)
	}

	sort.Strings(languages)
	return languages
}

// GetLevelsForLanguage returns the levels for a specific language
func (c *Config) GetLevelsForLanguage(language string) []string {
	if c.LanguageLevels == nil {
		return []string{}
	}

	langConfig, exists := c.LanguageLevels[language]
	if !exists {
		return []string{}
	}

	return langConfig.Levels
}

// GetLevelDescriptionsForLanguage returns the level descriptions for a specific language
func (c *Config) GetLevelDescriptionsForLanguage(language string) map[string]string {
	if c.LanguageLevels == nil {
		return map[string]string{}
	}

	langConfig, exists := c.LanguageLevels[language]
	if !exists {
		return map[string]string{}
	}

	return langConfig.Descriptions
}

// GetAllLevels returns all unique levels across all languages
func (c *Config) GetAllLevels() []string {
	if c.LanguageLevels == nil {
		return []string{}
	}

	levelSet := make(map[string]bool)
	for _, langConfig := range c.LanguageLevels {
		for _, level := range langConfig.Levels {
			levelSet[level] = true
		}
	}

	levels := make([]string, 0, len(levelSet))
	for level := range levelSet {
		levels = append(levels, level)
	}

	sort.Strings(levels)
	return levels
}

// GetAllLevelDescriptions returns all unique level descriptions across all languages
func (c *Config) GetAllLevelDescriptions() map[string]string {
	if c.LanguageLevels == nil {
		return map[string]string{}
	}

	descriptions := make(map[string]string)
	for _, langConfig := range c.LanguageLevels {
		for level, description := range langConfig.Descriptions {
			descriptions[level] = description
		}
	}

	return descriptions
}

// Languages returns all supported languages
func (c *Config) Languages() []string {
	return c.GetLanguages()
}

// Levels returns all unique levels
func (c *Config) Levels() []string {
	return c.GetAllLevels()
}

// LevelDescriptions returns all unique level descriptions
func (c *Config) LevelDescriptions() map[string]string {
	return c.GetAllLevelDescriptions()
}

// IsSignupDisabled returns whether signups are disabled based on configuration
func (c *Config) IsSignupDisabled() bool {
	if c.System == nil {
		return false // Default to enabled if no config
	}
	return c.System.Auth.SignupsDisabled
}

// IsEmailAllowed checks if an email is allowed for OAuth signup override
func (c *Config) IsEmailAllowed(email string) bool {
	if c.System == nil || c.System.Auth.AllowedEmails == nil {
		return false
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	for _, allowedEmail := range c.System.Auth.AllowedEmails {
		if strings.ToLower(strings.TrimSpace(allowedEmail)) == normalizedEmail {
			return true
		}
	}
	return false
}

// IsDomainAllowed checks if a domain is allowed for OAuth signup override
func (c *Config) IsDomainAllowed(domain string) bool {
	if c.System == nil || c.System.Auth.AllowedDomains == nil {
		return false
	}

	normalizedDomain := strings.ToLower(strings.TrimSpace(domain))
	for _, allowedDomain := range c.System.Auth.AllowedDomains {
		if strings.ToLower(strings.TrimSpace(allowedDomain)) == normalizedDomain {
			return true
		}
	}
	return false
}

// IsOAuthSignupAllowed checks if OAuth signup is allowed for a given email
func (c *Config) IsOAuthSignupAllowed(email string) bool {
	if c.System == nil {
		return false
	}

	// If signups are not disabled, OAuth signup is always allowed
	if !c.System.Auth.SignupsDisabled {
		return true
	}

	// If signups are disabled, check whitelist
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	// Use the shared email validation function
	if !contextutils.IsValidEmail(normalizedEmail) {
		return false
	}

	// Check if email is directly whitelisted
	if c.IsEmailAllowed(normalizedEmail) {
		return true
	}

	// Extract domain from email and check if domain is whitelisted
	parts := strings.Split(normalizedEmail, "@")
	domain := parts[1]
	return c.IsDomainAllowed(domain)
}

// OpenTelemetryConfig holds all OpenTelemetry-related configuration
type OpenTelemetryConfig struct {
	Endpoint       string            `json:"endpoint" yaml:"endpoint"`               // Default: "http://localhost:4317"
	Protocol       string            `json:"protocol" yaml:"protocol"`               // "grpc" or "http", default: "grpc"
	Insecure       bool              `json:"insecure" yaml:"insecure"`               // Default: true (for localhost)
	Headers        map[string]string `json:"headers" yaml:"headers"`                 // For authenticated endpoints
	ServiceName    string            `json:"service_name" yaml:"service_name"`       // Default: "quiz-backend" or "quiz-worker"
	ServiceVersion string            `json:"service_version" yaml:"service_version"` // From version package
	EnableTracing  bool              `json:"enable_tracing" yaml:"enable_tracing"`   // Default: true
	EnableMetrics  bool              `json:"enable_metrics" yaml:"enable_metrics"`   // Default: true
	EnableLogging  bool              `json:"enable_logging" yaml:"enable_logging"`   // Default: true (future)
	SamplingRate   float64           `json:"sampling_rate" yaml:"sampling_rate"`     // Default: 1.0 (100%)
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL             string        `json:"url" yaml:"url"`
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`       // Maximum number of open connections to the database
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`       // Maximum number of idle connections in the pool
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"` // Maximum amount of time a connection may be reused
}

// EmailConfig represents email/SMTP configuration
type EmailConfig struct {
	SMTP          SMTPConfig          `json:"smtp" yaml:"smtp"`
	DailyReminder DailyReminderConfig `json:"daily_reminder" yaml:"daily_reminder"`
	Enabled       bool                `json:"enabled" yaml:"enabled"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	FromAddress string `json:"from_address" yaml:"from_address"`
	FromName    string `json:"from_name" yaml:"from_name"`
}

// DailyReminderConfig represents daily reminder email configuration
type DailyReminderConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	Hour    int  `json:"hour" yaml:"hour"` // Hour of day to send (0-23)
}

// StorySectionLengthsConfig represents section length configuration by proficiency level
type StorySectionLengthsConfig struct {
	Beginner          map[string]int                       `json:"beginner" yaml:"beginner"`
	Elementary        map[string]int                       `json:"elementary" yaml:"elementary"`
	Intermediate      map[string]int                       `json:"intermediate" yaml:"intermediate"`
	UpperIntermediate map[string]int                       `json:"upper_intermediate" yaml:"upper_intermediate"`
	Advanced          map[string]int                       `json:"advanced" yaml:"advanced"`
	Proficient        map[string]int                       `json:"proficient" yaml:"proficient"`
	Overrides         map[string]map[string]map[string]int `json:"overrides" yaml:"overrides"`
}

// StoryConfig represents story mode configuration
type StoryConfig struct {
	MaxArchivedPerUser  int                       `json:"max_archived_per_user" yaml:"max_archived_per_user"`
	GenerationEnabled   bool                      `json:"generation_enabled" yaml:"generation_enabled"`
	SectionLengths      StorySectionLengthsConfig `json:"section_lengths" yaml:"section_lengths"`
	QuestionsPerSection int                       `json:"questions_per_section" yaml:"questions_per_section"`
	QuestionsShown      int                       `json:"questions_shown" yaml:"questions_shown"`
}

// NewConfig loads configuration from YAML file first, then overrides with environment variables
func NewConfig() (result0 *Config, err error) {
	// Load config from YAML file
	config, err := loadConfigWithOverrides()
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to load config: %w", err)
	}

	// Override with environment variables
	config.overrideFromEnv()

	return config, nil
}

// overrideFromEnv overrides config values with environment variables using reflection
func (c *Config) overrideFromEnv() {
	overrideStructFromEnv(c)
}

// overrideStructFromEnv recursively overrides struct fields with environment variables
func overrideStructFromEnv(v interface{}) {
	overrideStructFromEnvWithPrefix(v, "")
}

// overrideStructFromEnvWithPrefix recursively overrides struct fields with environment variables
func overrideStructFromEnvWithPrefix(v interface{}, prefix string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the yaml tag for the field
		yamlTag := fieldType.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Convert yaml tag to environment variable name
		envKey := strings.ToUpper(strings.ReplaceAll(yamlTag, "-", "_"))
		if prefix != "" {
			envKey = prefix + "_" + envKey
		}

		switch field.Kind() {
		case reflect.String:
			if envVal := os.Getenv(envKey); envVal != "" {
				field.SetString(envVal)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if envVal := os.Getenv(envKey); envVal != "" {
				if intVal, err := strconv.ParseInt(envVal, 10, 64); err == nil {
					field.SetInt(intVal)
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if envVal := os.Getenv(envKey); envVal != "" {
				if uintVal, err := strconv.ParseUint(envVal, 10, 64); err == nil {
					field.SetUint(uintVal)
				}
			}
		case reflect.Float32, reflect.Float64:
			if envVal := os.Getenv(envKey); envVal != "" {
				if floatVal, err := strconv.ParseFloat(envVal, 64); err == nil {
					field.SetFloat(floatVal)
				}
			}
		case reflect.Bool:
			if envVal := os.Getenv(envKey); envVal != "" {
				if boolVal, err := strconv.ParseBool(envVal); err == nil {
					field.SetBool(boolVal)
				}
			}
		case reflect.Slice:
			if envVal := os.Getenv(envKey); envVal != "" {
				// Handle string slices (like CORS_ORIGINS)
				if field.Type().Elem().Kind() == reflect.String {
					slice := strings.Split(envVal, ",")
					field.Set(reflect.ValueOf(slice))
				}
			}
		case reflect.Struct:
			// Recursively process nested structs with the field name as prefix
			if field.CanAddr() {
				fieldPrefix := strings.ToUpper(strings.ReplaceAll(yamlTag, "-", "_"))
				if prefix != "" {
					fieldPrefix = prefix + "_" + fieldPrefix
				}
				overrideStructFromEnvWithPrefix(field.Addr().Interface(), fieldPrefix)
			}
		case reflect.Ptr:
			// Handle pointer to struct
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				fieldPrefix := strings.ToUpper(strings.ReplaceAll(yamlTag, "-", "_"))
				if prefix != "" {
					fieldPrefix = prefix + "_" + fieldPrefix
				}
				overrideStructFromEnvWithPrefix(field.Interface(), fieldPrefix)
			}
		}
	}
}

// loadConfigWithOverrides loads the config file with potential local overrides
func loadConfigWithOverrides() (result0 *Config, err error) {
	// Try to load from environment variable first
	if envPath := os.Getenv("QUIZ_CONFIG_FILE"); envPath != "" {
		config, err := loadConfigFromFile(envPath)
		if err != nil {
			return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to load config from %s: %w", envPath, err)
		}
		return config, nil
	}

	// If no environment variable is set, try default config.yaml
	return loadConfigFromFile("config.yaml")
}

// loadConfigFromFile loads configuration from a specific file
func loadConfigFromFile(path string) (result0 *Config, err error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
