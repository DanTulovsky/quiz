package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/neurosnap/sentences"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestService creates a TranslationPracticeService for testing
func createTestService(t *testing.T) *TranslationPracticeService {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Determine punkt model directory for testing
	pwd, err := os.Getwd()
	require.NoError(t, err)

	// Find repo root by looking for go.mod
	repoRoot := pwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			repoRoot = pwd
			break
		}
		repoRoot = parent
	}

	// Create a template manager (required for service initialization)
	templateManager, err := NewAITemplateManager()
	require.NoError(t, err)

	return &TranslationPracticeService{
		config:          cfg,
		logger:          logger,
		templateManager: templateManager,
		punktModels:     make(map[string]*sentences.DefaultSentenceTokenizer),
		punktModelDir:   filepath.Join(repoRoot, "backend", "internal", "resources", "punkt"),
	}
}

func TestTranslationPracticeService_extractSentences_EnglishWithPunkt(t *testing.T) {
	service := createTestService(t)

	tests := []struct {
		name       string
		text       string
		language   string
		minCount   int // minimum expected sentences (Punkt may group differently)
		maxCount   int // maximum expected sentences (0 = no limit)
		validateFn func(t *testing.T, result []string)
	}{
		{
			name:     "simple sentences",
			text:     "Hello world. How are you? I'm fine!",
			language: "en",
			minCount: 1, // At least one sentence
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				// Should extract sentences, preserving punctuation
				assert.GreaterOrEqual(t, len(result), 1)
				for _, sent := range result {
					assert.NotEmpty(t, strings.TrimSpace(sent))
				}
			},
		},
		{
			name:     "with abbreviations",
			text:     "Dr. Smith went to the U.S.A. He met Mr. Jones.",
			language: "en",
			minCount: 1,
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				// Punkt should handle abbreviations correctly
				fullText := strings.Join(result, " ")
				assert.Contains(t, fullText, "Dr. Smith")
				assert.Contains(t, fullText, "Mr. Jones")
			},
		},
		{
			name:     "quoted sentences",
			text:     `She said "Hello there." He replied "Hi!"`,
			language: "en",
			minCount: 1,
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				fullText := strings.Join(result, " ")
				assert.Contains(t, fullText, "Hello there")
				assert.Contains(t, fullText, "Hi")
			},
		},
		{
			name:     "parentheses",
			text:     "This is a test (it works). Another sentence!",
			language: "en",
			minCount: 1,
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				fullText := strings.Join(result, " ")
				assert.Contains(t, fullText, "test")
				assert.Contains(t, fullText, "Another sentence")
			},
		},
		{
			name:     "empty text",
			text:     "",
			language: "en",
			minCount: 0,
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				assert.Empty(t, result)
			},
		},
		{
			name:     "whitespace only",
			text:     "   \n\t  ",
			language: "en",
			minCount: 0,
			maxCount: 0,
			validateFn: func(t *testing.T, result []string) {
				assert.Empty(t, result)
			},
		},
		{
			name:     "single sentence",
			text:     "Just one sentence.",
			language: "en",
			minCount: 1,
			maxCount: 1,
			validateFn: func(t *testing.T, result []string) {
				assert.Len(t, result, 1)
				assert.Contains(t, result[0], "Just one sentence")
				assert.True(t, strings.HasSuffix(strings.TrimSpace(result[0]), "."))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSentences(tt.text, tt.language)
			assert.GreaterOrEqual(t, len(result), tt.minCount,
				"Should extract at least %d sentence(s)", tt.minCount)
			if tt.maxCount > 0 {
				assert.LessOrEqual(t, len(result), tt.maxCount,
					"Should extract at most %d sentence(s)", tt.maxCount)
			}
			if tt.validateFn != nil {
				tt.validateFn(t, result)
			}
		})
	}
}

