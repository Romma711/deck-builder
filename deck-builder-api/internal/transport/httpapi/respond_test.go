package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/gin-gonic/gin"
)

func performErrorRequest(err error) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v3/decks/1", nil)
	RespondError(c, err)
	return w
}

func TestMapError_DominioAHTTP(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"invalid id", domain.ErrInvalidID, http.StatusBadRequest, "INVALID_ID"},
		{"conflict", domain.ErrConflict, http.StatusConflict, "CONFLICT"},
		{"validation", domain.ErrValidation, http.StatusBadRequest, "VALIDATION_ERROR"},
		{"unknown", fmt.Errorf("boom: sql connection refused"), http.StatusInternalServerError, "INTERNAL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := mapError(tc.err)
			if status != tc.wantStatus {
				t.Errorf("status = %d, esperaba %d", status, tc.wantStatus)
			}
			if body.Code != tc.wantCode {
				t.Errorf("code = %q, esperaba %q", body.Code, tc.wantCode)
			}
		})
	}
}

// El error desconocido NUNCA debe filtrar el detalle interno al cliente.
func TestMapError_NoExpondeDetalleInterno(t *testing.T) {
	_, body := mapError(fmt.Errorf("sql: contraseña incorrecta en postgres://user:pass@host"))
	if body.Message != "error interno del servidor" {
		t.Errorf("mensaje filtrado: %q", body.Message)
	}
}

func TestRespondError_FormaEstándar(t *testing.T) {
	w := performErrorRequest(fmt.Errorf("mazo: %w", domain.ErrNotFound))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, esperaba 404", w.Code)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if payload.Error.Code != "NOT_FOUND" {
		t.Errorf("code = %q, esperaba NOT_FOUND", payload.Error.Code)
	}
}

func TestValidationError_AportaDetails(t *testing.T) {
	err := &domain.ValidationError{Details: map[string]string{"Name": "requerido"}}
	status, body := mapError(err)

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", status)
	}
	if body.Details["Name"] != "requerido" {
		t.Errorf("details = %v, esperaba Name: requerido", body.Details)
	}
}
