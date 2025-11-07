//go:build integration

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreEnvironment restores the environment to its original state for tests
func restoreEnvironment(originalEnv []string) {
	// Clear all environment variables
	for _, env := range os.Environ() {
		if pair := strings.SplitN(env, "=", 2); len(pair) == 2 {
			_ = os.Unsetenv(pair[0])
		}
	}

	// Restore original environment
	for _, env := range originalEnv {
		if pair := strings.SplitN(env, "=", 2); len(pair) == 2 {
			_ = os.Setenv(pair[0], pair[1])
		}
	}
}

func TestNewConfig_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Set up test environment
	_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/testdb")
	_ = os.Setenv("SERVER_SESSION_SECRET", "test-secret-key")
	_ = os.Setenv("OPENAI_API_KEY", "test-openai-key")
	_ = os.Setenv("SERVER_PORT", "8080")

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "postgres://test:test@localhost:5432/testdb", cfg.Database.URL)
	assert.Equal(t, "test-secret-key", cfg.Server.SessionSecret)
	// OpenAI API key is not stored in config anymore - it's handled per user
	assert.Equal(t, "8080", cfg.Server.Port)
}

func TestNewConfig_Defaults_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Clear relevant environment variables
	envVars := []string{
		"DATABASE_URL", "SESSION_SECRET", "OPENAI_API_KEY", "PORT",
		"OPENAI_MODEL", "OPENAI_MAX_TOKENS", "AI_ENABLED", "QUESTION_CACHE_SIZE",
		"CORS_ORIGINS",
	}
	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults
	assert.Equal(t, "8080", cfg.Server.Port)
	// OpenAI model is not stored in config anymore - it's handled per user
	assert.Contains(t, cfg.Server.CORSOrigins, "http://localhost:3000")
}

func TestGetLanguages_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	languages := cfg.GetLanguages()
	assert.NotEmpty(t, languages)

	// Check that common languages are present
	expectedLanguages := []string{"italian", "french", "german", "japanese", "chinese", "russian"}
	for _, lang := range expectedLanguages {
		assert.Contains(t, languages, lang, "Language %s should be available", lang)
	}
}

func TestGetLevelsForLanguage_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	// Test Italian levels (CEFR)
	italianLevels := cfg.GetLevelsForLanguage("italian")
	assert.NotEmpty(t, italianLevels)
	expectedItalianLevels := []string{"A1", "A2", "B1", "B1+", "B1++", "B2", "C1", "C2"}
	for _, level := range expectedItalianLevels {
		assert.Contains(t, italianLevels, level, "Italian level %s should be available", level)
	}

	// Test Japanese levels (JLPT)
	japaneseLevels := cfg.GetLevelsForLanguage("japanese")
	assert.NotEmpty(t, japaneseLevels)
	expectedJapaneseLevels := []string{"N5", "N4", "N3", "N2", "N1"}
	for _, level := range expectedJapaneseLevels {
		assert.Contains(t, japaneseLevels, level, "Japanese level %s should be available", level)
	}

	// Test Chinese levels (HSK)
	chineseLevels := cfg.GetLevelsForLanguage("chinese")
	assert.NotEmpty(t, chineseLevels)
	expectedChineseLevels := []string{"HSK1", "HSK2", "HSK3", "HSK4", "HSK5", "HSK6"}
	for _, level := range expectedChineseLevels {
		assert.Contains(t, chineseLevels, level, "Chinese level %s should be available", level)
	}

	// Test unknown language
	unknownLevels := cfg.GetLevelsForLanguage("unknown")
	assert.Empty(t, unknownLevels)

	// Test language code lookups (should work the same as name lookups)
	italianLevelsByCode := cfg.GetLevelsForLanguage("it")
	assert.NotEmpty(t, italianLevelsByCode)
	assert.Equal(t, italianLevels, italianLevelsByCode, "Italian levels should be the same when looked up by code or name")

	japaneseLevelsByCode := cfg.GetLevelsForLanguage("ja")
	assert.NotEmpty(t, japaneseLevelsByCode)
	assert.Equal(t, japaneseLevels, japaneseLevelsByCode, "Japanese levels should be the same when looked up by code or name")

	chineseLevelsByCode := cfg.GetLevelsForLanguage("zh")
	assert.NotEmpty(t, chineseLevelsByCode)
	assert.Equal(t, chineseLevels, chineseLevelsByCode, "Chinese levels should be the same when looked up by code or name")
}

