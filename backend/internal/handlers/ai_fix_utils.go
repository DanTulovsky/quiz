package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MergeAISuggestion merges AI response into the original question map.
// It ensures top-level metadata from original are preserved and AI-provided
// content is merged into original["content"]. It moves top-level correct_answer
// and explanation into content to avoid duplicates.
func MergeAISuggestion(original, aiResp map[string]interface{}) map[string]interface{} {
	// copy original to avoid mutating caller's map
	out := map[string]interface{}{}
	b, _ := json.Marshal(original)
	_ = json.Unmarshal(b, &out)

	// ensure content map exists
	contentIface := out["content"]
	contentMap, _ := contentIface.(map[string]interface{})
	if contentMap == nil {
		contentMap = map[string]interface{}{}
		out["content"] = contentMap
	}

	// merge ai content
	if aiContentRaw, ok := aiResp["content"]; ok {
		if aiContentMap, ok2 := aiContentRaw.(map[string]interface{}); ok2 {
			for k, v := range aiContentMap {
				contentMap[k] = v
			}
		}
	}

	// move top-level fields into content
	if ca, ok := aiResp["correct_answer"]; ok {
		contentMap["correct_answer"] = ca
		delete(aiResp, "correct_answer")
	}
	if ex, ok := aiResp["explanation"]; ok {
		contentMap["explanation"] = ex
		delete(aiResp, "explanation")
	}

	if cr, ok := aiResp["change_reason"]; ok {
		out["change_reason"] = cr
	}

	NormalizeContent(contentMap)

	return out
}

// NormalizeContent attempts to sanitize content fields: options->[]string,
// correct_answer->int, trims duplicates and clamps indices.
func NormalizeContent(contentMap map[string]interface{}) {
	// normalize options
	if optsRaw, ok := contentMap["options"]; ok {
		switch opts := optsRaw.(type) {
		case []interface{}:
			seen := map[string]bool{}
			var out []string
			for _, it := range opts {
				s, ok := it.(string)
				if !ok {
					continue
				}
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				if !seen[s] {
					out = append(out, s)
					seen[s] = true
				}
			}
			contentMap["options"] = out
		case []string:
			// ok
		case string:
			var parsed []string
			if err := json.Unmarshal([]byte(opts), &parsed); err == nil {
				contentMap["options"] = parsed
			} else {
				parts := strings.FieldsFunc(opts, func(r rune) bool { return r == '\n' || r == ',' })
				var out []string
				seen := map[string]bool{}
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					if !seen[p] {
						out = append(out, p)
						seen[p] = true
					}
				}
				contentMap["options"] = out
			}
		default:
			delete(contentMap, "options")
		}
	}

	// ensure options slice is []string
	if optsI, ok := contentMap["options"].([]interface{}); ok {
		var out []string
		for _, it := range optsI {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		contentMap["options"] = out
	}

	// normalize correct_answer
	if ca, ok := contentMap["correct_answer"]; ok {
		switch v := ca.(type) {
		case float64:
			contentMap["correct_answer"] = int(v)
		case int:
			// ok
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				contentMap["correct_answer"] = n
			} else {
				delete(contentMap, "correct_answer")
			}
		default:
			delete(contentMap, "correct_answer")
		}
	}

	// clamp correct_answer to options length
	if ca, ok := contentMap["correct_answer"].(int); ok {
		if opts, ok := contentMap["options"].([]string); ok {
			if len(opts) == 0 {
				contentMap["correct_answer"] = 0
			} else if ca < 0 || ca >= len(opts) {
				contentMap["correct_answer"] = 0
			}
		}
	}

	// ensure simple string fields
	for _, k := range []string{"explanation", "question", "passage", "sentence"} {
		if v, ok := contentMap[k]; ok {
			switch t := v.(type) {
			case string:
				// ok
			default:
				contentMap[k] = fmt.Sprint(t)
			}
		}
	}
}
