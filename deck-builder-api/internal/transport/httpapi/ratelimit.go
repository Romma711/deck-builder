package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitParams configura el limitador de peticiones (token bucket por IP,
// en memoria, sin dependencias externas).
type RateLimitParams struct {
	// RPS/Burst: clase general para toda la API.
	RPS   float64
	Burst int
	// AuthRPS/AuthBurst: clase más estricta para /api/v3/auth/* (fuerza bruta).
	AuthRPS   float64
	AuthBurst int
}

// rateLimitExemptPaths: sondeos de salud/métricas y el spec OpenAPI no deben
// consumir cupo (un sondeo de Kubernetes o Prometheus no debe recibir 429).
var rateLimitExemptPaths = map[string]bool{
	"/healthz":      true,
	"/readyz":       true,
	"/metrics":      true,
	"/openapi.yaml": true,
}

// bucketEntry es un token bucket y su último uso (para barrido de huérfanos).
type bucketEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiter mantiene un bucket por clave (clase + IP). El barrido de entradas
// ocurre oportunistamente (cada N llamadas), evitando goroutines de fondo.
type rateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucketEntry
	params     RateLimitParams
	now        func() time.Time
	maxIdle    time.Duration
	calls      int
	sweepEvery int
}

func newRateLimiter(p RateLimitParams) *rateLimiter {
	// Normalización fail-safe: un parámetro en cero (config ausente o
	// inválida) jamás debe bloquear toda la API; se usan defaults sanos.
	if p.RPS <= 0 {
		p.RPS = 20
	}
	if p.Burst < 1 {
		p.Burst = 40
	}
	if p.AuthRPS <= 0 {
		p.AuthRPS = 5
	}
	if p.AuthBurst < 1 {
		p.AuthBurst = 10
	}
	return &rateLimiter{
		buckets:    make(map[string]*bucketEntry),
		params:     p,
		now:        time.Now,
		maxIdle:    10 * time.Minute,
		sweepEvery: 256,
	}
}

// allow toma un token del bucket de la clave; crea el bucket si no existe.
func (r *rateLimiter) allow(key string, limit rate.Limit, burst int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	r.calls++
	if r.calls%r.sweepEvery == 0 {
		r.sweep(now)
	}

	e, ok := r.buckets[key]
	if !ok {
		e = &bucketEntry{limiter: rate.NewLimiter(limit, burst)}
		r.buckets[key] = e
	}
	e.lastSeen = now
	return e.limiter.Allow()
}

func (r *rateLimiter) sweep(now time.Time) {
	for k, e := range r.buckets {
		if now.Sub(e.lastSeen) > r.maxIdle {
			delete(r.buckets, k)
		}
	}
}

// middleware aplica el rate limiting por IP con dos clases:
//   - auth: /api/v3/auth/* → AuthRPS/AuthBurst (más estricto).
//   - general: el resto → RPS/Burst.
//
// OPTIONS (preflight CORS) y rutas de infraestructura quedan exentas.
func (r *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions ||
			rateLimitExemptPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		limit, burst := rate.Limit(r.params.RPS), r.params.Burst
		class := "api"
		if strings.HasPrefix(c.Request.URL.Path, authPathPrefix) {
			limit, burst = rate.Limit(r.params.AuthRPS), r.params.AuthBurst
			class = "auth"
		}

		if !r.allow(class+"|"+c.ClientIP(), limit, burst) {
			// Retry-After en segundos: sugerimos cuánto esperar.
			c.Header("Retry-After", formatRetryAfter(limit))
			RespondError(c, domain.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}

// authPathPrefix agrupa los endpoints de autenticación bajo una clase
// de rate limiting más estricta.
const authPathPrefix = "/api/v3/auth/"

func formatRetryAfter(limit rate.Limit) string {
	secs := 1.0
	if limit > 0 {
		secs = math.Ceil(1.0 / float64(limit))
	}
	return strconv.Itoa(int(secs))
}
