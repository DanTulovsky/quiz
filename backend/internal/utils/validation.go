package contextutils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// IsValidEmail checks if an email address is valid using go-playground/validator
func IsValidEmail(email string) bool {
	return validate.Var(email, "email") == nil
}

// normalizeStringSlice normalizes a string slice from various input types
func normalizeStringSlice(raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// ExtractQuestionContent extracts question text and options from a content map
// It handles both flat content maps and nested content.content maps
// For question text, it checks "question" first, then "sentence" (for FillInBlank questions)
func ExtractQuestionContent(content map[string]interface{}) (questionText string, options []string) {
	if content == nil {
		return "", nil
	}

	nestedContent, _ := content["content"].(map[string]interface{})

	getString := func(key string) string {
		if v, ok := content[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
		if nestedContent != nil {
			if v, ok := nestedContent[key].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
		}
		return ""
	}

	// Check "question" first, then "sentence" (for FillInBlank questions)
	questionText = getString("question")
	if questionText == "" {
		questionText = getString("sentence")
	}

	getOptions := func() []string {
		if raw, ok := content["options"]; ok {
			return normalizeStringSlice(raw)
		}
		if nestedContent != nil {
			if raw, ok := nestedContent["options"]; ok {
				return normalizeStringSlice(raw)
			}
		}
		return nil
	}

	options = getOptions()
	return questionText, options
}

// ValidateQuestionContent validates that question content has required fields:
// - question text must be non-empty
// - options must have at least 4 items
// Returns an error if validation fails, nil if valid
// questionID is used for error messages but can be 0 if not available
func ValidateQuestionContent(content map[string]interface{}, questionID int) error {
	if content == nil {
		if questionID > 0 {
			return ErrorWithContextf("question %d: content is nil (missing 'question'/'sentence' field and 'options' field)", questionID)
		}
		return ErrorWithContextf("question content is nil (missing 'question'/'sentence' field and 'options' field)")
	}

	questionText, options := ExtractQuestionContent(content)

	// Build a list of missing fields for clearer error messages
	var missingFields []string
	if questionText == "" {
		missingFields = append(missingFields, "'question' or 'sentence'")
	}
	if len(options) < 4 {
		missingFields = append(missingFields, fmt.Sprintf("'options' (has %d, need at least 4)", len(options)))
	}

	if len(missingFields) > 0 {
		if questionID > 0 {
			return ErrorWithContextf("question %d: missing required field(s): %s", questionID, strings.Join(missingFields, ", "))
		}
		return ErrorWithContextf("question content missing required field(s): %s", strings.Join(missingFields, ", "))
	}

	return nil
}
