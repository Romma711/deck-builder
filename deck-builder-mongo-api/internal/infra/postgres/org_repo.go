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

// OrganizationRepository adapta ports.OrganizationRepository a PostgreSQL.
type OrganizationRepository struct {
	db *gorm.DB
}

var _ ports.OrganizationRepository = (*OrganizationRepository)(nil)

func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create inserta la org y su membership owner en UNA transacción (invariante:
// ninguna org sale de aquí sin owner).
func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization, ownerID string) error {
	uid, err := toUUID(ownerID)
	if err != nil {
		return err
	}

	model := OrganizationModel{
		ID:        uuid.Must(uuid.NewV7()),
		Slug:      org.Slug,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
	membership := MembershipModel{
		OrgID:     model.ID,
		UserID:    uid,
		Role:      string(domain.RoleOwner),
		CreatedAt: org.CreatedAt,
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("%w: iniciar transacción: %v", domain.ErrInternal, tx.Error)
	}
	if err := tx.Create(&model).Error; err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: slug ya en uso: %s", domain.ErrConflict, org.Slug)
		}
		return fmt.Errorf("crear organización: %w", err)
	}
	if err := tx.Create(&membership).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("%w: crear membership owner: %v", domain.ErrInternal, err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("%w: confirmar organización: %v", domain.ErrInternal, err)
	}

	org.ID = model.ID.String()
	return nil
}

func (r *OrganizationRepository) FindByID(ctx context.Context, id string) (*domain.Organization, error) {
	oid, err := toUUID(id)
	if err != nil {
		return nil, err
	}

	var model OrganizationModel
	dbErr := r.db.WithContext(ctx).First(&model, "id = ?", oid).Error
	return convertOrgErr(model, dbErr)
}

func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var model OrganizationModel
	dbErr := r.db.WithContext(ctx).First(&model, "slug = ?", slug).Error
	return convertOrgErr(model, dbErr)
}

func (r *OrganizationRepository) ListByUser(ctx context.Context, userID string) ([]domain.Organization, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return nil, err
	}

	// JOIN memberships: cada org sale con el rol del usuario consultante.
	type row struct {
		OrganizationModel
		Role string `gorm:"column:role"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Select("organizations.*, memberships.role").
		Joins("JOIN memberships ON memberships.org_id = organizations.id").
		Where("memberships.user_id = ?", uid).
		Order("organizations.name").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listar organizaciones: %w", err)
	}

	orgs := make([]domain.Organization, 0, len(rows))
	for _, rw := range rows {
		orgs = append(orgs, domain.Organization{
			ID:        rw.ID.String(),
			Slug:      rw.Slug,
			Name:      rw.Name,
			Role:      domain.Role(rw.Role),
			CreatedAt: rw.CreatedAt,
			UpdatedAt: rw.UpdatedAt,
		})
	}
	return orgs, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	oid, err := toUUID(org.ID)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Model(&OrganizationModel{}).
		Where("id = ?", oid).
		Updates(map[string]any{
			"slug":       org.Slug,
			"name":       org.Name,
			"updated_at": org.UpdatedAt,
		})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return fmt.Errorf("%w: slug ya en uso: %s", domain.ErrConflict, org.Slug)
		}
		return fmt.Errorf("actualizar organización: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete elimina la org; memberships caen en cascada y los mazos quedan
// personales (FK org_id ON DELETE SET NULL).
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	oid, err := toUUID(id)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Delete(&OrganizationModel{}, "id = ?", oid)
	if res.Error != nil {
		return fmt.Errorf("eliminar organización: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func convertOrgErr(model OrganizationModel, err error) (*domain.Organization, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar organización: %w", err)
	}
	return &domain.Organization{
		ID:        model.ID.String(),
		Slug:      model.Slug,
		Name:      model.Name,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}
