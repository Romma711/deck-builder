package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// MembershipRepository adapta ports.MembershipRepository a PostgreSQL.
type MembershipRepository struct {
	db *gorm.DB
}

var _ ports.MembershipRepository = (*MembershipRepository)(nil)

func NewMembershipRepository(db *gorm.DB) *MembershipRepository {
	return &MembershipRepository{db: db}
}

func (r *MembershipRepository) Add(ctx context.Context, m *domain.Membership) error {
	orgID, err := toUUID(m.OrgID)
	if err != nil {
		return err
	}
	userID, err := toUUID(m.UserID)
	if err != nil {
		return err
	}

	model := MembershipModel{
		OrgID:     orgID,
		UserID:    userID,
		Role:      string(m.Role),
		CreatedAt: m.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: el usuario ya es miembro de la organización", domain.ErrConflict)
		}
		if isForeignKeyViolation(err) {
			// Org o usuario inexistentes: para el cliente es "no encontrado".
			return domain.ErrNotFound
		}
		return fmt.Errorf("agregar miembro: %w", err)
	}
	return nil
}

func (r *MembershipRepository) FindByOrgAndUser(ctx context.Context, orgID, userID string) (*domain.Membership, error) {
	oid, err := toUUID(orgID)
	if err != nil {
		return nil, err
	}
	uid, err := toUUID(userID)
	if err != nil {
		return nil, err
	}

	var model MembershipModel
	dbErr := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", oid, uid).
		First(&model).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar membership: %w", dbErr)
	}
	return &domain.Membership{
		OrgID:     model.OrgID.String(),
		UserID:    model.UserID.String(),
		Role:      domain.Role(model.Role),
		CreatedAt: model.CreatedAt,
	}, nil
}

func (r *MembershipRepository) UpdateRole(ctx context.Context, orgID, userID string, role domain.Role) error {
	oid, err := toUUID(orgID)
	if err != nil {
		return err
	}
	uid, err := toUUID(userID)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Model(&MembershipModel{}).
		Where("org_id = ? AND user_id = ?", oid, uid).
		Update("role", string(role))
	if res.Error != nil {
		return fmt.Errorf("cambiar rol: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MembershipRepository) Remove(ctx context.Context, orgID, userID string) error {
	oid, err := toUUID(orgID)
	if err != nil {
		return err
	}
	uid, err := toUUID(userID)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", oid, uid).
		Delete(&MembershipModel{})
	if res.Error != nil {
		return fmt.Errorf("eliminar miembro: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MembershipRepository) CountOwners(ctx context.Context, orgID string) (int, error) {
	oid, err := toUUID(orgID)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&MembershipModel{}).
		Where("org_id = ? AND role = ?", oid, string(domain.RoleOwner)).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("contar owners: %w", err)
	}
	return int(count), nil
}

func (r *MembershipRepository) ListMembersByOrg(ctx context.Context, orgID string) ([]ports.MemberInfo, error) {
	oid, err := toUUID(orgID)
	if err != nil {
		return nil, err
	}

	// JOIN users: username/email en la misma query (sin N+1).
	type row struct {
		MembershipModel
		Username string `gorm:"column:username"`
		Email    string `gorm:"column:email"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Select("memberships.*, users.username, users.email").
		Joins("JOIN users ON users.id = memberships.user_id").
		Where("memberships.org_id = ?", oid).
		Order("memberships.created_at").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listar miembros: %w", err)
	}

	members := make([]ports.MemberInfo, 0, len(rows))
	for _, rw := range rows {
		members = append(members, ports.MemberInfo{
			UserID:   rw.UserID.String(),
			Username: rw.Username,
			Email:    rw.Email,
			Role:     domain.Role(rw.Role),
			JoinedAt: rw.CreatedAt,
		})
	}
	return members, nil
}

// isForeignKeyViolation detecta violaciones de FK (23503): org o usuario
// inexistentes al insertar una membership.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
