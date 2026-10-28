package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// DeckFilter acumula los criterios de búsqueda (AND entre campos no vacíos).
// La VISIBILIDAD (público/privado/ajeno) no se filtra aquí: la garantiza
// Row-Level Security en PostgreSQL según el usuario del contexto.
type DeckFilter struct {
	GameCode   string // "mtg", "pokemon"...
	OwnerID    string
	OrgID      string // mazos de una organización concreta
	Format     string
	LeaderName string // nombre del líder/commander
}

// DeckRepository persiste mazos. Las implementaciones deben devolver
// domain.ErrNotFound ante IDs inexistentes y domain.ErrInvalidID ante IDs
// malformados.
type DeckRepository interface {
	// Create persiste el mazo y le asigna su ID.
	Create(ctx context.Context, deck *domain.Deck) error
	FindByID(ctx context.Context, id string) (*domain.Deck, error)
	// Find devuelve los mazos que cumplen el filtro. Puede devolver los
	// mazos sin la lista de cartas (proyección liviana) para listados.
	Find(ctx context.Context, filter DeckFilter) ([]domain.Deck, error)
	Update(ctx context.Context, deck *domain.Deck) error
	Delete(ctx context.Context, id string) error
}
