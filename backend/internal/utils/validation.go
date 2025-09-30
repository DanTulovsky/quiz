package contextutils

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// IsValidEmail checks if an email address is valid using go-playground/validator
func IsValidEmail(email string) bool {
	return validate.Var(email, "email") == nil
}
