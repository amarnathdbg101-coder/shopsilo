package reuse

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

// ValidateStruct validates a struct and returns formatted error message
func ValidateStruct(s interface{}) error {
	if err := Validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMsgs []string
			for _, fieldErr := range validationErrors {
				switch fieldErr.Tag() {
				case "required":
					errMsgs = append(errMsgs, fmt.Sprintf("%s is required", fieldErr.Field()))
				case "email":
					errMsgs = append(errMsgs, fmt.Sprintf("%s must be a valid email address", fieldErr.Field()))
				case "min":
					errMsgs = append(errMsgs, fmt.Sprintf("%s must be at least %s characters long", fieldErr.Field(), fieldErr.Param()))
				case "max":
					errMsgs = append(errMsgs, fmt.Sprintf("%s cannot exceed %s characters", fieldErr.Field(), fieldErr.Param()))
				default:
					errMsgs = append(errMsgs, fmt.Sprintf("%s is invalid (%s)", fieldErr.Field(), fieldErr.Tag()))
				}
			}
			return fmt.Errorf("%s", strings.Join(errMsgs, ", "))
		}
		return err
	}
	return nil
}