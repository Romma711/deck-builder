package postgres

import (
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate aplica las migraciones pendientes (goose). Idempotente: solo
// ejecuta las que faltan. Los scripts van embebidos en el binario.
func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("obtener connection para migraciones: %w", err)
	}

	goose.SetBaseFS(migrationsFS)
	goose.SetDialect("postgres")

	if err := goose.Run("up", sqlDB, "migrations"); err != nil {
		return fmt.Errorf("aplicar migraciones: %w", err)
	}
	return nil
}
