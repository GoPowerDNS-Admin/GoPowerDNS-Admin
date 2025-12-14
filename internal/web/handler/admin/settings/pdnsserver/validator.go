package pdnsserver

import (
	"github.com/go-playground/validator/v10"
)

type (
	// ErrorResponse represents a validation error response.
	ErrorResponse struct {
		Error       bool
		FailedField string
		Tag         string
		Value       interface{}
	}

	// XValidator is a custom validator struct.
	XValidator struct {
		// validator *validator.Validate
	}

	// GlobalErrorHandlerResp represents a global error response structure.
	GlobalErrorHandlerResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

var validate = validator.New()

// Validate performs validation on the provided data and returns a slice of ErrorResponse.
func (v XValidator) Validate(data interface{}) []ErrorResponse {
	var validationErrors []ErrorResponse

	errs := validate.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) { //nolint:errorlint,errcheck // ok here
			// In this case data object is actually holding the User struct
			var elem ErrorResponse

			elem.FailedField = err.Field() // Export struct field name
			elem.Tag = err.Tag()           // Export struct tag
			elem.Value = err.Value()       // Export field value
			elem.Error = true

			validationErrors = append(validationErrors, elem)
		}
	}

	return validationErrors
}
