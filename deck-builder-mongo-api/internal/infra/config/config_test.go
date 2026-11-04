package config

import (
	"errors"
	"testing"
)

func TestValidate_OK(t *testing.T) {
	cfg := Config{
		Auth:     AuthConfig{SecretKey: "secreto"},
		Database: DatabaseConfig{DSN: "postgres://localhost:5432/deckbuilder"},
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() inesperadamente falló: %v", err)
	}
}

func TestValidate_FaltaSecretKey(t *testing.T) {
	cfg := Config{
		Auth:     AuthConfig{SecretKey: ""},
		Database: DatabaseConfig{DSN: "postgres://localhost:5432/deckbuilder"},
	}
	err := cfg.validate()
	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("esperaba ErrMissingEnv, obtuve: %v", err)
	}
}

func TestValidate_FaltaDSN(t *testing.T) {
	cfg := Config{Auth: AuthConfig{SecretKey: "s"}}
	err := cfg.validate()
	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("esperaba ErrMissingEnv, obtuve: %v", err)
	}
}

func TestLoad_ConVariablesDeEntorno(t *testing.T) {
	t.Setenv("SECRET_SIGNING_KEY", "clave-de-prueba")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("PORT", "9090")
	t.Setenv("DB_MAX_OPEN_CONNS", "25")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() falló: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, esperaba 9090", cfg.Port)
	}
	if cfg.Database.DSN == "" {
		t.Error("Database.DSN no se cargó")
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, esperaba 25", cfg.Database.MaxOpenConns)
	}
	if cfg.Env == "" {
		t.Errorf("Env no debería quedar vacío (fallback a development)")
	}
}

func TestLoad_RateLimitYCORS(t *testing.T) {
	t.Setenv("SECRET_SIGNING_KEY", "clave-de-prueba")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("RATE_LIMIT_RPS", "7.5")
	t.Setenv("RATE_LIMIT_BURST", "15")
	t.Setenv("RATE_LIMIT_AUTH_RPS", "2")
	t.Setenv("RATE_LIMIT_AUTH_BURST", "4")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.com, https://b.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() falló: %v", err)
	}
	if cfg.RateLimit.RPS != 7.5 || cfg.RateLimit.Burst != 15 {
		t.Errorf("RateLimit = %+v, esperaba RPS 7.5 / Burst 15", cfg.RateLimit)
	}
	if cfg.RateLimit.AuthRPS != 2 || cfg.RateLimit.AuthBurst != 4 {
		t.Errorf("RateLimit auth = %+v, esperaba RPS 2 / Burst 4", cfg.RateLimit)
	}
	wantOrigins := []string{"https://a.com", "https://b.com"}
	if len(cfg.CORS.AllowedOrigins) != 2 ||
		cfg.CORS.AllowedOrigins[0] != wantOrigins[0] ||
		cfg.CORS.AllowedOrigins[1] != wantOrigins[1] {
		t.Errorf("CORS = %v, esperaba %v (espacios recortados)", cfg.CORS.AllowedOrigins, wantOrigins)
	}
}

func TestLoad_RateLimitDefaultYMalo(t *testing.T) {
	t.Setenv("SECRET_SIGNING_KEY", "clave-de-prueba")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("RATE_LIMIT_RPS", "no-numero")
	t.Setenv("RATE_LIMIT_BURST", "9")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() falló: %v", err)
	}
	if cfg.RateLimit.RPS != 20 {
		t.Errorf("RPS = %v, esperaba el default 20 ante valor inválido", cfg.RateLimit.RPS)
	}
	if cfg.RateLimit.Burst != 9 {
		t.Errorf("Burst = %d, esperaba 9", cfg.RateLimit.Burst)
	}
}

func TestSplitList(t *testing.T) {
	if got := splitList(""); got != nil {
		t.Errorf("splitList(\"\") = %v, esperaba nil", got)
	}
	got := splitList(" a.com ,b.com,, c.com ")
	want := []string{"a.com", "b.com", "c.com"}
	if len(got) != len(want) {
		t.Fatalf("splitList = %v, esperaba %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitList[%d] = %q, esperaba %q", i, got[i], want[i])
		}
	}
}
