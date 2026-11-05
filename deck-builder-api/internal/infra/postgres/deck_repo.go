package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DeckRepository adapta ports.DeckRepository a PostgreSQL sobre el esquema
// normalizado: decks con referencias al catálogo (game_id, leader_card_id,
// partner_card_id), tenancy de la Fase 5 (org_id, is_private) y la tabla
// puente deck_cards.
//
// TODA operación corre dentro de una transacción con SET LOCAL app.user_id:
// así las políticas de Row-Level Security de la BD saben quién consulta
// (vacío = anónimo). Sin eso, el pool de conexiones dejaría el valor pegado
// a la conexión siguiente (fuga de identidad entre requests).
type DeckRepository struct {
	db *gorm.DB
}

var _ ports.DeckRepository = (*DeckRepository)(nil)

func NewDeckRepository(db *gorm.DB) *DeckRepository {
	return &DeckRepository{db: db}
}

// withRLS ejecuta fn en una transacción con la identidad del usuario
// seteada para las políticas RLS. SET LOCAL vuelve al valor anterior (vacío)
// al hacer commit/rollback: ninguna sesión queda contaminada.
func (r *DeckRepository) withRLS(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// set_config no admite NULL: "" se traduce a "sin usuario" vía
		// NULLIF dentro de las políticas.
		if err := tx.Exec(
			"SELECT set_config('app.user_id', ?, true)",
			domain.UserIDFromContext(ctx),
		).Error; err != nil {
			return fmt.Errorf("%w: setear identidad RLS: %v", domain.ErrInternal, err)
		}
		return fn(tx)
	})
}

