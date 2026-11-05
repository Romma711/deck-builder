package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CardRepository adapta ports.CardRepository al catálogo normalizado.
type CardRepository struct {
	db *gorm.DB
}

var _ ports.CardRepository = (*CardRepository)(nil)

func NewCardRepository(db *gorm.DB) *CardRepository {
	return &CardRepository{db: db}
}

// Upsert: identidad = (game_id, name, serial_set). Si existe, devuelve el
// registro persistido SIN pisar los atributos (el catálogo es autoritativo).
func (r *CardRepository) Upsert(ctx context.Context, card *domain.Card) (*domain.Card, error) {
	gameID, err := toUUID(card.GameID)
	if err != nil {
		return nil, err
	}

	attrs, err := marshalAttributes(card.Attributes)
	if err != nil {
		return nil, err
	}

	model := CardModel{
		ID:         uuid.Must(uuid.NewV7()),
		GameID:     gameID,
		Name:       card.Name,
		SerialSet:  card.SerialSet,
		Image:      card.Image,
		Attributes: attrs,
	}

	// ON CONFLICT sobre la identidad: si ya existe, no hacemos nada y
	// devolvemos la fila existente.
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "game_id"}, {Name: "name"}, {Name: "serial_set"}},
		DoNothing: true,
	}).Create(&model).Error
	if err != nil {
		return nil, fmt.Errorf("upsert de carta: %w", err)
	}

	// Recuperamos la fila real (puede ser la existente, con su ID original).
	var persisted CardModel
	if err := r.db.WithContext(ctx).
		Where("game_id = ? AND name = ? AND serial_set = ?", gameID, card.Name, card.SerialSet).
		First(&persisted).Error; err != nil {
		return nil, fmt.Errorf("recuperar carta tras upsert: %w", err)
	}

	return cardModelToDomain(persisted)
}

func (r *CardRepository) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	cid, err := toUUID(id)
	if err != nil {
		return nil, err
	}

	var model CardModel
	dbErr := r.db.WithContext(ctx).First(&model, "id = ?", cid).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar carta: %w", dbErr)
	}
	return cardModelToDomain(model)
}

func (r *CardRepository) Find(ctx context.Context, filter ports.CardFilter) ([]domain.Card, int, error) {
	query := r.db.WithContext(ctx).Model(&CardModel{})

	if filter.GameCode != "" {
		query = query.Where("game_id = (SELECT id FROM games WHERE code = ?)", filter.GameCode)
	}
	if filter.Query != "" {
		// Búsqueda por subcadena de nombre, case-insensitive.
		query = query.Where("name ILIKE ?", "%"+escapeLike(filter.Query)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("contar cartas: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var models []CardModel
	if err := query.
		Order("name, serial_set").
		Limit(limit).Offset(filter.Offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("listar cartas: %w", err)
	}

	cards := make([]domain.Card, 0, len(models))
	for _, model := range models {
		card, err := cardModelToDomain(model)
		if err != nil {
			return nil, 0, err
		}
		cards = append(cards, *card)
	}
	return cards, int(total), nil
}

// --- converters ---

func cardModelToDomain(model CardModel) (*domain.Card, error) {
	attrs, err := unmarshalAttributes(model.Attributes)
	if err != nil {
		return nil, err
	}
	return &domain.Card{
		ID:         model.ID.String(),
		GameID:     model.GameID.String(),
		Name:       model.Name,
		SerialSet:  model.SerialSet,
		Image:      model.Image,
		Attributes: attrs,
	}, nil
}

func escapeLike(s string) string {
	// Escapamos los caracteres comodín para que la búsqueda sea literal.
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
