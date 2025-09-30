package handlers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeAISuggestion_BasicMerge(t *testing.T) {
	original := map[string]interface{}{
		"id":   42,
		"type": "vocabulary",
		"content": map[string]interface{}{
			"question":       "Old?",
			"options":        []interface{}{"A", "B", "C", "D"},
			"correct_answer": 1,
		},
	}
	ai := map[string]interface{}{
		"content": map[string]interface{}{
			"question":       "New?",
			"options":        []interface{}{"A", "B", "C", "D"},
			"correct_answer": float64(2),
		},
		"change_reason": "Fixed clarity",
	}

	merged := MergeAISuggestion(original, ai)
	// id may be unmarshaled as float64; accept int or float64
	idVal := merged["id"]
	switch v := idVal.(type) {
	case float64:
		require.Equal(t, 42.0, v)
	case int:
		require.Equal(t, 42, v)
	default:
		t.Fatalf("unexpected id type: %T", v)
	}

	// content.correct_answer should be int 2
	content := merged["content"].(map[string]interface{})
	require.Equal(t, 2, content["correct_answer"].(int))
	require.Equal(t, "Fixed clarity", merged["change_reason"].(string))
}

func TestMergeAISuggestion_EdgeCases(t *testing.T) {
	// Original with full metadata and odd AI output with duplicated fields and mixed types
	original := map[string]interface{}{
		"id":               15,
		"type":             "reading_comprehension",
		"language":         "italian",
		"level":            "A1",
		"difficulty_score": 0,
		"status":           "reported",
		"content": map[string]interface{}{
			"passage":        "Al supermercato ho comprato delle mele. Erano rosse e lucide.",
			"question":       "Che colore avevano le mele?",
			"options":        []interface{}{"Erano verdi.", "Erano gialle.", "Erano rosse.", "Erano marroni."},
			"correct_answer": 2,
			"explanation":    "Il testo specifica: 'Erano rosse e lucide' riferendosi alle mele.",
		},
	}

	// Simulate messy AI output: duplicated keys, options as a string, and top-level correct_answer
	aiRaw := `{
        "id": 15,
        "type": "reading_comprehension",
        "correct_answer": "2",
        "explanation": "Il testo specifica: 'Eranorossee lucide' riferendosi alle mele.",
        "content": {
            "options": "Erano verdi.,Erano gialle.,Erano rosse.,Erano marroni.",
            "passage": "Alsupermercato ho comprato delle mele. Erano rosse e lucide.",
            "question": "Checoloreavevano le mele che ha comprato?",
            "explanation": "Il testo specifica: 'Erano rosse e lucide' riferendosi alle mele."
        },
        "change_reason": "AI corrected punctuation and options formatting"
    }`

	var ai map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(aiRaw), &ai))

	merged := MergeAISuggestion(original, ai)

	// Ensure top-level metadata preserved
	require.Equal(t, "reading_comprehension", merged["type"].(string))
	require.Equal(t, "italian", merged["language"].(string))

	// content.options should be parsed into []string and deduped
	content := merged["content"].(map[string]interface{})
	opts, ok := content["options"].([]string)
	require.True(t, ok)
	require.Len(t, opts, 4)
	require.Contains(t, opts, "Erano rosse.")

	// correct_answer should be int 2
	require.Equal(t, 2, content["correct_answer"].(int))

	// change_reason should be present
	require.Equal(t, "AI corrected punctuation and options formatting", merged["change_reason"].(string))
}
