package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// GameRepository accede al catálogo de juegos y formatos (tablas games y
// formats, sembradas por migración).
type GameRepository interface {
	// FindByCode devuelve el juego con sus formatos o domain.ErrNotFound.
	FindByCode(ctx context.Context, code string) (*domain.Game, error)
	// FindActive devuelve todos los juegos activos con sus formatos.
	FindActive(ctx context.Context) ([]domain.Game, error)
}
