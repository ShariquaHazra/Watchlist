package validator

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	// Custom validations register karo
	validate.RegisterValidation("no_only_spaces", noOnlySpaces)
	validate.RegisterValidation("strong_password", strongPassword)
	validate.RegisterValidation("valid_name", validName)
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Validate(s interface{}) []ValidationError {
	var errors []ValidationError
	err := validate.Struct(s)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errors = append(errors, ValidationError{
				Field:   e.Field(),
				Message: getErrorMessage(e),
			})
		}
	}
	return errors
}

// ── Custom Validators ──────────────────────────────────

// Sirf spaces nahi chalega
func noOnlySpaces(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

// Password strong hona chahiye
func strongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

// Name mein letters/numbers hone chahiye — sirf special chars nahi
func validName(fl validator.FieldLevel) bool {
	name := strings.TrimSpace(fl.Field().String())
	matched, _ := regexp.MatchString(`[a-zA-Z0-9]`, name)
	return matched
}

func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return "invalid email format"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	case "no_only_spaces":
		return e.Field() + " cannot be only spaces"
	case "strong_password":
		return "password must contain uppercase, lowercase, number and special character"
	case "valid_name":
		return e.Field() + " must contain at least one letter or number"
	default:
		return e.Field() + " is invalid"
	}
}
