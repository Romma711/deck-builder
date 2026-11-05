package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRequestIDEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(requestID())
	engine.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func TestRequestID_GeneraCuandoNoViene(t *testing.T) {
	engine := newRequestIDEngine(t)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	id := w.Header().Get(HeaderRequestID)
	if id == "" {
		t.Fatal("la respuesta no lleva X-Request-ID")
	}
	if len(id) != 32 { // hex de 16 bytes
		t.Errorf("id generado = %q, esperaba 32 caracteres hex", id)
	}
}

func TestRequestID_RespetaElEntranteValido(t *testing.T) {
	engine := newRequestIDEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderRequestID, "abc-123_XYZ")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get(HeaderRequestID); got != "abc-123_XYZ" {
		t.Errorf("id = %q, esperaba el entrante abc-123_XYZ", got)
	}
}

func TestRequestID_RechazaEntranteInvalido(t *testing.T) {
	engine := newRequestIDEngine(t)

	invalidos := []string{
		"con espacios",          // caracteres no permitidos
		"semi;colon",            // inyección
		strings.Repeat("a", 65), // demasiado largo
		"",                      // vacío
	}
	for _, id := range invalidos {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if id != "" {
			req.Header.Set(HeaderRequestID, id)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		got := w.Header().Get(HeaderRequestID)
		if got == id || got == "" {
			t.Errorf("entrada %q → id %q: esperaba reemplazo generado", id, got)
		}
		if !isValidRequestID(got) {
			t.Errorf("entrada %q → id %q inválido", id, got)
		}
	}
}
