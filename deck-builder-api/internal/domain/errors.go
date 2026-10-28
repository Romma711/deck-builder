// Package domain define las entidades y reglas centrales del negocio.
//
// REGLA: este paquete NO puede importar infraestructura (GORM, gin,
// configuración, etc.). Es el corazón puro de la aplicación.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Errores sentinela del dominio. La capa de transporte los interpreta y los
// convierte en códigos HTTP (ver internal/transport/httpapi.RespondError).
//
// Los servicios deben envolverlos con fmt.Errorf("...: %w", domain.ErrNotFound)
// para dar contexto sin perder la capacidad de comparar con errors.Is.
var (
	ErrNotFound        = errors.New("recurso no encontrado")
	ErrUnauthorized    = errors.New("no autenticado")
	ErrForbidden       = errors.New("no autorizado")
	ErrValidation      = errors.New("validación fallida")
	ErrConflict        = errors.New("conflicto")
	ErrInvalidID       = errors.New("identificador inválido")
	ErrInternal        = errors.New("error interno")
	ErrTooManyRequests = errors.New("demasiadas peticiones")
)

// ValidationError es el error tipado para fallos de validación con detalle
// por campo (útil para responder "details" en la API).
type ValidationError struct {
	// Details mapea nombre de campo → motivo de invalidez.
	Details map[string]string
	// Message opcional; si está vacío se usa el de ErrValidation.
	Message string
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if len(e.Details) == 0 {
		return ErrValidation.Error()
	}
	parts := make([]string, 0, len(e.Details))
	for field, reason := range e.Details {
		parts = append(parts, fmt.Sprintf("%s: %s", field, reason))
	}
	return fmt.Sprintf("%s (%s)", ErrValidation.Error(), strings.Join(parts, ", "))
}

// Unwrap permite errors.Is(err, domain.ErrValidation) == true.
func (e *ValidationError) Unwrap() error { return ErrValidation }
