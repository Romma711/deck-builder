// Package deck implementa el caso de uso de mazos multi-TCG: creación,
// consulta, actualización y borrado con verificación de propiedad.
//
// La validación específica de cada juego está DELEGADA a los GameSpec
// (patrón Strategy, ver internal/app/gamespec). Este servicio solo cubre lo
// genérico: formato existente, tamaño contra los límites del formato,
// presencia de líder si el formato lo exige, y persistencia del catálogo.
package deck

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Romma711/deck-builder/internal/app/gamespec"
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

type Service struct {
	decks   ports.DeckRepository
	cards   ports.CardRepository
	games   ports.GameRepository
	members ports.MembershipRepository // tenancy: verificar membresía de org
	specs   *gamespec.Registry
	now     func() time.Time // inyectable para tests
}

func NewService(
	decks ports.DeckRepository,
	cards ports.CardRepository,
	games ports.GameRepository,
	members ports.MembershipRepository,
	specs *gamespec.Registry,
) *Service {
	return &Service{decks: decks, cards: cards, games: games, members: members, specs: specs,
		now: func() time.Time { return time.Now().UTC() }}
}

// DeckInput es el payload de creación/actualización de un mazo.
type DeckInput struct {
	Name        string
	Description string
	DeckSize    int
	GameCode    string
	Format      string
	OrgID       string // "" = mazo personal
	IsPrivate   bool   // privado: solo dueño (y miembros de su org)
	Leader      *domain.Card
	Partner     *domain.Card
	Cards       []domain.CardEntry
}

// Create persiste un mazo nuevo owned por ownerID.
func (s *Service) Create(ctx context.Context, ownerID string, in DeckInput) (*domain.Deck, error) {
	if ownerID == "" {
		return nil, domain.ErrUnauthorized
	}

	deck, err := s.validate(ctx, ownerID, in)
	if err != nil {
		return nil, err
	}
	deck.OwnerID = ownerID

	// Normalizamos las cartas al catálogo (asigna IDs reales).
	if err := s.resolveCards(ctx, deck); err != nil {
		return nil, err
	}

	if err := s.decks.Create(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// GetByID devuelve el mazo completo con sus cartas (lectura pública).
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Deck, error) {
	return s.decks.FindByID(ctx, id)
}

// List aplica filtros de búsqueda (sin el array de cartas, proyección liviana).
func (s *Service) List(ctx context.Context, filter ports.DeckFilter) ([]domain.Deck, error) {
	filter.Format = strings.ToLower(strings.TrimSpace(filter.Format))
	filter.GameCode = strings.ToLower(strings.TrimSpace(filter.GameCode))
	return s.decks.Find(ctx, filter)
}

// Update modifica un mazo. Si el mazo existe pero pertenece a otro usuario
// responde ErrForbidden (las lecturas son públicas, así que ocultar la
// existencia no aporta nada).
func (s *Service) Update(ctx context.Context, id, ownerID string, in DeckInput) (*domain.Deck, error) {
	existing, err := s.decks.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != ownerID {
		return nil, fmt.Errorf("%w: el mazo pertenece a otro usuario", domain.ErrForbidden)
	}

	deck, err := s.validate(ctx, ownerID, in)
	if err != nil {
		return nil, err
	}
	deck.ID = existing.ID
	deck.OwnerID = ownerID
	deck.CreatedAt = existing.CreatedAt
	deck.UpdatedAt = s.now()

	if err := s.resolveCards(ctx, deck); err != nil {
		return nil, err
	}

	if err := s.decks.Update(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// Delete elimina un mazo (solo su dueño).
func (s *Service) Delete(ctx context.Context, id, ownerID string) error {
	existing, err := s.decks.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.OwnerID != ownerID {
		return fmt.Errorf("%w: el mazo pertenece a otro usuario", domain.ErrForbidden)
	}
	return s.decks.Delete(ctx, id)
}

// --- validación genérica + delegación a GameSpec ---

func (s *Service) validate(ctx context.Context, ownerID string, in DeckInput) (*domain.Deck, error) {
	in.GameCode = strings.ToLower(strings.TrimSpace(in.GameCode))
	in.Format = strings.ToLower(strings.TrimSpace(in.Format))
	in.OrgID = strings.TrimSpace(in.OrgID)

	details := map[string]string{}
	if strings.TrimSpace(in.Name) == "" {
		details["Name"] = "required"
	} else if len(in.Name) < 3 || len(in.Name) > 50 {
		details["Name"] = "min=3,max=50"
	}
	if len(in.Description) > 300 {
		details["Description"] = "max=300"
	}
	if len(in.Cards) == 0 {
		details["Cards"] = "required"
	}
	if len(details) > 0 {
		return nil, &domain.ValidationError{Details: details}
	}

	// Tenancy: un mazo de org requiere membership activa en esa org.
	// (RLS es la red de seguridad en BD; aquí damos el error claro 403.)
	if in.OrgID != "" {
		if _, err := s.members.FindByOrgAndUser(ctx, in.OrgID, ownerID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("%w: no eres miembro de la organización del mazo", domain.ErrForbidden)
			}
			return nil, err // ErrInvalidID si el org_id es basura → 400
		}
	}

	// Juego soportado (catálogo persistido).
	game, err := s.games.FindByCode(ctx, in.GameCode)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, &domain.ValidationError{
				Details: map[string]string{"Game": "unsupported"}, Message: "juego no soportado: " + in.GameCode,
			}
		}
		return nil, err
	}

	// Formato existente dentro del juego.
	format := game.FormatByCode(in.Format)
	if format == nil {
		return nil, &domain.ValidationError{
			Details: map[string]string{"Format": "unsupported_for_game"},
			Message: fmt.Sprintf("formato %q no existe en %s", in.Format, game.Code),
		}
	}

	// Tamaño contra los límites del formato.
	if !format.AllowsSize(in.DeckSize) {
		return nil, &domain.ValidationError{
			Details: map[string]string{
				"DeckSize": fmt.Sprintf("format_range=[%d,%d]", format.MinSize, format.MaxSize),
			},
		}
	}

	// Líder obligatorio según el formato.
	if format.RequiresLeader && in.Leader == nil {
		return nil, &domain.ValidationError{
			Details: map[string]string{"Leader": "required_for_" + format.Code},
		}
	}

	// Estrategia del juego (Strategy): atributos de cartas y reglas de mazo.
	spec, ok := s.specs.Spec(game.Code)
	if !ok {
		return nil, fmt.Errorf("%w: sin GameSpec para el juego %q", domain.ErrInternal, game.Code)
	}

	deck := &domain.Deck{
		GameID:      game.ID,
		GameCode:    game.Code,
		OrgID:       in.OrgID,
		IsPrivate:   in.IsPrivate,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		DeckSize:    in.DeckSize,
		Format:      format.Code,
		LeaderCard:  in.Leader,
		PartnerCard: in.Partner,
		Cards:       in.Cards,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}

	// Valida TODAS las cartas (líder, partner y mazo) contra el spec.
	if err := validateAllCards(spec, deck); err != nil {
		return nil, err
	}

	// Reglas de mazo propias del juego (copias, líder, etc.).
	if err := spec.ValidateDeck(deck, format); err != nil {
		return nil, err
	}

	return deck, nil
}