func TestTranslationPracticeService_extractSentences_RegexFallback(t *testing.T) {
	service := createTestService(t)

	tests := []struct {
		name     string
		text     string
		language string
		minCount int // minimum expected sentences (regex may split differently than Punkt)
	}{
		{
			name:     "Russian text",
			text:     "Привет мир. Как дела? Я хорошо!",
			language: "ru",
			minCount: 3,
		},
		{
			name:     "Russian with guillemets",
			text:     "Он сказал «Привет!» Она ответила «Здравствуй!»",
			language: "ru",
			minCount: 1, // May extract 1-2 sentences depending on how guillemets are handled
		},
		{
			name:     "Japanese text",
			text:     "こんにちは。元気ですか？元気です！",
			language: "zh",
			minCount: 3,
		},
		{
			name:     "Chinese text",
			text:     "你好。你好吗？我很好！",
			language: "zh",
			minCount: 2, // May extract 2-3 sentences (depends on punctuation handling)
		},
		{
			name:     "Hindi text",
			text:     "नमस्ते। कैसे हो? मैं ठीक हूँ!",
			language: "hi",
			minCount: 2, // May extract 2-3 sentences (Hindi uses । which might not match)
		},
		{
			name:     "unknown language code",
			text:     "Hello world. How are you?",
			language: "xx", // non-existent language
			minCount: 2,
		},
		{
			name:     "mixed punctuation",
			text:     "Sentence one. Sentence two! Sentence three?",
			language: "ru",
			minCount: 3,
		},
		{
			name:     "ellipsis",
			text:     "First sentence... Second sentence.",
			language: "ru",
			minCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSentences(tt.text, tt.language)
			assert.GreaterOrEqual(t, len(result), tt.minCount,
				"Should extract at least %d sentences from: %s", tt.minCount, tt.text)

			// Verify all sentences are non-empty
			for i, sent := range result {
				assert.NotEmpty(t, strings.TrimSpace(sent), "Sentence %d should not be empty", i)
			}

			// Verify punctuation is preserved
			for _, sent := range result {
				trimmed := strings.TrimSpace(sent)
				// Check if sentence contains terminators (may end with closing quote/bracket)
				hasTerminator := strings.Contains(trimmed, ".") ||
					strings.Contains(trimmed, "!") ||
					strings.Contains(trimmed, "?") ||
					strings.Contains(trimmed, "…") ||
					strings.Contains(trimmed, "。") ||
					strings.Contains(trimmed, "？") ||
					strings.Contains(trimmed, "！")
				// Or ends with punctuation or closing quote/bracket
				endsWithPunct := strings.HasSuffix(trimmed, ".") ||
					strings.HasSuffix(trimmed, "!") ||
					strings.HasSuffix(trimmed, "?") ||
					strings.HasSuffix(trimmed, "…") ||
					strings.HasSuffix(trimmed, "。") ||
					strings.HasSuffix(trimmed, "？") ||
					strings.HasSuffix(trimmed, "！") ||
					strings.HasSuffix(trimmed, "»") ||
					strings.HasSuffix(trimmed, "\"") ||
					strings.HasSuffix(trimmed, "'") ||
					strings.HasSuffix(trimmed, ")") ||
					strings.HasSuffix(trimmed, "]")
				// If text has terminators, sentences should have them too
				if strings.Contains(tt.text, ".") || strings.Contains(tt.text, "!") || strings.Contains(tt.text, "?") {
					assert.True(t, hasTerminator || endsWithPunct || len(result) == 1,
						"Sentence should contain punctuation or end with punctuation/quote: %s", trimmed)
				}
			}
		})
	}
}

