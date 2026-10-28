package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/gin-gonic/gin"
)

// Forma estándar de error de la API v3:
//
//	{"error": {"code": "NOT_FOUND", "message": "...", "details": {...}}}
type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

// RespondError traduce un error de dominio a su respuesta HTTP y la escribe.
// Los errores 5xx se loguean con detalle (nunca se filtran al cliente).
func RespondError(c *gin.Context, err error) {
	status, body := mapError(err)

	if status >= http.StatusInternalServerError {
		slog.Error("error interno manejado",
			"error", err,
			"code", body.Code,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
	}

	if c.Writer.Written() {
		return // los headers ya salieron; no hay nada que responder
	}
	c.AbortWithStatusJSON(status, errorEnvelope{Error: body})
}

// mapError implementa el mapeo centralizado dominio → HTTP.
func mapError(err error) (int, errorBody) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, errorBody{Code: "NOT_FOUND", Message: err.Error()}
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, errorBody{Code: "UNAUTHORIZED", Message: err.Error()}
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, errorBody{Code: "FORBIDDEN", Message: err.Error()}
	case errors.Is(err, domain.ErrInvalidID):
		return http.StatusBadRequest, errorBody{Code: "INVALID_ID", Message: err.Error()}
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, errorBody{Code: "CONFLICT", Message: err.Error()}
	case errors.Is(err, domain.ErrTooManyRequests):
		return http.StatusTooManyRequests, errorBody{Code: "RATE_LIMITED", Message: err.Error()}
	case errors.Is(err, domain.ErrValidation):
		body := errorBody{Code: "VALIDATION_ERROR", Message: err.Error()}
		var ve *domain.ValidationError
		if errors.As(err, &ve) && len(ve.Details) > 0 {
			body.Details = ve.Details
		}
		return http.StatusBadRequest, body
	default:
		// Error desconocido: 500 genérico, sin exponer el detalle interno.
		return http.StatusInternalServerError, errorBody{
			Code:    "INTERNAL",
			Message: "error interno del servidor",
		}
	}
}
