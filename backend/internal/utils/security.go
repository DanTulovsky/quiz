package contextutils

import (
	"strings"
)

// MaskAPIKey masks an API key for logging purposes to prevent exposure
// Returns a masked version that shows only first 4 and last 4 characters
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "[EMPTY]"
	}

	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}

	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
