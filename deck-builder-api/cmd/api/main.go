// Punto de entrada de la API.
//
// Responsabilidad ÚNICA: componer la aplicación (DI manual por constructores)
// y gestionar su ciclo de vida (arranque, señales, apagado ordenado).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authapp "github.com/Romma711/deck-builder/internal/app/auth"
	deckapp "github.com/Romma711/deck-builder/internal/app/deck"
	"github.com/Romma711/deck-builder/internal/app/gamespec"
	orgapp "github.com/Romma711/deck-builder/internal/app/org"
	authinfra "github.com/Romma711/deck-builder/internal/infra/auth"
	"github.com/Romma711/deck-builder/internal/infra/config"
	"github.com/Romma711/deck-builder/internal/infra/postgres"
	"github.com/Romma711/deck-builder/internal/transport/httpapi"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Configuración (una sola vez, validada, sin log.Fatal en librerías).
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuración inválida: %w", err)
	}

	// 2. Logging estructurado.
	logger := newLogger(cfg.Env)
	slog.SetDefault(logger)

	// 3. Modo de gin según entorno.
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 4. Base de datos: UN pool de conexiones (pgx vía GORM).
	//    Fallo de conexión = error de arranque claro (fail fast).
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer dbCancel()

	db, err := postgres.Open(dbCtx, postgres.Options{
		DSN:          cfg.Database.DSN,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		ConnMaxLife:  cfg.Database.ConnMaxLife,
		Debug:        cfg.Env != "production",
	})
	if err != nil {
		return fmt.Errorf("base de datos: %w", err)
	}

	// Migraciones versionadas (goose, embebidas en el binario).
	if err := postgres.Migrate(db); err != nil {
		return fmt.Errorf("migraciones: %w", err)
	}
	logger.Info("conexión a PostgreSQL establecida y migraciones aplicadas")

	// 5. Composición de dependencias (DI manual por constructores).
	userRepo := postgres.NewUserRepository(db)
	refreshRepo := postgres.NewRefreshTokenRepository(db)
	deckRepo := postgres.NewDeckRepository(db)
	cardRepo := postgres.NewCardRepository(db)
	gameRepo := postgres.NewGameRepository(db)
	orgRepo := postgres.NewOrganizationRepository(db)
	membershipRepo := postgres.NewMembershipRepository(db)
	passwordHasher := authinfra.NewPasswordHasher(cfg.Auth.PasswordHashCost)
	tokenService := authinfra.NewTokenService(cfg.Auth.SecretKey, cfg.Auth.AccessTokenTTL)
	authService := authapp.NewService(
		userRepo, refreshRepo, tokenService, passwordHasher,
		cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL,
	)

	// Estrategias de validación por juego (Strategy): alta aquí. Un juego
	// nuevo = implementar ports.GameSpec + añadirlo a esta lista.
	specRegistry, err := gamespec.NewRegistry(
		gamespec.MTGSpec{},
		gamespec.PokemonSpec{},
		gamespec.YuGiOhSpec{},
		gamespec.OnePieceSpec{},
	)
	if err != nil {
		return fmt.Errorf("registry de GameSpec: %w", err)
	}

	deckService := deckapp.NewService(deckRepo, cardRepo, gameRepo, membershipRepo, specRegistry)

	// Organizaciones (Fase 5): roles owner > admin > member con lookup RBAC
	// por request (siempre fresco; ver internal/app/org).
	orgService := orgapp.NewService(orgRepo, membershipRepo, userRepo)

	// 6. Motor HTTP: logging, panics, request IDs, CORS, rate limiting,
	//    métricas Prometheus y healthchecks con ping a la base de datos.
	engine := httpapi.NewEngine(httpapi.EngineOptions{
		Logger: logger,
		Readiness: func(ctx context.Context) error {
			return postgres.Ping(ctx, db)
		},
		RateLimit: httpapi.RateLimitParams{
			RPS:       cfg.RateLimit.RPS,
			Burst:     cfg.RateLimit.Burst,
			AuthRPS:   cfg.RateLimit.AuthRPS,
			AuthBurst: cfg.RateLimit.AuthBurst,
		},
		CORSOrigins: cfg.CORS.AllowedOrigins,
	})

	// 7. Registro de rutas (API v3).
	v3 := engine.Group("/api/v3")
	httpapi.RegisterAuthRoutes(v3, authService, tokenService)
	httpapi.RegisterCatalogRoutes(v3, gameRepo, cardRepo)
	httpapi.RegisterDeckRoutes(v3, deckService, tokenService)
	httpapi.RegisterOrgRoutes(v3, orgService, membershipRepo, tokenService)

	// 8. Servidor con timeouts (protección contra lentitud/larga espera).
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 9. Señales de sistema para el apagado ordenado.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("servidor iniciado", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("servidor: %w", err)
	case <-ctx.Done():
		logger.Info("señal de apagado recibida, cerrando conexiones...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("apagado limpio: %w", err)
	}

	// Cerramos el pool de conexiones después de que el servidor dejó de
	// aceptar tráfico.
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	logger.Info("servidor detenido correctamente")
	return nil
}

// newLogger crea el logger estructurado: JSON en producción, texto en dev.
func newLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env != "production" {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}
