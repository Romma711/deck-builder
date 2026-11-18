package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InvitationRepository adapta ports.InvitationRepository a PostgreSQL.
type InvitationRepository struct {
	db *gorm.DB
}

var _ ports.InvitationRepository = (*InvitationRepository)(nil)

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// activeAt filtra invitaciones vigentes a `now`.
func activeAt(tx *gorm.DB, now time.Time) *gorm.DB {
	return tx.Where("org_invitations.used_at IS NULL AND org_invitations.revoked_at IS NULL AND org_invitations.expires_at > ?", now)
}

// invitationRow es una invitación con datos de la org y del invitante.
type invitationRow struct {
	InvitationModel
	OrgName     string `gorm:"column:org_name"`
	OrgSlug     string `gorm:"column:org_slug"`
	InviterName string `gorm:"column:inviter_name"`
}

func (r *InvitationRepository) withDetails(tx *gorm.DB) *gorm.DB {
	return tx.Model(&InvitationModel{}).
		Select("org_invitations.*, organizations.name AS org_name, organizations.slug AS org_slug, users.username AS inviter_name").
		Joins("JOIN organizations ON organizations.id = org_invitations.org_id").
		Joins("LEFT JOIN users ON users.id = org_invitations.created_by")
}

func (row invitationRow) toDomain() domain.Invitation {
	inv := domain.Invitation{
		ID:          row.ID.String(),
		OrgID:       row.OrgID.String(),
		Role:        domain.Role(row.Role),
		ExpiresAt:   row.ExpiresAt,
		CreatedAt:   row.CreatedAt,
		OrgName:     row.OrgName,
		OrgSlug:     row.OrgSlug,
		InviterName: row.InviterName,
	}
	if row.CreatedBy != nil {
		inv.CreatedBy = row.CreatedBy.String()
	}
	return inv
}

func (r *InvitationRepository) Create(ctx context.Context, inv *domain.Invitation, tokenHash string) error {
	orgID, err := toUUID(inv.OrgID)
	if err != nil {
		return err
	}
	createdBy, err := optionalID(inv.CreatedBy)
	if err != nil {
		return err
	}
	model := InvitationModel{
		ID:        uuid.Must(uuid.NewV7()),
		OrgID:     orgID,
		TokenHash: tokenHash,
		Role:      string(inv.Role),
		CreatedBy: createdBy,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isForeignKeyViolation(err) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("crear invitación: %w", err)
	}
	inv.ID = model.ID.String()
	return nil
}

func (r *InvitationRepository) ListActiveByOrg(ctx context.Context, orgID string, now time.Time) ([]domain.Invitation, error) {
	oid, err := toUUID(orgID)
	if err != nil {
		return nil, err
	}
	var rows []invitationRow
	query := activeAt(r.withDetails(r.db.WithContext(ctx)), now).
		Where("org_invitations.org_id = ?", oid).
		Order("org_invitations.created_at DESC")
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listar invitaciones: %w", err)
	}
	out := make([]domain.Invitation, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *InvitationRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Invitation, error) {
	var row invitationRow
	err := activeAt(r.withDetails(r.db.WithContext(ctx)), now).
		Where("org_invitations.token_hash = ?", tokenHash).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar invitación: %w", err)
	}
	inv := row.toDomain()
	return &inv, nil
}

func (r *InvitationRepository) Revoke(ctx context.Context, orgID, id string, now time.Time) error {
	oid, err := toUUID(orgID)
	if err != nil {
		return err
	}
	iid, err := toUUID(id)
	if err != nil {
		return err
	}
	res := activeAt(r.db.WithContext(ctx).Model(&InvitationModel{}), now).
		Where("id = ? AND org_id = ?", iid, oid).
		Update("revoked_at", now)
	if res.Error != nil {
		return fmt.Errorf("revocar invitación: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *InvitationRepository) Accept(ctx context.Context, tokenHash, userID string, now time.Time) (*domain.Invitation, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return nil, err
	}

	var accepted *domain.Invitation
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE: dos aceptaciones simultáneas del mismo link se
		// serializan; la segunda ya no la ve activa.
		var model InvitationModel
		err := activeAt(tx.Clauses(clause.Locking{Strength: "UPDATE"}), now).
			Where("token_hash = ?", tokenHash).
			Take(&model).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("buscar invitación: %w", err)
		}

		membership := MembershipModel{OrgID: model.OrgID, UserID: uid, Role: model.Role, CreatedAt: now}
		if err := tx.Create(&membership).Error; err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: ya sos miembro de la organización", domain.ErrConflict)
			}
			return fmt.Errorf("agregar miembro: %w", err)
		}

		if err := tx.Model(&model).Updates(map[string]any{"used_at": now, "used_by": uid}).Error; err != nil {
			return fmt.Errorf("consumir invitación: %w", err)
		}

		inv := invitationRow{InvitationModel: model}.toDomain()
		accepted = &inv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return accepted, nil
}
