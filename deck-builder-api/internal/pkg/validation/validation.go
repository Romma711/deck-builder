// Package validation centraliza la validación de structs con tags `validate:`
// y la convierte en domain.ValidationError (errores 400 con detalle por campo).
package validation

import (
	"reflect"
	"strings"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/go-playground/validator/v10"
)

// Struct valida un struct y devuelve un *domain.ValidationError con el
// detalle por campo si algo falla.
func Struct(payload any) error {
	if err := validate.Struct(payload); err != nil {
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			return &domain.ValidationError{Message: "payload inválido"}
		}
		details := make(map[string]string, len(validationErrs))
		for _, fieldErr := range validationErrs {
			details[fieldPath(fieldErr)] = fieldErr.Tag()
		}
		return &domain.ValidationError{Details: details}
	}
	return nil
}

// validate usa los nombres del JSON en los errores (`deck_size`, no `DeckSize`):
// son los que conoce el cliente. Se crea una sola vez (cachea los structs).
var validate = func() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	return v
}()

// fieldPath devuelve la ruta JSON del campo sin el struct raíz:
// "deckRequest.cards[0].quantity" → "cards[0].quantity".
func fieldPath(fieldErr validator.FieldError) string {
	_, path, found := strings.Cut(fieldErr.Namespace(), ".")
	if !found {
		return fieldErr.Field()
	}
	return path
}
