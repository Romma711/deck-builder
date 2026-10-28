package postgres

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Modelos GORM: separados de las entidades de dominio para no acoplar el
// corazón del negocio al ORM. Los converters (toDomain/fromDomain) viven en
// cada repositorio.

// UserModel mapea la tabla users.
type UserModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username     string    `gorm:"type:varchar(30);uniqueIndex;not null"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserModel) TableName() string { return "users" }

// RefreshTokenModel mapea la tabla refresh_tokens.
type RefreshTokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	TokenHash string    `gorm:"type:char(64);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (RefreshTokenModel) TableName() string { return "refresh_tokens" }

// GameModel mapea la tabla games (catálogo de TCGs).
type GameModel struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code   string    `gorm:"type:varchar(20);uniqueIndex;not null"`
	Name   string    `gorm:"type:varchar(100);not null"`
	Active bool      `gorm:"not null;default:true"`
}

func (GameModel) TableName() string { return "games" }

// FormatModel mapea la tabla formats (formatos por juego).
type FormatModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	GameID         uuid.UUID `gorm:"type:uuid;index;not null"`
	Code           string    `gorm:"type:varchar(30);not null"`
	Name           string    `gorm:"type:varchar(100);not null"`
	DeckSize       int       `gorm:"column:deck_size"`
	MinSize        int       `gorm:"column:min_size"`
	MaxSize        int       `gorm:"column:max_size"`
	RequiresLeader bool      `gorm:"column:requires_leader;not null;default:false"`
}

func (FormatModel) TableName() string { return "formats" }

// CardModel mapea la tabla cards: columnas comunes + attributes JSONB.
type CardModel struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	GameID     uuid.UUID      `gorm:"type:uuid;index;not null"`
	Name       string         `gorm:"type:varchar(100);not null"`
	SerialSet  string         `gorm:"column:serial_set;type:varchar(50);not null"`
	Image      string         `gorm:"type:text;not null"`
	Attributes datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (CardModel) TableName() string { return "cards" }

// OrganizationModel mapea la tabla organizations.
type OrganizationModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Slug      string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Name      string    `gorm:"type:varchar(100);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (OrganizationModel) TableName() string { return "organizations" }

// MembershipModel mapea la tabla memberships (PK compuesta org_id+user_id).
type MembershipModel struct {
	OrgID     uuid.UUID `gorm:"type:uuid;primaryKey;column:org_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	Role      string    `gorm:"type:varchar(10);not null"`
	CreatedAt time.Time
}

func (MembershipModel) TableName() string { return "memberships" }

// DeckModel mapea la tabla decks con referencias al catálogo de cartas y al
// tenancy de la Fase 5 (org_id + is_private; RLS lo aplica en la BD).
type DeckModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	OwnerID       uuid.UUID  `gorm:"type:uuid;index;not null"`
	GameID        uuid.UUID  `gorm:"type:uuid;index;not null"`
	OrgID         *uuid.UUID `gorm:"type:uuid;column:org_id;index"`
	IsPrivate     bool       `gorm:"column:is_private;not null;default:false"`
	Name          string     `gorm:"type:varchar(50);not null"`
	Description   string     `gorm:"type:varchar(300);not null;default:''"`
	DeckSize      int        `gorm:"column:deck_size;not null"`
	Format        string     `gorm:"type:varchar(30);index;not null"`
	LeaderCardID  *uuid.UUID `gorm:"type:uuid;column:leader_card_id"`
	PartnerCardID *uuid.UUID `gorm:"type:uuid;column:partner_card_id"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DeckModel) TableName() string { return "decks" }

// DeckCardModel mapea la tabla puente deck_cards.
type DeckCardModel struct {
	DeckID   uuid.UUID `gorm:"type:uuid;primaryKey;column:deck_id"`
	CardID   uuid.UUID `gorm:"type:uuid;primaryKey;column:card_id"`
	Quantity int       `gorm:"not null"`
}

func (DeckCardModel) TableName() string { return "deck_cards" }
