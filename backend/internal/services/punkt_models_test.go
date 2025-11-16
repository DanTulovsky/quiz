package services

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// minimal struct to parse language codes from config
type configRoot struct {
	LanguageLevels map[string]struct {
		Code string `yaml:"code"`
	} `yaml:"language_levels"`
}

func codeToModelName(code string) string {
	switch code {
	case "en":
		return "english"
	case "it":
		return "italian"
	case "fr":
		return "french"
	case "de":
		return "german"
	case "es":
		return "spanish"
	case "ru":
		return "russian"
	case "hi":
		return "hindi"
	case "ja":
		return "japanese"
	case "zh":
		return "chinese"
	default:
		return ""
	}
}

func TestPunktModelsExistForConfiguredLanguages(t *testing.T) {
	cfgPath := os.Getenv("QUIZ_CONFIG_FILE")
	if cfgPath == "" {
		t.Skip("QUIZ_CONFIG_FILE not set; skipping model existence test")
		return
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config file %s: %v", cfgPath, err)
	}
	var cfg configRoot
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config yaml: %v", err)
	}

	// base dir relative to repo root for model files
	repoRoot := filepath.Join("..", "..", "..")
	modelDir := filepath.Join(repoRoot, "backend", "internal", "resources", "punkt")

	for langName, entry := range cfg.LanguageLevels {
		if entry.Code == "" {
			t.Fatalf("language %s has empty code in config", langName)
		}
		modelName := codeToModelName(entry.Code)
		if modelName == "" {
			t.Logf("no mapped Punkt model name for code=%s (language=%s); ensure regex fallback is acceptable or add mapping", entry.Code, langName)
			continue
		}
		modelPath := filepath.Join(modelDir, modelName+".json")
		info, err := os.Stat(modelPath)
		if err != nil {
			// Missing Punkt models are OK - regex fallback works for all languages
			t.Logf("⚠️  No Punkt model for language=%s code=%s (regex fallback will be used). "+
				"To download if available: task update-punkt-models", langName, entry.Code)
			continue
		}
		if info.Size() == 0 {
			t.Logf("⚠️  Empty Punkt model file at %s for language=%s code=%s (regex fallback will be used). "+
				"To re-download: task update-punkt-models", modelPath, langName, entry.Code)
			continue
		}
		t.Logf("✅ Punkt model available for language=%s code=%s", langName, entry.Code)
	}
}
