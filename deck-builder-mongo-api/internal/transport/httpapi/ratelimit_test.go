package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func newRateLimitedEngine(t *testing.T, p RateLimitParams) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	_ = engine.SetTrustedProxies(nil)
	engine.Use(newRateLimiter(p).middleware())
	engine.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.POST("/api/v3/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

// slowLimiter usa un refill tan lento que el burst dura para toda la prueba.
func slowBurst(n int) RateLimitParams {
	return RateLimitParams{RPS: 0.0001, Burst: n, AuthRPS: 0.0001, AuthBurst: n}
}

func doReq(engine *gin.Engine, method, path, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if ip != "" {
		req.RemoteAddr = ip + ":1234"
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestRateLimit_BurstAgotadoResponde429(t *testing.T) {
	engine := newRateLimitedEngine(t, slowBurst(2))

	for i := 0; i < 2; i++ {
		if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.1"); w.Code != http.StatusOK {
			t.Fatalf("petición %d = %d, esperaba 200", i+1, w.Code)
		}
	}

	w := doReq(engine, http.MethodGet, "/ping", "10.0.0.1")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, esperaba 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Error("falta el header Retry-After en la respuesta 429")
	}

	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body 429 no es JSON: %v", err)
	}
	if body.Error.Code != "RATE_LIMITED" {
		t.Errorf("code = %q, esperaba RATE_LIMITED", body.Error.Code)
	}
}

func TestRateLimit_IPsIndependientes(t *testing.T) {
	engine := newRateLimitedEngine(t, slowBurst(1))

	if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.1"); w.Code != http.StatusOK {
		t.Fatalf("IP A, petición 1 = %d, esperaba 200", w.Code)
	}
	if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.1"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("IP A, petición 2 = %d, esperaba 429", w.Code)
	}
	// Otra IP no debe heredar el bloqueo de la primera.
	if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.2"); w.Code != http.StatusOK {
		t.Fatalf("IP B = %d, esperaba 200 (bucket independiente)", w.Code)
	}
}

func TestRateLimit_RutasExentasNoConsumenCupo(t *testing.T) {
	engine := newRateLimitedEngine(t, slowBurst(1))

	for i := 0; i < 5; i++ {
		if w := doReq(engine, http.MethodGet, "/healthz", "10.0.0.9"); w.Code != http.StatusOK {
			t.Fatalf("healthz %d = %d, sondeo exento no debe limitarse", i+1, w.Code)
		}
	}
}

func TestRateLimit_AuthMasEstrictoQueElResto(t *testing.T) {
	engine := newRateLimitedEngine(t, RateLimitParams{
		RPS: 1000, Burst: 100, // clase general: holgada
		AuthRPS: 0.0001, AuthBurst: 1, // clase auth: durísima
	})

	if w := doReq(engine, http.MethodPost, "/api/v3/auth/login", "10.0.0.5"); w.Code != http.StatusOK {
		t.Fatalf("login 1 = %d, esperaba 200", w.Code)
	}
	if w := doReq(engine, http.MethodPost, "/api/v3/auth/login", "10.0.0.5"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("login 2 = %d, esperaba 429 (clase auth agotada)", w.Code)
	}
	// La misma IP sigue teniendo cupo en la clase general.
	if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.5"); w.Code != http.StatusOK {
		t.Fatalf("ping = %d, esperaba 200 (clase general independiente)", w.Code)
	}
}

func TestRateLimiter_SweepEliminaBucketsInactivos(t *testing.T) {
	r := newRateLimiter(slowBurst(5))

	fakeNow := time.Now()
	r.now = func() time.Time { return fakeNow }

	if !r.allow("api|10.0.0.1", rateOf(r, "api"), r.params.Burst) {
		t.Fatal("la primera petición no debía estar limitada")
	}
	if len(r.buckets) != 1 {
		t.Fatalf("buckets = %d, esperaba 1", len(r.buckets))
	}

	fakeNow = fakeNow.Add(r.maxIdle + time.Minute)
	r.sweep(r.now())

	if len(r.buckets) != 0 {
		t.Errorf("buckets tras sweep = %d, esperaba 0 (huérfanos barridos)", len(r.buckets))
	}
}

func TestRateLimit_ParamsEnCeroUsanDefaultsSeguros(t *testing.T) {
	// Un config en cero (env ausentes o inválidas) no debe bloquear la API:
	// newRateLimiter normaliza a defaults sanos.
	r := newRateLimiter(RateLimitParams{})
	if r.params.RPS <= 0 || r.params.Burst < 1 || r.params.AuthRPS <= 0 || r.params.AuthBurst < 1 {
		t.Fatalf("params normalizados = %+v, todos deben ser > 0", r.params)
	}

	engine := gin.New()
	_ = engine.SetTrustedProxies(nil)
	engine.Use(r.middleware())
	engine.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	if w := doReq(engine, http.MethodGet, "/ping", "10.0.0.1"); w.Code != http.StatusOK {
		t.Fatalf("con params en cero, la primera petición = %d, esperaba 200", w.Code)
	}
}

// rateOf evita duplicar la conversión de límite por clase en los tests.
func rateOf(r *rateLimiter, class string) rate.Limit {
	if class == "auth" {
		return rate.Limit(r.params.AuthRPS)
	}
	return rate.Limit(r.params.RPS)
}
