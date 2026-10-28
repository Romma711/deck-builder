package domain

import "context"

// Clave de contexto NO exportada: evita colisiones y que otros paquetes
// "adivinen" el tipo del valor (patrón idiomático de context.WithValue).
type ctxKey struct{}

// WithUserID adjunta el ID del usuario autenticado al contexto del request.
//
// Uso: lo setea el middleware de autenticación en el transporte y lo lee la
// infraestructura de persistencia (RLS en PostgreSQL necesita saber quién
// consulta en cada transacción). Si no hay usuario (anónimo), el valor se
// omite y UserIDFromContext devuelve "".
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserIDFromContext devuelve el ID del usuario autenticado o "" si es anónimo.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
