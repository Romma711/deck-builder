// Package postgres implementa los repositorios del dominio sobre PostgreSQL
// con GORM (driver pgx). Es el adaptador DEFINITIVO de persistencia.
package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Options configura la conexión al pool.
type Options struct {
	DSN          string // postgres://user:pass@host:port/db?sslmode=disable
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
	Debug        bool
}

// Open conecta a PostgreSQL y ajusta el pool. Falla rápido si no hay
// conexión disponible.
func Open(ctx context.Context, opts Options) (*gorm.DB, error) {
	level := gormlogger.Warn
	if opts.Debug {
		level = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(opts.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
	})
	if err != nil {
		return nil, fmt.Errorf("conectar a PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obtener connection pool: %w", err)
	}

	if opts.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	}
	if opts.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	}
	if opts.ConnMaxLife > 0 {
		sqlDB.SetConnMaxLifetime(opts.ConnMaxLife)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping a PostgreSQL: %w", err)
	}

	return db, nil
}

// Ping verifica que la base esté disponible (usado por /readyz).
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
