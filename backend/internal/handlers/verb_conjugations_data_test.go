package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type testVerb struct {
	Tenses []struct {
		Conjugations []struct{} `json:"conjugations"`
	} `json:"tenses"`
}

func TestHindiRomanizedFilesHaveFullTenses(t *testing.T) {
	baseDir := "../handlers/data/verb-conjugations/hi"
	// absolute path fallback when running from repo root
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		baseDir = "internal/handlers/data/verb-conjugations/hi"
	}

	targets := []string{"aana.json", "dena.json", "dekhana.json", "hona.json", "jana.json", "janana.json", "kahana.json", "lena.json", "sochana.json"}
	for _, fname := range targets {
		path := filepath.Join(baseDir, fname)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed reading %s: %v", fname, err)
		}
		var v testVerb
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("failed parsing %s: %v", fname, err)
		}
		if len(v.Tenses) != 12 {
			t.Fatalf("%s expected 12 tenses, got %d", fname, len(v.Tenses))
		}
		for i, ts := range v.Tenses {
			if len(ts.Conjugations) != 6 {
				t.Fatalf("%s tense #%d expected 6 conjugations, got %d", fname, i, len(ts.Conjugations))
			}
		}
	}
}

