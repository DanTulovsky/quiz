package contextutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty API key",
			apiKey:   "",
			expected: "[EMPTY]",
		},
		{
			name:     "short API key (4 chars)",
			apiKey:   "abcd",
			expected: "****",
		},
		{
			name:     "short API key (8 chars)",
			apiKey:   "abcdefgh",
			expected: "********",
		},
		{
			name:     "medium API key (12 chars)",
			apiKey:   "abcdefghijkl",
			expected: "abcd****ijkl",
		},
		{
			name:     "long API key (20 chars)",
			apiKey:   "abcdefghijklmnopqrst",
			expected: "abcd************qrst",
		},
		{
			name:     "very long API key (32 chars)",
			apiKey:   "abcdefghijklmnopqrstuvwxyz123456",
			expected: "abcd************************3456",
		},
		{
			name:     "API key with special characters",
			apiKey:   "sk-1234567890abcdef",
			expected: "sk-1***********cdef",
		},
		{
			name:     "API key with numbers only",
			apiKey:   "1234567890123456",
			expected: "1234********3456",
		},
		{
			name:     "API key with mixed case",
			apiKey:   "Sk-1234567890AbCdEf",
			expected: "Sk-1***********CdEf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.apiKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskAPIKey_SecurityProperties(t *testing.T) {
	// Test that masking preserves length
	apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"
	masked := MaskAPIKey(apiKey)
	assert.Equal(t, len(apiKey), len(masked), "Masked key should have same length as original")

	// Test that first 4 and last 4 characters are preserved
	assert.Equal(t, apiKey[:4], masked[:4], "First 4 characters should be preserved")
	assert.Equal(t, apiKey[len(apiKey)-4:], masked[len(masked)-4:], "Last 4 characters should be preserved")

	// Test that middle characters are masked
	middleMasked := masked[4 : len(masked)-4]
	for _, char := range middleMasked {
		assert.Equal(t, '*', char, "Middle characters should be masked with asterisks")
	}
}

func TestMaskAPIKey_EdgeCases(t *testing.T) {
	// Test with exactly 8 characters (boundary case)
	apiKey8 := "12345678"
	masked8 := MaskAPIKey(apiKey8)
	assert.Equal(t, "********", masked8, "8-character key should be fully masked")

	// Test with 9 characters (should show first 4 and last 4)
	apiKey9 := "123456789"
	masked9 := MaskAPIKey(apiKey9)
	assert.Equal(t, "1234*6789", masked9, "9-character key should show first 4 and last 4 with 1 asterisk")

	// Test with unicode characters
	unicodeKey := "sk-测试1234567890测试"
	maskedUnicode := MaskAPIKey(unicodeKey)
	assert.Equal(t, len(unicodeKey), len(maskedUnicode), "Unicode key should maintain length")
	assert.Equal(t, unicodeKey[:4], maskedUnicode[:4], "First 4 unicode characters should be preserved")
	assert.Equal(t, unicodeKey[len(unicodeKey)-4:], maskedUnicode[len(maskedUnicode)-4:], "Last 4 unicode characters should be preserved")
}

func TestMaskAPIKey_Consistency(t *testing.T) {
	// Test that masking is consistent for the same input
	apiKey := "sk-1234567890abcdef"
	masked1 := MaskAPIKey(apiKey)
	masked2 := MaskAPIKey(apiKey)
	assert.Equal(t, masked1, masked2, "Masking should be consistent for same input")

	// Test that different inputs produce different masked outputs
	apiKey1 := "sk-1234567890abcdef"
	apiKey2 := "sk-9876543210fedcba"
	maskedResult1 := MaskAPIKey(apiKey1)
	maskedResult2 := MaskAPIKey(apiKey2)
	assert.NotEqual(t, maskedResult1, maskedResult2, "Different inputs should produce different masked outputs")
}
