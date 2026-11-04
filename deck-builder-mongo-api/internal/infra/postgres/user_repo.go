package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// isUniqueViolation detecta violaciones de constraint únicos (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func toUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", domain.ErrInvalidID, id)
	}
	return parsed, nil
}

// --- UserRepository ---

type UserRepository struct {
	db *gorm.DB
}

var _ ports.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	model := UserModel{
		ID:           uuid.Must(uuid.NewV7()),
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: username o email ya registrado", domain.ErrConflict)
		}
		return fmt.Errorf("crear usuario: %w", err)
	}

	user.ID = model.ID.String()
	return nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&model).Error
	return convertUserErr(model, err)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := toUUID(id)
	if err != nil {
		return nil, err
	}

	var model UserModel
	dbErr := r.db.WithContext(ctx).First(&model, "id = ?", uid).Error
	return convertUserErr(model, dbErr)
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	uid, err := toUUID(user.ID)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", uid).
		Updates(map[string]any{
			"username":   user.Username,
			"email":      user.Email,
			"updated_at": user.UpdatedAt,
		})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return fmt.Errorf("%w: username o email ya en uso", domain.ErrConflict)
		}
		return fmt.Errorf("actualizar usuario: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	uid, err := toUUID(id)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Delete(&UserModel{}, "id = ?", uid)
	if res.Error != nil {
		return fmt.Errorf("eliminar usuario: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func convertUserErr(model UserModel, err error) (*domain.User, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar usuario: %w", err)
	}
	return &domain.User{
		ID:           model.ID.String(),
		Username:     model.Username,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}
