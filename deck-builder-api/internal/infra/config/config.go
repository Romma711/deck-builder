// Package config carga y valida la configuración de la aplicación UNA sola vez
// al arrancar. Nunca usa log.Fatal: devuelve errores y decide el caller.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ErrMissingEnv indica que faltan variables de entorno requeridas.
var ErrMissingEnv = errors.New("faltan variables de entorno requeridas")

// Config es la configuración tipada de toda la aplicación.
type Config struct {
	Env       string
	Port      string
	Auth      AuthConfig
	Database  DatabaseConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
}

// RateLimitConfig controla el limitador de peticiones por IP (token bucket,
// en memoria). AuthRPS/AuthBurst aplican a /api/v3/auth/* (más estrictos,
// para frenar fuerza bruta de credenciales).
type RateLimitConfig struct {
	RPS       float64
	Burst     int
	AuthRPS   float64
	AuthBurst int
}

// CORSConfig lista de orígenes permitidos para el navegador (CORS).
// Vacío = sin middleware CORS (por defecto: la API es same-origin).
type CORSConfig struct {
	AllowedOrigins []string
}

type AuthConfig struct {
	SecretKey        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	PasswordHashCost int
}

type DatabaseConfig struct {
	// DSN: postgres://user:pass@host:port/dbname?sslmode=disable
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
}

// Load lee .env (best-effort: si no existe, usa las variables del entorno),
// parsea y valida la configuración.
func Load() (Config, error) {
	// Si no hay archivo .env, no es un error: las variables pueden venir del
	// entorno (docker, CI, systemd).
	_ = godotenv.Load()

	cfg := Config{
		Env:  getEnv("APP_ENV", "development"),
		Port: getEnv("PORT", "8080"),
		Auth: AuthConfig{
			SecretKey:        os.Getenv("SECRET_SIGNING_KEY"),
			AccessTokenTTL:   getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL:  getDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
			PasswordHashCost: getInt("BCRYPT_COST", 10),
		},
		Database: DatabaseConfig{
			DSN:          os.Getenv("DATABASE_URL"),
			MaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns: getInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLife:  getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		RateLimit: RateLimitConfig{
			RPS:       getFloat("RATE_LIMIT_RPS", 20),
			Burst:     getInt("RATE_LIMIT_BURST", 40),
			AuthRPS:   getFloat("RATE_LIMIT_AUTH_RPS", 5),
			AuthBurst: getInt("RATE_LIMIT_AUTH_BURST", 10),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitList(os.Getenv("CORS_ALLOWED_ORIGINS")),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var missing []string
	if c.Auth.SecretKey == "" {
		missing = append(missing, "SECRET_SIGNING_KEY")
	}
	if c.Database.DSN == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s (copiá .env.example a .env y completalo)",
			ErrMissingEnv, strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return fallback
	}
	return f
}

// splitList parsea una lista separada por comas (p. ej. orígenes CORS),
// descartando elementos vacíos.
func splitList(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(v, ",") {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}