// Create persiste el mazo y sus cartas en una transacción. Las cartas deben
// venir ya resueltas al catálogo (con ID) — el caso de uso hace el upsert.
func (r *DeckRepository) Create(ctx context.Context, deck *domain.Deck) error {
	ownerID, err := toUUID(deck.OwnerID)
	if err != nil {
		return err
	}
	gameID, err := toUUID(deck.GameID)
	if err != nil {
		return err
	}
	orgID, err := optionalID(deck.OrgID)
	if err != nil {
		return err
	}
	leaderID, err := optionalUUID(deck.LeaderCard)
	if err != nil {
		return err
	}
	partnerID, err := optionalUUID(deck.PartnerCard)
	if err != nil {
		return err
	}

	model := DeckModel{
		ID:            uuid.Must(uuid.NewV7()),
		OwnerID:       ownerID,
		GameID:        gameID,
		OrgID:         orgID,
		IsPrivate:     deck.IsPrivate,
		Name:          deck.Name,
		Description:   deck.Description,
		DeckSize:      deck.DeckSize,
		Format:        deck.Format,
		LeaderCardID:  leaderID,
		PartnerCardID: partnerID,
		CreatedAt:     deck.CreatedAt,
		UpdatedAt:     deck.UpdatedAt,
	}
	entries, err := deckCardModels(model.ID, deck.Cards)
	if err != nil {
		return err
	}

	err = r.withRLS(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return fmt.Errorf("crear mazo: %w", err)
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return fmt.Errorf("crear cartas del mazo: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	deck.ID = model.ID.String()
	return nil
}

// FindByID carga el mazo completo con líder/partner y sus cartas. Devuelve
// domain.ErrNotFound si RLS no permite verlo (privado ajeno = no existe).
func (r *DeckRepository) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	did, err := toUUID(id)
	if err != nil {
		return nil, err
	}

	var deck *domain.Deck
	err = r.withRLS(ctx, func(tx *gorm.DB) error {
		var row deckWithGame
		dbErr := tx.
			Select("decks.*, games.code AS game_code").
			Joins("JOIN games ON games.id = decks.game_id").
			Where("decks.id = ?", did).
			First(&row).Error
		if dbErr != nil {
			if errors.Is(dbErr, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("buscar mazo: %w", dbErr)
		}

		loaded, err := row.toDeck()
		if err != nil {
			return err
		}
		if err := loadLeaders(tx, []*domain.Deck{loaded}); err != nil {
			return err
		}
		if err := loadCards(tx, loaded); err != nil {
			return err
		}
		deck = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return deck, nil
}

// Find lista mazos con filtros. Proyección liviana: incluye líder/partner
// (resueltos en lote) pero no el array completo de cartas. La visibilidad
// (público/propio/org) la aplica RLS: un privado ajeno jamás aparece.
func (r *DeckRepository) Find(ctx context.Context, filter ports.DeckFilter) ([]domain.Deck, error) {
	// Validamos IDs ANTES de transaccionar (error claro sin abrir tx).
	if filter.OwnerID != "" {
		if _, err := toUUID(filter.OwnerID); err != nil {
			return nil, err
		}
	}
	if filter.OrgID != "" {
		if _, err := toUUID(filter.OrgID); err != nil {
			return nil, err
		}
	}

	var decks []domain.Deck
	err := r.withRLS(ctx, func(tx *gorm.DB) error {
		query := tx.
			Select("decks.*, games.code AS game_code").
			Joins("JOIN games ON games.id = decks.game_id")

		if filter.OwnerID != "" {
			ownerID, _ := toUUID(filter.OwnerID)
			query = query.Where("decks.owner_id = ?", ownerID)
		}
		if filter.OrgID != "" {
			orgID, _ := toUUID(filter.OrgID)
			query = query.Where("decks.org_id = ?", orgID)
		}
		if filter.GameCode != "" {
			query = query.Where("decks.game_id = (SELECT id FROM games WHERE code = ?)", filter.GameCode)
		}
		if filter.Format != "" {
			query = query.Where("decks.format = ?", filter.Format)
		}
		if filter.LeaderName != "" {
			query = query.Where(
				"EXISTS (SELECT 1 FROM cards c WHERE c.id = decks.leader_card_id AND c.name = ?)",
				filter.LeaderName)
		}
		query = query.Order("decks.created_at DESC")

		var rows []deckWithGame
		if err := query.Find(&rows).Error; err != nil {
			return fmt.Errorf("listar mazos: %w", err)
		}

		decks = make([]domain.Deck, 0, len(rows))
		for _, row := range rows {
			loaded, err := row.toDeck()
			if err != nil {
				return err
			}
			decks = append(decks, *loaded)
		}
		// Punteros a los elementos del slice para mutarlos en loadLeaders.
		pointers := make([]*domain.Deck, 0, len(decks))
		for i := range decks {
			pointers = append(pointers, &decks[i])
		}
		return loadLeaders(tx, pointers)
	})
	if err != nil {
		return nil, err
	}
	return decks, nil
}

// Update reemplaza los datos del mazo y su lista de cartas en una transacción.
// RLS garantiza que solo el dueño actual alcance la fila (y no se reapropie
// cambiando owner_id, que ni siquiera se actualiza).
func (r *DeckRepository) Update(ctx context.Context, deck *domain.Deck) error {
	did, err := toUUID(deck.ID)
	if err != nil {
		return err
	}
	gameID, err := toUUID(deck.GameID)
	if err != nil {
		return err
	}
	orgID, err := optionalID(deck.OrgID)
	if err != nil {
		return err
	}
	leaderID, err := optionalUUID(deck.LeaderCard)
	if err != nil {
		return err
	}
	partnerID, err := optionalUUID(deck.PartnerCard)
	if err != nil {
		return err
	}
	entries, err := deckCardModels(did, deck.Cards)
	if err != nil {
		return err
	}

	return r.withRLS(ctx, func(tx *gorm.DB) error {
		res := tx.Model(&DeckModel{}).Where("id = ?", did).Updates(map[string]any{
			"game_id":         gameID,
			"org_id":          orgID,
			"is_private":      deck.IsPrivate,
			"name":            deck.Name,
			"description":     deck.Description,
			"deck_size":       deck.DeckSize,
			"format":          deck.Format,
			"leader_card_id":  leaderID,
			"partner_card_id": partnerID,
			"updated_at":      deck.UpdatedAt,
		})
		if res.Error != nil {
			return fmt.Errorf("actualizar mazo: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			// No existe O RLS no lo deja ver (privado ajeno: mismo 404).
			return domain.ErrNotFound
		}

		if err := tx.Where("deck_id = ?", did).Delete(&DeckCardModel{}).Error; err != nil {
			return fmt.Errorf("limpiar cartas del mazo: %w", err)
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return fmt.Errorf("guardar cartas del mazo: %w", err)
			}
		}
		return nil
	})
}

func (r *DeckRepository) Delete(ctx context.Context, id string) error {
	did, err := toUUID(id)
	if err != nil {
		return err
	}

	return r.withRLS(ctx, func(tx *gorm.DB) error {
		// deck_cards se elimina en cascada (y también por políticas RLS).
		res := tx.Delete(&DeckModel{}, "id = ?", did)
		if res.Error != nil {
			return fmt.Errorf("eliminar mazo: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// --- loaders (evitan N+1 con consultas en lote; corren DENTRO de la tx RLS) ---

// loadLeaders resuelve leader/partner de N mazos en un solo query. Acepta
// punteros para mutar los mazos originales (no copias).
func loadLeaders(tx *gorm.DB, decks []*domain.Deck) error {
	// Los mazos traen el ID de la carta en LeaderCard.ID/PartnerCard.ID
	// (resuelto desde la fila); recogemos todos y cargamos en un solo query.
	idSet := map[uuid.UUID]bool{}
	for _, d := range decks {
		for _, card := range []*domain.Card{d.LeaderCard, d.PartnerCard} {
			if card == nil || card.ID == "" {
				continue
			}
			if cid, err := uuid.Parse(card.ID); err == nil {
				idSet[cid] = true
			}
		}
	}
	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	var models []CardModel
	if err := tx.Where("id IN ?", ids).Find(&models).Error; err != nil {
		return fmt.Errorf("cargar líderes: %w", err)
	}
	byID := make(map[string]domain.Card, len(models))
	for _, m := range models {
		card, err := cardModelToDomain(m)
		if err != nil {
			return err
		}
		byID[card.ID] = *card
	}

	for _, d := range decks {
		if d.LeaderCard != nil {
			if full, ok := byID[d.LeaderCard.ID]; ok {
				d.LeaderCard = &full
			}
		}
		if d.PartnerCard != nil {
			if full, ok := byID[d.PartnerCard.ID]; ok {
				d.PartnerCard = &full
			}
		}
	}
	return nil
}

// loadCards carga el array completo de cartas del mazo (join con el catálogo).
// RLS en deck_cards hereda la visibilidad del mazo.
func loadCards(tx *gorm.DB, deck *domain.Deck) error {
	did, err := uuid.Parse(deck.ID)
	if err != nil {
		return fmt.Errorf("%w: %s", domain.ErrInvalidID, deck.ID)
	}

	// Struct plano para el join deck_cards → cards (GORM necesita columnas
	// explícitas cuando la tabla principal no coincide con el modelo).
	type row struct {
		ID         string         `gorm:"column:id"`
		GameID     string         `gorm:"column:game_id"`
		Name       string         `gorm:"column:name"`
		SerialSet  string         `gorm:"column:serial_set"`
		Image      string         `gorm:"column:image"`
		Attributes datatypes.JSON `gorm:"column:attributes"`
		Quantity   int            `gorm:"column:quantity"`
	}
	var rows []row
	if err := tx.
		Table("deck_cards").
		Select("c.*, deck_cards.quantity").
		Joins("JOIN cards AS c ON c.id = deck_cards.card_id").
		Where("deck_cards.deck_id = ?", did).
		Order("c.name").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("cargar cartas del mazo: %w", err)
	}

	entries := make([]domain.CardEntry, 0, len(rows))
	for _, rw := range rows {
		attrs, err := unmarshalAttributes(rw.Attributes)
		if err != nil {
			return err
		}
		entries = append(entries, domain.CardEntry{
			Card: domain.Card{
				ID:         rw.ID,
				GameID:     rw.GameID,
				Name:       rw.Name,
				SerialSet:  rw.SerialSet,
				Image:      rw.Image,
				Attributes: attrs,
			},
			Quantity: rw.Quantity,
		})
	}
	deck.Cards = entries
	return nil
}

// --- converters ---

// deckWithGame extiende la fila del mazo con el código del juego (JOIN).
type deckWithGame struct {
	DeckModel
	GameCode string `gorm:"column:game_code"`
}

func (row deckWithGame) toDeck() (*domain.Deck, error) {
	deck := &domain.Deck{
		ID:          row.ID.String(),
		OwnerID:     row.OwnerID.String(),
		GameID:      row.GameID.String(),
		GameCode:    row.GameCode,
		IsPrivate:   row.IsPrivate,
		Name:        row.Name,
		Description: row.Description,
		DeckSize:    row.DeckSize,
		Format:      row.Format,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.OrgID != nil {
		deck.OrgID = row.OrgID.String()
	}
	// Líder/partner vienen solo con su ID desde la fila; loadLeaders los
	// resuelve contra el catálogo en lote.
	if row.LeaderCardID != nil {
		deck.LeaderCard = &domain.Card{ID: row.LeaderCardID.String(), GameID: deck.GameID}
	}
	if row.PartnerCardID != nil {
		deck.PartnerCard = &domain.Card{ID: row.PartnerCardID.String(), GameID: deck.GameID}
	}
	return deck, nil
}

func optionalUUID(card *domain.Card) (*uuid.UUID, error) {
	if card == nil || card.ID == "" {
		return nil, nil
	}
	id, err := toUUID(card.ID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func optionalID(id string) (*uuid.UUID, error) {
	if id == "" {
		return nil, nil
	}
	parsed, err := toUUID(id)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func deckCardModels(deckID uuid.UUID, entries []domain.CardEntry) ([]DeckCardModel, error) {
	models := make([]DeckCardModel, 0, len(entries))
	for _, entry := range entries {
		cid, err := toUUID(entry.Card.ID)
		if err != nil {
			return nil, err
		}
		models = append(models, DeckCardModel{
			DeckID:   deckID,
			CardID:   cid,
			Quantity: entry.Quantity,
		})
	}
	return models, nil
}
