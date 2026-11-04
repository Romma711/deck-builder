package ports

import (
	"context"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
)

// MemberInfo es una membership enriquecida con datos del usuario (una sola
// query, sin N+1) para el listado de miembros.
type MemberInfo struct {
	UserID   string
	Username string
	Email    string
	Role     domain.Role
	JoinedAt time.Time
}

// MembershipRepository persiste la relación usuario ↔ organización.
// Todas las implementaciones deben devolver domain.ErrNotFound cuando la
// membership o la org no existen.
type MembershipRepository interface {
	// Add inserta la membership; domain.ErrConflict si ya existe.
	Add(ctx context.Context, m *domain.Membership) error

	// FindByOrgAndUser devuelve la membership o domain.ErrNotFound.
	FindByOrgAndUser(ctx context.Context, orgID, userID string) (*domain.Membership, error)

	// UpdateRole cambia el rol; domain.ErrNotFound si no hay membership.
	UpdateRole(ctx context.Context, orgID, userID string, role domain.Role) error

	// Remove elimina la membership; domain.ErrNotFound si no existe.
	Remove(ctx context.Context, orgID, userID string) error

	// CountOwners cuenta owners de la org (guarda de "último owner").
	CountOwners(ctx context.Context, orgID string) (int, error)

	// ListMembersByOrg lista los miembros con username/email de la org.
	ListMembersByOrg(ctx context.Context, orgID string) ([]MemberInfo, error)
}
