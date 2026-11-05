package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// RefreshTokenRepository almacena refresh tokens por su hash.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}
