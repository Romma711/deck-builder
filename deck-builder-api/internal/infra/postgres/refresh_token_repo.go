package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository adapta ports.RefreshTokenRepository a PostgreSQL.
type RefreshTokenRepository struct {
	db *gorm.DB
}

var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	userID, err := toUUID(token.UserID)
	if err != nil {
		return err
	}

	model := RefreshTokenModel{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt,
		CreatedAt: token.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("guardar refresh token: %w", err)
	}

	token.ID = model.ID.String()
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var model RefreshTokenModel
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar refresh token: %w", err)
	}

	return &domain.RefreshToken{
		ID:        model.ID.String(),
		UserID:    model.UserID.String(),
		TokenHash: model.TokenHash,
		ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt,
		CreatedAt: model.CreatedAt,
	}, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	tid, err := toUUID(id)
	if err != nil {
		return err
	}

	err = r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("id = ? AND revoked_at IS NULL", tid).
		Update("revoked_at", gorm.Expr("now()")).Error
	if err != nil {
		return fmt.Errorf("revocar refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	uid, err := toUUID(userID)
	if err != nil {
		return err
	}

	err = r.db.WithContext(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND revoked_at IS NULL", uid).
		Update("revoked_at", gorm.Expr("now()")).Error
	if err != nil {
		return fmt.Errorf("revocar refresh tokens del usuario: %w", err)
	}
	return nil
}