func TestGetLevelDescriptionsForLanguage_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	// Test Italian descriptions
	italianDescs := cfg.GetLevelDescriptionsForLanguage("italian")
	assert.NotEmpty(t, italianDescs)
	assert.Contains(t, italianDescs, "A1")
	assert.Contains(t, italianDescs, "C2")

	// Verify descriptions contain meaningful text
	a1Desc := italianDescs["A1"]
	assert.NotEmpty(t, a1Desc)
	assert.Contains(t, strings.ToLower(a1Desc), "beginner")

	// Test Japanese descriptions
	japaneseDescs := cfg.GetLevelDescriptionsForLanguage("japanese")
	assert.NotEmpty(t, japaneseDescs)
	assert.Contains(t, japaneseDescs, "N5")
	assert.Contains(t, japaneseDescs, "N1")

	// Test unknown language
	unknownDescs := cfg.GetLevelDescriptionsForLanguage("unknown")
	assert.Empty(t, unknownDescs)

	// Test language code lookups (should work the same as name lookups)
	italianDescsByCode := cfg.GetLevelDescriptionsForLanguage("it")
	assert.NotEmpty(t, italianDescsByCode)
	assert.Equal(t, italianDescs, italianDescsByCode, "Italian descriptions should be the same when looked up by code or name")

	japaneseDescsByCode := cfg.GetLevelDescriptionsForLanguage("ja")
	assert.NotEmpty(t, japaneseDescsByCode)
	assert.Equal(t, japaneseDescs, japaneseDescsByCode, "Japanese descriptions should be the same when looked up by code or name")
}

func TestGetAllLevels_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	allLevels := cfg.GetAllLevels()
	assert.NotEmpty(t, allLevels)

	// Should contain levels from different systems
	assert.Contains(t, allLevels, "A1")   // CEFR
	assert.Contains(t, allLevels, "N5")   // JLPT
	assert.Contains(t, allLevels, "HSK1") // HSK
}

func TestGetAllLevelDescriptions_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	allDescs := cfg.GetAllLevelDescriptions()
	assert.NotEmpty(t, allDescs)

	// Should contain descriptions from different systems
	assert.Contains(t, allDescs, "A1")
	assert.Contains(t, allDescs, "N5")
	assert.Contains(t, allDescs, "HSK1")

	// Verify descriptions are meaningful
	a1Desc := allDescs["A1"]
	assert.NotEmpty(t, a1Desc)
	assert.Contains(t, strings.ToLower(a1Desc), "beginner")
}

func TestLanguages_Property_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	languages := cfg.Languages()
	assert.NotEmpty(t, languages)
	assert.IsType(t, []string{}, languages)
}

func TestLevels_Property_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	levels := cfg.Levels()
	assert.NotEmpty(t, levels)
	assert.IsType(t, []string{}, levels)
}

func TestLevelDescriptions_Property_Integration(t *testing.T) {
	cfg, err := NewConfig()
	require.NoError(t, err)

	levelDescs := cfg.LevelDescriptions()
	assert.NotEmpty(t, levelDescs)
	assert.IsType(t, map[string]string{}, levelDescs)
}

// TestLoadAppConfigFromEnv_Integration tests loading config from merged.config.yaml
func TestLoadAppConfigFromEnv_Integration(t *testing.T) {
	tempDir := t.TempDir()
	configContent := `providers:
  - name: "Test Provider"
    code: "test"
    url: "http://test.com"
    supports_grammar: true
language_levels:
  english:
    levels: ["A1", "A2"]
    descriptions:
      A1: "Beginner"
      A2: "Elementary"
system:
  auth:
    signups_disabled: false
`
	configPath := filepath.Join(tempDir, "merged.config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Change to temp directory so the config loader can find the merged config
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	_ = os.Chdir(tempDir)

	// Set QUIZ_CONFIG_FILE so loadAppConfigFromEnv uses our test config
	originalEnv := os.Getenv("QUIZ_CONFIG_FILE")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("QUIZ_CONFIG_FILE", originalEnv)
		} else {
			_ = os.Unsetenv("QUIZ_CONFIG_FILE")
		}
	}()
	_ = os.Setenv("QUIZ_CONFIG_FILE", configPath)

	config, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Len(t, config.Providers, 1)
	provider := config.Providers[0]
	assert.Equal(t, "Test Provider", provider.Name)
	assert.Equal(t, "test", provider.Code)
	assert.Equal(t, "http://test.com", provider.URL)
	assert.True(t, provider.SupportsGrammar)
	assert.Contains(t, config.LanguageLevels, "english")
	englishLevels := config.LanguageLevels["english"]
	assert.Equal(t, []string{"A1", "A2"}, englishLevels.Levels)
	assert.Equal(t, "Beginner", englishLevels.Descriptions["A1"])
}

