package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/gin-gonic/gin"
)

// ReadinessCheck verifica que la aplicación esté lista para servir tráfico
// (p. ej. ping a la base de datos). Puede ser nil mientras no haya dependencias.
type ReadinessCheck func(ctx context.Context) error

// EngineOptions agrupa las dependencias y ajustes del motor HTTP. Se usa una
// struct (no parámetros posicionales) para que crezca sin romper llamadas.
type EngineOptions struct {
	// Logger base de la API. Si es nil se usa slog.Default().
	Logger *slog.Logger
	// Readiness es el chequeo de /readyz (opcional).
	Readiness ReadinessCheck
	// RateLimit configura el limitador por IP (token bucket en memoria).
	RateLimit RateLimitParams
	// CORSOrigins son los orígenes permitidos para el navegador (vacío = none).
	CORSOrigins []string
}

// NewEngine crea el motor HTTP con logging estructurado, manejo de panics,
// request IDs, CORS, rate limiting, métricas Prometheus y endpoints de salud.
// Los handlers se registran fuera (en main), manteniendo la composición
// explícita.
func NewEngine(opts EngineOptions) *gin.Engine {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	engine := gin.New()
	// Sin proxies de confianza: ClientIP() usa RemoteAddr directo y nadie
	// puede spoofear X-Forwarded-For para evadir el rate limiting. Detrás de
	// un proxy real, configurar los CIDRs de confianza explícitamente.
	_ = engine.SetTrustedProxies(nil)

	metrics := NewMetrics()
	engine.Use(
		requestID(),
		requestLogger(opts.Logger),
		recovery(opts.Logger),
		metrics.middleware(),
		cors(opts.CORSOrigins),
		newRateLimiter(opts.RateLimit).middleware(),
	)

	engine.GET("/metrics", gin.WrapH(metrics.Handler()))
	registerOpenAPIRoute(engine)
	registerHealthRoutes(engine, opts.Readiness)
	return engine
}

// requestLogger reemplaza gin.Logger por logs estructurados (slog).
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"client_ip", c.ClientIP(),
			"request_id", RequestID(c),
		)
	}
}

// recovery captura panics y responde 500 sin tumbar el proceso.
func recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec) // abortos deliberados del servidor se propagan
				}
				logger.Error("panic atendido",
					"error", rec,
					"stack", string(debug.Stack()),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"request_id", RequestID(c),
				)
				RespondError(c, fmt.Errorf("%w: %v", domain.ErrInternal, rec))
				c.Abort()
			}
		}()
		c.Next()
	}
}

// registerHealthRoutes expone los endpoints de salud y el spec OpenAPI.
func registerHealthRoutes(engine *gin.Engine, readiness ReadinessCheck) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.GET("/readyz", func(c *gin.Context) {
		if readiness == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
			return
		}
		if err := readiness(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unavailable",
				"error":  "no se pudo verificar la base de datos",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}