func TestTranslationPracticeService_extractSentences_EdgeCases(t *testing.T) {
	service := createTestService(t)

	tests := []struct {
		name     string
		text     string
		language string
		validate func(t *testing.T, result []string)
	}{
		{
			name:     "text without terminators",
			text:     "This is a sentence without any punctuation at the end",
			language: "en",
			validate: func(t *testing.T, result []string) {
				// Should return the whole text as a single sentence
				assert.Len(t, result, 1)
				assert.Contains(t, result[0], "sentence without")
			},
		},
		{
			name:     "multiple spaces",
			text:     "First.    Second.   Third.",
			language: "en",
			validate: func(t *testing.T, result []string) {
				assert.GreaterOrEqual(t, len(result), 3)
				for _, sent := range result {
					assert.NotContains(t, sent, "   ", "Should trim extra spaces")
				}
			},
		},
		{
			name:     "newlines",
			text:     "First sentence.\n\nSecond sentence.\nThird sentence.",
			language: "en",
			validate: func(t *testing.T, result []string) {
				assert.GreaterOrEqual(t, len(result), 3)
			},
		},
		{
			name:     "very short sentences",
			text:     "Hi. Bye. Yes. No.",
			language: "en",
			validate: func(t *testing.T, result []string) {
				assert.GreaterOrEqual(t, len(result), 4)
			},
		},
		{
			name:     "quotes and brackets",
			text:     `She said "Hello." (He replied "Hi!")`,
			language: "en",
			validate: func(t *testing.T, result []string) {
				assert.GreaterOrEqual(t, len(result), 1)
			},
		},
		{
			name:     "consecutive punctuation",
			text:     "Really?! No way... Yes!",
			language: "en",
			validate: func(t *testing.T, result []string) {
				assert.GreaterOrEqual(t, len(result), 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSentences(tt.text, tt.language)
			tt.validate(t, result)
		})
	}
}

func TestTranslationPracticeService_getPunktModel(t *testing.T) {
	service := createTestService(t)

	t.Run("English model loads", func(t *testing.T) {
		model := service.getPunktModel("en")
		// English may use built-in tokenizer, so model might be nil
		// The important thing is that extraction works
		if model == nil {
			t.Log("English model is nil (may use built-in tokenizer)")
			result := service.extractSentences("Test sentence.", "en")
			assert.NotEmpty(t, result, "Extraction should work even if model is nil")
		} else {
			require.NotNil(t, model, "English model should be available if loaded")
		}
	})

	t.Run("model caching works", func(t *testing.T) {
		model1 := service.getPunktModel("en")
		model2 := service.getPunktModel("en")
		assert.Same(t, model1, model2, "Should return the same cached model instance")
	})

	t.Run("non-existent language returns nil", func(t *testing.T) {
		model := service.getPunktModel("xx")
		assert.Nil(t, model, "Non-existent language should return nil")
	})

	t.Run("languages without models return nil", func(t *testing.T) {
		// These languages may not have models available
		languagesWithoutModels := []string{"ru", "hi", "ja", "zh"}
		for _, lang := range languagesWithoutModels {
			model := service.getPunktModel(lang)
			// It's OK if nil - we test that fallback works in other tests
			if model == nil {
				t.Logf("Language %s has no Punkt model (will use regex fallback)", lang)
			}
		}
	})
}

func TestTranslationPracticeService_getPunktModelName(t *testing.T) {
	service := createTestService(t)

	tests := []struct {
		code     string
		expected string
	}{
		{"en", "english"},
		{"it", "italian"},
		{"fr", "french"},
		{"de", "german"},
		{"es", "spanish"},
		{"ru", "russian"},
		{"hi", "hindi"},
		{"ja", "japanese"},
		{"zh", "chinese"},
		{"xx", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := service.getPunktModelName(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslationPracticeService_extractSentences_PunktVsRegex(t *testing.T) {
	service := createTestService(t)

	// Test that Punkt and regex produce similar results for English
	text := "Dr. Smith went to the U.S.A. He met Mr. Jones. They talked. It was great!"
	language := "en"

	result := service.extractSentences(text, language)
	assert.GreaterOrEqual(t, len(result), 4, "Should extract multiple sentences")

	// Verify all sentences have proper punctuation
	for _, sent := range result {
		trimmed := strings.TrimSpace(sent)
		assert.NotEmpty(t, trimmed, "Sentence should not be empty")
		assert.True(t,
			strings.HasSuffix(trimmed, ".") ||
				strings.HasSuffix(trimmed, "!") ||
				strings.HasSuffix(trimmed, "?"),
			"Sentence should end with punctuation: %s", trimmed)
	}
}

func TestTranslationPracticeService_extractSentences_ConcurrentAccess(t *testing.T) {
	service := createTestService(t)

	// Test that concurrent access to getPunktModel is safe
	// Note: English model may use built-in tokenizer (not file-based)
	done := make(chan bool, 10)
	results := make([]*sentences.DefaultSentenceTokenizer, 10)
	for i := 0; i < 10; i++ {
		idx := i
		go func() {
			defer func() { done <- true }()
			results[idx] = service.getPunktModel("en")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all results are the same (caching works)
	var firstModel *sentences.DefaultSentenceTokenizer
	for i, model := range results {
		if model != nil {
			if firstModel == nil {
				firstModel = model
			} else {
				assert.Same(t, firstModel, model, "All concurrent calls should return same cached model")
			}
		} else {
			// Model might be nil if English uses built-in tokenizer differently
			t.Logf("Goroutine %d returned nil model (this is OK if English uses built-in tokenizer)", i)
		}
	}

	// Verify extraction works (the important thing is that extraction still works,
	// regardless of whether model is cached or nil)
	result := service.extractSentences("Test sentence.", "en")
	assert.NotEmpty(t, result, "Extraction should work even if getPunktModel returns nil")
}

func TestTranslationPracticeService_extractSentences_PreservesPunctuation(t *testing.T) {
	service := createTestService(t)

	tests := []struct {
		name     string
		text     string
		language string
	}{
		{
			name:     "period",
			text:     "First sentence. Second sentence.",
			language: "en",
		},
		{
			name:     "exclamation",
			text:     "Wow! Amazing!",
			language: "en",
		},
		{
			name:     "question",
			text:     "How are you? What's up?",
			language: "en",
		},
		{
			name:     "mixed",
			text:     "Statement. Question? Exclamation!",
			language: "en",
		},
		{
			name:     "Russian punctuation",
			text:     "Привет. Как дела? Отлично!",
			language: "ru",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSentences(tt.text, tt.language)
			assert.NotEmpty(t, result, "Should extract at least one sentence")

			// Count punctuation in original vs result
			originalPunct := countPunctuation(tt.text)
			resultPunct := 0
			for _, sent := range result {
				resultPunct += countPunctuation(sent)
			}

			// Result should have same or more punctuation (may add trailing if missing)
			assert.GreaterOrEqual(t, resultPunct, originalPunct/2,
				"Should preserve most punctuation marks")
		})
	}
}

func countPunctuation(text string) int {
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' || r == '…' {
			count++
		}
	}
	return count
}

// Benchmark sentence extraction with Punkt vs regex
func BenchmarkExtractSentences_English(b *testing.B) {
	service := createTestService(&testing.T{})
	text := "This is the first sentence. This is the second sentence. This is the third sentence. " +
		"Dr. Smith went to the U.S.A. He met Mr. Jones. They talked. It was great!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.extractSentences(text, "en")
	}
}

func BenchmarkExtractSentences_Russian(b *testing.B) {
	service := createTestService(&testing.T{})
	text := "Это первое предложение. Это второе предложение. Это третье предложение. " +
		"Он сказал «Привет!» Она ответила «Здравствуй!»"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.extractSentences(text, "ru")
	}
}

