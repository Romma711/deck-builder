// Package validation centraliza la validación de structs con tags `validate:`
// y la convierte en domain.ValidationError (errores 400 con detalle por campo).
package validation

import (
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/go-playground/validator/v10"
)

// Struct valida un struct y devuelve un *domain.ValidationError con el
// detalle por campo si algo falla.
func Struct(payload any) error {
	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			return &domain.ValidationError{Message: "payload inválido"}
		}
		details := make(map[string]string, len(validationErrs))
		for _, fieldErr := range validationErrs {
			details[fieldErr.Field()] = fieldErr.Tag()
		}
		return &domain.ValidationError{Details: details}
	}
	return nil
}
