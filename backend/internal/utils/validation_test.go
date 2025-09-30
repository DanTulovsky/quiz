package contextutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	// Test valid emails
	assert.True(t, IsValidEmail("user@example.com"))
	assert.True(t, IsValidEmail("user.name@example.com"))
	assert.True(t, IsValidEmail("user+tag@example.com"))
	assert.True(t, IsValidEmail("user@example.co.uk"))
	assert.True(t, IsValidEmail("user@subdomain.example.com"))

	// Test invalid emails
	assert.False(t, IsValidEmail(""))
	assert.False(t, IsValidEmail("invalid-email"))
	assert.False(t, IsValidEmail("@example.com"))
	assert.False(t, IsValidEmail("user@"))
	assert.False(t, IsValidEmail("user@.com"))
	assert.False(t, IsValidEmail("user@example"))
	assert.False(t, IsValidEmail("user@example."))
	assert.False(t, IsValidEmail("user name@example.com"))

	assert.False(t, IsValidEmail("user@example..com"))
}
