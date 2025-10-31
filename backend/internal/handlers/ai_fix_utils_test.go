package handlers

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMergeAISuggestion_TopLevelAnswerAndExplanation(t *testing.T) {
	original := map[string]interface{}{
		"content": map[string]interface{}{
			"question": "What is 2+2?",
			"options":  []string{"3", "4", "5"},
		},
		// Original values that should be allowed to change
		"correct_answer": 0,
		"explanation":    "wrong",
	}

	// AI response provides updated answer/explanation and also (incorrectly) nests duplicates
	aiResp := map[string]interface{}{
		"content": map[string]interface{}{
			"question":       "What is 2 + 2?",
			"options":        []interface{}{"3", "4", "5"},
			"correct_answer": 1,
			"explanation":    "2 + 2 = 4",
		},
		"correct_answer": 1,
		"explanation":    "2 + 2 = 4",
		"change_reason":  "fix answer and minor wording",
	}

	merged := MergeAISuggestion(original, aiResp)

	// Top-level fields should be updated
	if got, ok := merged["correct_answer"].(int); !ok || got != 1 {
		t.Fatalf("expected top-level correct_answer=1, got %#v", merged["correct_answer"])
	}
	if got, ok := merged["explanation"].(string); !ok || got == "" || got != "2 + 2 = 4" {
		t.Fatalf("expected top-level explanation '2 + 2 = 4', got %#v", merged["explanation"])
	}

	// Content should not duplicate these fields
	content, _ := merged["content"].(map[string]interface{})
	if _, ok := content["correct_answer"]; ok {
		t.Fatalf("content.correct_answer should be removed, found: %#v", content["correct_answer"])
	}
	if _, ok := content["explanation"]; ok {
		t.Fatalf("content.explanation should be removed, found: %#v", content["explanation"])
	}

	// Options should be normalized to []string
	if opts, ok := content["options"].([]string); !ok || !reflect.DeepEqual(opts, []string{"3", "4", "5"}) {
		b, _ := json.Marshal(content["options"])
		t.Fatalf("expected options to be [3 4 5], got %s", string(b))
	}
}
