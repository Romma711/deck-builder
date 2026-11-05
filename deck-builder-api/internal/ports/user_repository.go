// Package ports define las INTERFACES (puertos) por las que el dominio y los
// casos de uso acceden a la infraestructura. El detalle (GORM, HTTP, etc.)
// vive en internal/infra y se inyecta desde cmd/api.
package ports

import (
	"context"

	"github.com/Romma711/deck-builder/internal/domain"
)

// UserRepository persiste usuarios. Las implementaciones deben devolver
// domain.ErrNotFound cuando el usuario no exista y domain.ErrConflict ante
// duplicados (username/email únicos).
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}
