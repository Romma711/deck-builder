package ports

import (
	"context"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
)

// InvitationRepository persiste invitaciones a organizaciones. Una
// invitación "activa" no fue usada ni revocada y no venció a `now`.
type InvitationRepository interface {
	// Create persiste la invitación con el hash de su token y le asigna ID.
	Create(ctx context.Context, inv *domain.Invitation, tokenHash string) error

	// ListActiveByOrg lista las invitaciones activas de la org.
	ListActiveByOrg(ctx context.Context, orgID string, now time.Time) ([]domain.Invitation, error)

	// FindActiveByTokenHash devuelve la invitación activa (con nombre de org
	// e invitante) o domain.ErrNotFound.
	FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Invitation, error)

	// Revoke revoca una invitación activa de la org; domain.ErrNotFound si no hay.
	Revoke(ctx context.Context, orgID, id string, now time.Time) error

	// Accept marca la invitación como usada y crea la membership en una
	// única transacción. domain.ErrNotFound si no está activa;
	// domain.ErrConflict si el usuario ya es miembro (y no la consume).
	Accept(ctx context.Context, tokenHash, userID string, now time.Time) (*domain.Invitation, error)
}