func validateAllCards(spec ports.GameSpec, deck *domain.Deck) error {
	collector := newCardErrorCollector()

	if deck.LeaderCard != nil {
		collector.collect("leader", spec.ValidateCard(deck.LeaderCard))
	}
	if deck.PartnerCard != nil {
		collector.collect("partner", spec.ValidateCard(deck.PartnerCard))
	}
	for i, entry := range deck.Cards {
		collector.collect(fmt.Sprintf("cards[%d]", i), spec.ValidateCard(&entry.Card))
	}
	return collector.fail()
}

// cardErrorCollector fusiona los errores de muchas cartas en un único
// *domain.ValidationError con prefijo de índice de carta.
type cardErrorCollector struct {
	details map[string]string
}

func newCardErrorCollector() *cardErrorCollector {
	return &cardErrorCollector{details: map[string]string{}}
}

func (c *cardErrorCollector) collect(prefix string, err error) {
	if err == nil {
		return
	}
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		for field, reason := range ve.Details {
			c.details[prefix+"."+field] = reason
		}
		return
	}
	c.details[prefix] = err.Error()
}

func (c *cardErrorCollector) fail() error {
	if len(c.details) == 0 {
		return nil
	}
	return &domain.ValidationError{Details: c.details}
}

// --- persistencia del catálogo de cartas ---

// resolveCards hace upsert de cada carta al catálogo (asignando IDs) antes
// de persistir el mazo. Deduplica entradas repetidas del payload sumando
// cantidades.
func (s *Service) resolveCards(ctx context.Context, deck *domain.Deck) error {
	// Todas las cartas del payload se catalogan bajo el juego del mazo.
	if deck.LeaderCard != nil {
		deck.LeaderCard.GameID = deck.GameID
		resolved, err := s.cards.Upsert(ctx, deck.LeaderCard)
		if err != nil {
			return err
		}
		deck.LeaderCard = resolved
	}
	if deck.PartnerCard != nil {
		deck.PartnerCard.GameID = deck.GameID
		resolved, err := s.cards.Upsert(ctx, deck.PartnerCard)
		if err != nil {
			return err
		}
		deck.PartnerCard = resolved
	}

	// Deduplicamos por identidad de catálogo (nombre + set) sumando cantidad.
	type key struct{ name, serialSet string }
	merged := map[key]*domain.CardEntry{}
	order := []key{}

	for i := range deck.Cards {
		entry := &deck.Cards[i]
		entry.Card.GameID = deck.GameID
		resolved, err := s.cards.Upsert(ctx, &entry.Card)
		if err != nil {
			return err
		}
		entry.Card = *resolved

		k := key{resolved.Name, resolved.SerialSet}
		if existing, exists := merged[k]; exists {
			existing.Quantity += entry.Quantity
			continue
		}
		merged[k] = entry
		order = append(order, k)
	}

	deduped := make([]domain.CardEntry, 0, len(order))
	for _, k := range order {
		deduped = append(deduped, *merged[k])
	}
	deck.Cards = deduped
	return nil
}