func TestLoadAppConfig_CustomFile_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Create temporary merged config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "merged.config.yaml")

	customConfig := `providers:
  - name: "Test Provider"
    code: "test"
    models:
      - name: "Test Model"
        code: "test-model"
`

	err := os.WriteFile(configFile, []byte(customConfig), 0o644)
	require.NoError(t, err)

	// Change to temp directory so the config loader can find the merged config
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	_ = os.Chdir(tempDir)

	// Set QUIZ_CONFIG_FILE so NewConfig uses our test config
	originalQuizConfig := os.Getenv("QUIZ_CONFIG_FILE")
	defer func() {
		if originalQuizConfig != "" {
			_ = os.Setenv("QUIZ_CONFIG_FILE", originalQuizConfig)
		} else {
			_ = os.Unsetenv("QUIZ_CONFIG_FILE")
		}
	}()
	_ = os.Setenv("QUIZ_CONFIG_FILE", configFile)

	cfg, err := NewConfig()
	require.NoError(t, err)

	// Verify custom config was loaded
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, "Test Provider", cfg.Providers[0].Name)
	assert.Equal(t, "test", cfg.Providers[0].Code)
	assert.Len(t, cfg.Providers[0].Models, 1)
	assert.Equal(t, "Test Model", cfg.Providers[0].Models[0].Name)
}

func TestConfig_EnvironmentOverrides_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Create a temporary merged config file for the test
	tempDir := t.TempDir()
	configContent := `providers:
  - name: "Test Provider"
    code: "test"
    url: "http://test.com"
    supports_grammar: true
language_levels:
  english:
    levels: ["A1", "A2"]
    descriptions:
      A1: "Beginner"
      A2: "Elementary"
system:
  auth:
    signups_disabled: false
`
	configPath := filepath.Join(tempDir, "merged.config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Change to temp directory so the config loader can find the merged config
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	_ = os.Chdir(tempDir)

	// Set comprehensive environment variables
	envVars := map[string]string{
		"DATABASE_URL":          "postgres://env:env@localhost:5432/envdb",
		"SERVER_SESSION_SECRET": "env-session-secret",
		"SERVER_PORT":           "9000",
		"SERVER_CORS_ORIGINS":   "https://prod.example.com,https://api.example.com",
	}

	for key, value := range envVars {
		_ = os.Setenv(key, value)
	}

	cfg, err := NewConfig()
	require.NoError(t, err)

	// Verify all environment variables are respected
	assert.Equal(t, "postgres://env:env@localhost:5432/envdb", cfg.Database.URL)
	assert.Equal(t, "env-session-secret", cfg.Server.SessionSecret)
	// OpenAI API key and model are not stored in config anymore - they're handled per user
	assert.Equal(t, "9000", cfg.Server.Port)

	expectedOrigins := []string{"https://prod.example.com", "https://api.example.com"}
	assert.Equal(t, expectedOrigins, cfg.Server.CORSOrigins)
}

func TestConfig_MissingAppConfigFile_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Set QUIZ_CONFIG_FILE to a non-existent file
	originalQuizConfig := os.Getenv("QUIZ_CONFIG_FILE")
	defer func() {
		if originalQuizConfig != "" {
			_ = os.Setenv("QUIZ_CONFIG_FILE", originalQuizConfig)
		} else {
			_ = os.Unsetenv("QUIZ_CONFIG_FILE")
		}
	}()
	_ = os.Setenv("QUIZ_CONFIG_FILE", "/non/existent/merged.config.yaml")

	// Should fail when no config file is found
	_, err := NewConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config from /non/existent/merged.config.yaml")
}

func TestConfig_EmptySessionSecret_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Create a temporary merged config file for the test
	tempDir := t.TempDir()
	configContent := `providers:
  - name: "Test Provider"
    code: "test"
    url: "http://test.com"
    supports_grammar: true
language_levels:
  english:
    levels: ["A1", "A2"]
    descriptions:
      A1: "Beginner"
      A2: "Elementary"
system:
  auth:
    signups_disabled: false
`
	configPath := filepath.Join(tempDir, "merged.config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Change to temp directory so the config loader can find the merged config
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	_ = os.Chdir(tempDir)

	// Clear session secret
	_ = os.Unsetenv("SESSION_SECRET")

	cfg, err := NewConfig()
	require.NoError(t, err)

	// Should have a default session secret (even if empty)
	assert.NotNil(t, cfg)
	// Session secret might be empty in test environment, which is okay
}

func TestConfig_LocalOverride_Integration(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer restoreEnvironment(originalEnv)

	// Create a temporary merged config file that simulates the result of merging
	// main config (signups_disabled: true) with local config (signups_disabled: false)
	tempDir := t.TempDir()
	mergedConfigContent := `providers:
  - name: "Test Provider"
    code: "test"
    url: "http://test.com"
    supports_grammar: true
language_levels:
  english:
    levels: ["A1", "A2"]
    descriptions:
      A1: "Beginner"
      A2: "Elementary"
system:
  auth:
    signups_disabled: false
`
	configPath := filepath.Join(tempDir, "merged.config.yaml")
	err := os.WriteFile(configPath, []byte(mergedConfigContent), 0o644)
	require.NoError(t, err)

	// Change to temp directory so the config loader can find the merged config
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	_ = os.Chdir(tempDir)

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.System)
	assert.False(t, cfg.System.Auth.SignupsDisabled, "Merged config should have signups_disabled set to false")
}
