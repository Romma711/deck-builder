package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newMetricsEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := NewEngine(EngineOptions{})
	// Ruta registrada como en producción: la etiqueta debe ser la plantilla.
	engine.GET("/api/v3/decks/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func scrapeMetrics(t *testing.T, engine *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/metrics = %d, esperaba 200", w.Code)
	}
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("leyendo /metrics: %v", err)
	}
	return string(body)
}

func TestMetrics_CuentaPeticionesConRutaPlantilla(t *testing.T) {
	engine := newMetricsEngine(t)

	// Dos peticiones a la MISMA ruta con IDs distintos: si la etiqueta fuera
	// la URL cruda, aparecerían dos series; con la plantilla, una sola.
	engine.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v3/decks/aaa", nil))
	engine.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v3/decks/bbb", nil))

	body := scrapeMetrics(t, engine)
	want := `http_requests_total{method="GET",route="/api/v3/decks/:id",status="200"} 2`
	if !strings.Contains(body, want) {
		t.Errorf("no encontré %q en:\n%s", want, body)
	}
	if strings.Contains(body, "aaa") || strings.Contains(body, "bbb") {
		t.Error("las etiquetas contienen IDs crudos: la cardinalidad explotaría")
	}
}

func TestMetrics_MarcaRutasNoRegistradasComoUnmatched(t *testing.T) {
	engine := newMetricsEngine(t)

	engine.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/ruta/inexistente", nil))

	body := scrapeMetrics(t, engine)
	want := `http_requests_total{method="GET",route="unmatched",status="404"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("no encontré %q en:\n%s", want, body)
	}
}

func TestMetrics_IncluyeColectoresDeGoYProceso(t *testing.T) {
	engine := newMetricsEngine(t)

	body := scrapeMetrics(t, engine)
	for _, metric := range []string{"go_goroutines", "process_open_fds"} {
		if !strings.Contains(body, metric) {
			t.Errorf("falta el colector %q en /metrics", metric)
		}
	}
}

func TestMetrics_HistogramaDeDuracion(t *testing.T) {
	engine := newMetricsEngine(t)

	engine.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v3/decks/x", nil))

	body := scrapeMetrics(t, engine)
	if !strings.Contains(body, `http_request_duration_seconds_bucket{method="GET",route="/api/v3/decks/:id"`) {
		t.Errorf("falta el histograma de duración con ruta plantilla en:\n%s", body)
	}
}
