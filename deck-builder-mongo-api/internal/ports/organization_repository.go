package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// OrganizationRepository persiste organizaciones.
type OrganizationRepository interface {
	// Create persiste la org e inserta atómicamente (transacción) la
	// membership inicial con rol owner. Si la org queda sin owner, no sale
	// de aquí: es un invariante del agregado.
	Create(ctx context.Context, org *domain.Organization, ownerID string) error

	// FindByID devuelve la org o domain.ErrNotFound.
	FindByID(ctx context.Context, id string) (*domain.Organization, error)

	// FindBySlug devuelve la org por su slug único o domain.ErrNotFound.
	FindBySlug(ctx context.Context, slug string) (*domain.Organization, error)

	// ListByUser devuelve las orgs de las que el usuario es miembro, con
	// Role = rol de ese usuario en cada una.
	ListByUser(ctx context.Context, userID string) ([]domain.Organization, error)

	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id string) error
}
