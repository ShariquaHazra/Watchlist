package validator

import "github.com/go-playground/validator/v10"

var validate = validator.New()

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
	default:
		return e.Field() + " is invalid"
	}
}
