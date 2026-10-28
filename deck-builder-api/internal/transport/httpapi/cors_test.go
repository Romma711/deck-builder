package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSEngine(t *testing.T, origins []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(cors(origins))
	engine.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func TestCORS_SinOrigenesConfigurados_NoAgregaHeaders(t *testing.T) {
	engine := newCORSEngine(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://ejemplo.com")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, esperaba vacío (sin configuración no hay CORS)", got)
	}
}

func TestCORS_SinOrigenes_PreflightResponde204SinHeaders(t *testing.T) {
	// Sin CORS configurado el preflight no debe caer en un 404: se contesta
	// 204 sin headers CORS y el navegador bloquea la petición.
	engine := newCORSEngine(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://ejemplo.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, esperaba 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, esperaba vacío sin configurar", got)
	}
}

func TestCORS_PreflightPermitido(t *testing.T) {
	engine := newCORSEngine(t, []string{"https://app.ejemplo.com"})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://app.ejemplo.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, esperaba 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.ejemplo.com" {
		t.Errorf("ACAO = %q, esperaba el origen permitido", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("falta Access-Control-Allow-Methods en el preflight")
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, esperaba Origin", got)
	}
}

func TestCORS_PreflightOrigenNoPermitido_NoExpongaHeaders(t *testing.T) {
	engine := newCORSEngine(t, []string{"https://app.ejemplo.com"})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://atacante.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q para origen no permitido, esperaba vacío", got)
	}
}

func TestCORS_PeticionRealConOrigenPermitido(t *testing.T) {
	engine := newCORSEngine(t, []string{"https://app.ejemplo.com"})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.ejemplo.com")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.ejemplo.com" {
		t.Errorf("ACAO = %q, esperaba el origen permitido", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, esperaba 200 (CORS no debe cortar la petición)", w.Code)
	}
}

func TestCORS_ComodinPermiteCualquierOrigen(t *testing.T) {
	engine := newCORSEngine(t, []string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://cualquiera.com")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("ACAO = %q, esperaba *", got)
	}
}
