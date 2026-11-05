package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// CardFilter acumula criterios de búsqueda del catálogo (AND).
type CardFilter struct {
	GameCode string // "mtg", "pokemon"...
	Query    string // búsqueda por nombre (subcadena)
	Limit    int    // 0 = default
	Offset   int
}

// CardRepository gestiona el catálogo de cartas normalizado.
type CardRepository interface {
	// Upsert inserta la carta si no existe (identidad: game_id + name +
	// serial_set) y devuelve siempre el registro persistido con su ID.
	Upsert(ctx context.Context, card *domain.Card) (*domain.Card, error)
	FindByID(ctx context.Context, id string) (*domain.Card, error)
	Find(ctx context.Context, filter CardFilter) ([]domain.Card, int, error) // cards, total
}
